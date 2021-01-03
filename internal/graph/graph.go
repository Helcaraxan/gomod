package graph

import (
	"errors"
)

var (
	ErrNilGraph          = errors.New("cannot operate on nil-graph")
	ErrNilNode           = errors.New("cannot operate on nil-node")
	ErrNodeAlreadyExists = errors.New("node with identical hash already exists in graph")
	ErrNodeNotFound      = errors.New("node not found")
	ErrEdgeSelf          = errors.New("self-edges are not allowed")
	ErrEdgeCrossLevel    = errors.New("edges not allowed between nodes of different hierarchical levels")
)

type HierarchicalDigraph struct {
	members NodeRefs
}

func NewHierarchicalDigraph() *HierarchicalDigraph {
	return &HierarchicalDigraph{
		members: NewNodeRefs(),
	}
}

func (g HierarchicalDigraph) GetNode(hash string) (Node, error) {
	n, _ := g.members.Get(hash)
	if n == nil {
		return nil, ErrNodeNotFound
	}
	return n, nil
}

func (g *HierarchicalDigraph) AddNode(node Node) error {
	if g == nil {
		return ErrNilGraph
	} else if nodeIsNil(node) {
		return ErrNilNode
	}

	if n, _ := g.members.Get(node.Hash()); n != nil {
		return ErrNodeAlreadyExists
	}

	if p := node.Parent(); !nodeIsNil(p) {
		if n, _ := g.members.Get(p.Hash()); nodeIsNil(n) {
			return ErrNodeNotFound
		}
		p.Children().Add(node)
	}
	g.members.Add(node)

	return nil
}

func (g *HierarchicalDigraph) DeleteNode(hash string) error {
	if g == nil {
		return ErrNilGraph
	}

	target, _ := g.members.Get(hash)
	if target == nil {
		return ErrNodeNotFound
	}

	if target.Children() != nil {
		for _, child := range target.Children().List() {
			if err := g.DeleteNode(child.Hash()); err != nil {
				return err
			}
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

	if p := target.Parent(); !nodeIsNil(p) {
		p.Children().Delete(hash)
		if p.Children().Len() == 0 {
			if err := g.DeleteNode(p.Hash()); err != nil {
				return err
			}
		}
	}

	g.members.Delete(hash)
	return nil
}

func (g *HierarchicalDigraph) AddEdge(src Node, dst Node) error {
	if g == nil {
		return ErrNilGraph
	} else if nodeIsNil(src) || nodeIsNil(dst) {
		return ErrNilNode
	}

	if src.Hash() == dst.Hash() {
		return ErrEdgeSelf
	}

	if _, w := g.members.Get(src.Hash()); w == 0 {
		return ErrNodeNotFound
	} else if _, w := g.members.Get(dst.Hash()); w == 0 {
		return ErrNodeNotFound
	}

	if nodeDepth(src) != nodeDepth(dst) {
		return ErrEdgeCrossLevel
	}

	for !nodeIsNil(src) && !nodeIsNil(dst) && src.Hash() != dst.Hash() {
		src.Successors().Add(dst)
		dst.Predecessors().Add(src)

		src = src.Parent()
		dst = dst.Parent()
	}
	return nil
}

func (g *HierarchicalDigraph) DeleteEdge(src Node, dst Node) error {
	if g == nil {
		return ErrNilGraph
	} else if nodeIsNil(src) || nodeIsNil(dst) {
		return ErrNilNode
	}

	if _, w := g.members.Get(src.Hash()); w == 0 {
		return ErrNodeNotFound
	} else if _, w := g.members.Get(dst.Hash()); w == 0 {
		return ErrNodeNotFound
	}

	if src.Children() != nil {
		for _, child := range src.Children().List() {
			for _, succ := range child.Successors().List() {
				if sp := succ.Parent(); !nodeIsNil(sp) && sp == dst {
					if err := g.DeleteEdge(child, succ); err != nil {
						return err
					}
				}
			}
		}
	}

	for !nodeIsNil(src) && !nodeIsNil(dst) && src.Hash() != dst.Hash() {
		src.Successors().Delete(dst.Hash())
		dst.Predecessors().Delete(src.Hash())

		src = src.Parent()
		dst = dst.Parent()
	}
	return nil
}

func (g HierarchicalDigraph) GetLevel(level int) NodeRefs {
	refs := NewNodeRefs()
	for _, n := range g.members.nodeList {
		if nodeDepth(n) == level {
			refs.Add(n)
		}
	}
	return refs
}
