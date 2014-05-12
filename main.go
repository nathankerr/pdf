package main

import (
	"github.com/juju/errgo"
	"github.com/nathankerr/pdf/file"
	"log"
	"os"
)

func main() {
	log.SetFlags(log.Lshortfile)

	if len(os.Args) != 2 {
		log.Fatalln("Usage: pdf [file.pdf]")
	}

	filename := os.Args[1]

	pdf, err := file.Open(filename)
	if err != nil {
		// log.Fatalf("%s: %s\n", filename, err)
		log.Fatalln(errgo.Details(err))
	}

	err = pdf.Close()
	if err != nil {
		log.Fatalln(err)
	}

	log.Println(pdf)
}
