package state

import (
	"fmt"
	"gscanner/internal/smt"
	"math"
	"os"

	"github.com/ethereum/go-ethereum/params"
	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
)

const STACK_SIZE = 1024

type MachineStack struct {
	stack [STACK_SIZE]smt.StorableType
	index int
}

func NewMachineStack() *MachineStack {
	return &MachineStack{}
}

func (mstack *MachineStack) Size() int {
	return mstack.index
}

func (mstack *MachineStack) Push(element smt.StorableType) error {
	if mstack.index >= STACK_SIZE {
		return fmt.Errorf("ErrorStackOverflow")
	}
	mstack.stack[mstack.index] = element
	mstack.index++
	return nil
}

func (mstack *MachineStack) Dup(index int) error {
	if mstack.index >= STACK_SIZE {
		return fmt.Errorf("ErrorStackOverflow")
	}
	if index-1 < 0 || index-1 >= mstack.index {
		return fmt.Errorf("invalid dup index")
	}
	var (
		// 栈顶+1
		a = mstack.index

		// 第i个元素
		b = mstack.index - index
	)
	mstack.stack[a] = mstack.stack[b].Clone()
	mstack.index++
	return nil
}

func (mstack *MachineStack) Top() (smt.StorableType, error) {
	if mstack.index < 0 {
		return nil, fmt.Errorf("ErrorStackUnderflow")
	}
	term := mstack.stack[mstack.index-1]
	return term, nil
}

func (mstack *MachineStack) Get(index int) (smt.StorableType, error) {
	if mstack.index < 0 || index < 0 {
		return nil, fmt.Errorf("ErrorStackUnderflow")
	}
	if mstack.index <= index {
		return nil, fmt.Errorf("ErrorStackOverflow")
	}
	return mstack.stack[index], nil
}

func (mstack *MachineStack) Swap(index int) error {
	if index > STACK_SIZE {
		return fmt.Errorf("ErrorStackOverflow")
	}
	if index < 0 || mstack.index-1 < 0 {
		return fmt.Errorf("ErrorStackUnderflow")
	}
	// 1 2 3 4 5
	// mstack.index=5 index=3
	// a = 4
	// p_index = 2
	var (
		// 栈顶
		a = mstack.index - 1

		// 第i个元素
		b = mstack.index - index - 1
	)
	fmt.Printf("size: %d, %dth, swaping %d and %d\n", mstack.index, index, a, b)
	mstack.stack[a], mstack.stack[b] = mstack.stack[b], mstack.stack[a]
	return nil
}

func (mstack *MachineStack) Pop() (smt.StorableType, error) {
	if mstack.index-1 < 0 {
		return nil, fmt.Errorf("ErrorStackUnderflow")
	}
	mstack.index--
	element := mstack.stack[mstack.index]
	return element, nil
}

func (mstack *MachineStack) Clone() *MachineStack {
	s := &MachineStack{}
	s.index = mstack.index
	copy(s.stack[:], mstack.stack[:])
	return s
}

type MachineState struct {
	gasLimit        int64
	pc              int
	gasUsedMin      int64
	gasUsedMax      int64
	depth           int
	memory          *Memory
	stack           *MachineStack
	subroutineStack *MachineStack
}

func NewMachineState(gasLimit int64) *MachineState {
	return &MachineState{
		gasLimit:        gasLimit,
		pc:              0,
		gasUsedMin:      0,
		gasUsedMax:      0,
		depth:           0,
		memory:          NewMemory(),
		stack:           NewMachineStack(),
		subroutineStack: NewMachineStack(),
	}
}

func (ms *MachineState) GasUsedMinAdd(gas int64) {
	ms.gasUsedMin += gas
}

func (ms *MachineState) GasUsedMaxAdd(gas int64) {
	ms.gasUsedMax += gas
}

func (ms *MachineState) GetGasUsedMin() int64 {
	return ms.gasUsedMin
}

func (ms *MachineState) GetGasUsedMax() int64 {
	return ms.gasUsedMax
}

func (ms *MachineState) GetGasLimit() int64 {
	return ms.gasLimit
}

func (ms *MachineState) IncreasePC() {
	ms.pc += 1
}

func (ms *MachineState) Jump(dest int) {
	ms.pc = dest
}

func (ms *MachineState) GetPC() int {
	return ms.pc
}

func (ms *MachineState) CalcExtensionSize(start, end int64) int64 {
	if ms.memory.Size() > start+end {
		return 0
	}
	newSize := Ceil32(start+end) / 32
	old_size := ms.memory.Size()
	return (newSize - old_size) * 32
}

func (ms *MachineState) CalcMemoryGas(start, end int64) int64 {
	oldSize := ms.memory.Size() / 32
	oldTotalFee := oldSize*int64(params.MemoryGas) + int64(math.Pow(float64(oldSize), 2))/int64(params.QuadCoeffDiv)
	newSize := Ceil32(start+end) / 32
	newTotalFee := newSize*int64(params.MemoryGas) + int64(math.Pow(float64(newSize), 2))/int64(params.QuadCoeffDiv)
	return newTotalFee - oldTotalFee
}

func (ms *MachineState) CheckGas() error {
	if ms.gasUsedMin > ms.gasLimit {
		return fmt.Errorf("ErrorOutOfGas")
	}
	return nil
}

func (ms *MachineState) MemExtend(start, size int64) error {
	extendSize := ms.CalcExtensionSize(start, size)
	if extendSize == 0 {
		return nil
	}
	extendGas := ms.CalcMemoryGas(start, size)
	ms.gasUsedMin += extendGas
	ms.gasUsedMax += extendGas
	err := ms.CheckGas()
	if err != nil {
		return err
	}
	ms.memory.Extend(extendSize)
	return nil
}

