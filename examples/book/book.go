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
	pagesRef := book.Get(book.Root).(pdf.Dictionary)["Pages"].(pdf.ObjectReference)
	pages := getPages(book, pagesRef)

	// assuming that all pages are the same size, figure out the
	// media box that will be the bbox of the xobject
	mediaBoxObj := pages[0]["MediaBox"]
	var mediaBox pdf.Array
	if mediaBoxObj == nil {
		// the first page inherits its MediaBox, therefore get it from the root
		pages := book.Get(pagesRef)
		mediaBox = pages.(pdf.Dictionary)["MediaBox"].(pdf.Array)
	} else {
		mediaBox = mediaBoxObj.(pdf.Array)
	}

	// change the pages to xobjects
	pageXobjects := []pdf.ObjectReference{}
	for _, page := range pages {
		page["Type"] = pdf.Name("XObject")
		page["Subtype"] = pdf.Name("Form")
		page["BBox"] = mediaBox

		// consolidate the contents into the xobject stream
		contents := []byte{}
		switch typed := page["Contents"].(type) {
		case pdf.ObjectReference:
			pageContents := book.Get(typed).(pdf.Stream)
			contents = pageContents.Stream
			page["Filter"] = pageContents.Dictionary["Filter"]
		case pdf.Array:
			if len(typed) == 1 {
				pageContents := book.Get(typed[0].(pdf.ObjectReference)).(pdf.Stream)
				contents = pageContents.Stream
				page["Filter"] = pageContents.Dictionary["Filter"]
			} else {
				for _, pageContentsRef := range typed {
					pageContents := book.Get(pageContentsRef.(pdf.ObjectReference)).(pdf.Stream)
					decoded, err := pageContents.Decode()
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
		xobjRef, err := book.Add(pdf.Stream{
			Dictionary: page,
			Stream:     contents,
		})
		if err != nil {
			log.Fatalln(err)
		}
		pageXobjects = append(pageXobjects, xobjRef)
	}

	// figure out how many pages to layout for
	numDocumentPages := len(pages)
	numPagesToLayout := numDocumentPages
	switch *binding {
	case "perfect", "chapbook":
		if (numPagesToLayout % 4) != 0 {
			numPagesToLayout = numDocumentPages + (4 - (numDocumentPages % 4))
		}
	case "none":
		numDocumentPages++
		if (numPagesToLayout % 2) != 0 {
			numPagesToLayout++
		}
	}

	// layout on landscape version of page size
	paperHeight := toFloat64(mediaBox[3])      // same height as the original page
	paperWidth := toFloat64(mediaBox[2]) * 2.0 // twice the width of the original page

	// layout the pages
	layedOutPages := pdf.Array{}
	stream := &bytes.Buffer{}
	xobjects := pdf.Dictionary{}
	showPage := false
	flipNextPage := true
	for pageToLayout := 0; pageToLayout < numPagesToLayout; pageToLayout++ {
		var pageNum int
		switch *binding {
		case "perfect":
			// determine the real page number for perfect bound books
			pageNum = pageToLayout - 1
			if pageToLayout%4 == 0 {
				pageNum += 4
			}
		case "chapbook":
			// determine the real page number for chapbooks
			pageNum = pageToLayout / 2
			if pageToLayout%2 == 1 {
				pageNum = numPagesToLayout - pageNum - 1
			}
		case "none":
			pageNum = pageToLayout - 1
			flipNextPage = false
		default:
			log.Println("unhandled binding:", *binding)
			usage()
		}

		// only render non-blank pages
		if pageNum < numDocumentPages && pageNum >= 0 {
			fmt.Fprintf(stream, "q ")
			// horizontal offset for recto (odd) pages
			// this correctly handles 0 based indexes for 1 based page numbers
			if pageNum%2 == 0 {
				fmt.Fprintf(stream, "1 0 0 1 %v %v cm ", paperWidth/2.0, 0)
			}

			// render the page
			pageName := fmt.Sprintf("Page%d", pageNum)
			xobjects[pdf.Name(pageName)] = pageXobjects[pageNum]
			fmt.Fprintf(stream, "/%v Do Q ", pageName)
		}

		// emit layouts after drawing both pages
		if showPage {
			// content for book page
			contents := pdf.Stream{
				Stream: stream.Bytes(),
			}
			contentsRef, err := book.Add(contents)
			if err != nil {
				log.Fatalln(err)
			}

			// add page
			bookPage := pdf.Dictionary{
				pdf.Name("Type"):   pdf.Name("Page"),
				pdf.Name("Parent"): pagesRef,
				pdf.Name("Resources"): pdf.Dictionary{
					pdf.Name("XObject"): xobjects,
				},
				pdf.Name("Contents"): contentsRef,
			}
			bookPageRef, err := book.Add(bookPage)
			if err != nil {
				log.Fatalln(err)
			}
			layedOutPages = append(layedOutPages, bookPageRef)

			// reset the stream and xobjects
			stream = &bytes.Buffer{}
			xobjects = pdf.Dictionary{}

			// flip the next page over
			if flipNextPage {
				fmt.Fprintf(stream, "%f %f %f %f %v %v cm ",
					math.Cos(math.Pi),
					math.Sin(math.Pi),
					-math.Sin(math.Pi),
					math.Cos(math.Pi),
					paperWidth,
					paperHeight,
				)
			}
			flipNextPage = !flipNextPage
		}
		showPage = !showPage
	}

	// Page tree for book
	bookPages := pdf.Dictionary{
		"Type":  pdf.Name("Pages"),
		"Kids":  layedOutPages,
		"Count": pdf.Integer(len(layedOutPages)),
		"MediaBox": pdf.Array{
			pdf.Integer(0),
			pdf.Integer(0),
			pdf.Real(paperWidth),  // width
			pdf.Real(paperHeight), // height
		},
	}
	_, err = book.Add(pdf.IndirectObject{
		ObjectReference: pagesRef,
		Object:          bookPages,
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
	pageNode := file.Get(ref).(pdf.Dictionary)

	switch pageNode["Type"] {
	case pdf.Name("Pages"):
		for _, kidRef := range pageNode["Kids"].(pdf.Array) {
			kidPages := getPages(file, kidRef.(pdf.ObjectReference))
			pages = append(pages, kidPages...)
		}
	case pdf.Name("Page"):
		pages = append(pages, pageNode)
	default:
		panic(string(pageNode["Type"].(pdf.Name)))
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
