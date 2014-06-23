package main

import (
	"bytes"
	"fmt"
	"github.com/juju/errgo"
	pdf "github.com/nathankerr/pdf/file"
	"log"
	"os"
	"reflect"
)

func main() {
	log.SetFlags(log.Lshortfile)

	// Process arguments
	if len(os.Args) != 3 {
		log.Fatalln("Usage: single-ref [input.pdf] [output.pdf]")
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

	page_refs := []pdf.ObjectReference{}
	xobjects := pdf.Dictionary{}
	stream := bytes.Buffer{}
	for page_num, page := range pages {
		page := page.Object.(pdf.Dictionary)

		// change the dict. values
		page[pdf.Name("Type")] = pdf.Name("XObject")
		page[pdf.Name("Subtype")] = pdf.Name("Form")
		if bbox, ok := page[pdf.Name("MediaBox")]; ok {
			page[pdf.Name("BBox")] = bbox
		} else {
			page[pdf.Name("BBox")] = pdf.Array{
				pdf.Integer(0),
				pdf.Integer(0),
				pdf.Real(419.53),
				pdf.Real(595.224),
			}
		}

		// consolidate the contents
		contents := []byte{}
		switch typed := page[pdf.Name("Contents")].(type) {
		case pdf.ObjectReference:
			page_contents_obj := single.Get(typed)
			decoded, err := page_contents_obj.(pdf.Stream).Decode()
			if err != nil {
				log.Fatalln(errgo.Details(err))
			}

			contents = append(contents, decoded...)
		case pdf.Array:
			for _, page_contents_ref := range typed {
				log.Println(page_contents_ref)
				page_contents_obj := single.Get(page_contents_ref.(pdf.ObjectReference))
				decoded, err := page_contents_obj.(pdf.Stream).Decode()
				if err != nil {
					log.Fatalln(errgo.Details(err))
				}

				contents = append(contents, decoded...)
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

		page_refs = append(page_refs, xobj_ref)

		page_name := fmt.Sprintf("Page%d", page_num)
		xobjects[pdf.Name(page_name)] = xobj_ref
		stream.WriteString("/" + page_name + " Do")
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
		// Stream: []byte("/Page0 Do"),
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
			// pdf.Name("XObject"): pdf.Dictionary{
			// 	pdf.Name("Page0"): page_refs[0],
			// },
		},
		pdf.Name("MediaBox"): pdf.Array{
			pdf.Integer(0),
			pdf.Integer(0),
			pdf.Real(595.224), // width
			pdf.Real(841.824), // height
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
