package module

import (
	"fmt"
	"gscanner/internal/ethereum/state"
	"gscanner/internal/issuse"
	"gscanner/internal/smt"

	"github.com/ethereum/go-ethereum/common/math"
	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type TxOrigin struct {
	*BaseModule
}

func NewTxOrigin() *TxOrigin {
	txOrigin := &TxOrigin{
		BaseModule: &BaseModule{
			swcData:    SWCDataMap["115"],
			entryPoint: CallbackEntryPoint,
			preHooks:   []string{"JUMPI"},
			postHooks:  []string{"ORIGIN"},
			Issuses:    make([]*issuse.Issuse, 0),
		},
	}
	return txOrigin
}

func (txOrigin *TxOrigin) Execute(globalState *state.GlobalState) (issuses []*issuse.Issuse, err error) {
	log.Info("Entering TxOrigin")
	defer log.Info("Exiting TxOrigin")

	defer func() {
		txOrigin.Issuses = append(txOrigin.Issuses, issuses...)
	}()

	instruction, err := globalState.GetCurrentInstruction()
	if err != nil {
		return nil, err
	}
	if instruction.OPCode == "JUMPI" {
		// JUMPI pre hook
		var (
			index = globalState.MachineState.StackSize() - 2
		)
		op, err := globalState.MachineState.Get(index)
		if err != nil {
			return nil, errors.Wrapf(err, "Get")
		}
		var anotations *smt.Set
		if top, ok := op.(*smt.BitVec); ok {
			anotations = top.GetAnnotations()
		} else if top, ok := op.(*smt.Bool); ok {
			anotations = top.GetAnnotations()
		} else {
			panic("unkonwn storablet ype")
		}
		for ano := range anotations.GetElements() {
			if _, ok := ano.(*smt.TxOriginAnnotation); !ok {
				continue
			}
			log.Infof("try find issuse")
			txs := checkIssuse(globalState)
			if len(txs) == 0 {
				continue
			}
			issuses = append(issuses, &issuse.Issuse{
				ID:          txOrigin.swcData.ID,
				Title:       txOrigin.swcData.Title,
				Description: txOrigin.swcData.Description,
				Address:     instruction.Address,
			})
		}
		return issuses, nil
	} else {
		// ORIGIN post hook
		log.Infof("Origin marked")
		top, err := globalState.MachineState.StackTop()
		if err != nil {
			return nil, err
		}
		top.Anotate(&smt.TxOriginAnnotation{})
		fmt.Println()
	}
	return issuses, nil
}

func checkIssuse(globalState *state.GlobalState) []state.Transaction {
	constraint := globalState.WorldState.GetConstraint()
	constraint = constraint.Clone()

	var newConstraints []yices2.TermT

	for _, tx := range globalState.WorldState.TransactionSequence {
		maxCalldataSize := smt.NewBitVecValInt64(5000, 256)
		f1 := maxCalldataSize.Uge(tx.GetCalldata().CalldataSize())
		constraint.AppendBool(*f1)

		maxBalance := smt.NewBitVecValFromBigInt(math.BigPow(10, 20), 256)
		callerBalance, err := globalState.WorldState.GetBalance(tx.GetCaller())
		if err != nil {
			fmt.Println("GetBalance", err)
			continue
		}
		f2 := maxBalance.Uge(callerBalance)
		constraint.AppendBool(*f2)

		newConstraints = append(newConstraints, f1.GetRaw())
		newConstraints = append(newConstraints, f2.GetRaw())
		// yices2.PpTerm(os.Stdout, f1.GetRaw(), 1000, 80, 0)
		// yices2.PpTerm(os.Stdout, f2.GetRaw(), 1000, 80, 0)
	}
	fmt.Println("---------------")
	for _, account := range globalState.WorldState.GetAccounts() {
		maxBalance := smt.NewBitVecValFromBigInt(math.BigPow(10, 20), 256)
		accountBalance, err := globalState.WorldState.GetBalance(account.Address)
		if err != nil {
			fmt.Println("GetBalance", err)
			continue
		}
		f := maxBalance.Uge(accountBalance)
		constraint.AppendBool(*f)

		newConstraints = append(newConstraints, f.GetRaw())
		// yices2.PpTerm(os.Stdout, f.GetRaw(), 1000, 80, 0)
	}

	var (
		solver = smt.NewSolver()
	)
	_, model, err := solver.Check(constraint.GetAllConstraintTerms()...)
	if err != nil {
		fmt.Println("Check", err)
		return nil
	}

	if model == nil {
		return nil
	}

	//TODO: fill tx
	return []state.Transaction{nil}
}
