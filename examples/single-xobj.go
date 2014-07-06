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
		log.Fatalln("Usage: single-xobj [input.pdf] [output.pdf]")
	}

	input_filename := os.Args[1]
	output_filename := os.Args[2]

	// open pdf document
	single, err := pdf.Open(input_filename)
	if err != nil {
		log.Fatalln(errgo.Details(err))
	}

	// create references to input pages
	catalog := single.Get(single.Root).(pdf.Dictionary)
	pages := getPages(single, catalog[pdf.Name("Pages")].(pdf.ObjectReference))

	// output to A4
	paper_width := 595.224
	paper_height := 841.824

	// assume that all pages are the same size
	media_box_obj := pages[0].Object.(pdf.Dictionary)[pdf.Name("MediaBox")]
	var media_box pdf.Array
	if media_box_obj == nil {
		// the first page inherits its MediaBox, therefore get it from the root
		pages_ref := catalog[pdf.Name("Pages")].(pdf.ObjectReference)
		pages := single.Get(pages_ref)
		media_box = pages.(pdf.Dictionary)[pdf.Name("MediaBox")].(pdf.Array)
	} else {
		media_box = media_box_obj.(pdf.Array)
	}

	var page_width float64
	switch typed := media_box[2].(type) {
	case pdf.Real:
		page_width = float64(typed)
	case pdf.Integer:
		page_width = float64(typed)
	default:
		panic(reflect.TypeOf(typed).Name())
	}

	var page_height float64
	switch typed := media_box[3].(type) {
	case pdf.Real:
		page_height = float64(typed)
	case pdf.Integer:
		page_height = float64(typed)
	default:
		panic(reflect.TypeOf(typed).Name())
	}

	num_pages := len(pages)

	// assuming that all the pages are the same size
	// the sum of the page areas must fit in the paper area
	// paper_area >= scale_factor² * num_pages * page_area
	paper_area := paper_width * paper_height
	page_area := page_width * page_height
	scale_factor := math.Sqrt(paper_area / float64(num_pages) / page_area)
	scaled_page_width := scale_factor * page_width
	nx := int(math.Ceil(paper_width / scaled_page_width))
	ny := num_pages / nx
	for (nx * ny) < num_pages {
		ny++
	}

	// adjust scale_factor to fit the new page count
	scale_factor_width := paper_width / float64(nx) / page_width
	scale_factor_height := paper_height / float64(ny) / page_height
	if scale_factor_width > scale_factor_height {
		scale_factor = scale_factor_height
	} else {
		scale_factor = scale_factor_width
	}

	xobjects := pdf.Dictionary{}
	stream := &bytes.Buffer{} // content stream for the single page

	// move to upper left
	fmt.Fprintf(stream, "1 0 0 1 %v %v cm ", 0, paper_height-(page_height*scale_factor))

	// if the pages won't fill up the paper, center them on the paper
	top_margin := (paper_height - (scale_factor * page_height * float64(ny))) / 2.0
	left_margin := (paper_width - (scale_factor * page_width * float64(nx))) / 2.0
	fmt.Fprintf(stream, "1 0 0 1 %v %v cm ", left_margin, -top_margin)

	// scale the pages
	fmt.Fprintf(stream, "%v 0 0 %v 0 0 cm ", scale_factor, scale_factor)

	for page_num, page := range pages {
		page := page.Object.(pdf.Dictionary)

		// change the dict. values
		page[pdf.Name("Type")] = pdf.Name("XObject")
		page[pdf.Name("Subtype")] = pdf.Name("Form")
		page[pdf.Name("BBox")] = media_box

		// consolidate the contents
		contents := []byte{}
		switch typed := page[pdf.Name("Contents")].(type) {
		case pdf.ObjectReference:
			page_contents_obj := single.Get(typed)
			page_contents := page_contents_obj.(pdf.Stream)
			contents = page_contents.Stream
			page[pdf.Name("Filter")] = page_contents.Dictionary[pdf.Name("Filter")]
		case pdf.Array:
			if len(typed) == 1 {
				page_contents_obj := single.Get(typed[0].(pdf.ObjectReference))
				page_contents := page_contents_obj.(pdf.Stream)
				contents = page_contents.Stream
				page[pdf.Name("Filter")] = page_contents.Dictionary[pdf.Name("Filter")]
			} else {
				for _, page_contents_ref := range typed {
					page_contents_obj := single.Get(page_contents_ref.(pdf.ObjectReference))
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

		// add the xobject to the pdf
		xobj_ref, err := single.Add(pdf.Stream{
			Dictionary: page,
			Stream:     contents,
		})
		if err != nil {
			log.Fatalln(errgo.Details(err))
		}

		// add to the single page
		page_name := fmt.Sprintf("Page%d", page_num)
		xobjects[pdf.Name(page_name)] = xobj_ref

		// draw rectangle around the page
		fmt.Fprintf(stream, "0 0 %v %v re S ", page_width, page_height)

		// draw the page
		stream.WriteString("/" + page_name + " Do ")

		// move to where the next page goes
		if (page_num+1)%nx == 0 {
			// move to first page of next line of pages
			fmt.Fprintf(stream, "1 0 0 1 %v %v cm ", -page_width*float64(nx-1), -page_height)
		} else {
			// next page in same line
			fmt.Fprintf(stream, "1 0 0 1 %v %v cm ", page_width, 0)
		}
	}

	// Pages for single
	single_pages := pdf.Dictionary{
		pdf.Name("Type"): pdf.Name("Pages"),
	}
	single_pages_ref, err := single.Add(single_pages)
	if err != nil {
		log.Fatalln(errgo.Details(err))
	}

	// content for single page
	contents := pdf.Stream{
		Stream: stream.Bytes(),
	}
	contents_ref, err := single.Add(contents)
	if err != nil {
		log.Fatalln(errgo.Details(err))
	}

	// add page
	single_page := pdf.Dictionary{
		pdf.Name("Type"):   pdf.Name("Page"),
		pdf.Name("Parent"): single_pages_ref,
		pdf.Name("Resources"): pdf.Dictionary{
			pdf.Name("XObject"): xobjects,
		},
		pdf.Name("MediaBox"): pdf.Array{
			pdf.Integer(0),
			pdf.Integer(0),
			pdf.Real(paper_width),  // width
			pdf.Real(paper_height), // height
		},
		pdf.Name("Contents"): contents_ref,
	}
	single_page_ref, err := single.Add(single_page)
	if err != nil {
		log.Fatalln(errgo.Details(err))
	}

	// update pages list
	single_pages[pdf.Name("Kids")] = pdf.Array{single_page_ref}
	single_pages[pdf.Name("Count")] = pdf.Integer(1)
	_, err = single.Add(pdf.IndirectObject{
		ObjectReference: single_pages_ref,
		Object:          single_pages,
	})
	if err != nil {
		log.Fatalln(errgo.Details(err))
	}

	// catalog for single
	catalog[pdf.Name("Pages")] = single_pages_ref
	_, err = single.Add(pdf.IndirectObject{
		ObjectReference: single.Root,
		Object:          catalog,
	})
	if err != nil {
		log.Fatalln(errgo.Details(err))
	}

	// close files
	err = single.SaveAs(output_filename)
	if err != nil {
		log.Fatalln(errgo.Details(err))
	}

	err = single.Close()
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
