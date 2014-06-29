all:
	go fmt ./...
	go install -v github.com/nathankerr/pdf
	go run examples/chapbook.go paper.pdf paper-single.pdf
	open paper-single.pdf

test:
	go test -i github.com/nathankerr/pdf/file
	go test github.com/nathankerr/pdf/file

fmt:
	go fmt *.go
	go fmt file/*.go

vet:
	go vet *.go
	go vet file/*.go