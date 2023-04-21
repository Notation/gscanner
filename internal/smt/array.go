package smt

import (
	"fmt"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
)

// Yices的Array可以通过function实现

const (
	DefaultBitVecSize = 256
)

// Array implementation of symbolic array
type Array struct {
	Name string
	rng  uint32
	term yices2.TermT
}

func NewArray() Array {
	t1 := yices2.BvType(DefaultBitVecSize)
	t2 := yices2.BvType(DefaultBitVecSize)
	term := yices2.FunctionType1(t1, t2)
	arr := Array{
		rng:  DefaultBitVecSize,
		term: yices2.NewUninterpretedTerm(term),
	}
	// fmt.Println("NewArray ", yices2.TypeOfTerm(arr.term), " ", t1, " ", t2)
	return arr
}

func NewConcretArray() Array {
	t1 := yices2.BvType(DefaultBitVecSize)
	t2 := yices2.BvType(DefaultBitVecSize)
	term := yices2.FunctionType1(t1, t2)
	arr := Array{
		rng:  DefaultBitVecSize,
		term: yices2.NewUninterpretedTerm(term),
	}
	fmt.Println("NewConcretArray ", yices2.TypeOfTerm(arr.term), " ", t1, " ", t2)
	return arr
}

func NewArrayWithName(name string) Array {
	t1 := yices2.BvType(DefaultBitVecSize)
	t2 := yices2.BvType(DefaultBitVecSize)
	term := yices2.FunctionType1(t1, t2)
	yices2.SetTermName(yices2.TermT(term), name)
	arr := Array{
		rng:  DefaultBitVecSize,
		term: yices2.NewUninterpretedTerm(term),
	}
	fmt.Println("NewArrayWithName ", yices2.TypeOfTerm(arr.term), " ", t1, " ", t2)
	return arr
}

func NewArrayWithRange(rng uint32, size uint32) Array {
	t1 := yices2.BvType(DefaultBitVecSize)
	t2 := yices2.BvType(size)
	term := yices2.FunctionType1(t1, t2)
	arr := Array{
		rng:  uint32(rng),
		term: yices2.NewUninterpretedTerm(term),
	}
	fmt.Println("NewArrayWithRange ", yices2.TypeOfTerm(arr.term), " ", t1, " ", t2)
	return arr
}

func NewArrayWithNameAndRange(name string, rng uint32) Array {
	t1 := yices2.BvType(DefaultBitVecSize)
	t2 := yices2.BvType(rng)
	term := yices2.FunctionType1(t1, t2)
	// yices2.SetTermName(yices2.TermT(term), name)
	arr := Array{
		rng:  rng,
		term: yices2.NewUninterpretedTerm(term),
	}
	fmt.Println("NewArrayWithNameAndRange ", yices2.TypeOfTerm(arr.term), " ", t1, " ", t2)
	return arr
}

func NewConcreteArrayWithNameAndRange(name string, rng uint32) Array {
	t1 := yices2.BvType(DefaultBitVecSize)
	t2 := yices2.BvType(rng)
	term := yices2.FunctionType1(t1, t2)
	yices2.SetTermName(yices2.TermT(term), name)
	arr := Array{
		rng:  rng,
		term: yices2.NewUninterpretedTerm(term),
	}
	fmt.Println("NewConcreteArrayWithNameAndRange ", yices2.TypeOfTerm(arr.term), " ", t1, " ", t2)
	return arr
}

func (array *Array) GetRange() uint32 {
	return array.rng
}

func (array *Array) Get(index *BitVec) (*BitVec, error) {
	// type1 := yices2.TypeOfTerm(array.term)
	// type2 := yices2.TypeOfTerm(index.GetRaw())
	// fmt.Println("get: ", type1, " ", type2)
	term := yices2.Application1(array.term, index.GetRaw())
	if term == yices2.NullTerm {
		return nil, fmt.Errorf("%s", yices2.ErrorString())
	}

	// fmt.Println("Get ", yices2.TypeOfTerm(array.term), " ", yices2.TypeOfTerm(term))
	return NewBitVecFromTerm(term, array.rng), nil
}

func (array *Array) Set(index, value *BitVec) error {
	// fmt.Println("Set ", yices2.TypeOfTerm(array.term))
	// t1 := yices2.TypeOfTerm(array.term)
	// t2 := yices2.TypeOfTerm(index.value)
	// t3 := yices2.TypeOfTerm(value.value)
	// fmt.Println(t1, " ", t2, " ", t3, " index: ",
	// 	yices2.TypeIsBitvector(yices2.TypeOfTerm(index.value)), " ",
	// 	yices2.TypeIsBitvector(yices2.TypeOfTerm(index.value)))
	array.term = yices2.Update1(array.term, index.GetRaw(), value.GetRaw())
	errorcode := yices2.ErrorCode()
	if errorcode != 0 {
		fmt.Printf("array set: %s,index size %d, value size %d, index value %d, value type %d\n",
			yices2.ErrorString(), yices2.TermBitsize(index.GetRaw()), yices2.TermBitsize(value.GetRaw()), index.value, value.TermType())
	}
	bv, err := array.Get(index)
	if err != nil {
		panic(err)
	}
	fmt.Println("array set eq", bv.GetRaw() == value.GetRaw())
	return nil
}
