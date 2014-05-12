package main

import (
	"github.com/juju/errgo"
	pdf "github.com/nathankerr/pdf/file"
	"log"
)

func main() {
	createMinimalFile()
	// stage1()
	// stage2()
	// stage3()
	// stage4()
	log.Printf("done")
}

// create the minimal file described in H.2
func createMinimalFile() {
	log.Printf("createMinimalFile")

	minimal, err := pdf.Create("h7-minimal.pdf")
	if err != nil {
		log.Fatalln(errgo.Details(err))
	}
	defer minimal.Close()

	// catalog
	minimal.Add(pdf.IndirectObject{
		ObjectNumber: 1,
		Object: pdf.Dictionary{
			pdf.Name("Type"): pdf.Name("Catalog"),
			pdf.Name("Outlines"): pdf.ObjectReference{
				ObjectNumber: 2,
			},
			pdf.Name("Pages"): pdf.ObjectReference{
				ObjectNumber: 3,
			},
		},
	})

	// outlines
	minimal.Add(pdf.IndirectObject{
		ObjectNumber: 2,
		Object: pdf.Dictionary{
			pdf.Name("Type"):  pdf.Name("Outlines"),
			pdf.Name("Count"): pdf.Integer(0),
		},
	})

	// pages
	minimal.Add(pdf.IndirectObject{
		ObjectNumber: 3,
		Object: pdf.Dictionary{
			pdf.Name("Type"): pdf.Name("Pages"),
			pdf.Name("Kids"): pdf.Array{
				pdf.ObjectReference{
					ObjectNumber: 4,
				},
			},
			pdf.Name("Count"): pdf.Integer(1),
		},
	})

	// page
	minimal.Add(pdf.IndirectObject{
		ObjectNumber: 4,
		Object: pdf.Dictionary{
			pdf.Name("Type"): pdf.Name("Page"),
			pdf.Name("Parent"): pdf.ObjectReference{
				ObjectNumber: 3,
			},
			pdf.Name("MediaBox"): pdf.Array{
				pdf.Integer(0),
				pdf.Integer(0),
				pdf.Integer(612),
				pdf.Integer(792),
			},
			pdf.Name("Contents"): pdf.ObjectReference{
				ObjectNumber: 5,
			},
			pdf.Name("Resources"): pdf.Dictionary{
				pdf.Name("ProcSet"): pdf.ObjectReference{
					ObjectNumber: 6,
				},
			},
		},
	})

	// content stream
	minimal.Add(pdf.IndirectObject{
		ObjectNumber: 5,
		Object: pdf.Stream{
			Dictionary: pdf.Dictionary{
				pdf.Name("Length"): pdf.Integer(0),
			},
		},
	})

	// procset
	minimal.Add(pdf.IndirectObject{
		ObjectNumber: 6,
		Object: pdf.Array{
			pdf.Name("PDF"),
		},
	})

	err = minimal.Save()
	if err != nil {
		log.Fatalln(errgo.Details(err))
	}
}
