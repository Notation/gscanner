package state

import (
	"fmt"
	"gscanner/internal/smt"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
)

// Memory 内存
// EVM的内存操作单元32byte，大端序
// 这里连续的内存最小为1byte
// index -> byte，使用8bits的bitvec模拟1byte
type Memory struct {
	memory map[int64]*smt.BitVec
}

func NewMemory() *Memory {
	return &Memory{
		memory: make(map[int64]*smt.BitVec, 0),
	}
}

func (m *Memory) GetMemory() map[int64]*smt.BitVec {
	return m.memory
}

func (m *Memory) Size() int64 {
	return int64(len(m.memory))
}

func (m *Memory) Clone() *Memory {
	newMemory := &Memory{
		memory: make(map[int64]*smt.BitVec, 0),
	}
	for k, v := range m.memory {
		newMemory.memory[k] = v.Clone().AsBitVec()
	}
	return newMemory
}

func (m *Memory) Extend(size int64) {

}

// GetWordAt 返回index处长度为32byte的word
// 大端序
func (m *Memory) GetWordAt(index *smt.BitVec) (result *smt.BitVec) {
	for i := index.Value() + 31; i >= index.Value(); i-- {
		currentByte := m.memory[int64(i)]
		currentByte.RotateLeft()
		if result == nil {
			result = currentByte
			continue
		}
		result = smt.Concat(result, currentByte)
	}
	fmt.Println(yices2.TermToString(result.GetRaw(), 512, 30, 0))
	return result
}

func (m *Memory) WriteByteAt(index, value *smt.BitVec) error {
	if value.Size() != 8 {
		return fmt.Errorf("wrong value size: %d", value.Size())
	}
	// fmt.Println(index.Value(), "->", value.Value())
	m.memory[index.Value()] = value
	return nil
}

// writeWordAt 在index处写入长度为32byte的word
// write_word_at
// 布尔类型无法被存储，这里转换成整型再存储
func (m *Memory) WriteWordAt(index, value *smt.BitVec) error {
	termToWrite := value.GetRaw()
	termSize := yices2.TermBitsize(termToWrite)
	if yices2.TypeIsBool(yices2.TypeOfTerm(value.GetRaw())) {
		x := yices2.BvconstUint32(256, 1)
		y := yices2.BvconstUint32(256, 0)
		termToWrite = yices2.Ite(value.GetRaw(), x, y)
		termSize = yices2.TermBitsize(termToWrite)
	}
	// 32byte 256bit
	if termSize != uint32(256) {
		return fmt.Errorf("ErrorWrongParamType")
	}
	// 依次将数据放到连续的32个byte里
	// 大端序
	for i := 0; i < 32; i++ {
		k := index.Value()
		v := yices2.Bvextract(termToWrite, uint32(i*8), uint32(i*8+7))
		//fmt.Printf("%v -> %v, termsize %d\n", k, v, yices2.TermBitsize(v))
		m.memory[k] = smt.NewBitVecFromTerm(v, 8)
		index = index.AddInt64(int64(1))
	}
	return nil
}
