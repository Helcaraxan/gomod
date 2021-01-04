package depgraph

import (
	"fmt"
	"math"
	"strings"

	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/internal/graph"
	"github.com/Helcaraxan/gomod/internal/query"
)

func (g *DepGraph) ApplyQuery(q query.Expr, level Level) error {
	targetSet, err := g.computeSet(q, level)
	if err != nil {
		return err
	}

	for _, n := range g.Graph.GetLevel(int(level)).List() {
		if targetSet[n.Name()] {
			continue
		}
		if err = g.Graph.DeleteNode(n.Hash()); err != nil {
			return err
		}
	}
	return nil
}

func (ns nodeSet) String() string {
	var s []string
	for n := range ns {
		s = append(s, n)
	}
	return strings.Join(s, ", ")
}

func (g *DepGraph) computeSet(q query.Expr, level Level) (set nodeSet, err error) {
	set = nodeSet{}
	defer func() {
		g.log.Debug("Found nodeset.", zap.Stringer("nodes", set))
	}()

	g.log.Debug("Computing matching nodes.", zap.Stringer("query", q))
	switch tq := q.(type) {
	case *query.ExprArgsList, *query.ExprBool, *query.ExprInteger:
		return nil, fmt.Errorf("invalid query on graph %v (%v)", q, q.Pos())
	case *query.ExprString:
		return g.computeSetNameMatch(tq, level)
	case query.BinaryExpr:
		return g.computeSetBinaryOp(tq, level)
	case *query.ExprFunc:
		return g.computeSetFunc(tq, level)
	}

	return set, nil
}

func (g *DepGraph) computeSetNameMatch(expr *query.ExprString, level Level) (nodeSet, error) {
	var withTestDeps bool

	parts := strings.Split(expr.Value(), ":")
	if len(parts) > 2 {
		return nil, fmt.Errorf("query '%v' contains more than one ':'", expr)
	} else if len(parts) == 2 {
		switch parts[1] {
		case "test":
			withTestDeps = true
		default:
			return nil, fmt.Errorf("undefined annotation '%s' in path query '%v'", parts[1], expr)
		}
	}
	q := parts[0]

	var matcher func(hash string) bool
	if target := strings.TrimSuffix(q, "/..."); target != q {
		matcher = func(name string) bool {
			return strings.HasPrefix(name, target)
		}
	} else {
		matcher = func(name string) bool {
			return name == target
		}
	}

	set := nodeSet{}
	for _, node := range g.Graph.GetLevel(int(level)).List() {
		p, ok := node.(*Package)
		switch {
		case !withTestDeps && ((ok && strings.HasSuffix(p.Info.Name, "_test")) || node.(testAnnotated).isTestDependency()):
			g.log.Debug("Discarded node as it is a test dependency.", zap.String("name", node.Name()))
		case !matcher(node.Name()):
			g.log.Debug("Discarded node as its name did not match the filter.", zap.String("name", node.Name()))
		default:
			g.log.Debug("Match found.", zap.String("name", node.Name()))
			set[node.Name()] = true
		}
	}

	if len(set) == 0 {
		g.log.Warn("Empty query result.", zap.Stringer("query", expr))
	}
	return set, nil
}

func (g *DepGraph) computeSetBinaryOp(expr query.BinaryExpr, level Level) (set nodeSet, err error) {
	defer func() {
		if err == nil && len(set) == 0 {
			g.log.Warn("Empty query result.", zap.Stringer("query", expr))
		}
	}()

	lhs, err := g.computeSet(expr.Operands().LHS, level)
	if err != nil {
		return nil, err
	}
	rhs, err := g.computeSet(expr.Operands().RHS, level)
	if err != nil {
		return nil, err
	}

	switch expr.(type) {
	case *query.ExprDelta:
		return lhs.delta(rhs), nil
	case *query.ExprIntersect:
		return lhs.inter(rhs), nil
	case *query.ExprSubtract:
		return lhs.subtract(rhs), nil
	case *query.ExprUnion:
		return lhs.union(rhs), nil
	}
	return nil, fmt.Errorf("unexpected operator expression %v", expr)
}

func (g *DepGraph) computeSetFunc(expr query.FuncExpr, level Level) (nodeSet, error) {
	switch expr.Name() {
	case "deps":
		return g.computeSetGraphTraversal(expr, forwards, level)
	case "rdeps":
		return g.computeSetGraphTraversal(expr, backwards, level)
	case "shared":
		return g.sharedFunc(expr, level)
	default:
		return nil, fmt.Errorf("unknown function %q", expr.Name())
	}
}

type traversalDirection uint8

const (
	forwards traversalDirection = iota
	backwards
)

