all: pdf.leg.go
	go install github.com/nathankerr/pdf
	go install github.com/nathankerr/pdf/pdf
	pdf PDF32000_2008.pdf

%.leg.go: %.leg
	leg $< > $@