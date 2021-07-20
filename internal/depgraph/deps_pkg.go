package depgraph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/internal/graph"
	"github.com/Helcaraxan/gomod/internal/logger"
	"github.com/Helcaraxan/gomod/internal/modules"
	"github.com/Helcaraxan/gomod/internal/util"
)

func (g *DepGraph) buildImportGraph(dl *logger.Builder) error {
	log := dl.Domain(logger.PackageInfoDomain)
	log.Debug("Building initial dependency graph based on the import graph.")

	err := g.retrieveTransitiveImports(log, []string{fmt.Sprintf("%s/...", g.Main.Info.Path)})
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
				log.Error("Detected import of unknown package.", zap.String("package", imp))
				continue
			}

			log.Debug(
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

	if err = g.markNonTestDependencies(log); err != nil {
		return err
	}

	return nil
}

func (g *DepGraph) markNonTestDependencies(log *logger.Logger) error {
	log.Debug("Marking non-test dependencies.")

	var todo []graph.Node
	seen := map[string]bool{}

	for _, mainPkg := range g.Main.packages.List() {
		if strings.HasSuffix(mainPkg.(*Package).Info.Name, "_test") {
			log.Debug("Skipping main module package as it is a test-only package.", zap.String("package", mainPkg.Name()))
			continue
		}

		todo = append(todo, mainPkg)
		seen[mainPkg.Name()] = true
	}

	for len(todo) > 0 {
		next := todo[0]
		todo = todo[1:]

		log.Debug("Marking package as non-test dependency.", zap.String("package", next.Name()))
		next.(*Package).isNonTestDependency = true
		next.Parent().(*Module).isNonTestDependency = true

		for _, imp := range next.(*Package).Info.Imports {
			if isStandardLib(imp) {
				continue
			}

			dep, err := g.Graph.GetNode(packageHash(imp))
			if err != nil {
				return err
			}

			if !seen[dep.Name()] {
				todo = append(todo, dep)
				seen[dep.Name()] = true
			}
		}
	}
	return nil
}

func (g *DepGraph) retrieveTransitiveImports(log *logger.Logger, pkgs []string) error {
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

		imports, err := g.retrievePackageInfo(log, query)
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

func (g *DepGraph) retrievePackageInfo(log *logger.Logger, pkgs []string) (imports []string, err error) {
	stdout, _, err := util.RunCommand(log, g.Main.Info.Dir, "go", append([]string{"list", "-json", "-mod=mod"}, pkgs...)...)
	if err != nil {
		log.Error("Failed to list imports for packages.", zap.Strings("packages", pkgs), zap.Error(err))
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(stdout))

	for {
		pkgInfo := &modules.PackageInfo{}
		if err = dec.Decode(pkgInfo); err != nil {
			if err == io.EOF {
				break
			} else {
				log.Error("Failed to parse go list output.", zap.Error(err))
				return nil, err
			}
		}
		parentModule, ok := g.getModule(pkgInfo.Module.Path)
		if !ok {
			log.Error("Encountered package in unknown module.", zap.String("package", pkgInfo.ImportPath), zap.String("module", pkgInfo.Module.Path))
			continue
		}

		pkg := NewPackage(pkgInfo, parentModule)
		_ = g.Graph.AddNode(pkg)
		log.Debug("Added import information for package", zap.String("package", pkg.Name()), zap.String("module", parentModule.Name()))

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
