// Package strategy 实现状态处理的策略
package strategy

import (
	"gscanner/internal/ethereum/state"
)

type Strategy interface {
	Size() int
	HasNext() bool
	Pop() (*state.GlobalState, error)
	Push(...*state.GlobalState) error
}
