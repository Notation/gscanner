package state

import (
	"fmt"
	"gscanner/internal/smt"
	"math/big"
	"testing"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
)

// var testStorageData = []struct {
// 	InitialStorage map[int32]int32
// 	Key            int32
// }{
// 	{map[int32]int32{1: 5}, 1},
// 	{map[int32]int32{1: 5}, 2},
// 	{map[int32]int32{1: 5, 3: 10}, 2},
// }

// func Test_storage(t *testing.T) {
// 	yices2.Init()

// 	var (
// 		storage       = NewConcreteStorage()
// 		k, v    int64 = 1, 1
// 		k1, v1  int64 = 1, 666
// 	)
// 	err := storage.Set(smt.NewBitVecValInt64(k, 256), smt.NewBitVecValInt64(v, 256))
// 	assert.Nil(t, err, "set error not nil")
// 	value, err := storage.Get(smt.NewBitVecValInt64(k, 256))
// 	assert.Nil(t, err, "get error not nil")
// 	assert.Equal(t, smt.NewBitVecValInt64(v, 256), value, "get value not equal")

// 	err = storage.Set(smt.NewBitVecValInt64(k1, 256), smt.NewBitVecValInt64(v1, 256))
// 	assert.Nil(t, err, "set error not nil")
// 	value, err = storage.Get(smt.NewBitVecValInt64(k1, 256))
// 	assert.Nil(t, err, "get error not nil")
// 	assert.Equal(t, smt.NewBitVecValInt64(v1, 256), value, "get value not equal")

// 	yices2.Exit()
// }

func Test_Balance(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()

	a1 := smt.NewArray()
	a2 := smt.NewArrayWithName("a2")
	a3 := smt.NewArrayWithNameAndRange("a3", 256)
	a4 := smt.NewArrayWithRange(256, 256)

	fmt.Println(a1, a2, a3, a4)

	a := smt.NewBitVecValInt64(1, 256)
	fmt.Println("a ", yices2.TypeOfTerm(a.GetRaw()))

	b := smt.NewBitVecValFromBigInt(big.NewInt(0), 256)
	fmt.Println("b ", yices2.TypeOfTerm(b.GetRaw()))

	array := smt.NewArrayWithNameAndRange("balance", 256)
	err := array.Set(a, b)
	if err != nil {
		fmt.Println(err)
	}

	err = array.Set(smt.NewBitVecValInt64(int64(0), 256).Clone().AsBitVec(), smt.NewBitVecValInt64(int64(0), 256))
	if err != nil {
		fmt.Println(err)
	}

	// b1, _ := hex.DecodeString("AFFEAFFEAFFEAFFEAFFEAFFEAFFEAFFEAFFEAFFE")
	// b2, _ := hex.DecodeString("DEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEF")
	// b3, _ := hex.DecodeString("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	// Actors := map[string]*smt.BitVec{
	// 	"CREATOR":  smt.NewBitVecValFromBytes(b1, 256),
	// 	"ATTACKER": smt.NewBitVecValFromBytes(b2, 256),
	// 	"SOMEGUY":  smt.NewBitVecValFromBytes(b3, 256),
	// }
	Actors := map[string]*smt.BitVec{
		"CREATOR":  smt.NewBitVecValInt64(9999, 256),
		"ATTACKER": smt.NewBitVecValInt64(8888, 256),
		"SOMEGUY":  smt.NewBitVecValInt64(7777, 256),
	}
	creator := Actors["CREATOR"]
	err = array.Set(creator, b)
	if err != nil {
		fmt.Println(err)
	}

	worldState := NewWorldState()
	account := worldState.CreateAccount(
		0,
		smt.NewBitVecValInt64(1, 256),
		true,
		nil,
		nil,
		0)
	_ = smt.NewArray()
	account.SetBalance(smt.NewBitVecValInt64(int64(0), 256))
	d := smt.NewBitVec("d", 256)
	account.SetBalance(d)
	account.SetBalance(a)
	account.SetBalance(b)
}
