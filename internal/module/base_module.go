package module

import (
	"gscanner/internal/ethereum/state"
	"gscanner/internal/issuse"
)

const (
	PostEntryPoint     = 0 // 模块执行完成之后，再判断和取issue
	CallbackEntryPoint = 1
)

type BaseModule struct {
	swcData    *SWCData // SWC信息
	entryPoint int      // issue获取入口
	preHooks   []string // 在这些指令执行前，执行本模块的hook
	postHooks  []string // 在这些指令执行后，执行本模块的hook
	Issuses    []*issuse.Issuse
}

func (bm *BaseModule) Execute(globalState *state.GlobalState) ([]*issuse.Issuse, error) {
	return nil, nil
}

func (bm *BaseModule) GetPreHooks() []string {
	return bm.preHooks
}

func (bm *BaseModule) GetPostHooks() []string {
	return bm.postHooks
}

func (bm *BaseModule) GetEntryPoint() int {
	return bm.entryPoint
}

func (bm *BaseModule) GetSWCData() *SWCData {
	return bm.swcData
}

func (bm *BaseModule) GetIssuses() []*issuse.Issuse {
	return bm.Issuses
}

type DetectionModule interface {
	Execute(*state.GlobalState) ([]*issuse.Issuse, error)
	GetPreHooks() []string
	GetPostHooks() []string
	GetEntryPoint() int
	GetSWCData() *SWCData
	GetIssuses() []*issuse.Issuse
}
