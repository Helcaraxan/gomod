package graph

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testNode struct {
	name     string
	preds    NodeRefs
	succs    NodeRefs
	parent   *testNode
	children NodeRefs
}

func (n *testNode) Name() string { return n.name }
func (n *testNode) Hash() string { return n.name }
func (n *testNode) String() string {
	return fmt.Sprintf("%s, preds: [%s], succs: [%s]", n.name, n.preds, n.succs)
}
func (n *testNode) Predecessors() *NodeRefs { return &n.preds }
func (n *testNode) Successors() *NodeRefs   { return &n.succs }
func (n *testNode) Parent() Node            { return n.parent }
func (n *testNode) Children() *NodeRefs     { return &n.children }

func newTestNode(name string, parent *testNode) *testNode {
	return &testNode{
		name:     name,
		preds:    NewNodeRefs(),
		succs:    NewNodeRefs(),
		parent:   parent,
		children: NewNodeRefs(),
	}
}

func TestEdgesNew(t *testing.T) {
	t.Parallel()

	newMap := NewNodeRefs()
	assert.NotNil(t, newMap.nodeMap)
	assert.NotNil(t, newMap.nodeList)
}

func TestEdgesAdd(t *testing.T) {
	t.Parallel()

	dependencyA := testNode{name: "dependency_a"}

	edges := NewNodeRefs()
	edges.Add(&dependencyA)
	_, w := edges.Get("dependency_a")
	assert.Equal(t, 1, w)
	edges.Add(&dependencyA)
	_, w = edges.Get("dependency_a")
	assert.Equal(t, 2, w)
}

func TestEdgesDelete(t *testing.T) {
	t.Parallel()

	dependencyA := testNode{name: "dependency_a"}

	edges := NewNodeRefs()

	edges.Delete("dependency_a")
	assert.Equal(t, 0, edges.Len())

	edges.Add(&dependencyA)
	edges.Delete("dependency_a")
	assert.Equal(t, 0, edges.Len())
}

func TestEdgesLen(t *testing.T) {
	t.Parallel()

	dependencyA := testNode{name: "dependency_a"}
	dependencyB := testNode{name: "dependency_b"}

	edges := NewNodeRefs()
	assert.Equal(t, 0, edges.Len())

	edges.Add(&dependencyA)
	assert.Equal(t, 1, edges.Len())

	edges.Add(&dependencyA)
	assert.Equal(t, 1, edges.Len())

	edges.Add(&dependencyB)
	assert.Equal(t, 2, edges.Len())

	edges.Delete("dependency_a")
	assert.Equal(t, 2, edges.Len())

	edges.Delete("dependency_b")
	assert.Equal(t, 1, edges.Len())
}

func TestEdgesList(t *testing.T) {
	t.Parallel()

	dependencyA := testNode{name: "dependency_a"}
	dependencyB := testNode{name: "dependency_b"}

	edges := NewNodeRefs()
	edges.Add(&dependencyB)
	edges.Add(&dependencyA)

	list := edges.List()
	assert.True(t, sort.SliceIsSorted(list, func(i int, j int) bool { return list[i].Name() < list[j].Name() }))
}
