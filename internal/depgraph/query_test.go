package depgraph

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Helcaraxan/gomod/internal/graph"
	"github.com/Helcaraxan/gomod/internal/modules"
	"github.com/Helcaraxan/gomod/internal/query"
	"github.com/Helcaraxan/gomod/internal/testutil"
)

type queryTestGraph struct {
	nodes []queryTestNode
	edges []queryTestEdge
}
type queryTestNode struct {
	name   string
	isTest bool
}
type queryTestEdge struct {
	s string
	e string
}

func instantiateQueryTestGraph(t *testing.T, testGraph queryTestGraph) DepGraph {
	g := DepGraph{
		Graph: graph.NewHierarchicalDigraph(testutil.TestLogger(t).Log()),
	}

	nodes := map[string]graph.Node{}
	for _, node := range testGraph.nodes {
		module := NewModule(&modules.ModuleInfo{
			Path: node.name,
		})
		module.isNonTestDependency = !node.isTest
		nodes[node.name] = module
		require.NoError(t, g.Graph.AddNode(module))
	}
	for _, edge := range testGraph.edges {
		require.NoError(t, g.Graph.AddEdge(nodes[edge.s], nodes[edge.e]))
	}
	return g
}

func TestQueryInvalid(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		query             string
		expectedErrString string
	}{
		"Boolean": {
			query:             "true",
			expectedErrString: "boolean",
		},
		"Integer": {
			query:             "42",
			expectedErrString: "integer",
		},
		"StringUnknownAnnotation": {
			query:             "foo:bar",
			expectedErrString: "undefined",
		},
		"StringTooManyAnnotations": {
			query:             "test:foo:bar",
			expectedErrString: "more than one",
		},
		"DepsFuncTooManyArgs": {
			query:             "deps(foo, bar, dead, beef)",
			expectedErrString: "at most",
		},
		"DepsFuncWrongTypeSecondArgument": {
			query:             "deps(foo, bar)",
			expectedErrString: "expected an integer",
		},
		"RDepsFuncTooManyArgs": {
			query:             "rdeps(foo, bar, dead, beef)",
			expectedErrString: "at most",
		},
		"RDepsFuncWrongTypeSecondArgument": {
			query:             "rdeps(foo, bar)",
			expectedErrString: "expected an integer",
		},
		"SharedFuncBoolean": {
			query:             "shared(false)",
			expectedErrString: "boolean",
		},
		"SharedFuncInteger": {
			query:             "shared(42)",
			expectedErrString: "integer",
		},
		"SharedFuncTooManyArgs": {
			query:             "shared(foo, bar, com)",
			expectedErrString: "single argument",
		},
		"UnknownFunc": {
			query:             "foo(bar)",
			expectedErrString: "unknown function",
		},
	}

	for name := range testcases {
		testcase := testcases[name]
		t.Run(name, func(t *testing.T) {
			log := testutil.TestLogger(t)
			g := DepGraph{
				Graph: graph.NewHierarchicalDigraph(log.Log()),
			}

			q, err := query.Parse(log, testcase.query)
			require.NoError(t, err)
			set, err := g.computeSet(log.Log(), q, LevelModules)
			require.True(t, errors.Is(err, ErrInvalidQuery))
			assert.Contains(t, err.Error(), testcase.expectedErrString)
			assert.Empty(t, set)
		})
	}
}

func TestQueryNameMatch(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		graph       queryTestGraph
		query       string
		expectedSet nodeSet
	}{
		"Exact": {
			graph: queryTestGraph{
				nodes: []queryTestNode{
					{name: "test.com/module"},
					{name: "test.com/foo"},
				},
			},
			query: "test.com/module",
			expectedSet: nodeSet{
				"test.com/module": true,
			},
		},
		"ExactWithTest": {
			graph: queryTestGraph{
				nodes: []queryTestNode{
					{name: "test.com/module"},
					{name: "test.com/foo", isTest: true},
				},
			},
			query: "test.com/foo:test",
			expectedSet: nodeSet{
				"test.com/foo": true,
			},
		},
		"ExactWithTestNoMatch": {
			graph: queryTestGraph{
				nodes: []queryTestNode{
					{name: "test.com/module"},
					{name: "test.com/foo"},
				},
			},
			query: "test.com/foo:test",
			expectedSet: nodeSet{
				"test.com/foo": true,
			},
		},
		"Prefix": {
			graph: queryTestGraph{
				nodes: []queryTestNode{
					{name: "test.com/module"},
					{name: "test.com/foo", isTest: true},
				},
			},
			query: "test.com/...",
			expectedSet: nodeSet{
				"test.com/module": true,
			},
		},
		"PrefixWithTest": {
			graph: queryTestGraph{
				nodes: []queryTestNode{
					{name: "test.com/module"},
					{name: "test.com/foo", isTest: true},
				},
			},
			query: "test.com/...:test",
			expectedSet: nodeSet{
				"test.com/module": true,
				"test.com/foo":    true,
			},
		},
	}

	for name := range testcases {
		testcase := testcases[name]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			log := testutil.TestLogger(t)
			g := instantiateQueryTestGraph(t, testcase.graph)

			q, err := query.Parse(log, testcase.query)
			require.NoError(t, err)
			require.IsType(t, &query.ExprString{}, q)

			set, err := g.computeSet(log.Log(), q, LevelModules)
			require.NoError(t, err)
			assert.Equal(t, testcase.expectedSet, set)
		})
	}
}

