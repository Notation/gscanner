package smt

import (
	"fmt"
	"os"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
)

type Model struct {
	ctxs []yices2.ContextT
}

func NewModel(ctxs ...yices2.ContextT) *Model {
	m := &Model{
		ctxs: make([]yices2.ContextT, 0, len(ctxs)),
	}
	m.ctxs = append(m.ctxs, ctxs...)
	return m
}

func (m *Model) Add(ctx yices2.ContextT) {
	m.ctxs = append(m.ctxs, ctx)
}

func (m *Model) Decls() []yices2.TermT {
	return nil
}

func (m *Model) Eval(term yices2.TermT) (yices2.SmtStatusT, *yices2.ModelT, error) {
	for i := range m.ctxs {
		isLastModel := i == len(m.ctxs)-1
		if isLastModel || isTermInContext(term, m.ctxs[i]) {
			return eval(m.ctxs[i], term, false)
		}
	}
	return yices2.StatusError, nil, nil
}

func (m *Model) ModelCompletionEval(term yices2.TermT) (yices2.SmtStatusT, *yices2.ModelT, error) {
	for i := range m.ctxs {
		isLastModel := i == len(m.ctxs)-1
		if isLastModel || isTermInContext(term, m.ctxs[i]) {
			return eval(m.ctxs[i], term, true)
		}
	}
	return yices2.StatusError, nil, nil
}

func eval(ctx yices2.ContextT, term yices2.TermT, addToModel bool) (yices2.SmtStatusT, *yices2.ModelT, error) {
	if !addToModel {
		yices2.Push(ctx)
	}

	// check
	errorcode := yices2.AssertFormula(ctx, term)
	if errorcode < 0 {
		return yices2.StatusError, nil, fmt.Errorf("%s", yices2.ErrorString())
	}
	status := yices2.CheckContext(ctx, yices2.ParamT{})
	if status != yices2.StatusSat {
		return yices2.StatusError, nil, fmt.Errorf("unsat")
	}
	model := yices2.GetModel(ctx, 1)
	if model == nil {
		return yices2.StatusError, nil, fmt.Errorf("get model error")
	}

	if !addToModel {
		yices2.Pop(ctx)
	}

	return status, model, nil
}

func isTermInContext(term yices2.TermT, ctx yices2.ContextT) bool {
	model := yices2.GetModel(ctx, 1)
	terms := yices2.ModelCollectDefinedTerms(*model)
	for i := range terms {
		if terms[i] == term {
			yices2.PpTerm(os.Stdout, terms[i], 200, 80, 0)
			fmt.Println("found term!")
			return true
		}
	}
	return false
}

func GetInt64Value(model *yices2.ModelT, term yices2.TermT) int64 {
	var val int64
	errcode := yices2.GetInt64Value(*model, term, &val)
	if errcode != 0 {
		fmt.Println(yices2.ErrorString())
	}
	return val
}

func GetBvValue(model *yices2.ModelT, term yices2.TermT) int64 {
	return GetBitVecTermValue(model, term)
}
