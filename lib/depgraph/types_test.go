package depgraph

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Helcaraxan/gomod/lib/modules"
)

func TestEdgesNew(t *testing.T) {
	newMap := NewEdges()
	assert.NotNil(t, newMap.dependencyMap)
	assert.NotNil(t, newMap.dependencyList)
}

func TestEdgesCopy(t *testing.T) {
	dependencyA := &Module{Info: &modules.ModuleInfo{Path: "dependency_a"}}
	dependencyB := &Module{Info: &modules.ModuleInfo{Path: "dependency_b"}}

	originalEdges := NewEdges()
	originalEdges.Add(&ModuleReference{Module: dependencyA})
	copiedEdges := originalEdges.Copy()
	originalEdges.Add(&ModuleReference{Module: dependencyB})

	_, okA := originalEdges.Get("dependency_a")
	_, okB := originalEdges.Get("dependency_b")
	assert.True(t, okA)
	assert.True(t, okB)

	_, okA = copiedEdges.Get("dependency_a")
	_, okB = copiedEdges.Get("dependency_b")
	assert.True(t, okA)
	assert.False(t, okB)
}

func TestEdgesAdd(t *testing.T) {
	dependencyA := &Module{Info: &modules.ModuleInfo{Path: "dependency_a"}}

	edges := NewEdges()
	edges.Add(&ModuleReference{Module: dependencyA})
	_, ok := edges.Get("dependency_a")
	assert.True(t, ok)
	edges.Add(&ModuleReference{Module: dependencyA})
	_, ok = edges.Get("dependency_a")
	assert.True(t, ok)
}

func TestEdgesDelete(t *testing.T) {
	dependencyA := &Module{Info: &modules.ModuleInfo{Path: "dependency_a"}}

	edges := NewEdges()

	edges.Delete("dependency_a")

	edges.Add(&ModuleReference{Module: dependencyA})
	edges.Delete("dependency_a")
	assert.NotContains(t, edges.List(), &Module{Info: &modules.ModuleInfo{Path: "dependency_a"}})
}

func TestEdgesLen(t *testing.T) {
	dependencyA := &Module{Info: &modules.ModuleInfo{Path: "dependency_a"}}
	dependencyB := &Module{Info: &modules.ModuleInfo{Path: "dependency_b"}}

	edges := NewEdges()
	assert.Equal(t, 0, edges.Len())

	edges.Add(&ModuleReference{Module: dependencyA})
	assert.Equal(t, 1, edges.Len())

	edges.Add(&ModuleReference{Module: dependencyA})
	assert.Equal(t, 1, edges.Len())

	edges.Add(&ModuleReference{Module: dependencyB})
	assert.Equal(t, 2, edges.Len())

	edges.Delete("dependency_a")
	assert.Equal(t, 1, edges.Len())
}

func TestEdgesList(t *testing.T) {
	dependencyA := &Module{Info: &modules.ModuleInfo{Path: "dependency_a"}}
	dependencyB := &Module{Info: &modules.ModuleInfo{Path: "dependency_b"}}

	edges := NewEdges()
	edges.Add(&ModuleReference{Module: dependencyB})
	edges.Add(&ModuleReference{Module: dependencyA})

	list := edges.List()
	assert.True(t, sort.SliceIsSorted(list, func(i int, j int) bool { return list[i].Name() < list[j].Name() }))
}
