package file

import (
	"bytes"
	"errors"
	"github.com/edsrzf/mmap-go"
	"os"
	"strconv"
)

type File struct {
	filename string
	file     *os.File
	mmap     mmap.MMap
}

func Open(filename string) (*File, error) {
	file := &File{
		filename: filename,
	}

	var err error
	file.file, err = os.Open(filename)
	if err != nil {
		return nil, err
	}

	file.mmap, err = mmap.Map(file.file, mmap.RDONLY, 0)
	if err != nil {
		return nil, err
	}

	// check pdf file header
	if !bytes.Equal(file.mmap[:7], []byte("%PDF-1.")) {
		return nil, errors.New("file does not have PDF header")
	}

	// check ending
	eofOffset := bytes.LastIndex(file.mmap, []byte("%%EOF"))
	if eofOffset == -1 {
		return nil, errors.New("file does not have PDF ending")
	}

	// find last startxref
	startxrefOffset := bytes.LastIndex(file.mmap, []byte("startxref"))
	if startxrefOffset == -1 {
		return nil, errors.New("could not find startxref")
	}

	digits := "0123456789"
	xrefStart := bytes.IndexAny(file.mmap[startxrefOffset:], digits)
	if xrefStart == -1 {
		return nil, errors.New("could not find beginning of startxref reference")
	}
	xrefStart += startxrefOffset
	xrefEnd := bytes.LastIndexAny(file.mmap[xrefStart:eofOffset], digits)
	if xrefEnd == -1 {
		return nil, errors.New("could not find end of startxref reference")
	}
	xrefEnd += xrefStart + 1

	xrefOffset, err := strconv.ParseUint(string(file.mmap[xrefStart:xrefEnd]), 10, 64)
	if err != nil {
		return nil, err
	}

	println(string(file.mmap[xrefStart:xrefEnd]), xrefOffset)
	println(string(file.mmap[xrefOffset : xrefOffset+200]))

	return file, nil
}

func (f *File) Close() error {
	err := f.mmap.Unmap()
	if err != nil {
		return err
	}

	err = f.file.Close()
	if err != nil {
		return err
	}

	return nil
}
