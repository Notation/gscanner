package module

import (
	"fmt"
	"gscanner/internal/ethereum/state"
	"gscanner/internal/issuse"
	"gscanner/internal/smt"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type ArbitraryJump struct {
	*BaseModule
}

func NewArbitraryJump() *ArbitraryJump {
	arbitraryJump := &ArbitraryJump{
		BaseModule: &BaseModule{
			swcData:    SWCDataMap["127"],
			entryPoint: CallbackEntryPoint,
			preHooks:   []string{"JUMP", "JUMPI"},
			Issuses:    make([]*issuse.Issuse, 0),
		},
	}
	return arbitraryJump
}

func (arbitraryJump *ArbitraryJump) Execute(globalState *state.GlobalState) (issuses []*issuse.Issuse, err error) {
	log.Info("Entering ArbitraryJump")
	defer log.Info("Exiting ArbitraryJump")

	defer func() {
		arbitraryJump.Issuses = append(arbitraryJump.Issuses, issuses...)
	}()

	jumpAddress, err := globalState.MachineState.StackTop()
	if err != nil {
		return nil, errors.Wrapf(err, "Get")
	}
	if !jumpAddress.IsSymbolic() {
		return nil, nil
	}

	if isUniqueJump(globalState, jumpAddress) {
		return nil, nil
	}

	currentInstruction, err := globalState.GetCurrentInstruction()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCurrentInstruction")
	}

	return []*issuse.Issuse{{
		ID:          arbitraryJump.swcData.ID,
		Title:       arbitraryJump.swcData.Title,
		Description: arbitraryJump.swcData.Description,
		Address:     currentInstruction.Address,
	}}, nil
}

func isUniqueJump(globalState *state.GlobalState, jumpAddress *smt.BitVec) bool {
	var (
		solver     = smt.NewSolver()
		constraint = globalState.WorldState.GetConstraint()
	)
	_, model, err := solver.Check(constraint.GetAllConstraintTerms()...)
	if err != nil {
		fmt.Println("Check", err)
		return true
	}
	if model == nil {
		return true
	}
	concreteJumpAddress := smt.GetBvValue(model, jumpAddress.GetRaw())
	f := yices2.Neq(jumpAddress.GetRaw(), smt.NewBitVecValFromInt64(concreteJumpAddress, 256).GetRaw())
	_, model, err = solver.Check(f)
	if err != nil {
		fmt.Println("Check", err)
		return true
	}
	if model == nil {
		return true
	}
	return false
}
