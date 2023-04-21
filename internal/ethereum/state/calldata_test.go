package state

import (
	"gscanner/internal/smt"
	"testing"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
	"github.com/stretchr/testify/assert"
)

func Test_emptycalldata(t *testing.T) {
	yices2.Init()

	calldata := NewConcreteCalldata("0", []byte{})

	v1, err := calldata.GetWordAt(smt.NewBitVecValInt64(100, 256))
	assert.NotNil(t, err)
	assert.Equal(t, nil, v1)

	yices2.Exit()
}

func Test_calldatasize(t *testing.T) {
	yices2.Init()

	calldata := NewConcreteCalldata("0", []byte{1, 2, 3})
	s := calldata.Size()
	assert.Equal(t, int64(3), s.Value())

	yices2.Exit()
}

func Test_calldataConstrainIndex(t *testing.T) {
	yices2.Init()

	arr := []byte{1, 2, 3, 5, 5}

	calldata := NewConcreteCalldata("0", arr)
	s := calldata.Size()
	assert.Equal(t, int64(5), s.Value())

	for i := range arr {
		v, err := calldata.GetByteAt(smt.NewBitVecValInt64(int64(i), 256))
		assert.Nil(t, err)
		assert.Equal(t, int64(arr[i]), v.Value())
	}

	yices2.Exit()
}

func Test_symbolicCalldata(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()
	calldata := NewSymbolicCalldata("0")

	a := smt.NewBitVecValInt64(2, 256)
	b := smt.NewBitVecValInt64(2, 256)
	c := smt.NewBitVecValInt64(3, 256)

	av, err := calldata.GetByteAt(a)
	assert.Nil(t, err)
	bv, err := calldata.GetByteAt(b)
	assert.Nil(t, err)

	s := smt.NewSolver()
	f := yices2.BvneqAtom(av.GetRaw(), bv.GetRaw())

	status, model, err := s.Check(f)
	assert.Nil(t, err)
	assert.Nil(t, model)
	assert.Equal(t, yices2.StatusUnsat, status)

	calldata2 := NewSymbolicCalldata("1")
	err = calldata2.calldata.Set(a, c)
	assert.Nil(t, err)
}
