package file

import (
	"os"
)

type File struct {
	filename string
	file     *os.File
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

	return file, nil
}

func (f *File) Close() error {
	err := f.file.Close()
	if err != nil {
		return err
	}

	return nil
}
