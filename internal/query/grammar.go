package query

import (
	"fmt"
	"strings"
)

type Expr interface {
	String() string
	Pos() Position
	_expr()
}

type ValueExpr interface {
	Expr
	_valueExpr()
}

type ExprBool struct {
	v bool
	p Position
}
type ExprInteger struct {
	v int
	p Position
}
type ExprString struct {
	v string
	p Position
}

func (e *ExprBool) Value() bool       { return e.v }
func (e *ExprInteger) Value() int     { return e.v }
func (e *ExprString) Value() string   { return e.v }
func (e *ExprBool) String() string    { return fmt.Sprintf("%v", e.v) }
func (e *ExprInteger) String() string { return fmt.Sprintf("%v", e.v) }
func (e *ExprString) String() string  { return fmt.Sprintf("%v", e.v) }
func (e *ExprBool) Pos() Position     { return e.p }
func (e *ExprInteger) Pos() Position  { return e.p }
func (e *ExprString) Pos() Position   { return e.p }
func (e *ExprBool) _expr()            {}
func (e *ExprInteger) _expr()         {}
func (e *ExprString) _expr()          {}
func (e *ExprBool) _valueExpr()       {}
func (e *ExprInteger) _valueExpr()    {}
func (e *ExprString) _valueExpr()     {}

type BinaryExpr interface {
	Expr
	Operands() *BinaryOperands
}
type BinaryOperands struct {
	LHS Expr
	RHS Expr
}

type ExprDelta struct {
	BinaryOperands
	p Position
}
type ExprIntersect struct {
	BinaryOperands
	p Position
}
type ExprSubtract struct {
	BinaryOperands
	p Position
}
type ExprUnion struct {
	BinaryOperands
	p Position
}

func (e *ExprDelta) Operands() *BinaryOperands     { return &e.BinaryOperands }
func (e *ExprIntersect) Operands() *BinaryOperands { return &e.BinaryOperands }
func (e *ExprSubtract) Operands() *BinaryOperands  { return &e.BinaryOperands }
func (e *ExprUnion) Operands() *BinaryOperands     { return &e.BinaryOperands }
func (e *ExprDelta) String() string                { return fmt.Sprintf("(%v delta %v)", e.LHS, e.RHS) }
func (e *ExprIntersect) String() string            { return fmt.Sprintf("(%v inter %v)", e.LHS, e.RHS) }
func (e *ExprSubtract) String() string             { return fmt.Sprintf("(%v - %v)", e.LHS, e.RHS) }
func (e *ExprUnion) String() string                { return fmt.Sprintf("(%v + %v)", e.LHS, e.RHS) }
func (e *ExprDelta) Pos() Position                 { return e.p }
func (e *ExprIntersect) Pos() Position             { return e.p }
func (e *ExprSubtract) Pos() Position              { return e.p }
func (e *ExprUnion) Pos() Position                 { return e.p }
func (e *ExprDelta) _expr()                        {}
func (e *ExprIntersect) _expr()                    {}
func (e *ExprSubtract) _expr()                     {}
func (e *ExprUnion) _expr()                        {}

type FuncExpr interface {
	Expr
	Name() string
	Args() ArgsListExpr
}

type ExprFunc struct {
	name string
	args ArgsListExpr
	p    Position
}

func (e *ExprFunc) Name() string       { return e.name }
func (e *ExprFunc) Args() ArgsListExpr { return e.args }
func (e *ExprFunc) String() string     { return fmt.Sprintf("%s(%v)", e.name, e.args) }
func (e *ExprFunc) Pos() Position      { return e.p }
func (e *ExprFunc) _expr()             {}

type ArgsListExpr interface {
	Expr
	Args() []Expr
}

type ExprArgsList struct {
	values []Expr
	p      Position
}

func (e *ExprArgsList) Args() []Expr { return e.values }
func (e *ExprArgsList) String() string {
	var strArgs []string
	for _, arg := range e.values {
		strArgs = append(strArgs, arg.String())
	}
	return fmt.Sprintf("[%s]", strings.Join(strArgs, ", "))
}
func (e *ExprArgsList) Pos() Position { return e.p }
func (e *ExprArgsList) _expr()        {}

var (
	_ ValueExpr = &ExprBool{}
	_ ValueExpr = &ExprInteger{}
	_ ValueExpr = &ExprString{}

	_ BinaryExpr = &ExprSubtract{}
	_ BinaryExpr = &ExprUnion{}
	_ BinaryExpr = &ExprIntersect{}
	_ BinaryExpr = &ExprDelta{}

	_ FuncExpr = &ExprFunc{}

	_ ArgsListExpr = &ExprArgsList{}
)
