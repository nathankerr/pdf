package pdf

import (
	"encoding/ascii85"
	"errors"
	"fmt"
	"reflect"
	// "compress/lzw"
	"bytes"
	"compress/flate"
	"io/ioutil"
)

// TODO:
// - file external file processing

// Decode decodes the stream data using the filters in the stream's dictionary.
func (s Stream) Decode() ([]byte, error) {
	// when there are no filters, it is already decoded
	if _, ok := s.Dictionary["Filter"]; !ok {
		return s.Stream, nil
	}

	// extract the list of filters to use
	filters := []Name{}
	switch streamFilter := s.Dictionary[Name("Filter")].(type) {
	case Name:
		filters = append(filters, streamFilter)
	case Array:
		for _, filter := range streamFilter {
			filters = append(filters, filter.(Name))
		}
	default:
		panic(fmt.Sprintf("unhandled type: %v", reflect.TypeOf(streamFilter).String()))
	}

	// extract the filter parameters
	parameters := []Dictionary{}
	if dict, ok := s.Dictionary[Name("DecodeParms")]; ok {
		switch streamParameter := dict.(type) {
		case Dictionary:
			parameters = append(parameters, streamParameter)
		case Array:
			for _, parameter := range streamParameter {
				parameters = append(parameters, parameter.(Dictionary))
			}
		default:
			panic(fmt.Sprintf("unhandled type: %v", reflect.TypeOf(streamParameter).String()))
		}
	}

	// apply the filters
	stream := s.Stream
	for i, filter := range filters {
		decoder, ok := decoders[filter]
		if !ok {
			return nil, errors.New("No decoder for " + string(filter))
		}

		parameter := Dictionary{}
		if i < len(parameters) {
			parameter = parameters[i]
		}

		var err error
		stream, err = decoder(stream, parameter)
		if err != nil {
			return nil, errors.New(string(filter) + ": " + err.Error())
		}
	}

	return stream, nil
}

var decoders = map[Name]func([]byte, Dictionary) ([]byte, error){
	Name("ASCII85Decode"): func(encoded []byte, dict Dictionary) ([]byte, error) {
		// the -3 strips the end of data marker
		return ioutil.ReadAll(ascii85.NewDecoder(bytes.NewBuffer(encoded[:len(encoded)-3])))
	},
	Name("FlateDecode"): func(encoded []byte, dict Dictionary) ([]byte, error) {
		return ioutil.ReadAll(flate.NewReader(bytes.NewBuffer(encoded[2:])))
	},
	// There is some problem with LZWDecode and TestFilterExample3
	// Name("LZWDecode"): func(encoded []byte, dict Dictionary) ([]byte, error) {
	// 	return ioutil.ReadAll(lzw.NewReader(bytes.NewBuffer(encoded[:len(encoded)-3]), lzw.MSB, 8))
	// },
}
