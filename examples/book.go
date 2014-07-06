package main

import (
	"bytes"
	"fmt"
	"github.com/juju/errgo"
	"github.com/nathankerr/pdf"
	"log"
	"math"
	"os"
	"reflect"
)

func main() {
	log.SetFlags(log.Lshortfile)

	// Process arguments
	if len(os.Args) != 3 {
		log.Fatalln("Usage: chapbook [input.pdf] [output.pdf]")
	}

	input_filename := os.Args[1]
	output_filename := os.Args[2]

	// open pdf document
	chapbook, err := pdf.Open(input_filename)
	if err != nil {
		log.Fatalln(errgo.Details(err))
	}
	defer chapbook.Close()

	// get the pdf page references
	catalog := chapbook.Get(chapbook.Root).(pdf.Dictionary)
	pages := getPages(chapbook, catalog[pdf.Name("Pages")].(pdf.ObjectReference))

	// assuming that all pages are the same size, figure out the
	// media box that will be the bbox of the xobject
	media_box_obj := pages[0].Object.(pdf.Dictionary)[pdf.Name("MediaBox")]
	var media_box pdf.Array
	if media_box_obj == nil {
		// the first page inherits its MediaBox, therefore get it from the root
		pages_ref := catalog[pdf.Name("Pages")].(pdf.ObjectReference)
		pages := chapbook.Get(pages_ref)
		media_box = pages.(pdf.Dictionary)[pdf.Name("MediaBox")].(pdf.Array)
	} else {
		media_box = media_box_obj.(pdf.Array)
	}

	// change the pages to xobjects
	page_xobjects := []pdf.ObjectReference{}
	for _, page := range pages {
		page := page.Object.(pdf.Dictionary)

		// change the dict. values
		page[pdf.Name("Type")] = pdf.Name("XObject")
		page[pdf.Name("Subtype")] = pdf.Name("Form")
		page[pdf.Name("BBox")] = media_box

		// consolidate the contents
		contents := []byte{}
		switch typed := page[pdf.Name("Contents")].(type) {
		case pdf.ObjectReference:
			page_contents_obj := chapbook.Get(typed)
			page_contents := page_contents_obj.(pdf.Stream)
			contents = page_contents.Stream
			page[pdf.Name("Filter")] = page_contents.Dictionary[pdf.Name("Filter")]
		case pdf.Array:
			if len(typed) == 1 {
				page_contents_obj := chapbook.Get(typed[0].(pdf.ObjectReference))
				page_contents := page_contents_obj.(pdf.Stream)
				contents = page_contents.Stream
				page[pdf.Name("Filter")] = page_contents.Dictionary[pdf.Name("Filter")]
			} else {
				for _, page_contents_ref := range typed {
					page_contents_obj := chapbook.Get(page_contents_ref.(pdf.ObjectReference))
					decoded, err := page_contents_obj.(pdf.Stream).Decode()
					if err != nil {
						log.Fatalln(errgo.Details(err))
					}

					contents = append(contents, decoded...)
				}
			}
		default:
			panic(reflect.TypeOf(typed).Name())
		}
		delete(page, pdf.Name("Contents"))

		// add the xobject to the pdf
		xobj_ref, err := chapbook.Add(pdf.Stream{
			Dictionary: page,
			Stream:     contents,
		})
		if err != nil {
			log.Fatalln(errgo.Details(err))
		}
		page_xobjects = append(page_xobjects, xobj_ref)
	}

	// figure out how many pages to layout for
	num_document_pages := len(pages)
	num_pages_to_layout := num_document_pages
	if (num_pages_to_layout % 4) != 0 {
		num_pages_to_layout = num_document_pages + (4 - (num_document_pages % 4))
	}

	// layout on A4 landscape
	paper_height := 595.224
	paper_width := 841.824

	// layout the pages
	layed_out_pages := pdf.Array{}
	stream := &bytes.Buffer{}
	xobjects := pdf.Dictionary{}
	show_page := false
	flip_next_page := true
	for page_to_layout := 0; page_to_layout < num_pages_to_layout; page_to_layout++ {
		// determine the real page number for perfect bound books
		page_num := page_to_layout - 1
		if page_to_layout%4 == 0 {
			page_num += 4
		}
		page_name := fmt.Sprintf("Page%d", page_num)

		// only render non-blank pages
		if page_num < num_document_pages {
			fmt.Fprintf(stream, "1 0 0 1 0 0 cm ")
			// horizontal offset
			// recto pages have odd page numbers
			// this correctly handles 0 based indexes for 1 based page numbers
			if page_num%2 == 0 {
				fmt.Fprintf(stream, "1 0 0 1 %v %v cm ", paper_width/2.0, 0)
			}

			// render the page
			xobjects[pdf.Name(page_name)] = page_xobjects[page_num]
			fmt.Fprintf(stream, "/%v Do ", page_name)
		}

		// emit layouts after drawing both pages
		if show_page {
			// content for chapbook page
			contents := pdf.Stream{
				Stream: stream.Bytes(),
			}
			contents_ref, err := chapbook.Add(contents)
			if err != nil {
				log.Fatalln(errgo.Details(err))
			}

			// add page
			chapbook_page := pdf.Dictionary{
				pdf.Name("Type"):   pdf.Name("Page"),
				pdf.Name("Parent"): catalog[pdf.Name("Pages")],
				pdf.Name("Resources"): pdf.Dictionary{
					pdf.Name("XObject"): xobjects,
				},
				pdf.Name("Contents"): contents_ref,
			}
			chapbook_page_ref, err := chapbook.Add(chapbook_page)
			if err != nil {
				log.Fatalln(errgo.Details(err))
			}
			layed_out_pages = append(layed_out_pages, chapbook_page_ref)

			// rest the stream and xobjects
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

	// Pages for chapbook
	chapbook_pages := pdf.Dictionary{
		pdf.Name("Type"):  pdf.Name("Pages"),
		pdf.Name("Kids"):  layed_out_pages,
		pdf.Name("Count"): pdf.Integer(len(layed_out_pages)),
		pdf.Name("MediaBox"): pdf.Array{
			pdf.Integer(0),
			pdf.Integer(0),
			pdf.Real(paper_width),  // width
			pdf.Real(paper_height), // height
		},
	}
	_, err = chapbook.Add(pdf.IndirectObject{
		ObjectReference: catalog[pdf.Name("Pages")].(pdf.ObjectReference),
		Object:          chapbook_pages,
	})
	if err != nil {
		log.Fatalln(errgo.Details(err))
	}

	// update catalog for chapbook
	_, err = chapbook.Add(pdf.IndirectObject{
		ObjectReference: chapbook.Root,
		Object:          catalog,
	})
	if err != nil {
		log.Fatalln(errgo.Details(err))
	}

	// close files
	err = chapbook.SaveAs(output_filename)
	if err != nil {
		log.Fatalln(errgo.Details(err))
	}
}

// transforms the page tree from the file into an array of pages
func getPages(file *pdf.File, ref pdf.ObjectReference) []pdf.IndirectObject {
	pages := []pdf.IndirectObject{}
	page_node := file.Get(ref).(pdf.Dictionary)

	switch page_node[pdf.Name("Type")] {
	case pdf.Name("Pages"):
		for _, kid_ref := range page_node[pdf.Name("Kids")].(pdf.Array) {
			kid_pages := getPages(file, kid_ref.(pdf.ObjectReference))
			pages = append(pages, kid_pages...)
		}
	case pdf.Name("Page"):
		pages = append(pages, pdf.IndirectObject{
			ObjectReference: ref,
			Object:          page_node,
		})
	default:
		panic(string(page_node[pdf.Name("Type")].(pdf.Name)))
	}

	return pages
}
