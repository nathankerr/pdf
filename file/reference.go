package file

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strconv"
)

// CrossReference holds the data described in Table 18
// type 0 = f entries in cross-reference table
// type 1 = n entries in cross-reference table
// type 2 not in cross-reference table
// 0 number_of_next_free_object generation_number_if_used_again
// 1 byte_offset_of_object generation_number
// 2 object_number_of_object_stream_containing_this_object index_of_this_object_in_object_stream
type crossReference [3]uint

type crossReferences map[Integer]crossReference

// handles cross-references
//
// 	1. Cross-Reference Table (§7.5.4) and File Trailer (§7.5.5)
// 	2. Cross-Reference Streams (§7.5.8) (since PDF-1.5)
// 	3. Hybrid (§7.5.8.4) (since PDF-1.5)
//
// The method used can be determined by following the
// startxref reference. If the referenced position is an
// indirect object, then method 2 is used. Otherwise if the
// trailer has an XRefStm entry, then method 3 is used.
// Otherwise method 1 is used.
func (file *File) loadReferences() error {
	// find EOF tag to ignore junk in the file after it
	eofOffset := bytes.LastIndex(file.mmap, []byte("%%EOF"))
	if eofOffset == -1 {
		return errors.New("file does not have PDF ending")
	}

	// find last startxref
	startxrefOffset := bytes.LastIndex(file.mmap, []byte("startxref"))
	if startxrefOffset == -1 {
		return errors.New("could not find startxref")
	}

	digits := "0123456789"
	xrefStart := bytes.IndexAny(file.mmap[startxrefOffset:], digits)
	if xrefStart == -1 {
		return errors.New("could not find beginning of startxref reference")
	}
	xrefStart += startxrefOffset
	xrefEnd := bytes.LastIndexAny(file.mmap[xrefStart:eofOffset], digits)
	if xrefEnd == -1 {
		return errors.New("could not find end of startxref reference")
	}
	xrefEnd += xrefStart + 1

	xrefOffset64, err := strconv.ParseUint(string(file.mmap[xrefStart:xrefEnd]), 10, 64)
	if err != nil {
		return err
	}
	xrefOffset := int(xrefOffset64)

	refs, trailer, err := file.load_references(xrefOffset)
	if err != nil {
		return err
	}

	file.prev = Integer(xrefOffset)
	file.objects = refs

	root := trailer[Name("Root")]
	if root != nil {
		file.Root = root.(ObjectReference)
	}

	encrypt := trailer[Name("Encrypt")]
	if encrypt != nil {
		file.Encrypt = encrypt.(Dictionary)
	}

	info := trailer[Name("Info")]
	if info != nil {
		file.Info = info.(Dictionary)
	}

	id := trailer[Name("ID")]
	if id != nil {
		file.ID = id.(Array)
	}

	// println(string(file.mmap[xrefStart:xrefEnd]), xrefOffset)
	// println(string(file.mmap[xrefOffset : xrefOffset+200]))

	return nil
}

