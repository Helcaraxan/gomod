package query

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSingleTokens(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		input         string
		expectedToken token
		expectedErr   error
	}{
		"StringSimple": {
			input:         "foo",
			expectedToken: &tokenString{p: pos(0, 3), v: "foo"},
			expectedErr:   nil,
		},
		"StringWithInteger": {
			input:         "42foo",
			expectedToken: &tokenString{p: pos(0, 5), v: "42foo"},
			expectedErr:   nil,
		},
		"StringWithSpecialCharacters": {
			input:         "foo-bar+dead",
			expectedToken: &tokenString{p: pos(0, 12), v: "foo-bar+dead"},
			expectedErr:   nil,
		},
		"StringQuotedDouble": {
			input:         "\"foo\"",
			expectedToken: &tokenString{p: pos(0, 5), v: "foo"},
			expectedErr:   nil,
		},
		"StringQuotedSingle": {
			input:         "'foo'",
			expectedToken: &tokenString{p: pos(0, 5), v: "foo"},
			expectedErr:   nil,
		},
		"StringQuotedDoubleUnclosed": {
			input:         "\"foo",
			expectedToken: nil,
			expectedErr:   ErrUnclosedString,
		},
		"StringQuotedSingleUnclosed": {
			input:         "'foo",
			expectedToken: nil,
			expectedErr:   ErrUnclosedString,
		},
		"Integer": {
			input:         "42",
			expectedToken: &tokenInteger{p: pos(0, 2), v: 42},
			expectedErr:   nil,
		},
		"SubSign": {
			input:         "-",
			expectedToken: &tokenSubtract{p: pos(0, 1)},
			expectedErr:   nil,
		},
		"SubString": {
			input:         "minus",
			expectedToken: &tokenSubtract{p: pos(0, 5)},
			expectedErr:   nil,
		},
		"UnionSign": {
			input:         "+",
			expectedToken: &tokenUnion{p: pos(0, 1)},
			expectedErr:   nil,
		},
		"UnionString": {
			input:         "union",
			expectedToken: &tokenUnion{p: pos(0, 5)},
			expectedErr:   nil,
		},
		"Inter": {
			input:         "inter",
			expectedToken: &tokenIntersect{p: pos(0, 5)},
			expectedErr:   nil,
		},
		"Delta": {
			input:         "delta",
			expectedToken: &tokenDelta{p: pos(0, 5)},
			expectedErr:   nil,
		},
		"ParenthesisLeft": {
			input:         "(",
			expectedToken: &tokenParenLeft{p: pos(0, 1)},
			expectedErr:   nil,
		},
		"ParenthesisRight": {
			input:         ")",
			expectedToken: &tokenParenRight{p: pos(0, 1)},
			expectedErr:   nil,
		},
		"Comma": {
			input:         ",",
			expectedToken: &tokenComma{p: pos(0, 1)},
			expectedErr:   nil,
		},
		"True": {
			input:         "true",
			expectedToken: &tokenBoolean{p: pos(0, 4), v: true},
			expectedErr:   nil,
		},
		"False": {
			input:         "false",
			expectedToken: &tokenBoolean{p: pos(0, 5), v: false},
			expectedErr:   nil,
		},
	}

	for name := range testcases {
		testcase := testcases[name]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tkn := newTokenizer(testcase.input)
			token, err := tkn.next()
			assert.Equal(t, testcase.expectedToken, token)
			assert.True(t, errors.Is(err, testcase.expectedErr))

			if testcase.expectedErr == nil {
				token, err = tkn.next()
				assert.Nil(t, token)
				assert.Equal(t, io.EOF, err)
			}
		})
	}
}

func TestTokenStream(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		input          string
		expectedTokens []token
		expectedErr    error
	}{
		"ParenthesizedExpression": {
			input: "deps(foo)",
			expectedTokens: []token{
				&tokenString{p: pos(0, 4), v: "deps"},
				&tokenParenLeft{p: pos(4, 5)},
				&tokenString{p: pos(5, 8), v: "foo"},
				&tokenParenRight{p: pos(8, 9)},
			},
			expectedErr: io.EOF,
		},
		"CommaSeparatedValues": {
			input: "foo, 42, bar",
			expectedTokens: []token{
				&tokenString{p: pos(0, 3), v: "foo"},
				&tokenComma{p: pos(3, 4)},
				&tokenInteger{p: pos(5, 7), v: 42},
				&tokenComma{p: pos(7, 8)},
				&tokenString{p: pos(9, 12), v: "bar"},
			},
			expectedErr: io.EOF,
		},
		"ParenthesizedExpressionComplex": {
			input: `deps("foo", 2, true) union rdeps(bar - dead/beef)`,
			expectedTokens: []token{
				&tokenString{p: pos(0, 4), v: "deps"},
				&tokenParenLeft{p: pos(4, 5)},
				&tokenString{p: pos(5, 10), v: "foo"},
				&tokenComma{p: pos(10, 11)},
				&tokenInteger{p: pos(12, 13), v: 2},
				&tokenComma{p: pos(13, 14)},
				&tokenBoolean{p: pos(15, 19), v: true},
				&tokenParenRight{p: pos(19, 20)},
				&tokenUnion{p: pos(21, 26)},
				&tokenString{p: pos(27, 32), v: "rdeps"},
				&tokenParenLeft{p: pos(32, 33)},
				&tokenString{p: pos(33, 36), v: "bar"},
				&tokenSubtract{p: pos(37, 38)},
				&tokenString{p: pos(39, 48), v: "dead/beef"},
				&tokenParenRight{p: pos(48, 49)},
			},
			expectedErr: io.EOF,
		},
		"UnclosedString": {
			input: `rdeps union( foo, "bar)`,
			expectedTokens: []token{
				&tokenString{p: pos(0, 5), v: "rdeps"},
				&tokenUnion{p: pos(6, 11)},
				&tokenParenLeft{p: pos(11, 12)},
				&tokenString{p: pos(13, 16), v: "foo"},
				&tokenComma{p: pos(16, 17)},
			},
			expectedErr: ErrUnclosedString,
		},
	}

	for name := range testcases {
		testcase := testcases[name]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := newTokenizer(testcase.input)

			var err error
			var tokens []token
			for {
				var tkn token
				tkn, err = r.next()
				if err != nil {
					break
				}
				tokens = append(tokens, tkn)
			}
			assert.Equal(t, testcase.expectedTokens, tokens)
			assert.True(t, errors.Is(err, testcase.expectedErr))
		})
	}
}
