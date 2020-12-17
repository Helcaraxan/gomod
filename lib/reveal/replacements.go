package reveal

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/lib/depgraph"
	"github.com/Helcaraxan/gomod/lib/modules"
)

type Replacement struct {
	Offender *modules.ModuleInfo
	Original string
	Override string
	Version  string
}

type Replacements struct {
	main     string
	topLevel map[string]string

	replacedModules []string
	originToReplace map[string][]Replacement
}

func (r *Replacements) Print(log *zap.Logger, writer io.Writer, offenders []string, targets []string) error {
	filtered := r.FilterOnOffendingModule(offenders).FilterOnReplacedModule(targets)

	var (
		output     string
		matchFound bool
	)
	for _, origin := range filtered.replacedModules {
		newOutput, match := filtered.printModuleReplacements(origin)
		output += newOutput
		matchFound = matchFound || match
	}

	if matchFound {
		output += fmt.Sprintf("[✓] Match with a top-level replace in '%s'\n", r.main)
	}

	if _, err := writer.Write([]byte(output)); err != nil {
		return fmt.Errorf("failed to print replacements: %v", err)
	}
	return nil
}

func (r *Replacements) FilterOnOffendingModule(offenders []string) *Replacements {
	if len(offenders) == 0 {
		return r
	}
	sort.Strings(offenders)

	filtered := &Replacements{
		main:            r.main,
		topLevel:        map[string]string{},
		originToReplace: map[string][]Replacement{},
	}
	for k, v := range r.topLevel {
		filtered.topLevel[k] = v
	}

	for _, origin := range r.replacedModules {
		unfilteredReplaces := r.originToReplace[origin]

		var filteredReplaces []Replacement
		var rIdx, oIdx int
		for {
			if rIdx == len(unfilteredReplaces) || oIdx == len(offenders) {
				break
			}
			switch {
			case unfilteredReplaces[rIdx].Offender.Path == offenders[oIdx]:
				filteredReplaces = append(filteredReplaces, unfilteredReplaces[rIdx])
				rIdx++
				oIdx++
			case unfilteredReplaces[rIdx].Offender.Path < offenders[oIdx]:
				rIdx++
			case unfilteredReplaces[rIdx].Offender.Path > offenders[oIdx]:
				oIdx++
			}
		}
		if len(filteredReplaces) != 0 {
			filtered.replacedModules = append(filtered.replacedModules, origin)
			filtered.originToReplace[origin] = filteredReplaces
		}
	}
	return filtered
}

func (r *Replacements) FilterOnReplacedModule(originals []string) *Replacements {
	if len(originals) == 0 {
		return r
	}
	sort.Strings(originals)

	filtered := &Replacements{
		main:            r.main,
		topLevel:        map[string]string{},
		originToReplace: map[string][]Replacement{},
	}
	for k, v := range r.topLevel {
		filtered.topLevel[k] = v
	}
	for _, original := range originals {
		if len(r.originToReplace[original]) == 0 {
			continue
		}
		filtered.replacedModules = append(filtered.replacedModules, original)
		replaces := make([]Replacement, len(r.originToReplace[original]))
		copy(replaces, r.originToReplace[original])
		filtered.originToReplace[original] = replaces
	}
	return filtered
}

func (r *Replacements) printModuleReplacements(original string) (string, bool) {
	const (
		matchedMark   = " ✓ "
		unmatchedMark = "   "
	)
	var (
		maxOffenderLength int
		maxOverrideLength int
		maxVersionLength  int
	)

	for _, replacement := range r.originToReplace[original] {
		if len(replacement.Offender.Path) > maxOffenderLength {
			maxOffenderLength = len(replacement.Offender.Path)
		}
		if len(replacement.Override) > maxOverrideLength {
			maxOverrideLength = len(replacement.Override)
		}
		if len(replacement.Version) > maxVersionLength {
			maxVersionLength = len(replacement.Version)
		}
	}
	nonVersionedReplaceTemplate := fmt.Sprintf("%%-%ds -> %%-%ds", maxOffenderLength, maxOverrideLength)
	versionedReplaceTemplate := fmt.Sprintf("%s @ %%%ds", nonVersionedReplaceTemplate, maxVersionLength)

	output := fmt.Sprintf("'%s' is replaced:\n", original)

	var foundMatch bool
	for _, replacement := range r.originToReplace[original] {
		if topLevelOverride, ok := r.topLevel[replacement.Original]; ok && topLevelOverride == replacement.Override {
			output += matchedMark
			foundMatch = true
		} else {
			output += unmatchedMark
		}
		if replacement.Version != "" {
			output += fmt.Sprintf(versionedReplaceTemplate, replacement.Offender.Path, replacement.Override, replacement.Version)
		} else {
			output += fmt.Sprintf(nonVersionedReplaceTemplate, replacement.Offender.Path, replacement.Override)
		}
		output += "\n"
	}
	return output + "\n", foundMatch
}

