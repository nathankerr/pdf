package file

import (
	"bytes"
	"fmt"
	"github.com/edsrzf/mmap-go"
	"github.com/juju/errgo"
	"io"
	"os"
)

type File struct {
	filename string
	file     *os.File
	mmap     mmap.MMap

	objects []IndirectObject
}

func Open(filename string) (*File, error) {
	file := &File{
		filename: filename,
	}

	var err error
	file.file, err = os.Open(filename)
	if err != nil {
		return nil, errgo.Mask(err)
	}

	file.mmap, err = mmap.Map(file.file, mmap.RDONLY, 0)
	if err != nil {
		file.Close()
		return nil, errgo.Mask(err)
	}

	// check pdf file header
	if !bytes.Equal(file.mmap[:7], []byte("%PDF-1.")) {
		file.Close()
		return nil, errgo.New("file does not have PDF header")
	}

	err = file.loadReferences()
	if err != nil {
		file.Close()
		return nil, err
	}

	return file, nil
}

func Create(filename string) (*File, error) {
	file := &File{
		filename: filename,
	}

	// create enough of the pdf so that
	// appends will not break things
	f, err := os.Create(filename)
	if err != nil {
		return nil, errgo.Mask(err)
	}
	defer f.Close()
	f.Write([]byte("%PDF-1.4"))

	return file, nil
}

// finds the object "object_number generation_number"
// returns Null when object not found
func (f *File) Get(reference string) Object {
	return Null{}
}

// adds an object to the file, returns the object reference "object_number generation_number"
func (f *File) Add(obj Object) ObjectReference {
	ref := ObjectReference{}

	switch typed := obj.(type) {
	case IndirectObject:
		ref.ObjectNumber = typed.ObjectNumber
		ref.GenerationNumber = typed.GenerationNumber
		f.objects = append(f.objects, typed)
	default:
		panic(obj)
	}
	return ref
}

func writeLineBreakTo(w io.Writer) (int64, error) {
	n, err := w.Write([]byte{'\n', '\n'})
	return int64(n), err
}

// Writes the objects that have been put into the File to the file.
// A new object index will be written (taking up space)
// the File object is still usable after calling this. The effect will be as if the file was newly opened.
func (f *File) Save() error {
	info, err := os.Stat(f.filename)
	if err != nil {
		return errgo.Mask(err)
	}

	file, err := os.OpenFile(f.filename, os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return errgo.Mask(err)
	}
	defer file.Close()

	offset := info.Size() + 1

	n, err := writeLineBreakTo(file)
	if err != nil {
		return errgo.Mask(err)
	}
	offset += n

	xrefs := map[uint64]CrossReference{}

	xrefs[0] = CrossReference{0, 0, 65535}

	for i := range f.objects {
		// fmt.Println("writing object", i, "at", offset)
		xrefs[f.objects[i].ObjectNumber] = CrossReference{1, int(offset), int(f.objects[i].GenerationNumber)}
		n, err = f.objects[i].WriteTo(file)
		if err != nil {
			return errgo.Mask(err)
		}
		offset += n

		n, err = writeLineBreakTo(file)
		if err != nil {
			return errgo.Mask(err)
		}
		offset += n
	}

	// FIXME: this is not really a good way to generate an xref table
	// for example, ordering and grouping are not done
	fmt.Fprintf(file, "xref\n0 %d\n", len(xrefs))
	for _, xref := range xrefs {
		fmt.Fprintf(file, "%010d %05d ", xref[1], xref[2])
		switch xref[0] {
		case 0:
			// f entries
			fmt.Fprintf(file, "f\n")
		case 1:
			// n entries
			fmt.Fprintf(file, "f\n")
		case 2:
			panic("can't be in xref table")
		default:
			panic("unhandled case")
		}
	}

	fmt.Fprintf(file, "\ntrailer\n")
	trailer := Dictionary{
		Name("Size"): Integer(len(xrefs)),
		Name("Root"): ObjectReference{
			ObjectNumber: 1,
		}, // TODO: figure out how to actually handle root
	}
	_, err = trailer.WriteTo(file)
	if err != nil {
		return errgo.Mask(err)
	}

	fmt.Fprintf(file, "\nstartxref\n%d\n%%%%EOF", offset)

	return nil
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
