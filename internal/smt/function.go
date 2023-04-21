package smt

import (
	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
)

// Function dom1,dom2,dom3... -> rng
type Function struct {
	name       string
	domain     []uint32
	valueRange uint32
	raw        yices2.TermT
}

func NewFunction(name string, domain []uint32, valueRange uint32) *Function {
	f := &Function{
		name:       name,
		domain:     make([]uint32, len(domain)),
		valueRange: valueRange,
	}
	copy(f.domain, domain)
	dom := make([]yices2.TypeT, len(domain))
	for i := range domain {
		dom[i] = yices2.BvType(domain[i])
	}
	funcType := yices2.FunctionType(dom, yices2.BvType(valueRange))
	f.raw = yices2.NewUninterpretedTerm(funcType)
	return f
}

func (f *Function) Call(items ...*BitVec) *BitVec {
	terms := make([]yices2.TermT, len(items))
	for i := range items {
		terms[i] = items[i].GetRaw()
	}
	return NewBitVecFromTerm(yices2.Application(f.raw, terms), 256)
}

func (f *Function) GetRaw() yices2.TermT {
	return f.raw
}
