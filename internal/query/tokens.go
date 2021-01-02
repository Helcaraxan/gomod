package query

import "fmt"

type token interface {
	String() string
	Pos() Position
	_tokenImpl()
}

type Position struct {
	start int64
	end   int64
}

func pos(s int64, e int64) Position {
	return Position{
		start: s,
		end:   e,
	}
}

func (p *Position) String() string {
	if p.start != p.end {
		return fmt.Sprintf("%d-%d", p.start+1, p.end+1)
	}
	return fmt.Sprintf("%d", p.start+1)
}

type valueToken interface {
	token
	_valueTokenImpl()
}

type tokenBoolean struct {
	p Position
	v bool
}
type tokenInteger struct {
	p Position
	v int
}
type tokenString struct {
	p Position
	v string
}

func (t *tokenBoolean) Pos() Position    { return t.p }
func (t *tokenInteger) Pos() Position    { return t.p }
func (t *tokenString) Pos() Position     { return t.p }
func (t *tokenBoolean) String() string   { return fmt.Sprintf("%t", t.v) }
func (t *tokenInteger) String() string   { return fmt.Sprintf("%d", t.v) }
func (t *tokenString) String() string    { return t.v }
func (t *tokenBoolean) _tokenImpl()      {}
func (t *tokenInteger) _tokenImpl()      {}
func (t *tokenString) _tokenImpl()       {}
func (t *tokenBoolean) _valueTokenImpl() {}
func (t *tokenInteger) _valueTokenImpl() {}
func (t *tokenString) _valueTokenImpl()  {}

type punctuationToken interface {
	token
	_punctuationTokenImpl()
}

type tokenComma struct {
	p Position
}
type tokenParenLeft struct {
	p Position
}
type tokenParenRight struct {
	p Position
}

func (t *tokenComma) Pos() Position               { return t.p }
func (t *tokenParenLeft) Pos() Position           { return t.p }
func (t *tokenParenRight) Pos() Position          { return t.p }
func (t *tokenComma) String() string              { return ", " }
func (t *tokenParenLeft) String() string          { return "(" }
func (t *tokenParenRight) String() string         { return ")" }
func (t *tokenComma) _tokenImpl()                 {}
func (t *tokenParenLeft) _tokenImpl()             {}
func (t *tokenParenRight) _tokenImpl()            {}
func (t *tokenComma) _punctuationTokenImpl()      {}
func (t *tokenParenLeft) _punctuationTokenImpl()  {}
func (t *tokenParenRight) _punctuationTokenImpl() {}

type operatorToken interface {
	token
	_operatorTokenImpl()
}

type tokenDelta struct {
	p Position
}
type tokenIntersect struct {
	p Position
}
type tokenSubtract struct {
	p Position
}
type tokenUnion struct {
	p Position
}

func (t *tokenDelta) Pos() Position           { return t.p }
func (t *tokenIntersect) Pos() Position       { return t.p }
func (t *tokenSubtract) Pos() Position        { return t.p }
func (t *tokenUnion) Pos() Position           { return t.p }
func (t *tokenDelta) String() string          { return " delta " }
func (t *tokenIntersect) String() string      { return " inter " }
func (t *tokenSubtract) String() string       { return " - " }
func (t *tokenUnion) String() string          { return " + " }
func (t *tokenDelta) _tokenImpl()             {}
func (t *tokenIntersect) _tokenImpl()         {}
func (t *tokenSubtract) _tokenImpl()          {}
func (t *tokenUnion) _tokenImpl()             {}
func (t *tokenDelta) _operatorTokenImpl()     {}
func (t *tokenIntersect) _operatorTokenImpl() {}
func (t *tokenSubtract) _operatorTokenImpl()  {}
func (t *tokenUnion) _operatorTokenImpl()     {}

var (
	_ valueToken = &tokenBoolean{}
	_ valueToken = &tokenInteger{}
	_ valueToken = &tokenString{}

	_ punctuationToken = &tokenComma{}
	_ punctuationToken = &tokenParenLeft{}
	_ punctuationToken = &tokenParenRight{}

	_ operatorToken = &tokenDelta{}
	_ operatorToken = &tokenIntersect{}
	_ operatorToken = &tokenSubtract{}
	_ operatorToken = &tokenUnion{}
)
