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

func (g *ModuleGraph) buildImportGraph() error {
	g.log.Debug("Building initial dependency graph based on the import graph.")

	pkgs, err := g.retrieveTransitiveImports([]string{fmt.Sprintf("%s/...", g.Main.Info.Path)})
	if err != nil {
		return err
	}

	for pkg, info := range pkgs {
		source, ok := g.GetModule(info.module.Path)
		if !ok {
			g.log.Error("Encountered package in unknown module.", zap.String("package", pkg), zap.String("module", info.module.Path))
			continue
		}

		for _, imp := range info.imports {
			targetPkg, ok := pkgs[imp]
			if !ok {
				g.log.Error("Detected import of unknown package.", zap.String("package", imp))
				continue
			}
			target, ok := g.GetModule(targetPkg.module.Path)
			if !ok {
				g.log.Error("Encountered package in unknown module.", zap.String("package", pkg), zap.String("module", targetPkg.module.Path))
				continue
			} else if source.Name() == target.Name() {
				continue
			}

			source.Successors.Add(&ModuleReference{
				Module:            target,
				VersionConstraint: target.SelectedVersion(),
			})
			target.Predecessors.Add(&ModuleReference{
				Module:            source,
				VersionConstraint: source.SelectedVersion(),
			})
		}
	}

	return nil
}

type packageImports struct {
	module  *modules.ModuleInfo
	imports []string
}

func (g *ModuleGraph) retrieveTransitiveImports(pkgs []string) (map[string]packageImports, error) {
	const maxQueryLength = 950 // This is chosen conservatively to ensure we don't exceed maximum command lengths for 'go list' invocations.

	pkgInfos, queued := map[string]packageImports{}, map[string]bool{}
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

		importsMap, err := g.retrievePackageImports(query)
		if err != nil {
			return nil, err
		}
		for pkg := range importsMap {
			queued[pkg] = true
		}

		for pkg, info := range importsMap {
			g.log.Debug("Adding import information for package", zap.String("package", pkg), zap.String("module", info.module.Path))
			pkgInfos[pkg] = info

			for _, imp := range info.imports {
				if !queued[imp] {
					pkgs = append(pkgs, imp)
					queued[imp] = true
				}
			}
		}
	}
	return pkgInfos, nil
}

func (g *ModuleGraph) retrievePackageImports(packages []string) (map[string]packageImports, error) {
	stdout, _, err := util.RunCommand(g.log, g.Main.Info.Dir, "go", append([]string{"list", "-json"}, packages...)...)
	if err != nil {
		g.log.Error("Failed to list imports for packages.", zap.Strings("packages", packages), zap.Error(err))
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(stdout))

	type packageInfo struct {
		ImportPath   string
		Module       *modules.ModuleInfo
		Imports      []string
		TestImports  []string
		XTestImports []string
	}

	isStandardLib := func(pkg string) bool {
		return !strings.Contains(strings.Split(pkg, "/")[0], ".")
	}

	infos := map[string]packageImports{}
	for {
		info := packageInfo{}
		if err = dec.Decode(&info); err != nil {
			if err == io.EOF {
				break
			} else {
				g.log.Error("Failed to parse go list output.", zap.Error(err))
				return nil, err
			}
		}

		var imports []string
		for _, importList := range [][]string{info.Imports, info.TestImports, info.XTestImports} {
			for _, pkg := range importList {
				if !isStandardLib(pkg) {
					imports = append(imports, pkg)
				}
			}
		}

		infos[info.ImportPath] = packageImports{
			module:  info.Module,
			imports: imports,
		}
	}
	return infos, nil
}
