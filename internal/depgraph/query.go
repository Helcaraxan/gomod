package depgraph

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/bmatcuk/doublestar/v3"
	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/internal/graph"
	"github.com/Helcaraxan/gomod/internal/logger"
	"github.com/Helcaraxan/gomod/internal/query"
)

var ErrInvalidQuery = errors.New("invalid query")

type queryErr struct {
	err  string
	expr query.Expr
}

func (e queryErr) Error() string {
	return fmt.Sprintf("%v: %v - %s", e.expr.Pos(), e.expr, e.err)
}

func (e queryErr) Unwrap() error {
	return ErrInvalidQuery
}

func (g *DepGraph) ApplyQuery(dl *logger.Builder, q query.Expr, level Level) error {
	log := dl.Domain(logger.QueryDomain)

	targetSet, err := g.computeSet(log, q, level)
	if err != nil {
		return err
	}

	log.Debug("Removing unselected nodes.")
	for _, n := range g.Graph.GetLevel(int(level)).List() {
		if targetSet[n.Name()] {
			continue
		}
		log.Debug("Removing node.", zap.String("node", n.Name()))
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

func (g *DepGraph) computeSet(log *logger.Logger, expr query.Expr, level Level) (set nodeSet, err error) {
	set = nodeSet{}
	defer func() {
		log.Debug("Found nodeset.", zap.Stringer("query", expr), zap.Stringer("nodes", set))
	}()

	log.Debug("Computing matching nodes.", zap.Stringer("query", expr))
	switch tq := expr.(type) {
	case *query.ExprArgsList, *query.ExprBool, *query.ExprInteger:
		return nil, &queryErr{
			err:  "cannot process integers, booleans or lists",
			expr: tq,
		}
	case *query.ExprString:
		return g.computeSetNameMatch(log, tq, level)
	case query.BinaryExpr:
		return g.computeSetBinaryOp(log, tq, level)
	case *query.ExprFunc:
		return g.computeSetFunc(log, tq, level)
	default:
		return nil, &queryErr{
			err:  "unexpected query expression",
			expr: expr,
		}
	}
}

func (g *DepGraph) computeSetNameMatch(log *logger.Logger, expr *query.ExprString, level Level) (nodeSet, error) {
	var withTestDeps bool

	parts := strings.Split(expr.Value(), ":")
	if len(parts) > 2 {
		return nil, &queryErr{
			err:  fmt.Sprintf("expression contains more than one ':' character"),
			expr: expr,
		}
	} else if len(parts) == 2 {
		switch parts[1] {
		case "test":
			withTestDeps = true
		default:
			return nil, &queryErr{
				err:  fmt.Sprintf("undefined path annotation '%s'", parts[1]),
				expr: expr,
			}
		}
	}

	q := parts[0]
	if _, err := doublestar.Match(q, ""); err != nil {
		return nil, &queryErr{
			err:  fmt.Sprintf("invalid query: %v", err),
			expr: expr,
		}
	}

	set := nodeSet{}
	for _, node := range g.Graph.GetLevel(int(level)).List() {
		p, ok := node.(*Package)
		matches, _ := doublestar.Match(q, node.Name())
		switch {
		case !withTestDeps && ((ok && strings.HasSuffix(p.Info.Name, "_test")) || node.(testAnnotated).isTestDependency()):
			log.Debug("Discarded node as it is a test dependency.", zap.String("name", node.Name()))
		case !matches:
			log.Debug("Discarded node as its name did not match the filter.", zap.String("name", node.Name()))
		default:
			log.Debug("Match found.", zap.String("name", node.Name()))
			set[node.Name()] = true
		}
	}

	if len(set) == 0 {
		log.Warn("Empty query result.", zap.Stringer("query", expr))
	}
	return set, nil
}

func (g *DepGraph) computeSetBinaryOp(log *logger.Logger, expr query.BinaryExpr, level Level) (set nodeSet, err error) {
	defer func() {
		if err == nil && len(set) == 0 {
			log.Warn("Empty query result.", zap.Stringer("query", expr))
		}
	}()

	lhs, err := g.computeSet(log, expr.Operands().LHS, level)
	if err != nil {
		return nil, err
	}
	rhs, err := g.computeSet(log, expr.Operands().RHS, level)
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
	return nil, &queryErr{
		err:  fmt.Sprintf("unknown operator"),
		expr: expr,
	}
}

func (g *DepGraph) computeSetFunc(log *logger.Logger, expr query.FuncExpr, level Level) (nodeSet, error) {
	switch expr.Name() {
	case "deps":
		return g.computeSetGraphTraversal(log, expr, forwards, level)
	case "rdeps":
		return g.computeSetGraphTraversal(log, expr, backwards, level)
	case "shared":
		return g.sharedFunc(log, expr, level)
	default:
		return nil, &queryErr{
			err:  fmt.Sprintf("unknown function %q", expr.Name()),
			expr: expr,
		}
	}
}

type traversalDirection uint8

const (
	forwards traversalDirection = iota
	backwards
)

func (g *DepGraph) computeSetGraphTraversal(log *logger.Logger, expr query.FuncExpr, direction traversalDirection, level Level) (nodeSet, error) {
	args := expr.Args()
	if len(args.Args()) == 0 {
		return nil, &queryErr{
			err:  "expected at least one argument but received none",
			expr: expr,
		}
	} else if len(args.Args()) > 2 {
		return nil, &queryErr{
			err:  fmt.Sprintf("expected at most 2 arguments but received %d", len(args.Args())),
			expr: expr,
		}
	}

	maxDepth := math.MaxInt64
	if len(args.Args()) == 2 {
		v, ok := args.Args()[1].(*query.ExprInteger)
		if !ok {
			return nil, &queryErr{
				err:  fmt.Sprintf("expected an integer as second argument but got '%v'", args.Args()[0]),
				expr: expr,
			}
		}
		maxDepth = v.Value()
	}
	log.Debug("Maximum depths for traversals set.", zap.Int("maxDepth", maxDepth))

	var iterateFunc func(graph.Node) []graph.Node
	switch direction {
	case forwards:
		iterateFunc = func(n graph.Node) []graph.Node { return n.Successors().List() }
	case backwards:
		iterateFunc = func(n graph.Node) []graph.Node { return n.Predecessors().List() }
	}

	sources, err := g.computeSet(log, args.Args()[0], level)
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
				log.Debug("Maximum depth reached.", zap.String("node", next.n.Name()))
				continue
			}

			for _, dep := range iterateFunc(next.n) {
				if seen[dep.Name()] {
					continue
				}

				log.Debug("Enqueing new node.", zap.String("node", dep.Name()), zap.Int("depth", next.d+1))
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

func (g *DepGraph) sharedFunc(log *logger.Logger, expr query.FuncExpr, level Level) (nodeSet, error) {
	args := expr.Args()
	if len(args.Args()) != 1 {
		return nil, &queryErr{
			err:  fmt.Sprintf("expected a single argument but received '%v'", len(args.Args())),
			expr: expr,
		}
	}

	set, err := g.computeSet(log, args.Args()[0], level)
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

		log.Debug("Removing node from set.", zap.String("name", next.Name()))
		delete(set, next.Name())

		pred := next.Predecessors().List()[0]
		if nodesInSet(set, pred.Successors().List()) == 0 && nodesInSet(set, pred.Predecessors().List()) == 1 {
			todo = append(todo, pred)
		}
	}

	if len(set) == 0 {
		log.Warn("Empty query result.", zap.Stringer("query", expr))
	}
	return set, nil
}

type testAnnotated interface {
	isTestDependency() bool
}

var (
	_ testAnnotated = &Module{}
	_ testAnnotated = &Package{}
)

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
