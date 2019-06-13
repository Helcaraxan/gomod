package reveal

import (
	"fmt"
	"io"
	"io/ioutil"
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

type Replacements []Replacement

func (r Replacements) Print(writer io.Writer, offenders []string, targets []string) error {
	var (
		lastOriginal         string
		output               string
		offenderReplacements Replacements
	)

	filtered := r.FilterOnOffendingModule(offenders).FilterOnReplacedModule(targets)
	sort.Slice(r, r.lessThanOriginal)

	for idx, replacement := range filtered {
		if lastOriginal != replacement.Original {
			output += offenderReplacements.printModuleReplacements()
			offenderReplacements = Replacements{replacement}
			lastOriginal = replacement.Original
		} else {
			offenderReplacements = append(offenderReplacements, replacement)
		}
		if idx == len(filtered)-1 {
			output += offenderReplacements.printModuleReplacements()
		}
	}

	if _, err := writer.Write([]byte(output)); err != nil {
		return fmt.Errorf("failed to print replacements: %v", err)
	}
	return nil
}

func (r Replacements) printModuleReplacements() string {
	if len(r) == 0 {
		return ""
	}

	var (
		maxOffenderLength int
		maxOverrideLength int
	)
	for _, replacement := range r {
		if len(replacement.Offender.Path) > maxOffenderLength {
			maxOffenderLength = len(replacement.Offender.Path)
		}
		if len(replacement.Override) > maxOverrideLength {
			maxOverrideLength = len(replacement.Override)
		}
	}
	moduleLineTemplate := fmt.Sprintf(" %%-%ds -> %%-%ds @ %%s\n", maxOffenderLength, maxOverrideLength)

	output := fmt.Sprintf("%q is replaced:\n", r[0].Original)
	for _, replacement := range r {
		output += fmt.Sprintf(moduleLineTemplate, replacement.Offender.Path, replacement.Override, replacement.Version)
	}
	return output + "\n"
}

func (r Replacements) FilterOnOffendingModule(offenders []string) Replacements {
	if len(offenders) == 0 {
		return r
	}

	sort.Strings(offenders)
	sort.Slice(r, r.lessThanOffenders)

	filteredReplacements := make(Replacements, 0, len(r))
	var fIdx, rIdx int
	for {
		switch {
		case fIdx == len(offenders) || rIdx == len(r):
			return filteredReplacements
		case offenders[fIdx] < r[rIdx].Offender.Path:
			fIdx++
		case r[rIdx].Offender.Path < offenders[fIdx]:
			rIdx++
		default:
			filteredReplacements = append(filteredReplacements, r[rIdx])
			rIdx++
		}
	}
}

func (r Replacements) FilterOnReplacedModule(targets []string) Replacements {
	if len(targets) == 0 {
		return r
	}

	sort.Strings(targets)
	sort.Slice(r, r.lessThanOriginal)

	filteredReplacements := make(Replacements, 0, len(r))
	var fIdx, rIdx int
	for {
		switch {
		case fIdx == len(targets) || rIdx == len(r):
			return filteredReplacements
		case targets[fIdx] < r[rIdx].Original:
			fIdx++
		case r[rIdx].Original < targets[fIdx]:
			rIdx++
		default:
			filteredReplacements = append(filteredReplacements, r[rIdx])
			rIdx++
		}
	}
}

var (
	singleReplaceRE = regexp.MustCompile("replace ([^\n]+)")
	multiReplaceRE  = regexp.MustCompile("replace \\(([^)]+)\\)")
	replaceRE       = regexp.MustCompile("([^\\s]+) => ([^\\s]+) ([^\\s]+)")
)

func FindReplacements(logger *logrus.Logger, graph *depgraph.DepGraph) Replacements {
	var replacements []Replacement
	for _, module := range graph.Modules {
		if module.Main || module.GoMod == "" {
			continue
		}

		if module.Replace != nil {
			logger.Debugf("Following top-level replace for %q to %q", module.Path, module.Replace.Path)
			module = module.Replace
		}
		logger.Debugf("Parsing go.mod for %q at %q.", module.Path, module.GoMod)
		rawMod, err := ioutil.ReadFile(module.GoMod)
		if err != nil {
			logger.WithError(err).Warnf("Failed to read content from go.mod at %q.", module.GoMod)
		}
		replacements = append(replacements, parseGoModForReplacements(logger, module, string(rawMod))...)
	}
	return replacements
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

func (r Replacements) lessThanOffenders(i int, j int) bool {
	if r[i].Offender.Path != r[j].Offender.Path {
		return r[i].Offender.Path < r[j].Offender.Path
	}
	return r[i].Original < r[j].Original
}

func (r Replacements) lessThanOriginal(i int, j int) bool {
	if r[i].Original != r[j].Original {
		return r[i].Original < r[j].Original
	}
	return r[i].Offender.Path < r[j].Offender.Path
}
