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

type AccidentallyKillable struct {
	*BaseModule
}

func NewAccidentallyKillable() *AccidentallyKillable {
	ak := &AccidentallyKillable{
		BaseModule: &BaseModule{
			swcData:    SWCDataMap["106"],
			entryPoint: CallbackEntryPoint,
			preHooks:   []string{"SELFDESTRUCT"},
		},
	}
	return ak
}

func (ak *AccidentallyKillable) Execute(globalState *state.GlobalState) (issuses []*issuse.Issuse, err error) {
	log.Info("Entering AccidentallyKillable")
	defer log.Info("Exiting AccidentallyKillable")

	defer func() {
		ak.Issuses = append(ak.Issuses, issuses...)
	}()

	// currentInstruction, err := globalState.GetCurrentInstruction()
	// if err != nil {
	// 	return nil, errors.Wrapf(err, "GetCurrentInstruction")
	// }

	to, err := globalState.MachineState.StackTop()
	if err != nil {
		return nil, errors.Wrapf(err, "StackTop")
	}

	var attackerConstraint []yices2.TermT

	for _, tx := range globalState.WorldState.TransactionSequence {
		if _, ok := tx.(*state.ContractCreationTransaction); ok {
			continue
		}
		f1 := state.Actors["ATTACKER"].Eq(tx.GetCaller())
		f2 := tx.GetCaller().Eq(tx.GetOrigin())
		term := yices2.And2(f1.GetRaw(), f2.GetRaw())
		attackerConstraint = append(attackerConstraint, term)
	}

	constraint := globalState.WorldState.GetConstraint()
	conditionA := append(constraint.GetAllConstraintTerms(), attackerConstraint...)
	conditionA = append(conditionA, state.Actors["ATTACKER"].Eq(to).GetRaw())

	solverA := smt.NewSolver()
	_, model, err := solverA.Check(conditionA...)
	if err != nil {
		fmt.Println("Check", err)
		return nil, err
	}
	if model != nil {
		return []*issuse.Issuse{{
			ID:          ak.swcData.ID,
			Title:       ak.swcData.Title,
			Description: ak.swcData.Description,
		}}, nil
	}

	solverB := smt.NewSolver()
	conditionB := append(constraint.GetAllConstraintTerms(), attackerConstraint...)
	_, model, err = solverB.Check(conditionB...)
	if err != nil {
		fmt.Println("Check", err)
		return nil, err
	}
	if model == nil {
		return nil, nil
	}

	currentInstruction, err := globalState.GetCurrentInstruction()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCurrentInstruction")
	}
	return []*issuse.Issuse{{
		ID:          ak.swcData.ID,
		Title:       ak.swcData.Title,
		Description: ak.swcData.Description,
		Address:     currentInstruction.Address,
	}}, nil
}
