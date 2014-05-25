/*
Package file manages objects stored in a PDF file.

This package only concerns itself with file-level concerns.
These include:
	* locating and decoding indirect objects

A PDF file is a serialized set of indirect objects
which can be randomly accessed. The format allows for
append-only extentions to the set of objects.

Methods for random access:
	1. Cross-Reference Table (7.5.4) and File Trailer (7.5.5)
	2. Cross-Reference Streams (7.5.8) (since PDF-1.5)
	3. Hybrid (7.5.8.4) (since PDF-1.5)
The method used can be determined by following the
startxref reference. If the referenced position is an
indirect object, then method 2 is used. Otherwise if the
trailer has an XRefStm entry, then method 3 is used.
Otherwise method 1 is used.

*/
package file
