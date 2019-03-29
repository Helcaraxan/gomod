package depgraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionComparison(t *testing.T) {
	tests := []struct {
		newer string
		older string
	}{
		// Valid versus invalid version.
		{newer: "v1.0.0", older: "foobar"},
		// Newer release versus older release.
		{newer: "v2.0.0", older: "v1.0.0"},
		// Missing precedening 'v' character.
		{newer: "v2.0.0", older: "1.0.0"},
		// Differing minor versions.
		{newer: "v1.1.0", older: "v1.0.0"},
		// Differing patch versions.
		{newer: "v1.0.1", older: "v1.0.0"},
		// Release versus pre-release.
		{newer: "v1.0.0", older: "v1.0.0-pre0"},
		// Release versus preceding non-versioned commit.
		{newer: "v1.0.0", older: "v1.0.0-0.20190101120000-abcdef012345"},
		// Non-versioned commit versus older non-versioned commit (no preceding releases).
		{newer: "v0.0.0-20190101120100-abcdef012345", older: "v0.0.0-20190101120000-abcdef012345"},
		// Non-versioned commit versus older non-versioned commit (with preceding releases).
		{newer: "v1.0.0-0.20190101120100-abcdef012345", older: "v1.0.0-0.20190101120000-abcdef012345"},
		// Non-versioned commit versus non-versioned commit on older releases.
		{newer: "v1.0.1-0.20190101120000-abcdef012345", older: "v1.0.0-0.20190101120000-abcdef012345"},
		// Pre-release versus older non-versioned commit on same releases.
		{newer: "v1.0.0-rc1.0.20190101120100-abcdef012345", older: "v1.0.0-0.20190101120000-abcdef012345"},
		// Non-versioned commit versus older pre-release on same release.
		{newer: "v1.0.0-0.20190101120100-abcdef012345", older: "v1.0.0-pre.0.20190101120000-abcdef012345"},
	}

	for _, test := range tests {
		result := moduleMoreRecentThan(test.newer, test.older)
		assert.Truef(t, result, "Evaluating %q > %q returned an unexpected result.", test.newer, test.older)
		result = moduleMoreRecentThan(test.older, test.newer)
		assert.Falsef(t, result, "Evaluating %q > %q returned an unexpected result.", test.older, test.newer)
	}
}