// func (ms *MachineState) MemWrite(offset int, data []smt.StorableType) error {
// 	err := ms.MemExtend(offset, len(data))
// 	if err != nil {
// 		return err
// 	}
// 	return ms.memory.write(offset, data)
// }

func (ms *MachineState) StackSize() int {
	return ms.stack.Size()
}

func (ms *MachineState) MemSize() int64 {
	return ms.memory.Size()
}

func (ms *MachineState) MemWriteWordAt(index, value *smt.BitVec) error {
	return ms.memory.WriteWordAt(index, value)
}

func (ms *MachineState) MemWriteByte(index, value *smt.BitVec) error {
	return ms.memory.WriteByteAt(index, value)
}

func (ms *MachineState) MemWriteBytes(index *smt.BitVec, values ...*smt.BitVec) error {
	for i := 0; i < len(values); i++ {
		err := ms.memory.WriteByteAt(index.AddInt64(int64(i)), values[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func (ms *MachineState) MemGetWordAt(index *smt.BitVec) *smt.BitVec {
	return ms.memory.GetWordAt(index)
}

func (ms *MachineState) MemGetByteAt(index *smt.BitVec) *smt.BitVec {
	return ms.memory.memory[index.Value()]
}

func (ms *MachineState) PopStack() (smt.StorableType, error) {
	return ms.stack.Pop()
}

func (ms *MachineState) PushStack(element smt.StorableType) error {
	return ms.stack.Push(element)
}

func (ms *MachineState) Dup(index int) error {
	return ms.stack.Dup(index)
}

func (ms *MachineState) StackTop() (*smt.BitVec, error) {
	elem, err := ms.stack.Top()
	if err != nil {
		return nil, err
	}
	bv, ok := elem.(*smt.BitVec)
	if !ok {
		fmt.Println("GetBitVec ", elem.Type())
		return nil, fmt.Errorf("type missmatch: %s", elem.Type())
	}
	return bv, nil
}

func (ms *MachineState) GetBitVec(index int) (*smt.BitVec, error) {
	elem, err := ms.stack.Get(index)
	if err != nil {
		return nil, err
	}
	bv, ok := elem.(*smt.BitVec)
	if !ok {
		fmt.Println("GetBitVec ", elem.Type())
		return nil, fmt.Errorf("type missmatch: %s", elem.Type())
	}
	return bv, nil
}

func (ms *MachineState) Get(index int) (smt.StorableType, error) {
	return ms.stack.Get(index)
}

func (ms *MachineState) GetBitVec2(index1, index2 int) (*smt.BitVec, *smt.BitVec, error) {
	bv1, err := ms.GetBitVec(index1)
	if err != nil {
		return nil, nil, err
	}
	bv2, err := ms.GetBitVec(index2)
	if err != nil {
		return nil, nil, err
	}
	return bv1, bv2, nil
}

func (ms *MachineState) PopBitVec() (*smt.BitVec, error) {
	top, err := ms.stack.Top()
	if err != nil {
		return nil, err
	}
	bv, ok := top.(*smt.BitVec)
	if !ok {
		fmt.Println("PopBitVec ", top.Type())
		return nil, fmt.Errorf("type missmatch: %s", top.Type())
	}
	ms.stack.Pop()
	return bv, nil
}

func (ms *MachineState) Print(pc int) {
	fmt.Printf("\n###########stack info print start %d ###########\n", pc)
	for i := 0; i < ms.StackSize(); i++ {
		yices2.PpTerm(os.Stdout, ms.stack.stack[i].GetRaw(), 1000, 80, 0)
	}
	fmt.Printf("###########stack info print end   %d ###########\n", pc)
	fmt.Println()
}

func (ms *MachineState) PopBitVec2() (*smt.BitVec, *smt.BitVec, error) {
	bv1, err := ms.PopBitVec()
	if err != nil {
		return nil, nil, err
	}
	bv2, err := ms.PopBitVec()
	if err != nil {
		return nil, nil, err
	}
	return bv1, bv2, nil
}

func (ms *MachineState) PopBitVec3() (*smt.BitVec, *smt.BitVec, *smt.BitVec, error) {
	bv1, err := ms.PopBitVec()
	if err != nil {
		return nil, nil, nil, err
	}
	bv2, err := ms.PopBitVec()
	if err != nil {
		return nil, nil, nil, err
	}
	bv3, err := ms.PopBitVec()
	if err != nil {
		return nil, nil, nil, err
	}
	return bv1, bv2, bv3, nil
}

func (ms *MachineState) PopBool() (*smt.Bool, error) {
	top, err := ms.stack.Top()
	if err != nil {
		return nil, err
	}
	bv, ok := top.(*smt.Bool)
	if !ok {
		fmt.Println("PopBool ", top.Type())
		return nil, fmt.Errorf("type missmatch: %s", top.Type())
	}
	ms.stack.Pop()
	return bv, nil
}

func (ms *MachineState) Clone() *MachineState {
	return &MachineState{
		pc:              ms.pc,
		gasUsedMin:      ms.gasUsedMin,
		gasUsedMax:      ms.gasUsedMax,
		gasLimit:        ms.gasLimit,
		depth:           ms.depth,
		memory:          ms.memory.Clone(),
		stack:           ms.stack.Clone(),
		subroutineStack: ms.subroutineStack.Clone(),
	}
}
