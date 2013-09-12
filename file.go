package pdf

type File []FileObject

// defines the types representing a pdf file
// Header, Body, CrossReferenceTable, Trailer,
// ObjectStream, CrossReferenceStream
type FileObject interface{}

// 7.5.2 File Header
type Header string

// 7.5.3 File Body
type Body []BodyObject
type BodyObject interface{} // IndirectObject, ObjectStream

// 7.5.4 Cross-Reference Table
type CrossReferenceTable []CrossReferenceEntry
type CrossReferenceEntry struct {
	Offset uint
	GenerationNumber uint
	InUse bool // 'n' if true; else 'f'
}

// 7.5.5 File Trailer
type Trailer Dictionary

// 7.5.7 Object Streams
type ObjectStream Stream

// 7.5.8 Cross-Reference Streams
type CrossReferenceStream Stream