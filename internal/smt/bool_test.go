package smt

import (
	"fmt"
	"testing"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
)

func Test_Bool(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	a := NewBoolVal(true)
	b := NewBoolVal(false)
	fmt.Println(a.AsBitVec().Value())
	fmt.Println(b.AsBitVec().Value())

	var aval, bval int32
	yices2.BoolConstValue(a.GetRaw(), &aval)
	yices2.BoolConstValue(b.GetRaw(), &bval)
	fmt.Println(aval)
	fmt.Println(bval)
}
