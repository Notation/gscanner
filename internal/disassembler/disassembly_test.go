package disassembler

import (
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_easm(t *testing.T) {
	const (
		InputDir         = "../../testdata/inputs"
		OuputExpectedDir = "../../testdata/outputs_expected"
		OuputCurrentDir  = "../../testdata/outputs_current"
	)
	files, err := ioutil.ReadDir(InputDir)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		var (
			inputFile          = path.Join(InputDir, f.Name())
			outputExpectedFile = path.Join(OuputExpectedDir, f.Name()+".easm")
			ouputCurrentFile   = path.Join(OuputCurrentDir, f.Name()+".easm")
		)
		fmt.Println(f.Name())
		// Read
		inputCode, err := ioutil.ReadFile(inputFile)
		if err != nil {
			log.Fatal(err)
		}

		// Parse and save
		disassembly := NewDisassembly(string(inputCode))

		err = ioutil.WriteFile(ouputCurrentFile, []byte(disassembly.GetEASM()), 0644)
		if err != nil {
			fmt.Println(err)
			continue
		}

		// Compare
		outputExpectedCode, err := ioutil.ReadFile(outputExpectedFile)
		if err != nil {
			log.Fatal(err)
		}
		assert.Equal(t, string(outputExpectedCode), disassembly.GetEASM(), fmt.Sprintf("%s != %s", inputFile, outputExpectedFile))
	}
	fmt.Println("total ", len(files))
}
