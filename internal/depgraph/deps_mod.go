package depgraph

import (
	"bytes"
	"io/ioutil"
	"regexp"
	"strings"

	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/internal/logger"
	"github.com/Helcaraxan/gomod/internal/util"
)

func (g *DepGraph) overlayModuleDependencies(dl *logger.Builder) error {
	log := dl.Domain(logger.ModuleDependencyDomain)
	log.Debug("Overlaying module-based dependency information over the import dependency graph.")

	raw, _, err := util.RunCommand(log, g.Main.Info.Dir, "go", "mod", "graph")
	if err != nil {
		return err
	}

	for _, depString := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		log.Debug("Parsing dependency", zap.String("reference", depString))
		modDep, ok := g.parseDependency(log, depString)
		if !ok {
			continue
		}

		log.Debug(
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

	if err := g.markIndirects(log); err != nil {
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

func (g *DepGraph) parseDependency(log *logger.Logger, depString string) (*moduleDependency, bool) {
	depContent := depRE.FindStringSubmatch(depString)
	if len(depContent) == 0 {
		log.Warn("Skipping ill-formed line in 'go mod graph' output.", zap.String("line", depString))
		return nil, false
	}

	sourceName, sourceVersion := depContent[1], depContent[2]
	targetName, targetVersion := depContent[3], depContent[4]

	source, ok := g.getModule(sourceName)
	if !ok {
		log.Warn("Encountered a dependency edge starting at an unknown module.", zap.String("source", sourceName), zap.String("target", targetName))
		return nil, false
	}
	target, ok := g.getModule(targetName)
	if !ok {
		log.Warn("Encountered a dependency edge ending at an unknown module.", zap.String("source", sourceName), zap.String("target", targetName))
		return nil, false

	}

	if sourceVersion != source.Info.Version {
		log.Debug(
			"Skipping edge as we are not using the specified source version.",
			zap.String("source", sourceName),
			zap.String("version", sourceVersion),
			zap.String("target", targetName),
		)
		return nil, false
	}
	log.Debug(
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

func (g *DepGraph) markIndirects(log *logger.Logger) error {
	for _, node := range g.Graph.GetLevel(int(LevelModules)).List() {
		module := node.(*Module)

		log := log.With(zap.String("module", module.Name()))
		log.Debug("Finding indirect dependencies for module.")

		if module.Info.GoMod == "" {
			// This occurs when we are under tests and can be skipped safely.
			continue
		}

		modContent, err := ioutil.ReadFile(module.Info.GoMod)
		if err != nil {
			log.Error("Failed to read content of go.mod file.", zap.String("path", module.Info.GoMod), zap.Error(err))
			return err
		}

		indirectDepRE := regexp.MustCompile(`^	([^\s]+) [^\s]+ // indirect$`)
		for _, line := range bytes.Split(modContent, []byte("\n")) {
			if m := indirectDepRE.FindSubmatch(line); len(m) == 2 {
				log.Debug("Found indirect dependency.", zap.String("consumer", module.Name()), zap.String("dependency", string(m[1])))
				module.Indirects[string(m[1])] = true
			}
		}
	}
	return nil
}
