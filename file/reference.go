package file

import (
	"bytes"
	"errors"
	"strconv"
)

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

	_, err = ParseIndirectObject(file.mmap[xrefOffset:eofOffset])
	if err != nil {
		return err
	}

	// println(string(file.mmap[xrefStart:xrefEnd]), xrefOffset)
	// println(string(file.mmap[xrefOffset : xrefOffset+200]))

	return nil
}
