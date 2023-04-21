package smt

import (
	"fmt"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
)

type Bool struct {
	name        string
	value       yices2.TermT
	annotations *Set
}

func NewBoolVal(value bool, annotations ...Annotation) Bool {
	if value {
		return Bool{
			value:       yices2.True(),
			annotations: NewSet(annotations...),
		}
	}
	return Bool{
		value:       yices2.False(),
		annotations: NewSet(annotations...),
	}
}

func NewBool(name string, annotations ...Annotation) Bool {
	term := yices2.NewVariable(yices2.BoolType())
	errcode := yices2.SetTermName(term, name)
	if errcode < 0 {
		fmt.Println("set term name ", errcode)
	}
	return Bool{
		value:       term,
		annotations: NewSet(annotations...),
	}
}

func NewBoolFromTerm(term yices2.TermT, annotations ...Annotation) Bool {
	return Bool{
		value:       term,
		annotations: NewSet(annotations...),
	}
}

func (b *Bool) Clone() StorableType {
	return &Bool{
		name:        b.name,
		value:       b.value,
		annotations: b.annotations.Clone(),
	}
}

func (b *Bool) AsBitVec() *BitVec {
	var bv *BitVec
	if b.IsSymbolic() {
		term := yices2.Ite(b.value, yices2.BvconstInt64(256, 1), yices2.BvconstInt64(256, 0))
		bv = NewBitVecFromTerm(term, 256)
		bv.name = b.name
		bv.annotations = b.annotations.Clone()
	} else {
		if b.IsTrue() {
			bv = NewBitVecValInt64(1, 256)
		} else {
			bv = NewBitVecValInt64(0, 256)
		}
		bv.name = b.name
		bv.annotations = b.annotations.Clone()
	}
	return bv
}

func (b *Bool) AsBool() Bool {
	return *b
}

func (b *Bool) GetRaw() yices2.TermT {
	return b.value
}

func (bv *Bool) Type() string {
	return BoolType
}

func (bv *Bool) Size() uint32 {
	return 0
}

func (bv *Bool) Not() *Bool {
	return &Bool{
		name:        "",
		value:       yices2.Not(bv.value),
		annotations: bv.annotations.Clone(),
	}
}

func (b *Bool) IsFalse() bool {
	var val int32
	errcode := yices2.BoolConstValue(b.value, &val)
	if errcode != 0 {
		fmt.Println(yices2.ErrorString())
	}
	// return val == 0
	return val == 0
}

func (b *Bool) IsTrue() bool {
	var val int32
	errcode := yices2.BoolConstValue(b.value, &val)
	if errcode != 0 {
		fmt.Println("errocode ", errcode, ", ", yices2.ErrorString(), ", type ", yices2.TypeOfTerm(b.value))
	}
	// return val == 1
	return val != 0
}

func (b *Bool) Value() bool {
	if b.IsTrue() {
		return true
	}
	if b.IsFalse() {
		return false
	}
	return false
}

func (b *Bool) IsSymbolic() bool {
	termC := yices2.TermConstructor(b.value)
	return yices2.TrmCnstrBoolConstant != termC
}

func (b *Bool) GetAnnotations() *Set {
	return b.annotations
}
