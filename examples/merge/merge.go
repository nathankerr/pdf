package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/nathankerr/pdf"
)

func main() {
	log.SetFlags(log.Lshortfile)

	output := flag.String("o", "merged.pdf", ".pdf to output merged pdfs to")
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatalln("needs at least some pdfs to merge ")
	}

	fmt.Println("writing to", *output)

	appendPDF(*output, flag.Args())
}

func appendPDF(newPDFfilename string, filenames []string) {
	merged, err := pdf.Create(newPDFfilename)
	if err != nil {
		log.Fatalln(err)
	}

	// add the contents of each pdf into the merged pdf
	// collects the roots of each pdf
	roots := make([]pdf.ObjectReference, 0, len(filenames))
	for _, filename := range filenames {
		file, err := pdf.Open(filename)
		if err != nil {
			log.Fatalln(err)
		}
		// because pdf files are mmap'ed and objects are zero copied
		// the files must remain open until merged is saved
		defer func() {
			err := file.Close()
			if err != nil {
				log.Fatalln(err)
			}
		}()

		_, root := copyReferencedObjects(map[pdf.ObjectReference]pdf.ObjectReference{}, merged, file, file.Root)
		roots = append(roots, root.(pdf.ObjectReference))
		merged.Root = root.(pdf.ObjectReference)
	}

	// get the catalogs for each of the pdfs for merging their contents
	catalogs := make([]pdf.Dictionary, 0, len(roots))
	for _, root := range roots {
		catalogs = append(catalogs, merged.Get(root).(pdf.Dictionary))
	}

	// merge the page trees
	pageTreeRef := mergePageTrees(merged, catalogs)

	// create a new root
	merged.Root, err = merged.Add(pdf.Dictionary{
		"Type":  pdf.Name("Catalog"),
		"Pages": pageTreeRef,
	})
	if err != nil {
		log.Fatalln(err)
	}

	err = merged.Save()
	if err != nil {
		log.Fatalln(err)
	}
}

func copyReferencedObjects(refs map[pdf.ObjectReference]pdf.ObjectReference, dst, src *pdf.File, obj pdf.Object) (map[pdf.ObjectReference]pdf.ObjectReference, pdf.Object) {
	var merge = func(newRefs map[pdf.ObjectReference]pdf.ObjectReference) {
		for k, v := range newRefs {
			refs[k] = v
		}
	}

	switch t := obj.(type) {
	case pdf.ObjectReference:
		if _, ok := refs[t]; ok {
			obj = refs[t]
			break
		}

		// get an object reference for the copied obj
		// needed to break reference cycles
		ref, err := dst.Add(pdf.Null{})
		if err != nil {
			log.Fatalln(err)
		}
		refs[t] = ref

		newRefs, newObj := copyReferencedObjects(refs, dst, src, src.Get(t))
		merge(newRefs)

		// now actually add the object to dst
		refs[t], err = dst.Add(pdf.IndirectObject{
			ObjectReference: ref,
			Object:          newObj,
		})
		if err != nil {
			log.Fatalln(err)
		}

		obj = refs[t]
	case pdf.Dictionary:
		for k, v := range t {
			var newRefs map[pdf.ObjectReference]pdf.ObjectReference
			newRefs, t[k] = copyReferencedObjects(refs, dst, src, v)

			merge(newRefs)
		}
		obj = t
	case pdf.Array:
		for i, v := range t {
			var newRefs map[pdf.ObjectReference]pdf.ObjectReference
			newRefs, t[i] = copyReferencedObjects(refs, dst, src, v)
			merge(newRefs)
		}
		obj = t
	case pdf.Stream:
		for k, v := range t.Dictionary {
			var newRefs map[pdf.ObjectReference]pdf.ObjectReference
			newRefs, t.Dictionary[k] = copyReferencedObjects(refs, dst, src, v)
			merge(newRefs)
		}
		obj = t
	case pdf.Name, pdf.Integer, pdf.String, pdf.Real:
		// these types can't have references
	default:
		log.Fatalf("unhandled %T", obj)
	}

	return refs, obj
}

func mergePageTrees(file *pdf.File, catalogs []pdf.Dictionary) pdf.ObjectReference {
	// reserve a reference for the new page tree root
	// needed to set the parent for the old page tree roots
	pageTreeRef, err := file.Add(pdf.Null{})
	if err != nil {
		log.Fatalln(err)
	}

	// use the old page tree roots as our page tree kids
	kids := pdf.Array{}
	pageCount := pdf.Integer(0)
	for _, catalog := range catalogs {
		// add the old page tree root to our list of kids
		pagesRef := catalog["Pages"].(pdf.ObjectReference)
		kids = append(kids, pagesRef)

		// now that the old page tree root is a kid, it needs a parent
		pages := file.Get(pagesRef).(pdf.Dictionary)
		pages["Parent"] = pageTreeRef
		_, err = file.Add(pdf.IndirectObject{
			ObjectReference: pagesRef,
			Object:          pages,
		})
		if err != nil {
			log.Fatalln(err)
		}

		pageCount += pages["Count"].(pdf.Integer)
	}

	// create the merged page tree
	_, err = file.Add(pdf.IndirectObject{
		ObjectReference: pageTreeRef,
		Object: pdf.Dictionary{
			"Type":  pdf.Name("Pages"),
			"Kids":  kids,
			"Count": pageCount,
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	return pageTreeRef
}
