package reveal

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/sirupsen/logrus"

	"github.com/Helcaraxan/gomod/lib/depgraph"
)

type Replacement struct {
	Offender *depgraph.Module
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

func (r *Replacements) Print(logger *logrus.Logger, writer io.Writer, offenders []string, targets []string) error {
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
	moduleLineTemplate := fmt.Sprintf("%%-%ds -> %%-%ds @ %%%ds", maxOffenderLength, maxOverrideLength, maxVersionLength)

	output := fmt.Sprintf("'%s' is replaced:\n", original)

	var foundMatch bool
	for _, replacement := range r.originToReplace[original] {
		if topLevelOverride, ok := r.topLevel[replacement.Original]; ok && topLevelOverride == replacement.Override {
			output += matchedMark
			foundMatch = true
		} else {
			output += unmatchedMark
		}
		output += fmt.Sprintf(moduleLineTemplate, replacement.Offender.Path, replacement.Override, replacement.Version)
		output += "\n"
	}
	return output + "\n", foundMatch
}

var (
	singleReplaceRE = regexp.MustCompile("replace ([^\n]+)")
	multiReplaceRE  = regexp.MustCompile("replace \\(([^)]+)\\)")
	replaceRE       = regexp.MustCompile("([^\\s]+) => ([^\\s]+) ([^\\s]+)")
)

func FindReplacements(logger *logrus.Logger, graph *depgraph.DepGraph) (*Replacements, error) {
	replacements := &Replacements{
		main:            graph.Name(),
		topLevel:        map[string]string{},
		originToReplace: map[string][]Replacement{},
	}

	replaces, err := parseGoMod(logger, graph.Module, replacements.topLevel, graph.Module)
	if err != nil {
		return nil, err
	}
	for _, replace := range replaces {
		replacements.topLevel[replace.Original] = replace.Override
	}

	for _, module := range graph.Modules {
		replaces, err = parseGoMod(logger, graph.Module, replacements.topLevel, module)
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
	logger *logrus.Logger,
	topLevelModule *depgraph.Module,
	topLevelReplaces map[string]string,
	module *depgraph.Module,
) ([]Replacement, error) {
	module, goModPath := findGoModFile(logger, module)
	if goModPath == "" {
		logger.Debugf("Skipping %q as no go.mod file was found.", module.Path)
		return nil, nil
	}

	logger.Debugf("Parsing go.mod for %q at %q.", module.Path, goModPath)
	rawGoMod, err := ioutil.ReadFile(goModPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read your module's go.mod file %q", goModPath)
	}

	replaces := parseGoModForReplacements(logger, module, string(rawGoMod))
	if module.Path == topLevelModule.Path {
		logger.Debugf(
			"Auto-dependency on %q detected at version %q. Filtering already known top-level dependencies.",
			topLevelModule.Path,
			module.Version,
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

func findGoModFile(logger *logrus.Logger, module *depgraph.Module) (*depgraph.Module, string) {
	if module == nil {
		return nil, ""
	} else if module.Replace != nil {
		logger.Debugf("Following top-level replace for %q to %q", module.Path, module.Replace.Path)
		module = module.Replace
	}

	if module.GoMod != "" {
		return module, module.GoMod
	}
	defaultPath := filepath.Join(module.Path, "go.mod")
	if _, err := os.Stat(defaultPath); err == nil {
		logger.Debugf("Found go.mod file at default path %q.", defaultPath)
		return module, defaultPath
	}
	return module, ""
}

func parseGoModForReplacements(logger *logrus.Logger, module *depgraph.Module, goModContent string) []Replacement {
	var replacements []Replacement
	for _, singleReplaceMatch := range singleReplaceRE.FindAllStringSubmatch(goModContent, -1) {
		replacements = append(replacements, parseReplacements(logger, module, singleReplaceMatch[1])...)
	}
	for _, multiReplaceMatch := range multiReplaceRE.FindAllStringSubmatch(goModContent, -1) {
		replacements = append(replacements, parseReplacements(logger, module, multiReplaceMatch[1])...)
	}
	return replacements
}

func parseReplacements(logger *logrus.Logger, module *depgraph.Module, replaceString string) []Replacement {
	var replacements []Replacement
	for _, replaceMatch := range replaceRE.FindAllStringSubmatch(replaceString, -1) {
		replace := Replacement{
			Offender: module,
			Original: replaceMatch[1],
			Override: replaceMatch[2],
			Version:  replaceMatch[3],
		}
		logger.Debugf(
			"Found hidden replace of %q by %q in dependency %q.",
			replace.Original,
			replace.Override,
			replace.Offender.Path,
		)
		replacements = append(replacements, replace)
	}
	return replacements
}

type orderedReplacements []Replacement

func (r orderedReplacements) Len() int               { return len(r) }
func (r orderedReplacements) Swap(i int, j int)      { r[i], r[j] = r[j], r[i] }
func (r orderedReplacements) Less(i int, j int) bool { return r[i].Offender.Path < r[j].Offender.Path }
