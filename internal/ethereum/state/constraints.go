package state

import (
	"fmt"
	funcmanager "gscanner/internal/ethereum/function_managers"
	"gscanner/internal/smt"

	yices2 "github.com/ianamason/yices2_go_bindings/yices_api"
)

type Constraint struct {
	constraints []smt.Bool
}

func NewConstraints(constraints ...smt.Bool) *Constraint {
	c := &Constraint{
		constraints: make([]smt.Bool, len(constraints)),
	}
	copy(c.constraints, constraints)
	return c
}

func (c *Constraint) IsPossible() bool {
	solver := smt.NewSolver()
	status, _, _ := solver.Check(c.GetAllConstraintTerms()...)
	return status != yices2.StatusUnsat
}

func (c *Constraint) AppendBoolVal(val bool) {
	c.constraints = append(c.constraints, smt.NewBoolVal(val))
}

func (c *Constraint) AppendBool(val smt.Bool) {
	c.constraints = append(c.constraints, val)
}

func (c *Constraint) AppendBools(values ...smt.Bool) {
	c.constraints = append(c.constraints, values...)
}

func (c *Constraint) GetConstraints() []smt.Bool {
	return c.constraints
}

func (c *Constraint) Clone() *Constraint {
	return NewConstraints(c.constraints...)
}

func (c *Constraint) GetAllConstraints() []smt.Bool {
	result := make([]smt.Bool, len(c.constraints)+1)
	copy(result, c.constraints)
	return append(result, []smt.Bool{funcmanager.Kfm.CreateConditions()}...) //funcmanager.Kfm.CreateConditions()
}

func (c *Constraint) GetAllConstraintTerms() []yices2.TermT {
	// fmt.Printf("###################### GetAllConstraintTerms print start %d ######################\n", len(c.constraints))
	result := make([]yices2.TermT, len(c.constraints))
	for i := range c.constraints {
		result[i] = c.constraints[i].GetRaw()
		// yices2.PpTerm(os.Stdout, result[i], 1000, 80, 0)
	}
	// fmt.Printf("###################### GetAllConstraintTerms print end  %d ######################\n", len(c.constraints))
	// fmt.Println()
	return result
}

func (c *Constraint) PrintAllConstraintTerms() []yices2.TermT {
	fmt.Printf("###################### GetAllConstraintTerms print start %d ######################\n", len(c.constraints))
	result := make([]yices2.TermT, len(c.constraints))
	for i := range c.constraints {
		result[i] = c.constraints[i].GetRaw()
		// yices2.PpTerm(os.Stdout, result[i], 1000, 80, 0)
	}
	fmt.Printf("###################### GetAllConstraintTerms print end  %d ######################\n", len(c.constraints))
	fmt.Println()
	return result
}
