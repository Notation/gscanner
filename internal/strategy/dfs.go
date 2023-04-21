// Package strategy 实现状态处理的策略
package strategy

import (
	"fmt"
	"gscanner/internal/ethereum/state"
)

// DFS 深度优先搜索策略
type DFS struct {
	states []*state.GlobalState
}

func NewDFS() *DFS {
	return &DFS{
		states: make([]*state.GlobalState, 0),
	}
}

func (dfs *DFS) Size() int {
	return len(dfs.states)
}

func (dfs *DFS) HasNext() bool {
	return len(dfs.states) > 0
}

func (dfs *DFS) Pop() (*state.GlobalState, error) {
	if len(dfs.states) <= 0 {
		return nil, fmt.Errorf("state queue is empty")
	}
	state := dfs.states[len(dfs.states)-1]
	dfs.states = dfs.states[:len(dfs.states)-1]
	return state, nil
}

func (dfs *DFS) Push(globalState ...*state.GlobalState) error {
	dfs.states = append(dfs.states, globalState...)
	return nil
}
