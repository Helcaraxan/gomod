package depgraph

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapNew(t *testing.T) {
	newMap := NewNodeMap()
	assert.NotNil(t, newMap.nodeMap)
	assert.NotNil(t, newMap.nodeList)
}

func TestMapCopy(t *testing.T) {
	nodeA := &Node{Module: &Module{Path: "node_a"}}
	nodeB := &Node{Module: &Module{Path: "node_b"}}

	originalMap := NewNodeMap()
	originalMap.Add(&NodeReference{Node: nodeA})
	copiedMap := originalMap.Copy()
	originalMap.Add(&NodeReference{Node: nodeB})

	_, okA := originalMap.Get("node_a")
	_, okB := originalMap.Get("node_b")
	assert.True(t, okA)
	assert.True(t, okB)

	_, okA = copiedMap.Get("node_a")
	_, okB = copiedMap.Get("node_b")
	assert.True(t, okA)
	assert.False(t, okB)
}

func TestMapAdd(t *testing.T) {
	nodeA := &Node{Module: &Module{Path: "node_a"}}

	newMap := NewNodeMap()
	newMap.Add(&NodeReference{Node: nodeA})
	_, ok := newMap.Get("node_a")
	assert.True(t, ok)
	newMap.Add(&NodeReference{Node: nodeA})
	_, ok = newMap.Get("node_a")
	assert.True(t, ok)
}

func TestMapDelete(t *testing.T) {
	nodeA := &Node{Module: &Module{Path: "node_a"}}

	newMap := NewNodeMap()

	newMap.Delete("node_a")

	newMap.Add(&NodeReference{Node: nodeA})
	newMap.Delete("node_a")
	assert.NotContains(t, newMap.List(), &Node{Module: &Module{Path: "node_a"}})
}

func TestMapLen(t *testing.T) {
	nodeA := &Node{Module: &Module{Path: "node_a"}}
	nodeB := &Node{Module: &Module{Path: "node_b"}}

	newMap := NewNodeMap()
	assert.Equal(t, 0, newMap.Len())

	newMap.Add(&NodeReference{Node: nodeA})
	assert.Equal(t, 1, newMap.Len())

	newMap.Add(&NodeReference{Node: nodeA})
	assert.Equal(t, 1, newMap.Len())

	newMap.Add(&NodeReference{Node: nodeB})
	assert.Equal(t, 2, newMap.Len())

	newMap.Delete("node_a")
	assert.Equal(t, 1, newMap.Len())
}

func TestMapList(t *testing.T) {
	nodeA := &Node{Module: &Module{Path: "node_a"}}
	nodeB := &Node{Module: &Module{Path: "node_b"}}

	newMap := NewNodeMap()
	newMap.Add(&NodeReference{Node: nodeB})
	newMap.Add(&NodeReference{Node: nodeA})

	list := newMap.List()
	isSorted := sort.SliceIsSorted(list, func(i int, j int) bool { return list[i].Name() < list[j].Name() })
	assert.True(t, isSorted)
}
