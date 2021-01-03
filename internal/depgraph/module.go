package depgraph

import (
	"fmt"
	"time"

	"github.com/Helcaraxan/gomod/internal/graph"
	"github.com/Helcaraxan/gomod/internal/modules"
)

// Module represents a module in a Go module's dependency graph.
type Module struct {
	Info               *modules.ModuleInfo
	Indirects          map[string]bool
	VersionConstraints map[string]VersionConstraint

	predecessors graph.NodeRefs
	successors   graph.NodeRefs

	packages            graph.NodeRefs
	isNonTestDependency bool
}

type VersionConstraint struct {
	Source string
	Target string
}

func NewModule(info *modules.ModuleInfo) *Module {
	return &Module{
		Info:               info,
		Indirects:          map[string]bool{},
		VersionConstraints: map[string]VersionConstraint{},
		predecessors:       graph.NewNodeRefs(),
		successors:         graph.NewNodeRefs(),
		packages:           graph.NewNodeRefs(),
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

func (m *Module) Parent() graph.Node {
	return nil
}

func (m *Module) Predecessors() *graph.NodeRefs {
	return &m.predecessors
}

func (m *Module) Successors() *graph.NodeRefs {
	return &m.successors
}

func (m *Module) Children() *graph.NodeRefs {
	return &m.packages
}

func (m *Module) NodeAttributes(annotate bool) []string {
	var annotations []string
	if annotate && m.SelectedVersion() != "" {
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

func (m *Module) EdgeAttributes(target graph.Node, annotate bool) []string {
	targetModule := target.(*Module)

	var annotations []string
	if m.Indirects[target.Name()] {
		annotations = append(annotations, "style=dashed")
	}
	if c, ok := m.VersionConstraints[targetModule.Hash()]; ok && annotate {
		annotations = append(annotations, fmt.Sprintf("label=<<font point-size=\"10\">%s</font>>", c.Target))
	}
	return annotations
}

var _ testAnnotated = &Module{}

func (m Module) isTestDependency() bool {
	return !m.isNonTestDependency
}
