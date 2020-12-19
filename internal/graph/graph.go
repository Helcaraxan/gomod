package graph

import (
	"errors"
	"fmt"
)

type HierarchicalDigraph struct {
	name     string
	members  NodeRefs
	children NodeRefs
}

func NewHierarchicalDigraph(name string) HierarchicalDigraph {
	return HierarchicalDigraph{
		name:     name,
		members:  NewNodeRefs(),
		children: NewNodeRefs(),
	}
}

func (g *HierarchicalDigraph) Name() string            { return g.name }
func (g *HierarchicalDigraph) Predecessors() *NodeRefs { return nil }
func (g *HierarchicalDigraph) Successors() *NodeRefs   { return nil }
func (g *HierarchicalDigraph) Parent() Node            { return nil }
func (g *HierarchicalDigraph) Children() *NodeRefs     { return &g.children }

func (g *HierarchicalDigraph) AddNode(node Node) error {
	if g == nil {
		return errors.New("cannot add node to nil graph")
	} else if n, _ := g.members.Get(node.Hash()); n != nil {
		return fmt.Errorf("could not add node %q to graph as there is already a node with that hash", node.Hash())
	}

	g.members.Add(node)
	if node.Parent() == nil {
		g.children.Add(node)
	}

	return nil
}

func (g *HierarchicalDigraph) DeleteNode(hash string) error {
	if g == nil {
		return errors.New("cannot delete node from nil graph")
	}

	target, _ := g.members.Get(hash)
	if target == nil {
		return fmt.Errorf("could not delete node %q from graph as there is no node with that name", hash)
	}

	for _, child := range target.Children().List() {
		if err := g.DeleteNode(child.Hash()); err != nil {
			return err
		}
	}

	for _, pred := range target.Predecessors().List() {
		if err := g.DeleteEdge(pred, target); err != nil {
			return err
		}
	}

	for _, succ := range target.Successors().List() {
		if err := g.DeleteEdge(target, succ); err != nil {
			return err
		}
	}

	g.members.Delete(hash)
	g.children.Delete(hash)

	return nil
}

func (g *HierarchicalDigraph) AddEdge(src Node, dst Node) error {
	if g == nil {
		return errors.New("cannot add edge to nil graph")
	} else if src.Hash() == dst.Hash() {
		return errors.New("cannot add self-edge")
	}

	if _, w := g.members.Get(src.Hash()); w == 0 {
		return fmt.Errorf("could not start edge from node %q as there is no node with that name in the graph", src.Name())
	} else if _, w := g.members.Get(dst.Hash()); w == 0 {
		return fmt.Errorf("could not end edge from node %q as there is no node with that name in the graph", dst.Name())
	}

	if nodeDepth(src) != nodeDepth(dst) {
		return fmt.Errorf(
			"could not create edge between %q (%d) and %q (%d) as they do not belong to the same hierarchical graph level",
			src.Name(),
			nodeDepth(src),
			dst.Name(),
			nodeDepth(dst),
		)
	}

	for src != nil && dst != nil && src.Hash() != dst.Hash() {
		src.Successors().Add(dst)
		dst.Predecessors().Add(src)

		src = src.Parent()
		dst = dst.Parent()
	}
	return nil
}

func (g *HierarchicalDigraph) DeleteEdge(src Node, dst Node) error {
	if g == nil {
		return errors.New("cannot delete edge from nil graph")
	}

	if _, w := g.members.Get(src.Hash()); w == 0 {
		return fmt.Errorf("could not remove edge starting at node %q as there is no node with that name in the graph", src.Name())
	} else if _, w := g.members.Get(dst.Hash()); w == 0 {
		return fmt.Errorf("could not remove edge ending at node %q as there is no node with that name in the graph", dst.Name())
	}

	for _, child := range src.Children().List() {
		for _, succ := range child.Successors().List() {
			if succ.Parent() == dst {
				if err := g.DeleteEdge(child, succ); err != nil {
					return err
				}
			}
		}
	}

	for src != nil && dst != nil && src.Hash() != dst.Hash() {
		src.Successors().Delete(dst.Hash())
		dst.Predecessors().Delete(src.Hash())

		src = src.Parent()
		dst = dst.Parent()
	}
	return nil
}

func (g *HierarchicalDigraph) GetLevel(level int) NodeRefs {
	refs := NewNodeRefs()
	for _, n := range g.members.nodeList {
		if nodeDepth(n) == level {
			refs.Add(n)
		}
	}
	return refs
}