// parse and recursively load and merge references and trailer
func (file *File) load_references(xrefOffset int) (map[uint]interface{}, Dictionary, error) {
	// fmt.Println("load_references", xrefOffset)

	// parse refs, trailer
	refs := map[uint]interface{}{}
	var trailer Dictionary

	switch file.mmap[xrefOffset] {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		// indirect object and therefore a cross-reference stream §7.5.8
		xrstreamAsObject, _, err := parseIndirectObject(file.mmap[xrefOffset:])
		if err != nil {
			return refs, trailer, err
		}
		xrstream := xrstreamAsObject.(IndirectObject).Object.(Stream)

		stream, err := xrstream.Decode()
		if err != nil {
			return refs, trailer, err
		}

		trailer = xrstream.Dictionary

		w := xrstream.Dictionary[Name("W")].(Array)
		size := int(xrstream.Dictionary[Name("Size")].(Integer))

		wi := []int{}
		stride := 0
		for _, integer := range w {
			stride += int(integer.(Integer))
			wi = append(wi, int(integer.(Integer)))
		}

		type index struct {
			objectNumber int
			size         int
		}
		indexes := []index{}

		indexArrayAsObject := xrstream.Dictionary[Name("Index")]
		if indexArrayAsObject == nil {
			// default when Index is not specified
			indexes = append(indexes, index{0, size})
		} else {
			indexArray := indexArrayAsObject.(Array)
			for i := 0; i < len(indexArray); i += 2 {
				indexes = append(indexes, index{
					int(indexArray[i].(Integer)),
					int(indexArray[i+1].(Integer)),
				})
			}
		}

		for _, index := range indexes {
			objectNumber := index.objectNumber
			offset := 0
			for n := 0; n < index.size; n++ {
				for offset < len(stream) {
					xref := crossReference{}
					ioffset := 0
					for i := 0; i < 2; i++ {
						width := wi[i]
						start := offset + ioffset
						xref[i] = uint(bytesToInt(stream[start : start+width]))
						ioffset += width
					}

					objectNumber++
					offset += stride
				}
			}
		}

	case 'x':
		// xref table §7.5.4
		i := xrefOffset

		token, n := nextToken(file.mmap[i:])
		if string(token) != "xref" {
			log.Fatalln("offset: ", i, "could not match xref")
		}
		i += n

		for {
			token, n := nextToken(file.mmap[i:])
			if string(token) == "trailer" {
				i += n
				break
			}

			xrefs, n := parseXrefBlock(file.mmap[i:])
			for objectNumber, xref := range xrefs {
				refs[uint(objectNumber)] = xref
			}
			i += n
		}

		trailerObj, n, err := parseObject(file.mmap[i:])
		if err != nil {
			fmt.Println("XREF TRAILER:", err)
		}
		i += n

		trailer = trailerObj.(Dictionary)

	default:
		fmt.Println(xrefOffset)
		println(string(file.mmap[xrefOffset : xrefOffset+20]))
		panic(file.mmap[xrefOffset])
	}

	prev, has_prev := trailer[Name("Prev")]
	if has_prev {
		prev_refs, prev_trailer, err := file.load_references(int(prev.(Integer)))
		if err != nil {
			return refs, trailer, err
		}

		for prev_ref := range prev_refs {
			if _, ok := refs[prev_ref]; !ok {
				refs[prev_ref] = prev_refs[prev_ref]
			}
		}

		for name := range prev_trailer {
			if _, ok := trailer[name]; !ok {
				trailer[name] = prev_trailer[name]
			}
		}
	}

	// TODO: hybrid

	return refs, trailer, nil
}

func bytesToInt(bytesOfInt []byte) int {
	value := 0
	for i, b := range bytesOfInt {
		shift := len(bytesOfInt) - i - 1
		value += int(b) << uint(8*shift)
	}
	return value
}

func parseXrefBlock(slice []byte) (crossReferences, int) {
	log.SetFlags(log.Lshortfile)
	var i int
	references := crossReferences{}

	// object number
	token, n := nextToken(slice[i:])
	objectNumber, err := strconv.ParseUint(string(token), 10, 64)
	if err != nil {
		log.Fatalln(err)
	}
	i += n

	// number of objects
	token, n = nextToken(slice[i:])
	nObjects, err := strconv.ParseUint(string(token), 10, 64)
	if err != nil {
		log.Fatalln(err)
	}
	i += n

	for j := 0; j < int(nObjects); j++ {
		// offset
		token, n = nextToken(slice[i:])
		offset, err := strconv.ParseUint(string(token), 10, 64)
		if err != nil {
			log.Fatalln(err)
		}
		i += n

		// generation number
		token, n = nextToken(slice[i:])
		generation, err := strconv.ParseUint(string(token), 10, 64)
		if err != nil {
			log.Fatalln(err)
		}
		i += n

		// type
		entryType, n := nextToken(slice[i:])
		i += n

		var xref crossReference
		switch entryType[0] {
		case 'f':
			xref[0] = 0
		case 'n':
			xref[0] = 1
		default:
			panic(string(entryType))
		}

		xref[1] = uint(offset)
		xref[2] = uint(generation)

		references[Integer(objectNumber)] = xref
		objectNumber++
	}

	return references, i
}
