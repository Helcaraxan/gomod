package depgraph

import (
	"fmt"
	"os"
	"regexp"

	"github.com/blang/semver"
)

var versionRE = regexp.MustCompile(`^v?([^-]+)(-.*)?$`)

type ModuleVersion string

func (v *ModuleVersion) MoreRecentThan(t ModuleVersion) bool {
	fmt.Fprintf(os.Stderr, "Comparing %q and %q\n", *v, t)
	vParsed := versionRE.FindStringSubmatch(string(*v))
	if len(vParsed) == 0 {
		fmt.Fprintf(os.Stderr, "Could not parse %q.\n", *v)
		return false
	}
	tParsed := versionRE.FindStringSubmatch(string(t))
	if len(tParsed) == 0 {
		fmt.Fprintf(os.Stderr, "Could not parse %q.\n", t)
		return false
	}

	vSemVer := semver.MustParse(vParsed[1])
	tSemVer := semver.MustParse(tParsed[1])
	if vSemVer.GT(tSemVer) {
		fmt.Fprintf(os.Stderr, "%q is newer than %q.\n", *v, t)
		return true
	} else if vSemVer.LT(tSemVer) {
		fmt.Fprintf(os.Stderr, "%q is older than %q.\n", *v, t)
		return false
	}
	if len(vParsed) == 2 && len(tParsed) == 3 {
		fmt.Fprintf(os.Stderr, "%q is newer than %q.\n", *v, t)
		return true
	} else if len(vParsed) == 3 && len(tParsed) == 3 {
		if vParsed[2] > tParsed[2] {
			fmt.Fprintf(os.Stderr, "%q is newer than %q.\n", *v, t)
		} else {
			fmt.Fprintf(os.Stderr, "%q is older than %q.\n", *v, t)
		}
		return vParsed[2] > tParsed[2]
	}
	fmt.Fprintf(os.Stderr, "%q is older than %q.\n", *v, t)
	return false
}
