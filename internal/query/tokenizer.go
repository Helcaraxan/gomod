package query

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

var ErrUnclosedString = errors.New("missing quotes to close of a string")

type unclosedStringErr struct {
	str string
	pos Position
}

func (e *unclosedStringErr) Error() string {
	return fmt.Sprintf("unclosed string at position %v: %s", e.pos, e.str)
}

func (e *unclosedStringErr) Unwrap() error {
	return ErrUnclosedString
}

var ErrTokenizer = errors.New("unexpected tokenizer error")

type tokenizerErr struct {
	err error
	pos Position
}

func (e *tokenizerErr) Error() string {
	return fmt.Sprintf("tokenizer error at position %v: %v", e.pos, e.err)
}

func (e *tokenizerErr) Unwrap() error {
	return ErrTokenizer
}

type tokenizer struct {
	s *strings.Reader
}

func newTokenizer(s string) *tokenizer {
	return &tokenizer{s: strings.NewReader(s)}
}

// nolint: gocyclo
func (t *tokenizer) next() (tkn token, err error) {
	var s tokenString
	var r rune
	var p int64

	for {
		p = t.s.Size() - int64(t.s.Len())
		r, _, err = t.s.ReadRune()
		if err == io.EOF {
			return nil, io.EOF
		} else if err != nil {
			return nil, &tokenizerErr{err: err, pos: pos(p, t.s.Size()-int64(t.s.Len()))}
		}
		if !unicode.IsSpace(r) {
			break
		}
	}

	switch r {
	// Special characters.
	case '(':
		return &tokenParenLeft{p: pos(p, p+1)}, nil
	case ')':
		return &tokenParenRight{p: pos(p, p+1)}, nil
	case '-':
		return &tokenSubtract{p: pos(p, p+1)}, nil
	case '+':
		return &tokenUnion{p: pos(p, p+1)}, nil
	case ',':
		return &tokenComma{p: pos(p, p+1)}, nil

	// Quoted string.
	case '"', '\'':
		s, err = readString(t.s, string(r))
		if err != nil {
			return nil, &tokenizerErr{err: err, pos: pos(p, t.s.Size()-int64(t.s.Len()))}
		}
		if _, _, err = t.s.ReadRune(); err == io.EOF {
			return nil, &unclosedStringErr{str: string(s.v), pos: pos(p, s.p.end)}
		} else if err != nil {
			return nil, &tokenizerErr{err: err, pos: pos(p, t.s.Size()-int64(t.s.Len()))}
		}
		s.p = pos(s.p.start-1, s.p.end+1)
		return &s, nil

	// String-based token.
	default:
		if err = t.s.UnreadRune(); err != nil {
			return nil, &tokenizerErr{err: err, pos: pos(p, t.s.Size()-int64(t.s.Len()))}
		}

		s, err := readString(t.s, "()=,\"' \t\n")
		if err != nil {
			return nil, &tokenizerErr{err: err, pos: pos(p, t.s.Size()-int64(t.s.Len()))}
		}

		switch s.v {
		case "true":
			return &tokenBoolean{p: pos(p, p+4), v: true}, nil
		case "false":
			return &tokenBoolean{p: pos(p, p+5), v: false}, nil
		case "minus":
			return &tokenSubtract{p: pos(p, p+5)}, nil
		case "union":
			return &tokenUnion{p: pos(p, p+5)}, nil
		case "inter":
			return &tokenIntersect{p: pos(p, p+5)}, nil
		case "delta":
			return &tokenDelta{p: pos(p, p+5)}, nil
		default:
			if v, intErr := strconv.Atoi(string(s.v)); intErr == nil {
				return &tokenInteger{
					p: s.p,
					v: v,
				}, nil
			}
			return &s, nil
		}
	}
}

func readString(s *strings.Reader, eos string) (tokenString, error) {
	acc := strings.Builder{}
	p := s.Size() - int64(s.Len())
	for {
		r, _, err := s.ReadRune()
		if err == io.EOF {
			return tokenString{
				p: pos(p, s.Size()),
				v: acc.String(),
			}, nil
		} else if err != nil {
			return tokenString{}, err
		}

		if strings.ContainsRune(eos, r) {
			if err = s.UnreadRune(); err != nil {
				return tokenString{}, err
			}
			return tokenString{
				p: pos(p, s.Size()-int64(s.Len())),
				v: acc.String(),
			}, nil
		}

		if _, err = acc.WriteRune(r); err != nil {
			return tokenString{}, err
		}
	}
}
