package reveal

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/Helcaraxan/gomod/lib/depgraph"
)

var (
	offender = &depgraph.Module{Path: "offender"}
	replaceA = Replacement{
		Offender: offender,
		Original: "originalA",
		Override: "overrideA",
		Version:  "v1.0.0",
	}
	replaceB = Replacement{
		Offender: offender,
		Original: "originalB",
		Override: "overrideB",
		Version:  "v1.0.0",
	}
	replaceC = Replacement{
		Offender: offender,
		Original: "originalC",
		Override: "overrideC",
		Version:  "v1.0.0",
	}
	replaceD = Replacement{
		Offender: &depgraph.Module{Path: "offender-bis"},
		Original: "originalB",
		Override: "overrideB-bis",
		Version:  "v2.0.0",
	}
	replaceE = Replacement{
		Offender: &depgraph.Module{Path: "offender-tertio"},
		Original: "originalA",
		Override: "overrideA-bis",
		Version:  "v2.0.0",
	}
)

func Test_ParseReplaces(t *testing.T) {
	logger := logrus.New()

	testcases := map[string]struct {
		input    string
		expected []Replacement
	}{
		"SingleReplace": {
			input:    "replace originalA => overrideA v1.0.0",
			expected: []Replacement{replaceA},
		},
		"MultiReplace": {
			input: `
replace (
	originalA => overrideA v1.0.0
	originalB => overrideB v1.0.0
)
`,
			expected: []Replacement{
				replaceA,
				replaceB,
			},
		},
		"MixedReplace": {
			input: `
replace (
	originalB => overrideB v1.0.0
	originalC => overrideC v1.0.0
)

replace originalA => overrideA v1.0.0
`,
			expected: []Replacement{
				replaceA,
				replaceB,
				replaceC,
			},
		},
		"FullGoMod": {
			input: `module github.com/foo/bar

go = 1.12.5

require (
	github.com/my-dep/A v1.2.0
	github.com/my-dep/B v1.9.2-201905291510-0123456789ab // indirect
	originalA v0.1.0
	originalB v0.4.3
	originalC v0.2.3
)

// Override this because it's upstream is broken.
replace originalA => overrideA v1.0.0

// Moar overrides.
replace (
	// Foo.
	originalB => overrideB v1.0.0
	originalC => overrideC v1.0.0 // Bar
)
`,
			expected: []Replacement{
				replaceA,
				replaceB,
				replaceC,
			},
		},
	}

	for name, test := range testcases {
		t.Run(name, func(t *testing.T) {
			output := parseGoModForReplacements(logger, offender, test.input)
			assert.Equal(t, test.expected, output)
		})
	}
}

func Test_FilterReplacements(t *testing.T) {
	testReplacements := Replacements{
		replaceE,
		replaceA,
		replaceD,
		replaceC,
		replaceB,
	}

	t.Run("OffenderEmpty", func(t *testing.T) {
		filtered := testReplacements.FilterOnOffendingModule(nil)
		assert.Equal(t, testReplacements, filtered, "Should return an identical array.")
	})
	t.Run("Offender", func(t *testing.T) {
		filtered := testReplacements.FilterOnOffendingModule([]string{"offender", "offender-tertio"})
		assert.Equal(t, Replacements{
			replaceA,
			replaceB,
			replaceC,
			replaceE,
		}, filtered, "Should filter out the expected replacements.")
	})

	t.Run("TargetEmpty", func(t *testing.T) {
		filtered := testReplacements.FilterOnReplacedModule(nil)
		assert.Equal(t, testReplacements, filtered, "Should return an identical array.")
	})
	t.Run("Target", func(t *testing.T) {
		filtered := testReplacements.FilterOnReplacedModule([]string{"originalA", "originalC"})
		assert.Equal(t, Replacements{
			replaceA,
			replaceE,
			replaceC,
		}, filtered, "Should filter out the expected replacements.")
	})
}

func Test_PrintReplacements(t *testing.T) {
	testReplacements := Replacements{
		replaceA,
		replaceB,
		replaceE,
	}
	const expectedOutput = `"originalA" is replaced:
 offender        -> overrideA     @ v1.0.0
 offender-tertio -> overrideA-bis @ v2.0.0

"originalB" is replaced:
 offender -> overrideB @ v1.0.0

`

	writer := &strings.Builder{}
	testReplacements.Print(writer, nil, nil)
	assert.Equal(t, expectedOutput, writer.String(), "Should print the expected output.")
}
