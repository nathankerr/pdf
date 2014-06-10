package file

import (
	"bytes"
	"fmt"
	"github.com/edsrzf/mmap-go"
	"github.com/juju/errgo"
	"io"
	"os"
	"reflect"
	"sort"
)

type freeObject uint // generation number for next use of the object number where this is stored

// File manages access to objects stored in a PDF file.
type File struct {
	filename string
	file     *os.File
	mmap     mmap.MMap

	// cross reference for existing objects
	// indirect object for new objects
	// free object for newly freed objects
	// map key is the object number
	// make sure generation number is >= existing generation number when modifying
	objects  map[uint]interface{}
	nextFree uint // object number of next free object
	size     uint // max object number + 1

	prev    Integer
	Trailer Dictionary

	// things from trailer that should be exported
	Root    ObjectReference
	Encrypt Dictionary
	Info    Dictionary
	ID      Array
}

// Open opens a PDF file for manipulation of its objects.
func Open(filename string) (*File, error) {
	file := &File{
		filename: filename,
		objects:  map[uint]interface{}{},
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

// Create creates a new PDF file with no objects.
func Create(filename string) (*File, error) {
	file := &File{
		filename: filename,
		Trailer:  Dictionary{},
		objects:  map[uint]interface{}{},
	}

	// create enough of the pdf so that
	// appends will not break things
	f, err := os.Create(filename)
	if err != nil {
		return nil, errgo.Mask(err)
	}
	defer f.Close()
	f.Write([]byte("%PDF-1.7"))

	return file, nil
}

// Get returns the referenced object.
// When the object does not exist, Null is returned.
func (f *File) Get(reference ObjectReference) Object {
	object, ok := f.objects[reference.ObjectNumber]
	if !ok {
		return Null{}
	}

	switch typed := object.(type) {
	case crossReference: // existing object
		switch typed[0] {
		case 0: // free entry
			return Null{}
		case 1: // normal
			offset := typed[1] - 1
			obj, _, err := parseIndirectObject(f.mmap[offset:])
			if err != nil {
				fmt.Println("file.Get:", err)
			}
			return obj.(IndirectObject).Object
		case 2: // in object stream
			panic("object streams not yet supported")
		default:
			panic(typed[0])
		}
	case IndirectObject: // new object
		return typed.Object
	case freeObject: // newly freed object
		return Null{}
	default:
		panic("unhandled type: " + reflect.TypeOf(object).Name())
	}
}

// Add returns the object reference of the object after adding it to the file.
// An IndirectObject's ObjectReference will be used,
// otherwise a free ObjectReference will be used.
//
// If an IndirectObject's ObjectReference also refers to an existing
// object, the newly added IndirectObject will mask the existing one.
// Only the most recently added object will be Saved to disk.
// GenerationNumber must be greater than or equal to the largest existing
// GenerationNumber for that ObjectNumber.
func (f *File) Add(obj Object) (ObjectReference, error) {
	// TODO: handle non indirect-objects
	ref := ObjectReference{}

	switch typed := obj.(type) {
	case IndirectObject:
		ref.ObjectNumber = typed.ObjectNumber
		ref.GenerationNumber = typed.GenerationNumber

		// check to see if the generation number works
		existing, ok := f.objects[ref.ObjectNumber]
		if ok {
			// determine the minimum allowed generation number
			var minGenerationNumber uint = 0
			switch typed := existing.(type) {
			case crossReference: // existing object
				switch typed[0] {
				case 0: // free entry
					minGenerationNumber = typed[2]
				case 1: // normal
					minGenerationNumber = typed[2]
				case 2: // in object stream
					// objects in object streams must have a
					// generation number of 0
					minGenerationNumber = 0
				default:
					panic(typed[0])
				}
			case IndirectObject: // new object
				minGenerationNumber = typed.GenerationNumber
			case freeObject: // newly freed object
				minGenerationNumber = uint(typed)
			default:
				panic("unhandled type: " + reflect.TypeOf(typed).Name())
			}

			if ref.GenerationNumber < minGenerationNumber {
				// TODO: make better error
				ref.GenerationNumber = minGenerationNumber
				return ref, errgo.New("Generation number is too small...")
			}
		}

		f.objects[ref.ObjectNumber] = typed
	default:
		panic(obj)
	}
	return ref, nil
}

func writeLineBreakTo(w io.Writer) (int64, error) {
	n, err := w.Write([]byte{'\n', '\n'})
	return int64(n), err
}

// Save appends the objects that have been added to the File
// to the file on disk. After saving, the File is still usable
// and will act as though it were just Open'ed.
//
// NOTE: A new object index will be written on each save,
// taking space in the file on disk
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

	xrefs := map[Integer]crossReference{}

	xrefs[0] = crossReference{0, 0, 65535}

	free := sort.IntSlice{}
	for i := range f.objects {
		switch typed := f.objects[i].(type) {
		case crossReference:
			// no-op, don't need to write unchanged objects to file
			// however, we do need to handle the free list
			if typed[0] == 0 {
				free = append(free, int(i))
			}
		case IndirectObject:
			xrefs[Integer(i)] = crossReference{1, uint(offset - 1), typed.GenerationNumber}
			n, err = typed.WriteTo(file)
			if err != nil {
				return errgo.Mask(err)
			}
			offset += n

			n, err = writeLineBreakTo(file)
			if err != nil {
				return errgo.Mask(err)
			}
			offset += n
		case freeObject:
			xrefs[Integer(i)] = crossReference{0, 0, uint(typed)}
			free = append(free, int(i))
		default:
			panic("unhandled type: " + reflect.TypeOf(typed).Name())
		}
	}

	// fill in the free linked list
	free.Sort()
	for i := 0; i < free.Len()-1; i++ {
		xref := xrefs[Integer(free[i])]
		xref[1] = uint(free[i+1])
		xrefs[Integer(free[i])] = xref
	}

	objects := make(sort.IntSlice, 0, len(xrefs))
	for objectNumber := range xrefs {
		objects = append(objects, int(objectNumber))
	}
	objects.Sort()

	// group into consecutive sets
	groups := []sort.IntSlice{}
	groupStart := 0
	for i := range objects {
		if i == 0 {
			continue
		}

		if objects[i] != objects[i-1]+1 {
			if groupStart+i-1 == groupStart {
				// handle single length groups
				groups = append(groups, objects[groupStart:groupStart+1])
			} else {
				groups = append(groups, objects[groupStart:groupStart+i-1])
			}
			groupStart = i
		}
	}
	// add remaining group
	groups = append(groups, objects[groupStart:])

	// write as an xref table to file
	fmt.Fprintf(file, "xref\n")
	for _, group := range groups {
		fmt.Fprintf(file, "%d %d\n", group[0], len(group))
		for _, objectNumber := range group {
			xref := xrefs[Integer(objectNumber)]
			fmt.Fprintf(file, "%010d %05d ", xref[1], xref[2])
			switch xref[0] {
			case 0:
				// f entries
				fmt.Fprintf(file, "f\n")
			case 1:
				// n entries
				fmt.Fprintf(file, "n\n")
			case 2:
				panic("can't be in xref table")
			default:
				panic("unhandled case")
			}
		}
	}

	// Write the file trailer
	fmt.Fprintf(file, "\ntrailer\n")
	trailer := Dictionary{}
	trailer[Name("Root")] = f.Root

	// Figure out the highest object number to set Size properly
	var maxObjNum uint
	for objNum := range f.objects {
		if objNum > maxObjNum {
			maxObjNum = objNum
		}
	}
	trailer[Name("Size")] = Integer(maxObjNum + 1)

	if f.prev != 0 {
		trailer[Name("Prev")] = f.prev
	}

	_, err = trailer.WriteTo(file)
	if err != nil {
		return errgo.Mask(err)
	}

	fmt.Fprintf(file, "\nstartxref\n%d\n%%%%EOF", offset-1)

	return nil
}

// Close the File, does not Save.
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

func (f *File) Free(objectNumber uint) {
	obj, ok := f.objects[objectNumber]
	if !ok {
		// object does not exist, and therefore is already free
		return
	}

	switch typed := obj.(type) {
	case crossReference: // existing object
		switch typed[0] {
		case 0: // free entry
			// no-op
			// the object is already free
		case 1: // normal
			f.objects[objectNumber] = freeObject(typed[2] + 1)
		case 2: // in object stream
			// objects in object streams must have a
			// generation number of 0
			f.objects[objectNumber] = freeObject(1)
		default:
			panic(typed[0])
		}
	case IndirectObject: // new object
		f.objects[objectNumber] = freeObject(typed.GenerationNumber + 1)
	case freeObject: // newly freed object
		// no-op
		// already free
	default:
		panic("unhandled type: " + reflect.TypeOf(typed).Name())
	}
}