func TestQueryBinaryOp(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		graph       queryTestGraph
		query       string
		expectedSet nodeSet
	}{
		"Union": {
			graph: queryTestGraph{
				nodes: []queryTestNode{
					{name: "test.com/module"},
					{name: "test.com/foo"},
				},
			},
			query: "test.com/module + test.com/foo",
			expectedSet: nodeSet{
				"test.com/module": true,
				"test.com/foo":    true,
			},
		},
		"Subtract": {
			graph: queryTestGraph{
				nodes: []queryTestNode{
					{name: "test.com/module"},
					{name: "test.com/foo"},
				},
			},
			query: "test.com/... - test.com/foo",
			expectedSet: nodeSet{
				"test.com/module": true,
			},
		},
		"Intersect": {
			graph: queryTestGraph{
				nodes: []queryTestNode{
					{name: "test.com/module"},
					{name: "test.com/foo"},
					{name: "test.com/foo/bar", isTest: true},
				},
			},
			query: "test.com/... inter test.com/foo/...:test",
			expectedSet: nodeSet{
				"test.com/foo": true,
			},
		},
		"Delta": {
			graph: queryTestGraph{
				nodes: []queryTestNode{
					{name: "test.com/module"},
					{name: "test.com/foo"},
					{name: "test.com/foo/bar", isTest: true},
				},
			},
			query: "test.com/... delta test.com/foo/...:test",
			expectedSet: nodeSet{
				"test.com/module":  true,
				"test.com/foo/bar": true,
			},
		},
	}

	for name := range testcases {
		testcase := testcases[name]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			log := testutil.TestLogger(t)
			g := instantiateQueryTestGraph(t, testcase.graph)

			q, err := query.Parse(log, testcase.query)
			require.NoError(t, err)
			require.Implements(t, (*query.BinaryExpr)(nil), q)

			set, err := g.computeSet(log.Log(), q, LevelModules)
			require.NoError(t, err)
			assert.Equal(t, testcase.expectedSet, set)
		})
	}
}

func TestQueryFuncs(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		graph       queryTestGraph
		query       string
		expectedSet nodeSet
	}{
		"DepsNoLimit": {
			graph: queryTestGraph{
				nodes: []queryTestNode{
					{name: "test.com/module"},
					{name: "test.com/foo"},
					{name: "test.com/bar"},
					{name: "test.com/beef"},
				},
				edges: []queryTestEdge{
					{s: "test.com/module", e: "test.com/foo"},
					{s: "test.com/foo", e: "test.com/bar"},
					{s: "test.com/bar", e: "test.com/foo"},
				},
			},
			query: "deps(test.com/module)",
			expectedSet: nodeSet{
				"test.com/module": true,
				"test.com/foo":    true,
				"test.com/bar":    true,
			},
		},
		"DepsLimit": {
			graph: queryTestGraph{
				nodes: []queryTestNode{
					{name: "test.com/module"},
					{name: "test.com/foo"},
					{name: "test.com/bar"},
				},
				edges: []queryTestEdge{
					{s: "test.com/module", e: "test.com/foo"},
					{s: "test.com/foo", e: "test.com/bar"},
				},
			},
			query: "deps(test.com/module, 1)",
			expectedSet: nodeSet{
				"test.com/module": true,
				"test.com/foo":    true,
			},
		},
		"ReverseDepsNoLimit": {
			graph: queryTestGraph{
				nodes: []queryTestNode{
					{name: "test.com/module"},
					{name: "test.com/foo"},
					{name: "test.com/bar"},
					{name: "test.com/beef"},
				},
				edges: []queryTestEdge{
					{s: "test.com/module", e: "test.com/foo"},
					{s: "test.com/foo", e: "test.com/bar"},
				},
			},
			query: "rdeps(test.com/bar)",
			expectedSet: nodeSet{
				"test.com/module": true,
				"test.com/foo":    true,
				"test.com/bar":    true,
			},
		},
		"ReverseDepsLimit": {
			graph: queryTestGraph{
				nodes: []queryTestNode{
					{name: "test.com/module"},
					{name: "test.com/foo"},
					{name: "test.com/bar"},
					{name: "test.com/beef"},
				},
				edges: []queryTestEdge{
					{s: "test.com/module", e: "test.com/foo"},
					{s: "test.com/foo", e: "test.com/bar"},
				},
			},
			query: "rdeps(test.com/bar, 1)",
			expectedSet: nodeSet{
				"test.com/foo": true,
				"test.com/bar": true,
			},
		},
		"Shared": {
			graph: queryTestGraph{
				nodes: []queryTestNode{
					{name: "test.com/module"},
					{name: "test.com/foo"},
					{name: "test.com/bar"},
					{name: "test.com/dead"},
					{name: "test.com/beef"},
				},
				edges: []queryTestEdge{
					{s: "test.com/module", e: "test.com/foo"},
					{s: "test.com/module", e: "test.com/bar"},
					{s: "test.com/foo", e: "test.com/bar"},
					{s: "test.com/bar", e: "test.com/dead"},
					{s: "test.com/dead", e: "test.com/beef"},
				},
			},
			query: "shared(test.com/...)",
			expectedSet: nodeSet{
				"test.com/module": true,
				"test.com/foo":    true,
				"test.com/bar":    true,
			},
		},
	}

	for name := range testcases {
		testcase := testcases[name]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			log := testutil.TestLogger(t)
			g := instantiateQueryTestGraph(t, testcase.graph)

			q, err := query.Parse(log, testcase.query)
			require.NoError(t, err)
			require.Implements(t, (*query.FuncExpr)(nil), q)

			set, err := g.computeSet(log.Log(), q, LevelModules)
			require.NoError(t, err)
			assert.Equal(t, testcase.expectedSet, set)
		})
	}
}
