package depgraph

import (
	"regexp"

	"github.com/blang/semver"
)

var versionRE = regexp.MustCompile(`^v?([^-]+)(-.*)?$`)

type ModuleVersion string

func (v *ModuleVersion) MoreRecentThan(t ModuleVersion) bool {
	vParsed := versionRE.FindStringSubmatch(string(*v))
	if len(vParsed) == 0 {
		return false
	}
	tParsed := versionRE.FindStringSubmatch(string(t))
	if len(tParsed) == 0 {
		return false
	}

	vSemVer := semver.MustParse(vParsed[1])
	tSemVer := semver.MustParse(tParsed[1])
	if vSemVer.GT(tSemVer) {
		return true
	} else if vSemVer.LT(tSemVer) {
		return false
	}
	if len(vParsed) == 2 && len(tParsed) == 3 {
		return true
	} else if len(vParsed) == 3 && len(tParsed) == 3 {
		return vParsed[2] > tParsed[2]
	}
	return false
}
