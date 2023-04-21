package state

import "gscanner/internal/smt"

type ReturnData struct {
	Data []*smt.BitVec
	Size *smt.BitVec
}

func (returnData *ReturnData) Clone() *ReturnData {
	result := &ReturnData{
		Data: make([]*smt.BitVec, len(returnData.Data)),
		Size: returnData.Size.Clone().AsBitVec(),
	}
	for i, data := range returnData.Data {
		result.Data[i] = data.Clone().AsBitVec()
	}
	return result
}
