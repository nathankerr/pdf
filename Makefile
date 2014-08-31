all:
	go fmt ./...
	go vet ./...
	go install -v github.com/nathankerr/pdf
	cp paper.pdf paper-modified.pdf
	go run examples/book/book.go paper-modified.pdf
	open -a "Adobe Reader" paper-modified.pdf