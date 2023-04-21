package gscanner

import (
	"fmt"
	"gscanner/internal/ethereum/state"
	"gscanner/internal/issuse"
	"gscanner/internal/module"
	"gscanner/internal/smt"
	"gscanner/internal/strategy"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Analyzer struct {
	moduleManager        *module.ModuleManager
	disassembler         *Disassembler
	worldStates          []*state.WorldState
	stateProcessStrategy strategy.Strategy
}

func NewAnalyzer(dm *module.ModuleManager, disassembler *Disassembler) *Analyzer {
	ma := &Analyzer{
		moduleManager:        dm,
		disassembler:         disassembler,
		worldStates:          make([]*state.WorldState, 0),
		stateProcessStrategy: strategy.NewDFS(),
	}
	return ma
}

// 执行合约函数
// 收集issuse
func (ma *Analyzer) Run() error {
	var (
		issuses []*issuse.Issuse
	)

	if len(ma.disassembler.contracts) == 0 {
		return fmt.Errorf("no contract found")
	}

	startTime := time.Now()

	for _, contract := range ma.disassembler.contracts {
		log.Infof("analyzing contract %s", contract.Name)
		var (
			worldState      = state.NewWorldState()
			creatorAccount  = state.NewAccount(state.Actors["CREATOR"], nil, smt.NewArray(), 0, "", false)
			attackerAccount = state.NewAccount(state.Actors["ATTACKER"], nil, smt.NewArray(), 0, "", false)
		)
		if contract.CreationCode != "" {
			worldState.PutAccount(creatorAccount)
			worldState.PutAccount(attackerAccount)
		} else {
			worldState.PutAccount(attackerAccount)
		}
		ma.executeContract("", worldState, contract.CreationCode, contract.Name)
		for _, is := range ma.RetrieveIssuses() {
			is.AddCodeInfo(contract)
			issuses = append(issuses, is)
		}
	}
	log.Infof("total issuses found: %d", len(issuses))
	for _, issuse := range issuses {
		fmt.Println(issuse)
	}
	fmt.Println("analyze time used: ", time.Since(startTime).Seconds())
	return nil
}

func (ma *Analyzer) RetrieveIssuses() []*issuse.Issuse {
	var result []*issuse.Issuse
	for _, module := range ma.moduleManager.CallbackModules {
		result = append(result, module.GetIssuses()...)
	}
	return result
}

func (ma Analyzer) executeContract(
	contractAddress string,
	worldState *state.WorldState,
	creationCode,
	contractName string) error {
	if contractAddress != "" {
		// 为链上运行预留的入口，链上已有合约，不需要再创建
		// return ma.executeMessageCallTx()
	} else {
		// 1. 创建合约
		contract, err := ma.executeContractCreationTx(creationCode, contractName, worldState)
		if err != nil {
			return err
		}
		fmt.Println("===========contract creation done============")
		// 2. 执行调用
		return ma.executeMessageCallTx(contract)
	}
	return nil
}

func (ma *Analyzer) executeContractCreationTx(
	contractCreationCode string,
	contractName string,
	worldState *state.WorldState) (contractAddress *state.Account, err error) {
	log.Infof("creating contract...")
	defer log.Infof("exit contract creating")
	globalState, newAccount, err := state.PrepareContractCreation(contractCreationCode, contractName, worldState)
	if err != nil {
		return nil, errors.Wrapf(err, "")
	}
	log.Infof("create contract: address[%s] bytecode[%s]", newAccount.Address.HexString(), newAccount.Code.GetBytecode())
	err = ma.exec([]*state.GlobalState{globalState})
	if err != nil {
		return nil, errors.Wrapf(err, "")
	}
	return newAccount.Clone(), nil
}

func (ma *Analyzer) executeMessageCallTx(contract *state.Account) error {
	newStates, err := state.PrepareMessageCall(ma.worldStates, contract.Address, nil)
	if err != nil {
		return errors.Wrapf(err, "PrepareMessageCall")
	}
	return ma.exec(newStates)
}

func (ma *Analyzer) exec(globalStates []*state.GlobalState) error {
	log.Infof("exec %d globalStates", len(globalStates))
	ma.stateProcessStrategy.Push(globalStates...)
	for {
		if !ma.stateProcessStrategy.HasNext() {
			break
		}
		globalState, _ := ma.stateProcessStrategy.Pop()
		newStates, _, err := ma.executeState(globalState)
		if err != nil {
			log.Errorf("executeState: %v", err)
			return errors.Wrapf(err, "executeState")
		}
		if len(newStates) > 1 {
			var filteredStates []*state.GlobalState
			for i := range newStates {
				if newStates[i].WorldState.IsConstraintPossible() {
					filteredStates = append(filteredStates, newStates[i])
				} else {
					fmt.Println()
				}
			}
		}
		ma.stateProcessStrategy.Push(newStates...)
	}

	return nil
}

func (ma *Analyzer) addWorldState(worldState *state.WorldState) {
	ma.worldStates = append(ma.worldStates, worldState)
}

func (ma *Analyzer) executeState(globalState *state.GlobalState) ([]*state.GlobalState, string, error) {
	instruction, err := globalState.GetCurrentInstruction()
	if err != nil {
		log.Infof("GetCurrentInstruction %v", err)
		ma.addWorldState(globalState.WorldState)
		return nil, "", nil
	}

	// 堆栈下溢：数据长度不足以执行指令
	if globalState.MachineState.StackSize() < instruction.RequiredArguments {
		return nil, instruction.OPCode, nil
	}

	var (
		preHooks  = ma.moduleManager.PreHooks[instruction.OPCode]
		postHooks = ma.moduleManager.PostHooks[instruction.OPCode]
		newStates = make([]*state.GlobalState, 0)
	)

	for _, hook := range preHooks {
		hook(globalState)
	}
	evaluateResult, err := state.NewInstruction(instruction.OPCode, nil, nil).Evaluate(globalState)
	if err != nil {
		log.Errorf("Evaluate %s: %v", instruction.OPCode, err)
		return nil, instruction.OPCode, nil
	}

	if evaluateResult.TxStart != nil {
		newGlobalState, err := evaluateResult.TxStart.Tx.InitialGlobalState()
		if err != nil {
			return nil, "", errors.Wrapf(err, "TxStart.Tx.InitialGlobalState")
		}
		newGlobalState.TransactionStack = globalState.TransactionStack.Clone()
		newGlobalState.TransactionStack.Push(&state.TxInfo{
			State: globalState,
			Tx:    evaluateResult.TxStart.Tx,
		})
		newGlobalState.WorldState.SetConstraint(globalState.WorldState.GetConstraint())
		log.Infof("starting new transaction %s", evaluateResult.TxStart.Tx.GetTxID())
		return []*state.GlobalState{newGlobalState}, instruction.OPCode, nil
	} else if evaluateResult.TxEnd != nil {
		txInfo := evaluateResult.TxEnd.GlobalState.GetCurrentTransaction()
		log.Infof("Ending %s", txInfo.Tx.String())
		if len(evaluateResult.GlobalStates) == 0 {
			_, isContractCreationTx := txInfo.Tx.(*state.ContractCreationTransaction)
			if (!isContractCreationTx || txInfo.Tx.GetReturnData() != "") && !evaluateResult.TxEnd.Revert {
				// some issuses should be checked here
				ma.addWorldState(evaluateResult.TxEnd.GlobalState.WorldState)
			}
		} else {
			var newAnnotations []smt.Annotation
			returnGlobalState := evaluateResult.TxEnd.GlobalState.Clone()
			for _, Annotation := range globalState.Annotations {
				if Annotation.PersistOverCalls() {
					newAnnotations = append(newAnnotations, Annotation)
				}
			}
			returnGlobalState.AddAnnotations(newAnnotations...)
			newStates, err = ma.endMessageCall(returnGlobalState, globalState,
				evaluateResult.TxEnd.Revert, evaluateResult.TxEnd.ReturnData)
			if err != nil {
				log.Errorf("endMessageCall %s: %v", instruction.OPCode, err)
				return nil, instruction.OPCode, nil
			}
		}
	} else {
		newStates = append(newStates, evaluateResult.GlobalStates...)
	}

	for _, hook := range postHooks {
		hook(globalState)
	}

	return newStates, instruction.OPCode, nil
}

func (ma *Analyzer) endMessageCall(
	returnState, globalState *state.GlobalState,
	revert bool,
	returnData *state.ReturnData,
) ([]*state.GlobalState, error) {
	log.Infof("ending message call")
	constraint := globalState.WorldState.GetConstraint()
	returnState.WorldState.AddConstraints(constraint.GetConstraints()...)
	instruction, err := returnState.GetCurrentInstruction()
	if err != nil {
		return nil, errors.Wrap(err, "GetCurrentInstruction")
	}
	returnState.LastReturnData = returnData.Clone()
	if !revert {
		returnState.WorldState = globalState.WorldState.Clone()
		account := globalState.GetAccounts()[returnState.Enviroment.ActiveAccount.Address.Value()].Clone()
		returnState.Enviroment.ActiveAccount = account
		txInfo := globalState.GetCurrentTransaction()
		if _, ok := txInfo.Tx.(*state.ContractCreationTransaction); ok {
			returnState.GasUsedMinAdd(globalState.MachineState.GetGasUsedMin())
			returnState.GasUsedMaxAdd(globalState.MachineState.GetGasUsedMax())
		}
	}
	evaluateResult, err := state.NewInstruction(instruction.OPCode+"Post", nil, nil).Evaluate(globalState)
	if err != nil {
		log.Errorf("endMessageCall Evaluate %s: %v", instruction.OPCode, err)
		return nil, nil
	}

	return evaluateResult.GlobalStates, nil
}
