package depgraph

import (
	"time"

	"github.com/Helcaraxan/gomod/lib/modules"
)

// Module represents a module in a Go module's dependency graph.
type Module struct {
	Info *modules.ModuleInfo

	predecessors Edges
	successors   Edges
}

// Name of the module represented by this Dependency in the Graph instance.
func (n *Module) Name() string {
	return n.Info.Path
}

func (n *Module) Predecessors() *Edges {
	return &n.predecessors
}

func (n *Module) Successors() *Edges {
	return &n.successors
}

// SelectedVersion corresponds to the version of the dependency represented by this Dependency which
// was selected for use.
func (n *Module) SelectedVersion() string {
	if n.Info.Replace != nil {
		return n.Info.Replace.Version
	}
	return n.Info.Version
}

// Timestamp returns the time corresponding to the creation of the version at which this dependency
// is used.
func (n *Module) Timestamp() *time.Time {
	if n.Info.Replace != nil {
		return n.Info.Replace.Time
	}
	return n.Info.Time
}

// ModuleReference represents an edge from one module to another in a dependency graph.
type ModuleReference struct {
	*Module
	VersionConstraint string
}
