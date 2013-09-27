package main

import (
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
		log.Fatalln(filename, err)
	}

	err = pdf.Close()
	if err != nil {
		log.Fatalln(err)
	}

	log.Println(pdf)
}
