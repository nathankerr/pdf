package main

import (
	"os"
	"log"
)

func main() {
	log.SetFlags(log.Lshortfile)

	if len(os.Args) != 2 {
		log.Fatalln("Usage: pdf [file.pdf]")
	}

	filename := os.Args[1]

	log.Println(filename)

	// file, err := pdf.Open(filename)
	// if err != nil {
	// 	log.Fatalln(filename, err)
	// }

	// log.Println(file)
}