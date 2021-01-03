package query

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"go.uber.org/zap"
)

var (
	ErrEmptyExpression       = errors.New("multiple expressions detected")
	ErrEmptyFuncCall         = errors.New("empty function call")
	ErrEmptyParenthesis      = errors.New("empty parenthesis")
	ErrInvalidArgument       = errors.New("invalid argument")
	ErrInvalidFuncName       = errors.New("invalid function name")
	ErrMissingArgument       = errors.New("missing argument")
	ErrMissingOperator       = errors.New("missing operator")
	ErrUnexpectedComma       = errors.New("unexpected comma")
	ErrUnexpectedOperator    = errors.New("unexpected operator")
	ErrUnexpectedParenthesis = errors.New("unexpected parenthesis")
)

func Parse(log *zap.Logger, query string) (Expr, error) {
	var stream []token

	r := newTokenizer(query)
	for {
		t, err := r.next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		stream = append(stream, t)
	}

	p := parser{
		log:       log,
		stream:    stream,
		streamIdx: 0,
		exprStack: []Expr{},
		ruleStack: []rule{},
	}
	return p.parse()
}

type parserError struct {
	err error
	pos Position
}

func (e *parserError) Error() string {
	return fmt.Sprintf("error at position %d: %s", e.pos, e.err)
}

func (e *parserError) Unwrap() error {
	return e.err
}

type parser struct {
	log       *zap.Logger
	stream    []token
	streamIdx int
	exprStack []Expr
	ruleStack []rule
}

func (p *parser) parse() (Expr, error) {
	for p.streamIdx < len(p.stream) {
		r, err := p.shift(p.stream[p.streamIdx])
		if err != nil {
			return nil, err
		}

		if r {
			if err = p.reduce(); err != nil {
				return nil, err
			}
		}

		if p.exprStackLength() < p.ruleStackLength() || p.exprStackLength() > p.ruleStackLength()+1 {
			return nil, &parserError{
				err: ErrMissingOperator,
				pos: p.stream[p.streamIdx].Pos(),
			}
		}
		p.streamIdx++
	}

	for len(p.ruleStack) > 0 {
		if err := p.reduce(); err != nil {
			return nil, err
		}
	}

	if len(p.exprStack) == 0 {
		return nil, &parserError{
			err: ErrEmptyExpression,
			pos: pos(0, 0),
		}
	}

	switch p.exprStack[0].(type) {
	case *ExprInteger, *ExprBool:
		return nil, &parserError{
			err: ErrInvalidArgument,
			pos: pos(0, p.stream[len(p.stream)-1].Pos().end),
		}
	default:
		return p.exprStack[0], nil
	}
}

type rule uint8

const (
	deltaRule     rule = iota // Expr delta Expr -> BinaryExpr
	intersectRule             // Expr inter Expr -> BinaryExpr
	unionRule                 // Expr + Expr -> BinaryExpr
	subtractRule              // Expr - Expr -> BinaryExpr
	argsListRule              // Expr, Expr -> ArgsListExpr
	funcRule                  // Expr(ArgListExpr) -> FuncExpr
	groupRule                 // (Expr) -> Expr
)

func (r rule) String() string {
	return map[rule]string{
		funcRule:      "func",
		groupRule:     "group",
		deltaRule:     "delta",
		intersectRule: "intersect",
		unionRule:     "union",
		subtractRule:  "subtract",
		argsListRule:  "arglist",
	}[r]
}

func (p *parser) shift(next token) (red bool, err error) {
	p.log.Debug("Computing shift.", zap.Stringer("token", next))
	switch v := next.(type) {
	case valueToken:
		return p.shiftValue(v)
	case punctuationToken:
		return p.shiftPunctuation(v)
	case operatorToken:
		return p.shiftOperator(v)
	default:
		return false, fmt.Errorf("unexpected token of type %T at %v", next, next.Pos())
	}
}

func (p *parser) shiftValue(next valueToken) (bool, error) {
	switch tv := next.(type) {
	case *tokenBoolean:
		p.exprStack = append(p.exprStack, &ExprBool{v: tv.v, p: next.Pos()})
	case *tokenInteger:
		p.exprStack = append(p.exprStack, &ExprInteger{v: tv.v, p: next.Pos()})
	case *tokenString:
		p.exprStack = append(p.exprStack, &ExprString{v: tv.v, p: next.Pos()})
	}
	p.log.Debug("Shifting value token onto stack.", zap.String("exprStack", p.exprStackString()))
	return false, nil
}

