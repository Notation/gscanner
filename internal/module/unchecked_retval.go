package module

import (
	"fmt"
	"gscanner/internal/ethereum/state"
	"gscanner/internal/issuse"
	"gscanner/internal/smt"
	"os"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type UncheckedRetval struct {
	*BaseModule
}

func NewUncheckedRetval() *UncheckedRetval {
	ur := &UncheckedRetval{
		BaseModule: &BaseModule{
			swcData:    SWCDataMap["104"],
			entryPoint: CallbackEntryPoint,
			preHooks:   []string{"STOP", "RETURN"},
			postHooks:  []string{"CALL", "DELEGATECALL", "STATICCALL", "CALLCODE"},
			Issuses:    make([]*issuse.Issuse, 0),
		},
	}
	return ur
}

func (ur *UncheckedRetval) Execute(globalState *state.GlobalState) (issuses []*issuse.Issuse, err error) {
	log.Info("Entering UncheckedRetval")
	defer log.Info("Exiting UncheckedRetval")

	defer func() {
		ur.Issuses = append(ur.Issuses, issuses...)
	}()

	currentInstruction, err := globalState.GetCurrentInstruction()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCurrentInstruction")
	}

	var (
		annotations         = globalState.GetAnnotations()
		uncheckedRetvalAnno *smt.UncheckedRetvalAnnotation
	)
	for _, ano := range annotations {
		if uano, ok := ano.(*smt.UncheckedRetvalAnnotation); ok {
			uncheckedRetvalAnno = uano
			break
		}
	}
	if uncheckedRetvalAnno == nil {
		uncheckedRetvalAnno = &smt.UncheckedRetvalAnnotation{}
		globalState.AddAnnotation(uncheckedRetvalAnno)
	}

	if currentInstruction.OPCode == "STOP" || currentInstruction.OPCode == "RETURN" {
		var (
			constraint = globalState.WorldState.GetConstraint()
		)
		fmt.Println("UncheckedRetval len", len(constraint.GetAllConstraintTerms()))
		constraint.PrintAllConstraintTerms()
		for _, val := range uncheckedRetvalAnno.ReturnValues {
			conditionA := yices2.Eq(val.Value.GetRaw(), smt.NewBitVecValInt64(1, val.Value.Size()).GetRaw())
			solverA := smt.NewSolver()
			_, model, err := solverA.Check(append(constraint.GetAllConstraintTerms(), conditionA)...)
			if err != nil {
				fmt.Println("Check", err)
				return nil, err
			}
			if model == nil {
				continue
			}

			solverB := smt.NewSolver()
			conditionB := yices2.Eq(val.Value.GetRaw(), smt.NewBitVecValInt64(0, val.Value.Size()).GetRaw())
			_, model, err = solverB.Check(append(constraint.GetAllConstraintTerms(), conditionB)...)
			if err != nil {
				fmt.Println("Check", err)
				return nil, err
			}
			if model == nil {
				continue
			}
			return []*issuse.Issuse{{
				ID:          ur.swcData.ID,
				Title:       ur.swcData.Title,
				Description: ur.swcData.Description,
				Address:     currentInstruction.Address,
			}}, nil
		}
	} else {
		fmt.Println("End of call, extracting retval")
		pc := globalState.MachineState.GetPC()
		instructions := globalState.Enviroment.Code.GetInstructions()
		previousInstruction := instructions[pc-1]
		if previousInstruction.OPCode != "CALL" &&
			previousInstruction.OPCode != "DELEGATECALL" &&
			previousInstruction.OPCode != "STATICCALL" &&
			previousInstruction.OPCode != "CALLCODE" {
			return nil, nil
		}
		retval, err := globalState.MachineState.StackTop()
		if err != nil {
			return nil, errors.Wrapf(err, "StackTop")
		}
		uncheckedRetvalAnno.ReturnValues = append(uncheckedRetvalAnno.ReturnValues, &smt.ReturnValue{
			Address: previousInstruction.Address,
			Value:   retval.Clone().AsBitVec(),
		})
		fmt.Printf("UncheckedRetval apppend: address %d, value ", previousInstruction.Address)
		yices2.PpTerm(os.Stdout, retval.GetRaw(), 1000, 80, 0)
		fmt.Printf(" size %d", retval.Size())
	}
	return nil, nil
}
