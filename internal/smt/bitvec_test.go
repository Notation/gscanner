package smt

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/math"
	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
	"github.com/stretchr/testify/assert"
)

func Test_Concat(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	a := NewBitVecValInt64(1, 8)
	b := NewBitVecValInt64(1, 8)
	c := NewBitVecValInt64(1, 8)
	z := Concats([]*BitVec{a, b, c}...)
	fmt.Println(a.Value(), b.Value(), z.HexString(), z.TermType())
}

func Test_getBitVecVal(t *testing.T) {
	yices2.Init()

	const v uint32 = 666

	bv := yices2.BvconstUint32(256, v)
	val := getBitVecValue(bv)
	assert.Equal(t, v, uint32(val))

	yices2.Exit()
}

func Test_GetBigBvValue(t *testing.T) {
	yices2.Init()

	var terms = make([]yices2.TermT, 0)
	for i := 0; i < 32; i++ {
		p := math.BigPow(256, int64(i))

		v := make([]int32, p.BitLen())
		for j := 0; j < p.BitLen(); j++ {
			v[j] = int32(p.Bit(j))
		}
		assert.Equal(t, p.BitLen(), len(v))
		terms = append(terms, yices2.BvconstFromArray(v))
	}
	assert.Equal(t, 32, len(terms))

	for i := 0; i < 32; i++ {
		p := math.BigPow(256, int64(i))
		v := GetBigBvValue(terms[i])
		assert.Equal(t, p.String(), v.String())
	}

	bv := NewBitVecValInt64(0xFF, 256)
	fmt.Println(bv.String())
	fmt.Println(bv.HexString())

	yices2.Exit()
}

func Test_BitVecType(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	a := NewBitVecValInt64(1, 256)
	fmt.Println("a ", yices2.TypeOfTerm(a.GetRaw()))

	b := newBitVecValFromBigInt(big.NewInt(0), 256)
	fmt.Println("b ", yices2.TypeOfTerm(b.GetRaw()))

	array := NewArrayWithNameAndRange("balance", 256)
	err := array.Set(a, b)
	if err != nil {
		fmt.Println(err)
	}

	err = array.Set(NewBitVecValInt64(int64(0), 256).Clone().AsBitVec(), NewBitVecValInt64(int64(0), 256))
	if err != nil {
		fmt.Println(err)
	}

	b1, _ := hex.DecodeString("AFFEAFFEAFFEAFFEAFFEAFFEAFFEAFFEAFFEAFFE")
	b2, _ := hex.DecodeString("DEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEF")
	b3, _ := hex.DecodeString("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	Actors := map[string]*BitVec{
		"CREATOR":  NewBitVecValFromBytes(b1, 256),
		"ATTACKER": NewBitVecValFromBytes(b2, 256),
		"SOMEGUY":  NewBitVecValFromBytes(b3, 256),
	}
	creator := Actors["CREATOR"]
	err = array.Set(creator, b)
	if err != nil {
		fmt.Println(err)
	}

	c := a.Sub(b)
	fmt.Println("c ", yices2.TypeOfTerm(c.GetRaw()))
	t1 := yices2.BvType(256)
	t2 := yices2.BvType(256)
	term := yices2.FunctionType1(t1, t2)
	f1 := yices2.NewUninterpretedTerm(term)
	fmt.Println(yices2.ErrorString())
	fmt.Println("f1 ", yices2.TypeOfTerm(f1))
	f1 = yices2.Update1(f1, a.GetRaw(), b.GetRaw())
	fmt.Println(yices2.ErrorString())
	fmt.Println("f1 ", yices2.TypeOfTerm(f1))
}
