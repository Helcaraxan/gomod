package depgraph

import (
	"fmt"

	"github.com/Helcaraxan/gomod/internal/graph"
	"github.com/Helcaraxan/gomod/internal/modules"
)

// Package represents a single Go package in a dependency graph.
type Package struct {
	Info *modules.PackageInfo

	predecessors graph.NodeRefs
	successors   graph.NodeRefs

	parent              *Module
	isNonTestDependency bool
}

func NewPackage(info *modules.PackageInfo, parent *Module) *Package {
	return &Package{
		Info:         info,
		predecessors: graph.NewNodeRefs(),
		successors:   graph.NewNodeRefs(),
		parent:       parent,
	}
}

// Name returns the import path of the package and not the value declared inside the package with
// the 'package' statement.
func (p *Package) Name() string {
	return p.Info.ImportPath
}

func (p *Package) Hash() string {
	return packageHash(p.Info.ImportPath)
}

func (p *Package) String() string {
	return fmt.Sprintf("%s, module: %s, preds: [%s], succs: [%s]", p.Hash(), p.parent.Name(), p.predecessors, p.successors)
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

func (p *Package) NodeAttributes(annotate bool) []string {
	return nil
}

func (p *Package) EdgeAttributes(target graph.Node, annotate bool) []string {
	return nil
}

func (p *Package) isTestDependency() bool {
	return !p.isNonTestDependency
}
