package state

import (
	"fmt"
	"gscanner/internal/smt"
	"testing"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
	"github.com/stretchr/testify/assert"
)

func Test_MachineStack_Overflow(t *testing.T) {
	var mstack MachineStack
	for i := 0; i < STACK_SIZE; i++ {
		mstack.Push(nil)
	}
	err := mstack.Push(nil)
	assert.Equal(t, fmt.Errorf("ErrorStackOverflow"), err)
	err = mstack.Push(nil)
	assert.Equal(t, fmt.Errorf("ErrorStackOverflow"), err)
}

func Test_MachineStack_Underflow(t *testing.T) {
	var mstack MachineStack
	_, err := mstack.Pop()
	assert.Equal(t, fmt.Errorf("ErrorStackUnderflow"), err)
}

func Test_MachineStackExtendSize(t *testing.T) {
	var testCases = []struct {
		InitialSize   int64
		Start         int64
		ExtensionSize int64
	}{
		{0, 0, 10},
		{0, 30, 10},
		{100, 22, 8},
	}
	for _, tc := range testCases {
		var mstate MachineState
		mstate.gasLimit = 10000000
		mstate.memory = NewMemory()
		mstate.memory.Extend(tc.InitialSize)

		mstate.MemExtend(tc.Start, tc.ExtensionSize)

		size := tc.InitialSize
		extSize := (Ceil32(tc.Start+tc.ExtensionSize) / 32) * 32
		if size < extSize {
			size = extSize
		}
		// (ceil32(start + extension_size) // 32) * 32
		assert.Equal(t, mstate.MemSize(), mstate.memory.Size())
		assert.Equal(t, mstate.MemSize(), size)
	}
}

func Test_MachineStackSwap(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()
	var mstack MachineStack
	var (
		a = smt.NewBitVecValInt64WithName("a", 1, 256)
		b = smt.NewBitVecValInt64WithName("b", 2, 256)
		c = smt.NewBitVecValInt64WithName("c", 3, 256)
		d = smt.NewBitVecValInt64WithName("d", 4, 256)
		e = smt.NewBitVecValInt64WithName("e", 5, 256)
	)
	err := mstack.Push(a)
	assert.Nil(t, err)
	err = mstack.Push(b)
	assert.Nil(t, err)

	err = mstack.Swap(1)
	assert.Nil(t, err)

	r1, err := mstack.Get(0)
	assert.Nil(t, err)
	r2, err := mstack.Get(1)
	assert.Nil(t, err)
	assert.Equal(t, a.Value(), r2.AsBitVec().Value())
	assert.Equal(t, b.Value(), r1.AsBitVec().Value())

	mstack.Pop()
	mstack.Pop()

	// 1 2 3
	err = mstack.Push(a)
	assert.Nil(t, err)
	err = mstack.Push(b)
	assert.Nil(t, err)
	err = mstack.Push(c)
	assert.Nil(t, err)

	err = mstack.Swap(2)
	assert.Nil(t, err)

	r1, err = mstack.Get(0)
	assert.Nil(t, err)
	r2, err = mstack.Get(2)
	assert.Nil(t, err)
	assert.Equal(t, a.Value(), r2.AsBitVec().Value())
	assert.Equal(t, c.Value(), r1.AsBitVec().Value())

	mstack.Pop()
	mstack.Pop()
	mstack.Pop()

	// 1 2 3 4 5
	err = mstack.Push(a)
	assert.Nil(t, err)
	err = mstack.Push(b)
	assert.Nil(t, err)
	err = mstack.Push(c)
	assert.Nil(t, err)
	err = mstack.Push(d)
	assert.Nil(t, err)
	err = mstack.Push(e)
	assert.Nil(t, err)

	err = mstack.Swap(2)
	assert.Nil(t, err)

	r1, err = mstack.Get(2)
	assert.Nil(t, err)
	r2, err = mstack.Get(4)
	assert.Nil(t, err)
	assert.Equal(t, c.Value(), r2.AsBitVec().Value())
	assert.Equal(t, e.Value(), r1.AsBitVec().Value())
}

func Test_MachineStackDup(t *testing.T) {
	yices2.Init()
	defer yices2.Exit()
	var mstack MachineStack
	var (
		a = smt.NewBitVecValInt64WithName("a", 1, 256)
		b = smt.NewBitVecValInt64WithName("b", 2, 256)
		c = smt.NewBitVecValInt64WithName("c", 3, 256)
		d = smt.NewBitVecValInt64WithName("d", 4, 256)
		e = smt.NewBitVecValInt64WithName("e", 5, 256)
	)

	// 1 2 3 4 5
	err := mstack.Push(a)
	assert.Nil(t, err)
	err = mstack.Push(b)
	assert.Nil(t, err)
	err = mstack.Push(c)
	assert.Nil(t, err)
	err = mstack.Push(d)
	assert.Nil(t, err)
	err = mstack.Push(e)
	assert.Nil(t, err)

	err = mstack.Dup(5)
	assert.Nil(t, err)
}
