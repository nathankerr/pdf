package file

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

// tables 15, 17, 19
// §7.5.4 Cross-Reference Table
// §7.5.5 File Trailer
// §7.5.8 Cross-Reference Streams
type Trailer struct {
	Size    Integer    // required, not an indirect reference
	Prev    Integer    // present only if the file has more than one cross-reference section
	Root    Dictionary // required, shall be an indirect reference
	Encrypt Dictionary // required if document is encrypted; PDF-1.1
	Info    Dictionary // optional, shall be an indirect reference
	ID      Array      // required if Encrypt entry is present; optional otherwise; PDF-1.1

	XRefStm Integer // optional

	Index Array // optional
	W     Array // required
}

// Table 18 defines the cross-reference stream type
// type 0 = f entries in cross-reference table
// type 1 = n entries in cross-reference table
// type 2 not in cross-reference table
type CrossReference [3]int

type CrossReferences map[string]CrossReference

// handles cross-references
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

	xrefOffset, err := strconv.ParseUint(string(file.mmap[xrefStart:xrefEnd]), 10, 64)
	if err != nil {
		return err
	}

	switch file.mmap[xrefOffset] {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		// indirect object and therefore a cross-reference stream §7.5.8
		xrstreamAsObject, _, err := parseIndirectObject(file.mmap[xrefOffset:eofOffset])
		if err != nil {
			return err
		}
		xrstream := xrstreamAsObject.(IndirectObject).Object.(Stream)

		stream, err := xrstream.Decode()
		if err != nil {
			return err
		}
		// fmt.Printf("%v\n", xrstream.Stream)
		// fmt.Printf("%v\n", stream)

		w := xrstream.Dictionary[Name("W")].(Array)
		wi := []int{}

		size := int(xrstream.Dictionary[Name("Size")].(Integer))

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
					xref := CrossReference{}
					ioffset := 0
					for i := 0; i < 2; i++ {
						width := wi[i]
						start := offset + ioffset
						xref[i] = bytesToInt(stream[start : start+width])
						ioffset += width
					}
					fmt.Println(objectNumber, xref)

					objectNumber++
					offset += stride
				}
			}
		}

		fmt.Println("Index:", len(stream)%stride, size, indexArrayAsObject, indexes)

		// fmt.Println("XREF: ", w, stride, len(xrstream.Stream), len(stream), len(stream) % stride, xrstream.Dictionary)

	case 'x':
		// xref table §7.5.4
		println("xref table")
	default:
		panic(file.mmap[xrefOffset])
	}

	// println(string(file.mmap[xrefStart:xrefEnd]), xrefOffset)
	// println(string(file.mmap[xrefOffset : xrefOffset+200]))

	return nil
}

func bytesToInt(bytesOfInt []byte) int {
	value := 0
	for i, b := range bytesOfInt {
		shift := len(bytesOfInt) - i - 1
		value += int(b) << uint(8*shift)
	}
	return value
}
