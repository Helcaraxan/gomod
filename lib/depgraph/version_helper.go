package depgraph

import (
	"regexp"

	"github.com/blang/semver"
)

var (
	// Regular expressions according to https://golang.org/cmd/go/#hdr-Pseudo_versions.
	versionRE       = regexp.MustCompile(`^v?(\d+\.\d+\.\d+)(?:-(.*))?$`)
	pseudoVersionRE = regexp.MustCompile(`^(?:(?:.*.)?0.)?(\d{14})-[0-9a-f]{12}$`)
)

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

	if len(lhsParsed[2]) == 0 || len(rhsParsed[2]) == 0 {
		// At least one of the version is a release so this comes down to whether it is the LHS.
		return len(lhsParsed[2]) == 0
	}

	// We are comparing two pre-release versions.
	pseudoLHS := pseudoVersionRE.FindStringSubmatch(lhsParsed[2])
	pseudoRHS := pseudoVersionRE.FindStringSubmatch(rhsParsed[2])
	return pseudoLHS[1] > pseudoRHS[1]
}
