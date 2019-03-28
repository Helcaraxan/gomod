package depgraph

import (
	"regexp"

	"github.com/blang/semver"
)

var versionRE = regexp.MustCompile(`^v?([^-]+)(-.*)?$`)

func moduleMoreRecentThan(lhs string, rhs string) bool {
	lhsParsed := versionRE.FindStringSubmatch(lhs)
	if len(lhsParsed) == 0 {
		return false
	}
	rhsParsed := versionRE.FindStringSubmatch(rhs)
	if len(rhsParsed) == 0 {
		return true
	}

	lhsSemVer := semver.MustParse(lhsParsed[1])
	rhsSemVer := semver.MustParse(rhsParsed[1])
	if lhsSemVer.GT(rhsSemVer) {
		return true
	} else if lhsSemVer.LT(rhsSemVer) {
		return false
	}
	if len(lhsParsed) == 2 && len(rhsParsed) == 3 {
		return true
	} else if len(lhsParsed) == 3 && len(rhsParsed) == 3 {
		return lhsParsed[2] > rhsParsed[2]
	}
	return false
}
