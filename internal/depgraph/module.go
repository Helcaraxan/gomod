package depgraph

import (
	"fmt"
	"time"

	"github.com/Helcaraxan/gomod/internal/graph"
	"github.com/Helcaraxan/gomod/internal/modules"
)

// Module represents a module in a Go module's dependency graph.
type Module struct {
	Info *modules.ModuleInfo

	predecessors graph.NodeRefs
	successors   graph.NodeRefs

	packages graph.NodeRefs
}

func NewModule(info *modules.ModuleInfo) *Module {
	return &Module{
		Info:         info,
		predecessors: graph.NewNodeRefs(),
		successors:   graph.NewNodeRefs(),
	}
}

// Name of the module represented by this Dependency in the Graph instance.
func (m *Module) Name() string {
	return m.Info.Path
}

func (m *Module) Hash() string {
	return moduleHash(m.Info.Path)
}

func moduleHash(name string) string {
	return "module " + name
}

// SelectedVersion corresponds to the version of the dependency represented by this Dependency which
// was selected for use.
func (m *Module) SelectedVersion() string {
	if m.Info.Replace != nil {
		return m.Info.Replace.Version
	}
	return m.Info.Version
}

// Timestamp returns the time corresponding to the creation of the version at which this dependency
// is used.
func (m *Module) Timestamp() *time.Time {
	if m.Info.Replace != nil {
		return m.Info.Replace.Time
	}
	return m.Info.Time
}

// ModuleReference represents an edge from one module to another in a dependency graph.
type ModuleReference struct {
	*Module
	VersionConstraint string
}

func (m *ModuleReference) Parent() graph.Node {
	return nil
}

func (m *ModuleReference) Predecessors() *graph.NodeRefs {
	return &m.predecessors
}

func (m *ModuleReference) Successors() *graph.NodeRefs {
	return &m.successors
}

func (m *ModuleReference) Children() *graph.NodeRefs {
	return &m.packages
}

func (m *ModuleReference) NodeAnnotations() []string {
	var annotations []string
	if m.SelectedVersion() != "" {
		var replacement string
		if m.Info.Replace != nil {
			replacement = m.Info.Replace.Path + "<br />"
		}
		annotations = append(
			annotations,
			fmt.Sprintf("label=<%s<br /><font point-size=\"10\">%s%s</font>>", m.Name(), replacement, m.SelectedVersion()),
		)
	}
	return annotations
}

func (m *ModuleReference) EdgeAnnotations() []string {
	var annotations []string
	if m.VersionConstraint != "" {
		annotations = append(annotations, fmt.Sprintf("label=<<font point-size=\"10\">%s</font>>", m.VersionConstraint))
	}
	return annotations
}
