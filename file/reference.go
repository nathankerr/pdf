package file

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

// tables 15, 19
// §7.5.4 Cross-Reference Table
// §7.5.5 File Trailer
// §7.5.8 Cross-Reference Streams
type Trailer {
	Size Integer // required, not an indirect reference
	Prev Integer // present only if the file has more than one cross-reference section
	Root Dictionary // required, shall be an indirect reference
	Encrypt Dictionary // required if document is encrypted; PDF-1.1
	Info Dictionary // optional, shall be an indirect reference
	ID Array // required if Encrypt entry is present; optional otherwise; PDF-1.1
	
	XRefStm Integer // optional

	Index Array // optional
	W Array // required
}


// Table 18 defines the cross-reference stream type
// type 0 = f entries in cross-reference table
// type 1 = n entries in cross-reference table
// type 2 nnot in cross-reference table
CrossReference [3]int

CrossReferences map[string]CrossReference

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
		xrstream, _, err := parseIndirectObject(file.mmap[xrefOffset:eofOffset])
		if err != nil {
			return err
		}
		fmt.Println(xrstream)
	case 'x':
		// xref table
		println("xref table")
	default:
		panic(file.mmap[xrefOffset])
	}

	// println(string(file.mmap[xrefStart:xrefEnd]), xrefOffset)
	// println(string(file.mmap[xrefOffset : xrefOffset+200]))

	return nil
}
