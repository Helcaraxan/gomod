package depgraph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/internal/graph"
	"github.com/Helcaraxan/gomod/internal/modules"
	"github.com/Helcaraxan/gomod/internal/util"
)

func (g *DepGraph) buildImportGraph() error {
	g.log.Debug("Building initial dependency graph based on the import graph.")

	err := g.retrieveTransitiveImports([]string{fmt.Sprintf("%s/...", g.Main.Info.Path)})
	if err != nil {
		return err
	}

	pkgs := g.Graph.GetLevel(int(LevelPackages))
	for _, node := range pkgs.List() {
		pkg := node.(*Package)

		imports := pkg.Info.Imports
		if pkg.parent.Name() == g.Main.Name() {
			imports = append(imports, pkg.Info.TestImports...)
			imports = append(imports, pkg.Info.XTestImports...)
		}

		for _, imp := range imports {
			if isStandardLib(imp) {
				continue
			}

			targetNode, _ := pkgs.Get(packageHash(imp))
			if targetNode == nil {
				g.log.Error("Detected import of unknown package.", zap.String("package", imp))
				continue
			}

			g.log.Debug(
				"Adding package dependency.",
				zap.String("source", pkg.Name()),
				zap.String("source-module", pkg.Parent().Name()),
				zap.String("target", targetNode.Name()),
				zap.String("target-module", targetNode.Parent().Name()),
			)
			targetPkg := targetNode.(*Package)
			if err = g.Graph.AddEdge(pkg, targetPkg); err != nil {
				return err
			}
		}
	}

	g.markNonTestDependencies()

	return nil
}

func (g *DepGraph) markNonTestDependencies() {
	var todo []graph.Node
	seen := map[string]bool{}

	mainPkgs := g.Main.packages.List()
	for _, mainPkg := range mainPkgs {
		if strings.HasSuffix(mainPkg.(*Package).Info.Name, "_test") {
			continue
		}

		todo = append(todo, mainPkg)
		seen[mainPkg.Name()] = true
	}

	for len(todo) > 0 {
		next := todo[0]
		todo = todo[1:]

		next.(*Package).isNonTestDependency = true
		next.Parent().(*Module).isNonTestDependency = true
		for _, dep := range next.Successors().List() {
			if !seen[dep.Name()] {
				todo = append(todo, dep)
				seen[dep.Name()] = true
			}
		}
	}
}

func (g *DepGraph) retrieveTransitiveImports(pkgs []string) error {
	const maxQueryLength = 950 // This is chosen conservatively to ensure we don't exceed maximum command lengths for 'go list' invocations.

	queued := map[string]bool{}
	for len(pkgs) > 0 {
		queryLength := 0

		cursor := 0
		for {
			if cursor == len(pkgs) || queryLength+len(pkgs[cursor]) > maxQueryLength {
				break
			}
			queryLength += len(pkgs[cursor]) + 1
			cursor++
		}
		query := pkgs[:cursor]
		pkgs = pkgs[cursor:]

		imports, err := g.retrievePackageInfo(query)
		if err != nil {
			return err
		}

		for _, pkg := range imports {
			if !queued[pkg] {
				queued[pkg] = true
				pkgs = append(pkgs, pkg)
			}
		}
	}
	return nil
}

func (g *DepGraph) retrievePackageInfo(packages []string) (imports []string, err error) {
	stdout, _, err := util.RunCommand(g.log, g.Main.Info.Dir, "go", append([]string{"list", "-json"}, packages...)...)
	if err != nil {
		g.log.Error("Failed to list imports for packages.", zap.Strings("packages", packages), zap.Error(err))
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(stdout))

	for {
		pkgInfo := &modules.PackageInfo{}
		if err = dec.Decode(pkgInfo); err != nil {
			if err == io.EOF {
				break
			} else {
				g.log.Error("Failed to parse go list output.", zap.Error(err))
				return nil, err
			}
		}
		parentModule, ok := g.getModule(pkgInfo.Module.Path)
		if !ok {
			g.log.Error("Encountered package in unknown module.", zap.String("package", pkgInfo.ImportPath), zap.String("module", pkgInfo.Module.Path))
			continue
		}

		pkg := NewPackage(pkgInfo, parentModule)
		_ = g.Graph.AddNode(pkg)
		g.log.Debug("Added import information for package", zap.String("package", pkg.Name()), zap.String("module", parentModule.Name()))

		importCandidates := make([]string, len(pkgInfo.Imports))
		copy(importCandidates, pkgInfo.Imports)
		if parentModule.Name() == g.Main.Name() {
			importCandidates = append(importCandidates, pkgInfo.TestImports...)
			importCandidates = append(importCandidates, pkgInfo.XTestImports...)
		}

		for _, candidate := range importCandidates {
			if !isStandardLib(candidate) {
				imports = append(imports, candidate)
			}
		}
	}
	return imports, nil
}

func isStandardLib(pkg string) bool {
	return !strings.Contains(strings.Split(pkg, "/")[0], ".")
}
