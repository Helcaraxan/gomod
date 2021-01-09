package graph

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Helcaraxan/gomod/internal/testutil"
)

func TestGraphNodes(t *testing.T) {
	var g *HierarchicalDigraph

	n := newTestNode("test-node", nil)
	nc1 := newTestNode("test-node-child-1", n)
	nc2 := newTestNode("test-node-child-2", n)

	t.Run("NilGraph", func(t *testing.T) {
		assert.Equal(t, ErrNilGraph, g.AddNode(n))
		assert.Equal(t, ErrNilGraph, g.DeleteNode(n.name))
	})

	g = NewHierarchicalDigraph(testutil.TestLogger(t).Log())

	t.Run("AddNode", func(t *testing.T) {
		assert.Equal(t, ErrNilNode, g.AddNode(nil))

		nf := newTestNode("non-member", nil)
		nfc := newTestNode("non-member-child", nf)
		assert.True(t, errors.Is(g.AddNode(nfc), ErrNodeNotFound))

		assert.NoError(t, g.AddNode(n))
		assert.Error(t, g.AddNode(n))

		assert.NoError(t, g.AddNode(nc1))
		assert.NoError(t, g.AddNode(nc2))
	})

	t.Run("GetNode", func(t *testing.T) {
		r, err := g.GetNode(n.name)
		assert.NoError(t, err)
		assert.Equal(t, n, r)

		l0 := g.GetLevel(0)
		assert.Equal(t, 1, l0.Len(), l0)
		m, _ := l0.Get(n.name)
		assert.Equal(t, n, m)

		l1 := g.GetLevel(1)
		assert.Equal(t, 2, l1.Len(), l1)
		m, _ = l1.Get(nc1.name)
		assert.Equal(t, nc1, m)
		m, _ = l1.Get(nc2.name)
		assert.Equal(t, nc2, m)
	})

	t.Run("DeleteNode", func(t *testing.T) {
		assert.NoError(t, g.DeleteNode(nc2.name))
		assert.True(t, errors.Is(g.DeleteNode(nc2.name), ErrNodeNotFound))

		assert.NoError(t, g.DeleteNode(n.name), g.members)
		_, err := g.GetNode(nc1.name)
		assert.True(t, errors.Is(err, ErrNodeNotFound))
	})
}

func TestGraphEdges(t *testing.T) {
	var g *HierarchicalDigraph

	t.Run("NilGraph", func(t *testing.T) {
		assert.Equal(t, ErrNilGraph, g.AddEdge(nil, nil))
		assert.Equal(t, ErrNilGraph, g.DeleteEdge(nil, nil))
	})

	g = NewHierarchicalDigraph(testutil.TestLogger(t).Log())

	n1 := newTestNode("test-node-1", nil)
	n2 := newTestNode("test-node-2", nil)
	nc1 := newTestNode("test-node-child-1", n1)
	nc2 := newTestNode("test-node-child-2", n2)

	require.NoError(t, g.AddNode(n1))
	require.NoError(t, g.AddNode(n2))
	require.NoError(t, g.AddNode(nc1))
	require.NoError(t, g.AddNode(nc2))

	t.Run("NilNodes", func(t *testing.T) {
		assert.Equal(t, ErrNilNode, g.AddEdge(nil, nil))
		assert.Equal(t, ErrNilNode, g.AddEdge(n1, nil))
		assert.Equal(t, ErrNilNode, g.AddEdge(nil, n2))

		assert.Equal(t, ErrNilNode, g.DeleteEdge(nil, nil))
		assert.Equal(t, ErrNilNode, g.DeleteEdge(n1, nil))
		assert.Equal(t, ErrNilNode, g.DeleteEdge(nil, n2))
	})

	t.Run("AddEdge", func(t *testing.T) {
		nf := newTestNode("non-member", nil)
		assert.True(t, errors.Is(g.AddEdge(n1, nf), ErrNodeNotFound))
		assert.True(t, errors.Is(g.AddEdge(nf, n1), ErrNodeNotFound))

		assert.True(t, errors.Is(g.AddEdge(n1, nc1), ErrEdgeCrossLevel))

		assert.NoError(t, g.AddEdge(nc1, nc2))

		m, w := nc1.Successors().Get(nc2.name)
		assert.Equal(t, nc2, m)
		assert.Equal(t, 1, w)
		m, w = nc2.Predecessors().Get(nc1.name)
		assert.Equal(t, nc1, m)
		assert.Equal(t, 1, w)
		m, w = n1.Successors().Get(n2.name)
		assert.Equal(t, n2, m)
		assert.Equal(t, 1, w)
		m, w = n2.Predecessors().Get(n1.name)
		assert.Equal(t, n1, m)
		assert.Equal(t, 1, w)

		assert.NoError(t, g.AddEdge(n1, n2))
		m, w = n1.Successors().Get(n2.name)
		assert.Equal(t, n2, m)
		assert.Equal(t, 2, w)
		m, w = n2.Predecessors().Get(n1.name)
		assert.Equal(t, n1, m)
		assert.Equal(t, 2, w)
	})

	t.Run("DeleteEdge", func(t *testing.T) {
		nf := newTestNode("non-member", nil)
		assert.True(t, errors.Is(g.DeleteEdge(n1, nf), ErrNodeNotFound))
		assert.True(t, errors.Is(g.DeleteEdge(nf, n1), ErrNodeNotFound))

		assert.NoError(t, g.DeleteEdge(nc2, nc1))
		m, w := nc1.Successors().Get(nc2.name)
		assert.Equal(t, nc2, m)
		assert.Equal(t, 1, w)
		m, w = nc2.Predecessors().Get(nc1.name)
		assert.Equal(t, nc1, m)
		assert.Equal(t, 1, w)
		m, w = n1.Successors().Get(n2.name)
		assert.Equal(t, n2, m)
		assert.Equal(t, 2, w)
		m, w = n2.Predecessors().Get(n1.name)
		assert.Equal(t, n1, m)
		assert.Equal(t, 2, w)

		assert.NoError(t, g.DeleteEdge(nc1, nc2))
		_, w = nc1.Successors().Get(nc2.name)
		assert.Equal(t, 0, w)
		_, w = nc2.Predecessors().Get(nc1.name)
		assert.Equal(t, 0, w)
		_, w = n1.Successors().Get(n2.name)
		assert.Equal(t, 1, w)
		_, w = n2.Predecessors().Get(n1.name)
		assert.Equal(t, 1, w)

		assert.NoError(t, g.AddEdge(nc1, nc2))
		assert.NoError(t, g.DeleteEdge(n1, n2))
		_, w = nc1.Successors().Get(nc2.name)
		assert.Equal(t, 0, w)
		_, w = nc2.Predecessors().Get(nc1.name)
		assert.Equal(t, 0, w)
		_, w = n1.Successors().Get(n2.name)
		assert.Equal(t, 0, w)
		_, w = n2.Predecessors().Get(n1.name)
		assert.Equal(t, 0, w)
	})

	t.Run("DeleteNode", func(t *testing.T) {
		assert.NoError(t, g.AddEdge(n1, n2))

		assert.NoError(t, g.DeleteNode(n2.name))
		_, w := n1.Successors().Get(n2.name)
		assert.Equal(t, 0, w)

		assert.NoError(t, g.AddNode(n2))
		assert.NoError(t, g.AddEdge(n1, n2))

		assert.NoError(t, g.DeleteNode(nc1.name))
		_, w = n2.Predecessors().Get(n1.name)
		assert.Equal(t, 0, w)
		_, err := g.GetNode(n1.name)
		assert.True(t, errors.Is(err, ErrNodeNotFound))
	})
}