// nolint: gocyclo
func (p *parser) shiftPunctuation(next punctuationToken) (bool, error) {
	switch next.(type) {
	case *tokenComma:
		if len(p.exprStack) == 0 {
			return false, &parserError{
				err: ErrUnexpectedComma,
				pos: next.Pos(),
			}
		}

		if len(p.ruleStack) > 0 && p.ruleStack[len(p.ruleStack)-1] < argsListRule {
			p.log.Debug("Triggering reduce and forcing reprocessing of token.")
			p.streamIdx--
			return true, nil
		}

		p.ruleStack = append(p.ruleStack, argsListRule)
		p.log.Debug("Appending arglist rule.", zap.String("ruleStack", p.ruleStackString()))
		return false, nil

	case *tokenParenLeft:
		p.log.Debug("Computing stack-lengths", zap.Int("exprStackLength", p.exprStackLength()), zap.Int("ruleStackLength", p.ruleStackLength()))
		if p.exprStackLength() == p.ruleStackLength() {
			p.ruleStack = append(p.ruleStack, groupRule)
			p.log.Debug("Appended group rule.", zap.String("ruleStack", p.ruleStackString()))
		} else if p.exprStackLength() == p.ruleStackLength()+1 {
			p.ruleStack = append(p.ruleStack, funcRule)
			p.log.Debug("Appended func rule.", zap.String("ruleStack", p.ruleStackString()))
		}
		return false, nil

	case *tokenParenRight:
		var valid bool
		for _, r := range p.ruleStack {
			if r == groupRule || r == funcRule {
				valid = true
				break
			}
		}
		if !valid {
			return false, &parserError{
				err: ErrUnexpectedParenthesis,
				pos: next.Pos(),
			}
		}

		if p.ruleStack[len(p.ruleStack)-1] != groupRule && p.ruleStack[len(p.ruleStack)-1] != funcRule {
			p.log.Debug("Triggering reduce and forcing reprocessing of token.")
			p.streamIdx--
		} else {
			p.log.Debug("Triggering reduce.")
		}

		return true, nil

	default:
		return false, fmt.Errorf("unknown punctuation token of type %T at %v", next, next.Pos())
	}
}

func (p *parser) shiftOperator(next operatorToken) (bool, error) {
	if len(p.exprStack) == 0 {
		return false, &parserError{
			err: ErrUnexpectedOperator,
			pos: next.Pos(),
		}
	}

	var r rule
	switch next.(type) {
	case *tokenDelta:
		r = deltaRule
	case *tokenIntersect:
		r = intersectRule
	case *tokenUnion:
		r = unionRule
	case *tokenSubtract:
		r = subtractRule
	}

	if len(p.ruleStack) > 0 && p.ruleStack[len(p.ruleStack)-1] <= r {
		p.log.Debug("Triggering reduce and forcing reprocessing of token.")
		p.streamIdx--
		return true, nil
	}
	p.ruleStack = append(p.ruleStack, r)
	p.log.Debug("Appending operator rule.", zap.String("ruleStack", p.ruleStackString()))
	return false, nil
}

func (p *parser) reduce() error {
	if len(p.ruleStack) == 0 {
		p.log.Debug("Not reducing as rule stack is empty.")
		return nil
	}
	p.log.Debug("Reducing.", zap.String("ruleStack", p.ruleStackString()))

	var reduceFunc func() error
	switch p.ruleStack[len(p.ruleStack)-1] {
	case funcRule:
		reduceFunc = p.reduceFuncRule
	case groupRule:
		reduceFunc = p.reduceGroupRule
	case deltaRule, intersectRule, unionRule, subtractRule:
		reduceFunc = p.reduceOperatorRule(p.ruleStack[len(p.ruleStack)-1])
	case argsListRule:
		reduceFunc = p.reduceArgsListRule
	}
	return reduceFunc()
}

// nolint: gocyclo
func (p *parser) reduceFuncRule() error {
	if len(p.exprStack) < 2 {
		return &parserError{
			err: ErrEmptyFuncCall,
			pos: p.stream[p.streamIdx-1].Pos(),
		}
	}

	name := p.exprStack[len(p.exprStack)-2]
	switch name.(type) {
	case *ExprString:
		// Expected
	default:
		return &parserError{
			err: ErrInvalidFuncName,
			pos: p.stream[p.streamIdx].Pos(),
		}
	}

	args, ok := p.exprStack[len(p.exprStack)-1].(ArgsListExpr)
	if !ok {
		args = &ExprArgsList{values: []Expr{p.exprStack[len(p.exprStack)-1]}}
	}

	p.exprStack[len(p.exprStack)-2] = &ExprFunc{
		name: name.String(),
		args: args,
		p:    pos(name.Pos().start, p.stream[p.streamIdx-1].Pos().end),
	}

	p.exprStack = p.exprStack[:len(p.exprStack)-1]
	p.ruleStack = p.ruleStack[:len(p.ruleStack)-1]
	p.log.Debug("Reduced func rule.", zap.String("exprStack", p.exprStackString()))
	return nil
}

