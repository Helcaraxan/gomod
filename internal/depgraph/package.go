package depgraph

import (
	"github.com/Helcaraxan/gomod/internal/graph"
	"github.com/Helcaraxan/gomod/internal/modules"
)

// Package represents a single Go package in a dependency graph.
type Package struct {
	Info *modules.PackageInfo

	predecessors graph.NodeRefs
	successors   graph.NodeRefs

	parent *ModuleReference
}

// Name returns the import path of the package and not the value declared inside the package with
// the 'package' statement.
func (p Package) Name() string {
	return p.Info.ImportPath
}

func (p Package) Hash() string {
	return packageHash(p.Info.ImportPath)
}

func packageHash(name string) string { return "package " + name }

func (p *Package) Predecessors() *graph.NodeRefs {
	return &p.predecessors
}

func (p *Package) Successors() *graph.NodeRefs {
	return &p.successors
}

func (p *Package) Children() *graph.NodeRefs {
	return nil
}

func (p *Package) Parent() graph.Node {
	return p.parent
}
