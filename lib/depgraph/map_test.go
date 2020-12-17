package depgraph

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Helcaraxan/gomod/lib/modules"
)

func TestMapNew(t *testing.T) {
	newMap := NewDependencyMap()
	assert.NotNil(t, newMap.dependencyMap)
	assert.NotNil(t, newMap.dependencyList)
}

func TestMapCopy(t *testing.T) {
	dependencyA := &Dependency{Module: &modules.ModuleInfo{Path: "dependency_a"}}
	dependencyB := &Dependency{Module: &modules.ModuleInfo{Path: "dependency_b"}}

	originalMap := NewDependencyMap()
	originalMap.Add(&DependencyReference{Dependency: dependencyA})
	copiedMap := originalMap.Copy()
	originalMap.Add(&DependencyReference{Dependency: dependencyB})

	_, okA := originalMap.Get("dependency_a")
	_, okB := originalMap.Get("dependency_b")
	assert.True(t, okA)
	assert.True(t, okB)

	_, okA = copiedMap.Get("dependency_a")
	_, okB = copiedMap.Get("dependency_b")
	assert.True(t, okA)
	assert.False(t, okB)
}

func TestMapAdd(t *testing.T) {
	dependencyA := &Dependency{Module: &modules.ModuleInfo{Path: "dependency_a"}}

	newMap := NewDependencyMap()
	newMap.Add(&DependencyReference{Dependency: dependencyA})
	_, ok := newMap.Get("dependency_a")
	assert.True(t, ok)
	newMap.Add(&DependencyReference{Dependency: dependencyA})
	_, ok = newMap.Get("dependency_a")
	assert.True(t, ok)
}

func TestMapDelete(t *testing.T) {
	dependencyA := &Dependency{Module: &modules.ModuleInfo{Path: "dependency_a"}}

	newMap := NewDependencyMap()

	newMap.Delete("dependency_a")

	newMap.Add(&DependencyReference{Dependency: dependencyA})
	newMap.Delete("dependency_a")
	assert.NotContains(t, newMap.List(), &Dependency{Module: &modules.ModuleInfo{Path: "dependency_a"}})
}

func TestMapLen(t *testing.T) {
	dependencyA := &Dependency{Module: &modules.ModuleInfo{Path: "dependency_a"}}
	dependencyB := &Dependency{Module: &modules.ModuleInfo{Path: "dependency_b"}}

	newMap := NewDependencyMap()
	assert.Equal(t, 0, newMap.Len())

	newMap.Add(&DependencyReference{Dependency: dependencyA})
	assert.Equal(t, 1, newMap.Len())

	newMap.Add(&DependencyReference{Dependency: dependencyA})
	assert.Equal(t, 1, newMap.Len())

	newMap.Add(&DependencyReference{Dependency: dependencyB})
	assert.Equal(t, 2, newMap.Len())

	newMap.Delete("dependency_a")
	assert.Equal(t, 1, newMap.Len())
}

func TestMapList(t *testing.T) {
	dependencyA := &Dependency{Module: &modules.ModuleInfo{Path: "dependency_a"}}
	dependencyB := &Dependency{Module: &modules.ModuleInfo{Path: "dependency_b"}}

	newMap := NewDependencyMap()
	newMap.Add(&DependencyReference{Dependency: dependencyB})
	newMap.Add(&DependencyReference{Dependency: dependencyA})

	list := newMap.List()
	isSorted := sort.SliceIsSorted(list, func(i int, j int) bool { return list[i].Name() < list[j].Name() })
	assert.True(t, isSorted)
}
