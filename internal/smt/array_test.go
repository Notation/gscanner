package smt

import (
	"fmt"
	"testing"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
	"github.com/stretchr/testify/assert"
)

func Test_array(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	var (
		ctx yices2.ContextT
		cfg yices2.ConfigT
	)
	yices2.InitConfig(&cfg)
	yices2.InitContext(cfg, &ctx)

	var (
		k1 = NewBitVecValInt64(1, 256)
		v1 = NewBitVecValInt64(100, 256)

		k2 = NewBitVecValInt64(10, 256)
		v2 = NewBitVecValInt64(1000, 256)

		v3 = NewBitVecValInt64(3333333, 256)
	)

	array := NewArray()
	err := array.Set(k1, v1)
	assert.Nil(t, err)
	err = array.Set(k2, v2)
	assert.Nil(t, err)
	vv1, err := array.Get(k1)
	assert.Nil(t, err)
	assert.Equal(t, v1.String(), vv1.String())

	brray := array

	// ============================
	vv1, _ = array.Get(k1)
	vv2, _ := array.Get(k2)
	fmt.Printf("%s %s\n", vv1.String(), vv2.String())

	vv1, _ = brray.Get(k1)
	vv2, _ = brray.Get(k2)
	fmt.Printf("%s %s\n", vv1.String(), vv2.String())
	fmt.Println("========1====================")

	// ============================
	err = brray.Set(k2, v3)
	vv1, _ = array.Get(k1)
	vv2, _ = array.Get(k2)
	fmt.Printf("%s %s\n", vv1.String(), vv2.String())

	vv1, _ = brray.Get(k1)
	vv2, _ = brray.Get(k2)
	fmt.Printf("%s %s\n", vv1.String(), vv2.String())
	fmt.Println("=========2===================")

	// ============================
	err = array.Set(k1, v3)
	vv1, _ = array.Get(k1)
	vv2, _ = array.Get(k2)
	fmt.Printf("%s %s\n", vv1.String(), vv2.String())

	vv1, _ = brray.Get(k1)
	vv2, _ = brray.Get(k2)
	fmt.Printf("%s %s\n", vv1.String(), vv2.String())
}

func Test_Type(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	// for i := uint32(0); i < 10; i++ {
	// 	t1 := yices2.BvType(DefaultBitVecSize)
	// 	t2 := yices2.BvType(DefaultBitVecSize)
	// 	term := yices2.FunctionType1(t1, t2)
	// 	f := yices2.NewUninterpretedTerm(term)
	// 	fmt.Println(f, t1, t2)
	// }

	a := NewBitVecValInt64(1, 128)
	b := NewBitVecValInt64(1, 128)
	c := Concats([]*BitVec{a, b}...)
	fmt.Println(a.TermType(), b.TermType(), c.TermType())
}
