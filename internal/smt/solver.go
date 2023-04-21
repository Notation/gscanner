package smt

import (
	"fmt"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
)

type Solver struct {
	ctx yices2.ContextT
}

func NewSolver() *Solver {
	s := &Solver{
		ctx: yices2.ContextT{},
	}
	yices2.InitContext(yices2.ConfigT{}, &s.ctx)
	return s
}

func (s *Solver) Check(terms ...yices2.TermT) (yices2.SmtStatusT, *yices2.ModelT, error) {
	errorcode := yices2.AssertFormulas(s.ctx, terms)
	if errorcode < 0 {
		return yices2.StatusError, nil, fmt.Errorf("%s", yices2.ErrorString())
	}
	status := yices2.CheckContext(s.ctx, yices2.ParamT{})
	switch status {
	case yices2.StatusSat:
		return status, yices2.GetModel(s.ctx, 1), nil
	case yices2.StatusUnsat:
		fallthrough
	case yices2.StatusIdle:
		fallthrough
	case yices2.StatusSearching:
		fallthrough
	case yices2.StatusInterrupted:
		fallthrough
	case yices2.StatusError:
		return status, nil, nil
	}
	return yices2.StatusError, nil, nil
}

func (s *Solver) GetContext() yices2.ContextT {
	return s.ctx
}

func (s *Solver) GetInt64Value(model *yices2.ModelT, term yices2.TermT) (int64, error) {
	var val int64
	errcode := yices2.GetInt64Value(*model, term, &val)
	if errcode != 0 {
		return 0, fmt.Errorf(yices2.ErrorString())
	}
	return val, nil
}
