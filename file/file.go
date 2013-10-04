package file

import (
	"bytes"
	"errors"
	"github.com/edsrzf/mmap-go"
	"os"
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

	err = file.loadReferences()
	if err != nil {
		return nil, err
	}

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
