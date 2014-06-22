package main

import (
	"github.com/juju/errgo"
	pdf "github.com/nathankerr/pdf/file"
	"log"
	"os"
)

func main() {
	log.SetFlags(log.Lshortfile)

	if len(os.Args) != 2 {
		log.Fatalln("Usage: single [pdf.pdf]")
	}

	filename := os.Args[1]

	file, err := pdf.Open(filename)
	if err != nil {
		log.Fatalln(errgo.Details(err))
	}

	catalog := file.Get(file.Root).(pdf.Dictionary)
	pages := getPages(file, catalog[pdf.Name("Pages")].(pdf.ObjectReference))

	// get a new root for the page tree
	page_list := file.Get(catalog[pdf.Name("Pages")].(pdf.ObjectReference)).(pdf.Dictionary)
	page_list[pdf.Name("Kids")] = pdf.Array{}
	page_list_ref, err := file.Add(page_list)

	// make the first page the only page
	single_page := pages[0].Object.(pdf.Dictionary)
	delete(single_page, pdf.Name("Parent"))

	single_page[pdf.Name("Parent")] = page_list_ref
	single_page_ref, err := file.Add(single_page)
	if err != nil {
		log.Fatalln(err)
	}
	page_list[pdf.Name("Kids")] = append(page_list[pdf.Name("Kids")].(pdf.Array), single_page_ref)

	// update page list count
	page_list[pdf.Name("Count")] = pdf.Integer(len(page_list[pdf.Name("Kids")].(pdf.Array)))
	file.Add(pdf.IndirectObject{
		ObjectReference: page_list_ref,
		Object:          page_list,
	})

	// update the catalog
	catalog[pdf.Name("Pages")] = page_list_ref
	new_catalog_ref, err := file.Add(catalog)
	if err != nil {
		log.Fatalln(err)
	}
	file.Root = new_catalog_ref

	// save as because can't mix xref and xref stream files
	// and xref stream writing is not supported (yet)
	err = file.SaveAs("copy-" + filename)
	if err != nil {
		log.Fatalln(errgo.Details(err))
	}

	err = file.Close()
	if err != nil {
		log.Fatalln(err)
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
