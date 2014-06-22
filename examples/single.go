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

	single_page := pages[0].Object.(pdf.Dictionary)
	delete(single_page, pdf.Name("Parent"))

	// rebuild the pdf structure
	page_list := file.Get(catalog[pdf.Name("Pages")].(pdf.ObjectReference)).(pdf.Dictionary)
	page_list[pdf.Name("Kids")] = pdf.Array{}
	page_list_ref, err := file.Add(page_list)

	single_page[pdf.Name("Parent")] = page_list_ref
	single_page_ref, err := file.Add(single_page)
	if err != nil {
		log.Fatalln(err)
	}
	page_list[pdf.Name("Kids")] = append(page_list[pdf.Name("Kids")].(pdf.Array), single_page_ref)

	// outlines
	outlines := pdf.Dictionary{
		pdf.Name("Type"):  pdf.Name("Outlines"),
		pdf.Name("Count"): pdf.Integer(0),
	}
	outlines_ref, err := file.Add(outlines)
	if err != nil {
		log.Fatalln(err)
	}

	// update page list count
	page_list[pdf.Name("Count")] = pdf.Integer(len(page_list[pdf.Name("Kids")].(pdf.Array)))
	file.Add(pdf.IndirectObject{
		ObjectReference: page_list_ref,
		Object:          page_list,
	})

	catalog[pdf.Name("Pages")] = page_list_ref
	catalog[pdf.Name("Outlines")] = outlines_ref
	new_catalog_ref, err := file.Add(catalog)
	if err != nil {
		log.Fatalln(err)
	}
	file.Root = new_catalog_ref

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
