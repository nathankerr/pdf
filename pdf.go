package pdf

import(
	"io/ioutil"
)

func Open(filename string) (File, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	p := yyParser {
		File: make(File, 0),
	}
	p.Init()
	p.ResetBuffer(string(data))

	err = p.Parse(ruleFile)
	if err != nil {
		return nil, err
	}

	return p.File, nil

}