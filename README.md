# Levels of API

file - handle serialization / de-serialization of pdf files (random access object streams)
pdf (the root package) - the pdf document abstraction (the catalog)
document - the document structure itself
compatibility - zombiezen, fpdf, itext, etc.

file and document must be able to do anything pdf itself can do and do it for any version of pdf. All versions and standards need to be accessible and usable such that any required pdf can be read/generated. This means a complete version-feature matrix will need to be created and used.