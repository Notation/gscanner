package solidity

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

const solTestFile = "../../solidity_examples/origin.sol"

const (
	InputDir = "../../testdata/input_contracts"
)

func Test_getSourceInfoWithoutName(t *testing.T) {
	inputFile := path.Join(InputDir, "multi_contracts.sol")
	contract, err := NewSolidityContract(inputFile, "")
	assert.Nil(t, err)
	codeInfo := contract.GetSourceInfo(116, false)
	assert.Equal(t, inputFile, contract.inputFile, "inputFile missmatch")
	assert.Equal(t, 14, codeInfo.LineNum, "wrong linenum")
	assert.Equal(t, "msg.sender.transfer(2 ether)", codeInfo.Code, "source code not equal")
}

func Test_getSourceInfoWithName(t *testing.T) {
	inputFile := path.Join(InputDir, "multi_contracts.sol")
	contract, err := NewSolidityContract(inputFile, "Transfer1")
	assert.Nil(t, err)
	codeInfo := contract.GetSourceInfo(116, false)
	assert.Equal(t, inputFile, contract.inputFile, "inputFile missmatch")
	assert.Equal(t, 6, codeInfo.LineNum, "wrong linenum")
	assert.Equal(t, "msg.sender.transfer(1 ether)", codeInfo.Code, "source code not equal")
}

func Test_getSourceInfoWithNameAndConstructor(t *testing.T) {
	inputFile := path.Join(InputDir, "constructor_assert.sol")
	contract, err := NewSolidityContract(inputFile, "AssertFail")
	assert.Nil(t, err)
	codeInfo := contract.GetSourceInfo(75, true)
	assert.Equal(t, inputFile, contract.inputFile, "inputFile missmatch")
	assert.Equal(t, 6, codeInfo.LineNum, "wrong linenum")
	assert.Equal(t, "assert(var1 > 0)", codeInfo.Code, "source code not equal")
}
