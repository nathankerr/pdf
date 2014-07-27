all:
	go fmt ./...
	go install -v github.com/nathankerr/pdf
	cp paper.pdf paper-modified.pdf
	go run examples/single.go paper-modified.pdf
	open paper-modified.pdf

test:
	go test -i github.com/nathankerr/pdf/file
	go test github.com/nathankerr/pdf/file

fmt:
	go fmt *.go
	go fmt file/*.go

vet:
	go vet *.go
	go vet file/*.go