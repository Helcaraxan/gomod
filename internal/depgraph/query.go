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
	targetSet, err := g.computeQuerySet(q, level)
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

type testAnnotated interface {
	isTestDependency() bool
}

type nodeSet map[string]bool

func (ns nodeSet) String() string {
	var s []string
	for n := range ns {
		s = append(s, n)
	}
	return strings.Join(s, ", ")
}

// nolint: gocyclo
func (g *DepGraph) computeQuerySet(q query.Expr, level Level) (set nodeSet, err error) {
	set = nodeSet{}
	defer func() {
		g.log.Debug("Found nodeset.", zap.Stringer("nodes", set))
	}()

	g.log.Debug("Computing matching nodes.", zap.Stringer("query", q))
	switch tq := q.(type) {
	case *query.ExprArgsList, *query.ExprBool, *query.ExprInteger:
		return nil, fmt.Errorf("invalid query on graph %v (%v)", q, q.Pos())

	case *query.ExprString:
		var matcher func(string) bool
		if target := strings.TrimSuffix(tq.Value(), "/..."); target != tq.Value() {
			matcher = func(hash string) bool {
				n, _ := g.Graph.GetNode(hash)
				return !n.(testAnnotated).isTestDependency() && strings.HasPrefix(n.Name(), target)
			}
		} else {
			matcher = func(hash string) bool {
				n, _ := g.Graph.GetNode(hash)
				return !n.(testAnnotated).isTestDependency() && n.Name() == target
			}
		}
		return g.matcherFunc(matcher, level), nil

	case query.BinaryExpr:
		lhs, err := g.computeQuerySet(tq.Operands().LHS, level)
		if err != nil {
			return nil, err
		}
		rhs, err := g.computeQuerySet(tq.Operands().RHS, level)
		if err != nil {
			return nil, err
		}

		switch tq.(type) {
		case *query.ExprDelta:
			for n := range lhs {
				if !rhs[n] {
					set[n] = true
				}
			}
			for n := range rhs {
				if !lhs[n] {
					set[n] = true
				}
			}

		case *query.ExprIntersect:
			for n := range lhs {
				if rhs[n] {
					set[n] = true
				}
			}

		case *query.ExprSubtract:
			for n := range lhs {
				if !rhs[n] {
					set[n] = true
				}
			}

		case *query.ExprUnion:
			for n := range lhs {
				set[n] = true
			}
			for n := range rhs {
				set[n] = true
			}
		}

	case *query.ExprFunc:
		switch tq.Name() {
		case "deps":
			return g.traversalFunc("deps", func(n graph.Node) []graph.Node { return n.Successors().List() }, tq.Args(), level)
		case "rdeps":
			return g.traversalFunc("rdeps", func(n graph.Node) []graph.Node { return n.Predecessors().List() }, tq.Args(), level)
		case "test":
			if len(tq.Args().Args()) != 1 {
				return nil, fmt.Errorf("the 'test' function takes only a single string argument but received %v (%v)", tq.Args(), tq.Args().Pos())
			}
			ts, ok := tq.Args().Args()[0].(*query.ExprString)
			if !ok {
				return nil, fmt.Errorf("the 'test' function takes only a single string argument but received %v (%v)", tq.Args(), tq.Args().Pos())
			}

			var matcher func(string) bool
			if target := strings.TrimSuffix(ts.Value(), "/..."); target != ts.Value() {
				matcher = func(hash string) bool {
					n, _ := g.Graph.GetNode(hash)
					return strings.HasPrefix(n.Name(), target)
				}
			} else {
				matcher = func(hash string) bool {
					n, _ := g.Graph.GetNode(hash)
					return n.Name() == target
				}
			}
			return g.matcherFunc(matcher, level), nil
		default:
			return nil, fmt.Errorf("unknown function %q", tq.Name())
		}
	}

	return set, nil
}

func (g *DepGraph) matcherFunc(matcher func(hash string) bool, level Level) nodeSet {
	set := nodeSet{}
	for _, node := range g.Graph.GetLevel(int(level)).List() {
		if matcher(node.Hash()) {
			set[node.Name()] = true
			g.log.Debug("Match found.", zap.String("name", node.Name()))
		} else {
			g.log.Debug("Discarded node.", zap.String("name", node.Name()))
		}
	}
	return set
}

func (g *DepGraph) traversalFunc(name string, iterate func(graph.Node) []graph.Node, args query.ArgsListExpr, level Level) (nodeSet, error) {
	if len(args.Args()) == 0 {
		return nil, fmt.Errorf("the '%s' function takes at least one argument but received none (%v)", name, args.Pos())
	} else if len(args.Args()) > 2 {
		return nil, fmt.Errorf("the '%s' function takes 2 arguments at most but received %d (%v)", name, len(args.Args()), args.Pos())
	}

	sources, err := g.computeQuerySet(args.Args()[0], level)
	if err != nil {
		return nil, err
	}

	maxDepth := math.MaxInt64
	if len(args.Args()) == 2 {
		v, ok := args.Args()[1].(*query.ExprInteger)
		if !ok {
			return nil, fmt.Errorf("the '%s' function takes an integer as second argument but got %v (%v)", name, args.Args()[1], args.Args()[1].Pos())
		}
		maxDepth = v.Value()
	}
	g.log.Debug("Maximum depths for traversals set.", zap.Int("maxDepth", maxDepth))

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

			for _, dep := range iterate(next.n) {
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
