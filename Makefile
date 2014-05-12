all:
	go fmt ./...
	go install -v github.com/nathankerr/pdf
	#pdf PDF32000_2008.pdf
	rm -f h7*.pdf
	go run examples/h7.go

test:
	go test -i github.com/nathankerr/pdf/file
	go test github.com/nathankerr/pdf/file

fmt:
	go fmt *.go
	go fmt file/*.go

vet:
	go vet *.go
	go vet file/*.go