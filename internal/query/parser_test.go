package query

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Helcaraxan/gomod/internal/testutil"
)

func TestParser(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		input        string
		expectedExpr Expr
	}{
		"SimpleString": {
			input:        "foo/bar",
			expectedExpr: &ExprString{v: "foo/bar"},
		},
		"SimpleOperator": {
			input: "foo/bar + dead/beef",
			expectedExpr: &ExprUnion{BinaryOperands: BinaryOperands{
				LHS: &ExprString{v: "foo/bar"},
				RHS: &ExprString{v: "dead/beef"},
			}},
		},
		"MultipleOperators": {
			input: "foo - bar delta dead + beef inter null",
			expectedExpr: &ExprSubtract{BinaryOperands: BinaryOperands{
				LHS: &ExprString{v: "foo"},
				RHS: &ExprUnion{BinaryOperands: BinaryOperands{
					LHS: &ExprDelta{BinaryOperands: BinaryOperands{
						LHS: &ExprString{v: "bar"},
						RHS: &ExprString{v: "dead"},
					}},
					RHS: &ExprIntersect{BinaryOperands: BinaryOperands{
						LHS: &ExprString{v: "beef"},
						RHS: &ExprString{v: "null"},
					}},
				}},
			}},
		},
		"OperatorsAndParenthesises": {
			input: "foo + bar delta (dead - beef) inter null",
			expectedExpr: &ExprUnion{BinaryOperands: BinaryOperands{
				LHS: &ExprString{v: "foo"},
				RHS: &ExprIntersect{BinaryOperands: BinaryOperands{
					LHS: &ExprDelta{BinaryOperands: BinaryOperands{
						LHS: &ExprString{v: "bar"},
						RHS: &ExprSubtract{BinaryOperands: BinaryOperands{
							LHS: &ExprString{v: "dead"},
							RHS: &ExprString{v: "beef"},
						}},
					}},
					RHS: &ExprString{v: "null"},
				}},
			}},
		},
		"SimpleFuncCallOneArg": {
			input: "deps(foo)",
			expectedExpr: &ExprFunc{
				name: "deps",
				args: &ExprArgsList{values: []Expr{
					&ExprString{v: "foo"},
				}},
			},
		},
		"SimpleFuncCallTwoArgs": {
			input: "deps(foo, 7)",
			expectedExpr: &ExprFunc{
				name: "deps",
				args: &ExprArgsList{values: []Expr{
					&ExprString{v: "foo"},
					&ExprInteger{v: 7},
				}},
			},
		},
		"SimpleFuncCallThreeArgs": {
			input: "deps(foo, 42, true)",
			expectedExpr: &ExprFunc{
				name: "deps",
				args: &ExprArgsList{values: []Expr{
					&ExprString{v: "foo"},
					&ExprInteger{v: 42},
					&ExprBool{v: true},
				}},
			},
		},
		"NestedFuncCalls": {
			input: "foo(bar(test))",
			expectedExpr: &ExprFunc{
				name: "foo",
				args: &ExprArgsList{values: []Expr{
					&ExprFunc{
						name: "bar",
						args: &ExprArgsList{values: []Expr{
							&ExprString{v: "test"},
						}},
					},
				}},
			},
		},
		"ComplexFuncCall": {
			input: "foo((bar - dead) delta beef + null, 3, true)",
			expectedExpr: &ExprFunc{
				name: "foo",
				args: &ExprArgsList{values: []Expr{
					&ExprUnion{BinaryOperands: BinaryOperands{
						LHS: &ExprDelta{BinaryOperands: BinaryOperands{
							LHS: &ExprSubtract{BinaryOperands: BinaryOperands{
								LHS: &ExprString{v: "bar"},
								RHS: &ExprString{v: "dead"},
							}},
							RHS: &ExprString{v: "beef"},
						}},
						RHS: &ExprString{v: "null"},
					}},
					&ExprInteger{v: 3},
					&ExprBool{v: true},
				}},
			},
		},
		"ComplexExpression": {
			input: "deps(foo) inter (rdeps(bar, 5, true) + dead) - beef",
			expectedExpr: &ExprSubtract{BinaryOperands: BinaryOperands{
				LHS: &ExprIntersect{BinaryOperands: BinaryOperands{
					LHS: &ExprFunc{
						name: "deps",
						args: &ExprArgsList{values: []Expr{
							&ExprString{v: "foo"},
						}},
					},
					RHS: &ExprUnion{BinaryOperands: BinaryOperands{
						LHS: &ExprFunc{
							name: "rdeps",
							args: &ExprArgsList{values: []Expr{
								&ExprString{v: "bar"},
								&ExprInteger{v: 5},
								&ExprBool{v: true},
							}},
						},
						RHS: &ExprString{v: "dead"},
					}},
				}},
				RHS: &ExprString{v: "beef"},
			}},
		},
	}

	for name := range testcases {
		testcase := testcases[name]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			expr, err := Parse(testutil.TestLogger(t), testcase.input)
			require.NoError(t, err)
			assert.Equal(t, testcase.expectedExpr.String(), expr.String())
		})
	}
}

func TestParserErrors(t *testing.T) {
	testcases := map[string]struct {
		input       string
		expectedErr error
	}{
		"EmptyExpression": {
			input:       "",
			expectedErr: ErrEmptyExpression,
		},
		"EmptyFuncCall": {
			input:       "foo()",
			expectedErr: ErrEmptyFuncCall,
		},
		"EmptyParenthesis": {
			input:       "()",
			expectedErr: ErrEmptyParenthesis,
		},
		"MissingArgument": {
			input:       "bar, foo,",
			expectedErr: ErrMissingArgument,
		},
		"MissingOperator": {
			input:       "foo bar",
			expectedErr: ErrMissingOperator,
		},
		"InvalidFuncName": {
			input:       "1(foo, 1, false)",
			expectedErr: ErrInvalidFuncName,
		},
		"UnexpectedComma": {
			input:       ",",
			expectedErr: ErrUnexpectedComma,
		},
		"UnexpectedOperator": {
			input:       "delta",
			expectedErr: ErrUnexpectedOperator,
		},
		"UnexpectedParenthesis": {
			input:       "(bar))",
			expectedErr: ErrUnexpectedParenthesis,
		},
		"MissingOperand": {
			input:       "foo union bar -",
			expectedErr: ErrMissingArgument,
		},
		"InvalidOperandLHS": {
			input:       "false inter bar",
			expectedErr: ErrInvalidArgument,
		},
		"InvalidOperandRHS": {
			input:       "foo delta 3",
			expectedErr: ErrInvalidArgument,
		},
	}

	for name := range testcases {
		testcase := testcases[name]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			expr, err := Parse(testutil.TestLogger(t), testcase.input)
			assert.True(t, errors.Is(err, testcase.expectedErr), err)
			assert.Nil(t, expr)
		})
	}
}
