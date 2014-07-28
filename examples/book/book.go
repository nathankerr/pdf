package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/nathankerr/pdf"
	"log"
	"math"
	"os"
	"reflect"
)

func usage() {
	fmt.Printf("Usage: book [options] <file.pdf>\n\nOptions:\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	log.SetFlags(log.Lshortfile)

	binding := flag.String("binding", "chapbook", "Type of binding to generate {perfect, chapbook, none}. Default is chapbook.")
	flag.Parse()

	switch *binding {
	case "chapbook", "perfect", "none":
		// no-op
	default:
		usage()
	}

	// Process arguments
	if flag.NArg() != 1 {
		usage()
	}
	filename := flag.Arg(0)

	// open pdf document
	book, err := pdf.Open(filename)
	if err != nil {
		log.Fatalln(err)
	}
	defer book.Close()

	// get the pdf page references
	catalog := book.Get(book.Root).(pdf.Dictionary)
	pages := getPages(book, catalog["Pages"].(pdf.ObjectReference))

	// assuming that all pages are the same size, figure out the
	// media box that will be the bbox of the xobject
	media_box_obj := pages[0]["MediaBox"]
	var media_box pdf.Array
	if media_box_obj == nil {
		// the first page inherits its MediaBox, therefore get it from the root
		pages_ref := catalog["Pages"].(pdf.ObjectReference)
		pages := book.Get(pages_ref)
		media_box = pages.(pdf.Dictionary)["MediaBox"].(pdf.Array)
	} else {
		media_box = media_box_obj.(pdf.Array)
	}

	// change the pages to xobjects
	page_xobjects := []pdf.ObjectReference{}
	for _, page := range pages {
		page["Type"] = pdf.Name("XObject")
		page["Subtype"] = pdf.Name("Form")
		page["BBox"] = media_box

		// consolidate the contents into the xobject stream
		contents := []byte{}
		switch typed := page["Contents"].(type) {
		case pdf.ObjectReference:
			page_contents := book.Get(typed).(pdf.Stream)
			contents = page_contents.Stream
			page["Filter"] = page_contents.Dictionary["Filter"]
		case pdf.Array:
			if len(typed) == 1 {
				page_contents := book.Get(typed[0].(pdf.ObjectReference)).(pdf.Stream)
				contents = page_contents.Stream
				page["Filter"] = page_contents.Dictionary["Filter"]
			} else {
				for _, page_contents_ref := range typed {
					page_contents := book.Get(page_contents_ref.(pdf.ObjectReference)).(pdf.Stream)
					decoded, err := page_contents.Decode()
					if err != nil {
						log.Fatalln(err)
					}

					contents = append(contents, decoded...)
				}
			}
		default:
			panic(reflect.TypeOf(typed).Name())
		}
		delete(page, "Contents")

		// add the xobject to the pdf
		xobj_ref, err := book.Add(pdf.Stream{
			Dictionary: page,
			Stream:     contents,
		})
		if err != nil {
			log.Fatalln(err)
		}
		page_xobjects = append(page_xobjects, xobj_ref)
	}

	// figure out how many pages to layout for
	num_document_pages := len(pages)
	num_pages_to_layout := num_document_pages
	switch *binding {
	case "perfect", "chapbook":
		if (num_pages_to_layout % 4) != 0 {
			num_pages_to_layout = num_document_pages + (4 - (num_document_pages % 4))
		}
	case "none":
		num_document_pages++
		if (num_pages_to_layout % 2) != 0 {
			num_pages_to_layout++
		}
	}

	// layout on landscape version of page size
	paper_height := toFloat64(media_box[3])      // same height as the original page
	paper_width := toFloat64(media_box[2]) * 2.0 // twice the width of the original page

	// layout the pages
	layed_out_pages := pdf.Array{}
	stream := &bytes.Buffer{}
	xobjects := pdf.Dictionary{}
	show_page := false
	flip_next_page := true
	for page_to_layout := 0; page_to_layout < num_pages_to_layout; page_to_layout++ {
		var page_num int
		switch *binding {
		case "perfect":
			// determine the real page number for perfect bound books
			page_num = page_to_layout - 1
			if page_to_layout%4 == 0 {
				page_num += 4
			}
		case "chapbook":
			// determine the real page number for chapbooks
			page_num = page_to_layout / 2
			if page_to_layout%2 == 1 {
				page_num = num_pages_to_layout - page_num - 1
			}
		case "none":
			page_num = page_to_layout - 1
			flip_next_page = false
		default:
			log.Println("unhandled binding:", *binding)
			usage()
		}

		// only render non-blank pages
		if page_num < num_document_pages && page_num >= 0 {
			fmt.Fprintf(stream, "q ")
			// horizontal offset for recto (odd) pages
			// this correctly handles 0 based indexes for 1 based page numbers
			if page_num%2 == 0 {
				fmt.Fprintf(stream, "1 0 0 1 %v %v cm ", paper_width/2.0, 0)
			}

			// render the page
			page_name := fmt.Sprintf("Page%d", page_num)
			xobjects[pdf.Name(page_name)] = page_xobjects[page_num]
			fmt.Fprintf(stream, "/%v Do Q ", page_name)
		}

		// emit layouts after drawing both pages
		if show_page {
			// content for book page
			contents := pdf.Stream{
				Stream: stream.Bytes(),
			}
			contents_ref, err := book.Add(contents)
			if err != nil {
				log.Fatalln(err)
			}

			// add page
			book_page := pdf.Dictionary{
				pdf.Name("Type"):   pdf.Name("Page"),
				pdf.Name("Parent"): catalog["Pages"],
				pdf.Name("Resources"): pdf.Dictionary{
					pdf.Name("XObject"): xobjects,
				},
				pdf.Name("Contents"): contents_ref,
			}
			book_page_ref, err := book.Add(book_page)
			if err != nil {
				log.Fatalln(err)
			}
			layed_out_pages = append(layed_out_pages, book_page_ref)

			// reset the stream and xobjects
			stream = &bytes.Buffer{}
			xobjects = pdf.Dictionary{}

			// flip the next page over
			if flip_next_page {
				fmt.Fprintf(stream, "%f %f %f %f %v %v cm ",
					math.Cos(math.Pi),
					math.Sin(math.Pi),
					-math.Sin(math.Pi),
					math.Cos(math.Pi),
					paper_width,
					paper_height,
				)
			}
			flip_next_page = !flip_next_page
		}
		show_page = !show_page
	}

	// Page tree for book
	book_pages := pdf.Dictionary{
		"Type":  pdf.Name("Pages"),
		"Kids":  layed_out_pages,
		"Count": pdf.Integer(len(layed_out_pages)),
		"MediaBox": pdf.Array{
			pdf.Integer(0),
			pdf.Integer(0),
			pdf.Real(paper_width),  // width
			pdf.Real(paper_height), // height
		},
	}
	_, err = book.Add(pdf.IndirectObject{
		ObjectReference: catalog["Pages"].(pdf.ObjectReference),
		Object:          book_pages,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// save
	err = book.Save()
	if err != nil {
		log.Fatalln(err)
	}
}

// transforms the page tree from the file into an array of pages
func getPages(file *pdf.File, ref pdf.ObjectReference) []pdf.Dictionary {
	pages := []pdf.Dictionary{}
	page_node := file.Get(ref).(pdf.Dictionary)

	switch page_node["Type"] {
	case pdf.Name("Pages"):
		for _, kid_ref := range page_node["Kids"].(pdf.Array) {
			kid_pages := getPages(file, kid_ref.(pdf.ObjectReference))
			pages = append(pages, kid_pages...)
		}
	case pdf.Name("Page"):
		pages = append(pages, page_node)
	default:
		panic(string(page_node["Type"].(pdf.Name)))
	}

	return pages
}

// Transforms a pdf.Object into a float64.
// Panics if this is not possible
func toFloat64(obj pdf.Object) float64 {
	switch typed := obj.(type) {
	case pdf.Real:
		return float64(typed)
	case pdf.Integer:
		return float64(typed)
	default:
		panic(reflect.TypeOf(typed).Name())
	}
}
