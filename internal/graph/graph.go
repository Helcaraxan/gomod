package graph

import (
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/internal/logger"
)

var (
	ErrNilGraph          = errors.New("cannot operate on nil-graph")
	ErrNilNode           = errors.New("cannot operate on nil-node")
	ErrNodeAlreadyExists = errors.New("node with identical hash already exists in graph")
	ErrNodeNotFound      = errors.New("node not found")
	ErrEdgeSelf          = errors.New("self-edges are not allowed")
	ErrEdgeCrossLevel    = errors.New("edges not allowed between nodes of different hierarchical levels")
)

type graphErr struct {
	err error
	ctx string
}

func (e graphErr) Error() string {
	return fmt.Sprintf("%s: %v", e.ctx, e.err)
}

func (e graphErr) Unwrap() error {
	return e.err
}

type HierarchicalDigraph struct {
	log     *logger.Logger
	members NodeRefs
}

func NewHierarchicalDigraph(log *logger.Logger) *HierarchicalDigraph {
	return &HierarchicalDigraph{
		log:     log,
		members: NewNodeRefs(),
	}
}

func (g HierarchicalDigraph) GetNode(hash string) (Node, error) {
	n, _ := g.members.Get(hash)
	if n == nil {
		return nil, &graphErr{
			err: ErrNodeNotFound,
			ctx: fmt.Sprintf("node hash %q", hash),
		}
	}
	return n, nil
}

func (g *HierarchicalDigraph) AddNode(node Node) error {
	if g == nil {
		return ErrNilGraph
	} else if nodeIsNil(node) {
		return ErrNilNode
	}
	g.log.Debug("Adding node to graph.", zap.Stringer("node", node))

	if n, _ := g.members.Get(node.Hash()); n != nil {
		return &graphErr{
			err: ErrNodeAlreadyExists,
			ctx: fmt.Sprintf("node hash %q", node.Hash()),
		}
	}

	if p := node.Parent(); !nodeIsNil(p) {
		if n, _ := g.members.Get(p.Hash()); nodeIsNil(n) {
			return &graphErr{
				err: ErrNodeNotFound,
				ctx: fmt.Sprintf("node hash %q", node.Hash()),
			}
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
	g.log.Debug("Deleting node from graph.", zap.String("hash", hash))
	g.log.AddIndent()
	defer g.log.RemoveIndent()

	target, _ := g.members.Get(hash)
	if target == nil {
		return &graphErr{
			err: ErrNodeNotFound,
			ctx: fmt.Sprintf("node hash %q", hash),
		}
	}

	for _, pred := range target.Predecessors().List() {
		g.disconnectNodeFromTarget(pred, target)
	}

	for _, succ := range target.Successors().List() {
		g.disconnectNodeFromTarget(target, succ)
	}

	g.deleteNode(target)

	for !nodeIsNil(target) {
		p := target.Parent()
		if nodeIsNil(p) || p.Children().Len() > 0 {
			break
		}

		g.log.AddIndent()
		defer g.log.RemoveIndent()

		if err := g.DeleteNode(p.Hash()); err != nil {
			return err
		}
		target = target.Parent()
	}
	return nil
}

func (g *HierarchicalDigraph) AddEdge(src Node, dst Node) error {
	if g == nil {
		return ErrNilGraph
	} else if nodeIsNil(src) || nodeIsNil(dst) {
		return ErrNilNode
	}

	if _, w := g.members.Get(src.Hash()); w == 0 {
		return &graphErr{
			err: ErrNodeNotFound,
			ctx: fmt.Sprintf("node hash %q", src.Hash()),
		}
	} else if _, w := g.members.Get(dst.Hash()); w == 0 {
		return &graphErr{
			err: ErrNodeNotFound,
			ctx: fmt.Sprintf("node hash %q", dst.Hash()),
		}
	}

	if nodeDepth(src) != nodeDepth(dst) {
		return &graphErr{
			err: ErrEdgeCrossLevel,
			ctx: fmt.Sprintf("node %q (%d) - node %q (%d)", src.Hash(), nodeDepth(src), dst.Hash(), nodeDepth(dst)),
		}
	}

	for {
		if nodeIsNil(src) || nodeIsNil(dst) || src.Hash() == dst.Hash() {
			break
		}

		g.log.Debug("Adding edge to graph.", zap.String("source-hash", src.Hash()), zap.String("target-hash", dst.Hash()))
		src.Successors().Add(dst)
		dst.Predecessors().Add(src)

		src = src.Parent()
		dst = dst.Parent()
		g.log.AddIndent()
		defer g.log.RemoveIndent()
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
		return &graphErr{
			err: ErrNodeNotFound,
			ctx: fmt.Sprintf("node hash %q", src.Hash()),
		}
	} else if _, w := g.members.Get(dst.Hash()); w == 0 {
		return &graphErr{
			err: ErrNodeNotFound,
			ctx: fmt.Sprintf("node hash %q", dst.Hash()),
		}
	}

	g.disconnectNodeFromTarget(src, dst)

	for {
		g.log.AddIndent()
		defer g.log.RemoveIndent()

		src = src.Parent()
		dst = dst.Parent()
		if nodeIsNil(src) || nodeIsNil(dst) || src.Hash() == dst.Hash() {
			break
		}

		g.log.Debug("Unregistring edge from node parents.", zap.String("source-hash", src.Hash()), zap.String("target-hash", dst.Hash()))
		src.Successors().Delete(dst.Hash())
		dst.Predecessors().Delete(src.Hash())
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

func (g HierarchicalDigraph) disconnectNodeFromTarget(n Node, target Node) {
	g.log.Debug("Disconnecting nodes.", zap.String("source-hash", n.Hash()), zap.String("target-hash", target.Hash()))
	g.log.AddIndent()
	defer g.log.RemoveIndent()

	if n.Children() != nil && n.Children().Len() > 0 {
		for _, child := range n.Children().List() {
			g.disconnectNodeFromTarget(child, target)
		}
	}

	for _, succ := range n.Successors().List() {
		if isChild(succ, target) {
			g.log.Debug("Deleting edge from graph.", zap.String("source-hash", n.Hash()), zap.String("target-hash", succ.Hash()))
			n.Successors().Wipe(succ.Hash())
			succ.Predecessors().Wipe(n.Hash())
			g.log.Debug("New successors.", zap.String("hash", n.Hash()), zap.Stringer("succs", n.Successors()))
		}
	}
}

func (g HierarchicalDigraph) deleteNode(n Node) {
	if n.Children() != nil && n.Children().Len() > 0 {
		g.log.AddIndent()
		for _, child := range n.Children().List() {
			g.deleteNode(child)
		}
		g.log.RemoveIndent()
	}

	g.log.Debug("Removing node from members list.", zap.String("hash", n.Hash()))
	g.members.Delete(n.Hash())
	if p := n.Parent(); !nodeIsNil(p) {
		p.Children().Delete(n.Hash())
	}
}

func isChild(n Node, p Node) bool {
	if nodeIsNil(p) {
		return false
	}
	for !nodeIsNil(n) {
		if n.Hash() == p.Hash() {
			return true
		}
		n = n.Parent()
	}
	return false
}
