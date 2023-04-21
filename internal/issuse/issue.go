package issuse

import (
	"fmt"
	"gscanner/internal/solidity"
)

type Issuse struct {
	ID          string
	Title       string
	Description string

	Address int
	File    string
	Line    int
	Code    string
}

func (is *Issuse) AddCodeInfo(contract *solidity.SolidityContract) {
	codeInfo := contract.GetSourceInfo(is.Address, false)
	if codeInfo == nil {
		is.File = "Internal file"
		return
	}
	is.File = codeInfo.FileName
	is.Line = codeInfo.LineNum
	is.Code = codeInfo.Code
}

func (is *Issuse) String() string {
	swcDescription := fmt.Sprintf("ID: %s\nTitle: %s\nDescription: %s\n\n",
		is.ID, is.Title, is.Description)
	swcDescription = Colour(31, swcDescription)

	codeInfo := fmt.Sprintf("In file: %s:%d\n%s\n", is.File, is.Line, is.Code)
	codeInfo = Colour(33, codeInfo)

	return fmt.Sprintf("%s%s", swcDescription, codeInfo)
}

func Colour(color int, str string) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", color, str)
}
