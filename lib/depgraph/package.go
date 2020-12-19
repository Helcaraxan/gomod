package depgraph

import "github.com/Helcaraxan/gomod/lib/modules"

// Package represents a single Go package in a dependency graph.
type Package struct {
	Info   *modules.PackageInfo
	Parent *Module

	predecessors Edges
	successors   Edges
}

// Name returns the import path of the package and not the value declared inside the package with
// the 'package' statement.
func (p *Package) Name() string {
	return p.Info.ImportPath
}

func (p *Package) Predecessors() *Edges {
	return &p.predecessors
}

func (p *Package) Successors() *Edges {
	return &p.successors
}
