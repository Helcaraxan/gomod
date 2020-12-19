package depgraph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/lib/internal/util"
	"github.com/Helcaraxan/gomod/lib/modules"
)

func (g *Graph) buildImportGraph() error {
	g.log.Debug("Building initial dependency graph based on the import graph.")

	err := g.retrieveTransitiveImports([]string{fmt.Sprintf("%s/...", g.Main.Info.Path)})
	if err != nil {
		return err
	}

	for _, node := range g.Packages.List() {
		pkg := node.(*Package)

		for _, imp := range pkg.Info.Imports {
			targetNode, ok := g.Packages.Get(imp)
			if !ok {
				g.log.Error("Detected import of unknown package.", zap.String("package", imp))
				continue
			}
			targetPkg := targetNode.(*Package)
			pkg.Successors().Add(targetPkg)
			targetPkg.Predecessors().Add(pkg)

			if pkg.Parent.Name() == targetPkg.Parent.Name() {
				continue
			}

			pkg.Parent.Successors().Add(&ModuleReference{
				Module:            targetPkg.Parent,
				VersionConstraint: targetPkg.Parent.SelectedVersion(),
			})
			targetPkg.Parent.Predecessors().Add(&ModuleReference{
				Module:            pkg.Parent,
				VersionConstraint: pkg.Parent.SelectedVersion(),
			})
		}
	}

	return nil
}

func (g *Graph) retrieveTransitiveImports(pkgs []string) error {
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

func (g *Graph) retrievePackageInfo(packages []string) (imports []string, err error) {
	stdout, _, err := util.RunCommand(g.log, g.Main.Info.Dir, "go", append([]string{"list", "-json"}, packages...)...)
	if err != nil {
		g.log.Error("Failed to list imports for packages.", zap.Strings("packages", packages), zap.Error(err))
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(stdout))

	isStandardLib := func(pkg string) bool {
		return !strings.Contains(strings.Split(pkg, "/")[0], ".")
	}

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
		parent, ok := g.GetModule(pkgInfo.Module.Path)
		if !ok {
			g.log.Error("Encountered package in unknown module.", zap.String("package", pkgInfo.ImportPath), zap.String("module", pkgInfo.Module.Path))
			continue
		}

		pkg := &Package{
			Info:         pkgInfo,
			Parent:       parent,
			predecessors: NewEdges(),
			successors:   NewEdges(),
		}
		g.Packages.Add(pkg)
		g.log.Debug("Added import information for package", zap.String("package", pkg.Name()), zap.String("module", parent.Name()))

		importCandidates := make([]string, len(pkgInfo.Imports))
		copy(importCandidates, pkgInfo.Imports)
		if parent.Name() == g.Main.Name() {
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
