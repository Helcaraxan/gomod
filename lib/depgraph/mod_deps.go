package depgraph

import (
	"bytes"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/lib/internal/util"
)

func (g *ModuleGraph) overlayModuleDependencies() error {
	g.log.Debug("Overlaying module-based dependency information over the import dependency graph.")

	indirectsMap, err := g.getIndirectDeps()
	if err != nil {
		return err
	}

	raw, _, err := util.RunCommand(g.log, g.Main.Info.Dir, "go", "mod", "graph")
	if err != nil {
		return err
	}

	for _, depString := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		g.log.Debug("Parsing dependency", zap.String("reference", depString))
		modDep, ok := g.parseDependency(depString)
		if !ok {
			continue
		}

		if indirectsMap[modDep.source.Name()][modDep.target.Name()] {
			g.log.Debug("Skipping indirect dependency.", zap.String("source", modDep.source.Name()), zap.String("target", modDep.target.Name()))
			continue
		}

		g.log.Debug(
			"Overlaying module dependency.",
			zap.String("version", modDep.targetVersion),
			zap.String("source", modDep.source.Name()),
			zap.String("target", modDep.target.Name()),
		)
		if dep, ok := modDep.source.Successors.Get(modDep.target.Name()); !ok {
			modDep.source.Successors.Add(&DependencyReference{
				Module:            modDep.target,
				VersionConstraint: modDep.targetVersion,
			})
		} else {
			dep.VersionConstraint = modDep.targetVersion
		}

		if dep, ok := modDep.target.Predecessors.Get(modDep.source.Name()); !ok {
			modDep.target.Predecessors.Add(&DependencyReference{
				Module:            modDep.source,
				VersionConstraint: modDep.sourceVersion,
			})
		} else {
			dep.VersionConstraint = modDep.sourceVersion
		}
	}

	return nil
}

type moduleDependency struct {
	source        *Module
	sourceVersion string
	target        *Module
	targetVersion string
}

func (g *ModuleGraph) parseDependency(depString string) (*moduleDependency, bool) {
	depContent := depRE.FindStringSubmatch(depString)
	if len(depContent) == 0 {
		g.log.Warn("Skipping ill-formed line in 'go mod graph' output.", zap.String("line", depString))
		return nil, false
	}

	sourceName, sourceVersion := depContent[1], depContent[2]
	targetName, targetVersion := depContent[3], depContent[4]

	source, ok := g.GetModule(sourceName)
	if !ok {
		g.log.Warn("Encountered a dependency edge starting at an unknown module.", zap.String("source", sourceName), zap.String("target", targetName))
		return nil, false
	}
	target, ok := g.GetModule(targetName)
	if !ok {
		g.log.Warn("Encountered a dependency edge ending at an unknown module.", zap.String("source", sourceName), zap.String("target", targetName))
		return nil, false

	}

	if sourceVersion != source.Info.Version {
		g.log.Debug(
			"Skipping edge as we are not using the specified source version.",
			zap.String("source", sourceName),
			zap.String("version", sourceVersion),
			zap.String("target", targetName),
		)
		return nil, false
	}

	return &moduleDependency{
		source:        source,
		sourceVersion: sourceVersion,
		target:        target,
		targetVersion: targetVersion,
	}, true
}

func (g *ModuleGraph) getIndirectDeps() (map[string]map[string]bool, error) {
	indirectsMap := map[string]map[string]bool{}

	for _, dep := range g.Dependencies.List() {
		log := g.log.With(zap.String("module", dep.Name()))
		log.Debug("Finding indirect dependencies for module.")

		modContent, err := ioutil.ReadFile(dep.Info.GoMod)
		if os.IsNotExist(err) {
			// This is mostly useful for tests where we don't want to write mod files for every test dependency.
			indirectsMap[dep.Name()] = map[string]bool{}
		} else if err != nil {
			g.log.Error("Failed to read content of go.mod file.", zap.String("path", dep.Info.GoMod), zap.Error(err))
			return nil, err
		}

		indirects := map[string]bool{}
		indirectDepRE := regexp.MustCompile(`^	([^\s]+) [^\s]+ // indirect$`)
		for _, line := range bytes.Split(modContent, []byte("\n")) {
			if m := indirectDepRE.FindSubmatch(line); len(m) == 2 {
				g.log.Debug("Found indirect dependency.", zap.String("dependency", string(m[1])))
				indirects[string(m[1])] = true
			}
		}
		indirectsMap[dep.Name()] = indirects
	}

	return indirectsMap, nil
}
