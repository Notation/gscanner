package funcmanager

import (
	"fmt"
	"gscanner/internal/smt"

	"github.com/ethereum/go-ethereum/common/math"
	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
)

type ExponentFunctionManager struct {
	concreteConstraints smt.Bool
}

func NewExponentFunctionManager() *ExponentFunctionManager {
	var (
		power     = smt.NewFunction("Power", []uint32{256, 256}, 256)
		number256 = smt.NewBitVecValInt64(256, 256)
		terms     = make([]yices2.TermT, 0, 32)
	)
	for i := 0; i < 32; i++ {
		p := math.BigPow(256, int64(i))
		v := make([]int32, p.BitLen())
		for j := 0; j < p.BitLen(); j++ {
			v[j] = int32(p.Bit(j))
		}
		if len(v) < 256 {
			v = append(v, make([]int32, 256-len(v))...)
		}
		t := yices2.Application2(power.GetRaw(), number256.GetRaw(), yices2.BvconstInt32(256, int32(i)))
		f := yices2.BveqAtom(t, yices2.BvconstFromArray(v))
		if f == yices2.NullTerm {
			fmt.Println(yices2.ErrorString())
		}
		terms = append(terms, f)
	}
	return &ExponentFunctionManager{
		concreteConstraints: smt.NewBoolFromTerm(yices2.And(terms)),
	}
}

func (efm *ExponentFunctionManager) CreateCondition(base, exponent *smt.BitVec) (*smt.BitVec, smt.Bool) {
	var (
		power          = smt.NewFunction("Power", []uint32{256, 256}, 256)
		exponentiation = yices2.Application2(power.GetRaw(), base.GetRaw(), exponent.GetRaw())
	)
	if !base.IsSymbolic() && !exponent.IsSymbolic() {
		fmt.Println("pow", base.Value(), exponent.Value())
		constExp := smt.NewBitVecValFromBigInt(math.Exp(base.GetBigInt(), exponent.GetBigInt()), 256)
		constraint := yices2.Eq(constExp.GetRaw(), exponentiation)
		return constExp, smt.NewBoolFromTerm(constraint)
	}
	f := yices2.BvgtAtom(exponentiation, yices2.Zero())
	// b := smt.NewBoolFromTerm(f)
	constraint := yices2.And2(f, efm.concreteConstraints.GetRaw())
	if base.Value() == 256 {

	}
	return smt.NewBitVecFromTerm(exponentiation, 256), smt.NewBoolFromTerm(constraint)
}
