package depgraph

import (
	"bytes"
	"io/ioutil"
	"regexp"
	"strings"

	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/internal/util"
)

func (g *Graph) overlayModuleDependencies() error {
	g.log.Debug("Overlaying module-based dependency information over the import dependency graph.")

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

		g.log.Debug(
			"Overlaying module dependency.",
			zap.String("version", modDep.targetVersion),
			zap.String("source", modDep.source.Name()),
			zap.String("target", modDep.target.Name()),
		)
		err = g.Graph.AddEdge(modDep.source, modDep.target)
		if err != nil {
			return err
		}
		modDep.source.VersionConstraints[modDep.target.Hash()] = VersionConstraint{
			Source: modDep.sourceVersion,
			Target: modDep.targetVersion,
		}
	}

	if err := g.markIndirects(); err != nil {
		return err
	}

	return nil
}

type moduleDependency struct {
	source        *Module
	sourceVersion string
	target        *Module
	targetVersion string
}

func (g *Graph) parseDependency(depString string) (*moduleDependency, bool) {
	depContent := depRE.FindStringSubmatch(depString)
	if len(depContent) == 0 {
		g.log.Warn("Skipping ill-formed line in 'go mod graph' output.", zap.String("line", depString))
		return nil, false
	}

	sourceName, sourceVersion := depContent[1], depContent[2]
	targetName, targetVersion := depContent[3], depContent[4]

	source, ok := g.getModule(sourceName)
	if !ok {
		g.log.Warn("Encountered a dependency edge starting at an unknown module.", zap.String("source", sourceName), zap.String("target", targetName))
		return nil, false
	}
	target, ok := g.getModule(targetName)
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
	g.log.Debug(
		"Recording module dependency.",
		zap.String("source", sourceName),
		zap.String("version", sourceVersion),
		zap.String("target", targetName),
	)

	return &moduleDependency{
		source:        source,
		sourceVersion: sourceVersion,
		target:        target,
		targetVersion: targetVersion,
	}, true
}

func (g *Graph) markIndirects() error {
	for _, node := range g.Graph.GetLevel(int(LevelModules)).List() {
		module := node.(*Module)

		log := g.log.With(zap.String("module", module.Name()))
		log.Debug("Finding indirect dependencies for module.")

		if module.Info.GoMod == "" {
			// This occurs when we are under tests and can be skipped safely.
			continue
		}

		modContent, err := ioutil.ReadFile(module.Info.GoMod)
		if err != nil {
			g.log.Error("Failed to read content of go.mod file.", zap.String("path", module.Info.GoMod), zap.Error(err))
			return err
		}

		indirectDepRE := regexp.MustCompile(`^	([^\s]+) [^\s]+ // indirect$`)
		for _, line := range bytes.Split(modContent, []byte("\n")) {
			if m := indirectDepRE.FindSubmatch(line); len(m) == 2 {
				g.log.Debug("Found indirect dependency.", zap.String("consumer", module.Name()), zap.String("dependency", string(m[1])))
				module.Indirects[string(m[1])] = true
			}
		}
	}
	return nil
}
