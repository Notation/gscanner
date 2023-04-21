package solidity

import (
	"fmt"
	"os"
	"strings"

	"github.com/Notation/solc-go"
	"github.com/pkg/errors"
)

// for older version, compiler wrapper is not standard
// less than version 0.5.0, use compileJSON
// greater or equal 0.5.0 and less than 0.6.0, use solidity_compile('string', 'number')
// greater or equal 0.6.0, use solidity_compile('string', 'number', 'number')

// solc compiler input & output docs:
// https://docs.soliditylang.org/en/v0.5.0/using-the-compiler.html#compiler-input-and-output-json-description

const (
	SolcBinaryDir      = "/Users/harry/workspace/gscanner/solc_binary/"
	SolcBinaryMetaFile = "list.json"
	SolcBinaryEndpoint = "https://raw.githubusercontent.com/ethereum/solc-bin/gh-pages/wasm/"
)

func PrepareSolcBinary(version string) (string, error) {
	solcMeta, err := NewSolcBinaryMeta()
	if err != nil {
		return "", errors.Wrap(err, "NewSolcBinaryMeta")
	}
	solcFile, err := solcMeta.GetSolcBinary(version)
	if err != nil {
		return "", errors.Wrap(err, "GetSolcBinary")
	}
	return solcFile, nil
}

func GetSolcJson(file string) (*solc.Output, error) {
	fileData, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	version, err := ExtractVersionFromData(fileData)
	if err != nil {
		return nil, fmt.Errorf("ExtractVersionFromData: %v", err)
	}
	solcFile, err := PrepareSolcBinary(version)
	if err != nil {
		return nil, fmt.Errorf("PrepareSolcBinary: %v", err)
	}
	compiler, err := solc.NewFromFile(solcFile, strings.TrimPrefix(version, "^"))
	if err != nil {
		return nil, err
	}
	// defer compiler.Close()
	input := &solc.Input{
		Language: "Solidity",
		Sources: map[string]solc.SourceIn{
			file: {Content: string(fileData)},
		},
		Settings: solc.Settings{
			Optimizer: solc.Optimizer{
				Enabled: false,
			},
			OutputSelection: map[string]map[string][]string{
				"*": {
					"*": []string{
						"metadata",
						"evm.bytecode",
						"evm.deployedBytecode",
						"evm.methodIdentifiers",
					},
					"": []string{
						"ast",
					},
				},
			},
		},
	}
	return compiler.Compile(input)
}

func GetRandomAddress() string {
	return ""
}

func GetIndexedString(index int) string {
	return "0x" + strings.Repeat(fmt.Sprintf("%0x", index), 40)
}

const PragmaSolidity = "pragma solidity "

// ExtractVersionFromFile 提取版本号
func ExtractVersionFromFile(file string) (string, error) {
	fileData, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	return ExtractVersionFromData(fileData)
}

// ExtractVersionFromData 提取版本号
func ExtractVersionFromData(fileData []byte) (string, error) {
	lines := strings.Split(string(fileData), "\n")
	for i := range lines {
		if strings.HasPrefix(lines[i], PragmaSolidity) {
			pre := strings.TrimPrefix(lines[i], PragmaSolidity)
			return strings.TrimRight(pre, ";"), nil
		}
	}
	return "", nil
}