var (
	singleReplaceRE = regexp.MustCompile(`replace ([^\n]+)`)
	multiReplaceRE  = regexp.MustCompile(`replace \(([^)]+)\)`)
	replaceRE       = regexp.MustCompile(`([^\s]+) => ([^\s]+)(?: (v[^\s]+))?`)
)

func FindReplacements(log *zap.Logger, graph *depgraph.DepGraph) (*Replacements, error) {
	replacements := &Replacements{
		main:            graph.Main.Name(),
		topLevel:        map[string]string{},
		originToReplace: map[string][]Replacement{},
	}

	replaces, err := parseGoMod(log, graph.Main.Module, replacements.topLevel, graph.Main.Module)
	if err != nil {
		return nil, err
	}
	for _, replace := range replaces {
		replacements.topLevel[replace.Original] = replace.Override
	}

	for _, node := range graph.Dependencies.List() {
		replaces, err = parseGoMod(log, graph.Main.Module, replacements.topLevel, node.Module)
		if err != nil {
			return nil, err
		}

		for _, replace := range replaces {
			replaces, ok := replacements.originToReplace[replace.Original]
			if !ok {
				replacements.replacedModules = append(replacements.replacedModules, replace.Original)
			}
			replacements.originToReplace[replace.Original] = append(replaces, replace)
		}
	}
	sort.Strings(replacements.replacedModules)
	for origin, replaces := range replacements.originToReplace {
		sort.Sort(orderedReplacements(replaces))
		replacements.originToReplace[origin] = replaces
	}
	return replacements, nil
}

func parseGoMod(
	log *zap.Logger,
	topLevelModule *modules.ModuleInfo,
	topLevelReplaces map[string]string,
	module *modules.ModuleInfo,
) ([]Replacement, error) {
	module, goModPath := findGoModFile(log, module)
	if goModPath == "" {
		log.Debug("Skipping dependency as no go.mod file was found.", zap.String("dependency", module.Path))
		return nil, nil
	}

	log.Debug("Parsing go.mod.", zap.String("self", module.Path), zap.String("path", goModPath))
	rawGoMod, err := ioutil.ReadFile(goModPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read your module's go.mod file %q", goModPath)
	}

	replaces := parseGoModForReplacements(log, module, string(rawGoMod))
	if module.Path == topLevelModule.Path {
		log.Debug(
			"Auto-dependency detected at version. Filtering already known top-level dependencies.",
			zap.String("self", topLevelModule.Path),
			zap.String("version", module.Version),
		)
		var filteredReplaces []Replacement
		for _, replace := range replaces {
			if _, ok := topLevelReplaces[replace.Original]; !ok {
				filteredReplaces = append(filteredReplaces, replace)
			}
		}
		replaces = filteredReplaces
	}
	return replaces, nil
}

func findGoModFile(log *zap.Logger, module *modules.ModuleInfo) (*modules.ModuleInfo, string) {
	if module == nil {
		return nil, ""
	} else if module.Replace != nil {
		log.Debug("Following top-level replace.", zap.String("source", module.Path), zap.String("target", module.Replace.Path))
		module = module.Replace
	}

	if module.GoMod != "" {
		return module, module.GoMod
	}
	defaultPath := filepath.Join(module.Path, "go.mod")
	if _, err := os.Stat(defaultPath); err == nil {
		log.Debug("Found go.mod file at default path.", zap.String("path", defaultPath))
		return module, defaultPath
	}
	return module, ""
}

func parseGoModForReplacements(log *zap.Logger, module *modules.ModuleInfo, goModContent string) []Replacement {
	var replacements []Replacement
	for _, singleReplaceMatch := range singleReplaceRE.FindAllStringSubmatch(goModContent, -1) {
		replacements = append(replacements, parseReplacements(log, module, singleReplaceMatch[1])...)
	}
	for _, multiReplaceMatch := range multiReplaceRE.FindAllStringSubmatch(goModContent, -1) {
		replacements = append(replacements, parseReplacements(log, module, multiReplaceMatch[1])...)
	}
	return replacements
}

func parseReplacements(log *zap.Logger, module *modules.ModuleInfo, replaceString string) []Replacement {
	var replacements []Replacement
	for _, replaceMatch := range replaceRE.FindAllStringSubmatch(replaceString, -1) {
		replace := Replacement{
			Offender: module,
			Original: replaceMatch[1],
			Override: replaceMatch[2],
			Version:  replaceMatch[3],
		}
		log.Debug(
			"Found hidden replace.",
			zap.String("source", replace.Original),
			zap.String("target", replace.Override),
			zap.String("location", replace.Offender.Path),
		)
		replacements = append(replacements, replace)
	}
	return replacements
}

type orderedReplacements []Replacement

func (r orderedReplacements) Len() int               { return len(r) }
func (r orderedReplacements) Swap(i int, j int)      { r[i], r[j] = r[j], r[i] }
func (r orderedReplacements) Less(i int, j int) bool { return r[i].Offender.Path < r[j].Offender.Path }