func (g *DepGraph) computeSetGraphTraversal(expr query.FuncExpr, direction traversalDirection, level Level) (nodeSet, error) {
	args := expr.Args()
	if len(args.Args()) == 0 {
		return nil, fmt.Errorf("function takes at least one argument but received none (%v)", args.Pos())
	} else if len(args.Args()) > 2 {
		return nil, fmt.Errorf("function takes at most 2 arguments but received %d (%v)", len(args.Args()), args.Pos())
	}

	maxDepth := math.MaxInt64
	if len(args.Args()) == 2 {
		v, ok := args.Args()[1].(*query.ExprInteger)
		if !ok {
			return nil, fmt.Errorf("function's second argument must be an integer but got %v (%v)", args.Args()[1], args.Args()[1].Pos())
		}
		maxDepth = v.Value()
	}
	g.log.Debug("Maximum depths for traversals set.", zap.Int("maxDepth", maxDepth))

	var iterateFunc func(graph.Node) []graph.Node
	switch direction {
	case forwards:
		iterateFunc = func(n graph.Node) []graph.Node { return n.Successors().List() }
	case backwards:
		iterateFunc = func(n graph.Node) []graph.Node { return n.Predecessors().List() }
	}

	sources, err := g.computeSet(args.Args()[0], level)
	if err != nil {
		return nil, err
	}

	set := nodeSet{}
	for src := range sources {
		var h string
		switch level {
		case LevelModules:
			h = moduleHash(src)
		case LevelPackages:
			h = packageHash(src)
		}
		node, _ := g.Graph.GetNode(h)

		seen := nodeSet{}
		todo := []struct {
			n graph.Node
			d int
		}{{node, 0}}
		for len(todo) > 0 {
			next := todo[0]
			todo = todo[1:]

			set[next.n.Name()] = true
			if next.d >= maxDepth {
				g.log.Debug("Maximum depth reached.", zap.String("node", next.n.Name()))
				continue
			}

			for _, dep := range iterateFunc(next.n) {
				if seen[dep.Name()] {
					continue
				}

				g.log.Debug("Enqueing new node.", zap.String("node", dep.Name()), zap.Int("depth", next.d+1))
				todo = append(todo, struct {
					n graph.Node
					d int
				}{dep, next.d + 1})
				seen[dep.Name()] = true
			}
		}
	}
	return set, nil
}

func (g *DepGraph) sharedFunc(expr query.FuncExpr, level Level) (nodeSet, error) {
	args := expr.Args()
	if len(args.Args()) != 1 {
		return nil, fmt.Errorf("the 'shared' function takes only a single argument but received %v (%v)", args, args.Pos())
	}

	set, err := g.computeSet(args.Args()[0], level)
	if err != nil {
		return nil, err
	}

	nodesInSet := func(set nodeSet, list []graph.Node) int {
		var c int
		for _, n := range list {
			if set[n.Name()] {
				c++
			}
		}
		return c
	}

	var todo []graph.Node
	for src := range set {
		var h string
		switch level {
		case LevelModules:
			h = moduleHash(src)
		case LevelPackages:
			h = packageHash(src)
		}
		n, _ := g.Graph.GetNode(h)

		if nodesInSet(set, n.Successors().List()) == 0 && nodesInSet(set, n.Predecessors().List()) == 1 {
			todo = append(todo, n)
		}
	}

	for len(todo) > 0 {
		next := todo[0]
		todo = todo[1:]

		g.log.Debug("Removing node from set.", zap.String("name", next.Name()))
		delete(set, next.Name())

		pred := next.Predecessors().List()[0]
		if err := g.Graph.DeleteNode(next.Hash()); err != nil {
			return nil, err
		}

		if nodesInSet(set, pred.Successors().List()) == 0 && nodesInSet(set, pred.Predecessors().List()) == 1 {
			todo = append(todo, pred)
		}
	}

	if len(set) == 0 {
		g.log.Warn("Empty query result.", zap.Stringer("query", expr))
	}
	return set, nil
}

type testAnnotated interface {
	isTestDependency() bool
}

type nodeSet map[string]bool

func (ns nodeSet) union(rhs nodeSet) nodeSet {
	set := nodeSet{}
	for k := range ns {
		set[k] = true
	}
	for k := range rhs {
		set[k] = true
	}
	return set
}

func (ns nodeSet) subtract(rhs nodeSet) nodeSet {
	set := nodeSet{}
	for k := range ns {
		if !rhs[k] {
			set[k] = true
		}
	}
	return set
}

func (ns nodeSet) inter(rhs nodeSet) nodeSet {
	set := nodeSet{}
	for k := range ns {
		if rhs[k] {
			set[k] = true
		}
	}
	return set
}

func (ns nodeSet) delta(rhs nodeSet) nodeSet {
	set := nodeSet{}
	for k := range ns {
		if !rhs[k] {
			set[k] = true
		}
	}
	for k := range rhs {
		if !ns[k] {
			set[k] = true
		}
	}
	return set
}