func (p *parser) reduceGroupRule() error {
	if len(p.exprStack) == 0 {
		return &parserError{
			err: ErrEmptyParenthesis,
			pos: p.stream[p.streamIdx].Pos(),
		}
	}

	p.ruleStack = p.ruleStack[:len(p.ruleStack)-1]
	p.log.Debug("Reduced group rule.", zap.String("ruleStack", p.ruleStackString()))
	return nil
}

func (p *parser) reduceOperatorRule(r rule) func() error {
	return func() error {
		if len(p.exprStack) < 2 {
			return &parserError{
				err: ErrMissingArgument,
				pos: p.stream[p.streamIdx-1].Pos(),
			}
		}

		operands := BinaryOperands{
			LHS: p.exprStack[len(p.exprStack)-2],
			RHS: p.exprStack[len(p.exprStack)-1],
		}
		npos := pos(operands.LHS.Pos().start, operands.RHS.Pos().end)

		for _, expr := range []Expr{operands.LHS, operands.RHS} {
			switch expr.(type) {
			case *ExprBool, *ExprInteger:
				return &parserError{
					err: ErrInvalidArgument,
					pos: expr.Pos(),
				}
			}
		}

		switch r {
		case deltaRule:
			p.exprStack[len(p.exprStack)-2] = &ExprDelta{BinaryOperands: operands, p: npos}
		case intersectRule:
			p.exprStack[len(p.exprStack)-2] = &ExprIntersect{BinaryOperands: operands, p: npos}
		case unionRule:
			p.exprStack[len(p.exprStack)-2] = &ExprUnion{BinaryOperands: operands, p: npos}
		case subtractRule:
			p.exprStack[len(p.exprStack)-2] = &ExprSubtract{BinaryOperands: operands, p: npos}
		}

		p.exprStack = p.exprStack[:len(p.exprStack)-1]
		p.ruleStack = p.ruleStack[:len(p.ruleStack)-1]
		p.log.Debug("Reduced top two arguments.", zap.String("exprStack", p.exprStackString()))
		return nil
	}
}

func (p *parser) reduceArgsListRule() error {
	if len(p.exprStack) < 2 {
		return &parserError{
			err: ErrMissingArgument,
			pos: p.stream[p.streamIdx-1].Pos(),
		}
	}

	arg0 := p.exprStack[len(p.exprStack)-2]
	argsList := &ExprArgsList{values: []Expr{arg0}}
	switch tExpr := p.exprStack[len(p.exprStack)-1].(type) {
	case ArgsListExpr:
		argsList.values = append(argsList.values, tExpr.Args()...)
	default:
		argsList.values = append(argsList.values, tExpr)
	}
	argsList.p = pos(argsList.values[0].Pos().start, argsList.values[len(argsList.values)-1].Pos().end)

	p.exprStack[len(p.exprStack)-2] = argsList
	p.exprStack = p.exprStack[:len(p.exprStack)-1]
	p.ruleStack = p.ruleStack[:len(p.ruleStack)-1]

	p.log.Debug("Reduced top-two expressions.", zap.String("exprStack", p.exprStackString()))
	return nil
}

func (p *parser) exprStackLength() int {
	return len(p.exprStack)
}

func (p *parser) ruleStackLength() int {
	var acc int
	for _, r := range p.ruleStack {
		switch r {
		case funcRule, deltaRule, intersectRule, unionRule, subtractRule, argsListRule:
			acc++
		default:
			// None
		}
	}
	return acc
}

func (p *parser) exprStackString() string {
	var strExpr []string
	for _, expr := range p.exprStack {
		strExpr = append(strExpr, expr.String())
	}
	return fmt.Sprintf("[%s]", strings.Join(strExpr, ", "))
}

func (p *parser) ruleStackString() string {
	var strRule []string
	for _, expr := range p.ruleStack {
		strRule = append(strRule, expr.String())
	}
	return fmt.Sprintf("[%s]", strings.Join(strRule, ", "))
}
