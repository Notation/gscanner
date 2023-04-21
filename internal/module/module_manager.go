package module

import (
	"gscanner/internal/ethereum/state"
	"gscanner/internal/issuse"
)

type Hook func(*state.GlobalState) ([]*issuse.Issuse, error)

type ModuleManager struct {
	PostModules     []DetectionModule
	CallbackModules []DetectionModule
	PreHooks        map[string][]Hook
	PostHooks       map[string][]Hook
}

func NewModuleManager() *ModuleManager {
	return &ModuleManager{
		PostModules:     make([]DetectionModule, 0),
		CallbackModules: make([]DetectionModule, 0),
		PreHooks:        make(map[string][]Hook),
		PostHooks:       make(map[string][]Hook),
	}
}

func (mm *ModuleManager) AddModule(dm DetectionModule) {
	if dm.GetEntryPoint() == PostEntryPoint {
		mm.PostModules = append(mm.PostModules, dm)
	} else if dm.GetEntryPoint() == CallbackEntryPoint {
		mm.CallbackModules = append(mm.CallbackModules, dm)
	}
	for _, opCode := range dm.GetPreHooks() {
		mm.PreHooks[opCode] = append(mm.PreHooks[opCode], dm.Execute)
	}
	for _, opCode := range dm.GetPostHooks() {
		mm.PostHooks[opCode] = append(mm.PostHooks[opCode], dm.Execute)
	}
}
