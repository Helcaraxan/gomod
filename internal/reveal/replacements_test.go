package reveal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Helcaraxan/gomod/internal/depgraph"
	"github.com/Helcaraxan/gomod/internal/logger"
	"github.com/Helcaraxan/gomod/internal/modules"
)

var (
	replaceA = Replacement{
		Offender: &modules.ModuleInfo{Path: "offender"},
		Original: "originalA",
		Override: "overrideA",
		Version:  "v1.0.0",
	}
	replaceB = Replacement{
		Offender: moduleA,
		Original: "originalB",
		Override: "overrideB",
		Version:  "v1.0.0",
	}
	replaceC = Replacement{
		Offender: moduleA,
		Original: "originalC",
		Override: "./overrideC",
	}
	replaceD = Replacement{
		Offender: moduleA,
		Original: "originalD",
		Override: "./overrideD",
	}
	replaceE = Replacement{
		Offender: &modules.ModuleInfo{Path: "offender-bis"},
		Original: "originalA",
		Override: "overrideA-bis",
		Version:  "v2.0.0",
	}
	replaceF = Replacement{
		Offender: &modules.ModuleInfo{Path: "offender-tertio"},
		Original: "originalB",
		Override: "overrideB-bis",
		Version:  "v2.0.0",
	}

	testReplacements = &Replacements{
		main: "test-module",
		topLevel: map[string]string{
			"originalA": "overrideA",
			"originalB": "overrideB-bis",
		},
		replacedModules: []string{
			"originalA",
			"originalB",
			"originalC",
		},
		originToReplace: map[string][]Replacement{
			"originalA": {replaceA, replaceE},
			"originalB": {replaceB, replaceF},
			"originalC": {replaceC},
		},
	}

	moduleA = &modules.ModuleInfo{
		Main:    false,
		Path:    "moduleA",
		Version: "v1.0.0",
		GoMod:   filepath.Join("testdata", "moduleA", "go.mod"),
	}
	moduleB = &modules.ModuleInfo{
		Main:    false,
		Path:    filepath.Join("testdata", "moduleB"),
		Version: "v1.1.0",
	}
	moduleC = &modules.ModuleInfo{
		Main:    false,
		Path:    "moduleA",
		Version: "v0.1.0",
		Replace: moduleA,
		GoMod:   "nowhere",
	}
	moduleD = &modules.ModuleInfo{
		Main:    false,
		Path:    "moduleD",
		Version: "v0.0.1",
		GoMod:   "",
	}
)

func createTestGraph() *depgraph.Graph {
	log := zap.NewNop()

	testGraph := depgraph.NewGraph(log, "", &modules.ModuleInfo{
		Main:  true,
		Path:  "test/module",
		GoMod: filepath.Join("testdata", "mainModule", "go.mod"),
	})
	for _, module := range []*modules.ModuleInfo{moduleA, moduleB, moduleC, moduleD} {
		_ = testGraph.Graph.AddNode(depgraph.NewModule(module))
		if module.Replace != nil {
			testGraph.Replaces[module.Path] = module.Replace.Path
		}
	}
	return testGraph
}

func Test_ParseReplaces(t *testing.T) {
	t.Parallel()

	log := zap.New(zapcore.NewCore(logger.NewGoModEncoder(), os.Stdout, zap.DebugLevel))

	testcases := map[string]struct {
		input    string
		offender *modules.ModuleInfo
		expected []Replacement
	}{
		"SingleReplace": {
			input:    "replace originalA => overrideA v1.0.0",
			offender: &modules.ModuleInfo{Path: "offender"},
			expected: []Replacement{replaceA},
		},
		"MultiReplace": {
			input: `
replace (
	originalB => overrideB v1.0.0
	originalC => ./overrideC
)
`,
			offender: moduleA,
			expected: []Replacement{
				replaceB,
				replaceC,
			},
		},
		"MixedReplace": {
			input: `
replace (
	originalB => overrideB v1.0.0
	originalC => ./overrideC
)

replace originalD => ./overrideD
`,
			offender: moduleA,
			expected: []Replacement{
				replaceD,
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
	originalB v0.4.3
	originalC v0.2.3
	originalD v0.1.0
)

// Override this because it's upstream is broken.
replace originalC => ./overrideC // Bar

// Moar overrides.
replace (
	// Foo.
	originalB => overrideB v1.0.0
	originalD => ./overrideD
)
`,
			offender: moduleA,
			expected: []Replacement{
				replaceC,
				replaceB,
				replaceD,
			},
		},
	}

	for name := range testcases {
		testcase := testcases[name]
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			output := parseGoModForReplacements(log, testcase.offender, testcase.input)
			assert.Equal(t, testcase.expected, output)
		})
	}
}

func Test_FindReplacements(t *testing.T) {
	t.Parallel()

	log := zap.NewNop()

	expectedReplacements := &Replacements{
		main:     "test/module",
		topLevel: map[string]string{"module/foo": "module/foo-bis"},
		replacedModules: []string{
			"originalB",
			"originalC",
			"originalD",
		},
		originToReplace: map[string][]Replacement{
			"originalB": {replaceB},
			"originalC": {replaceC},
			"originalD": {replaceD},
		},
	}

	replacements, err := FindReplacements(log, createTestGraph())
	assert.NoError(t, err, "Should not error while searching for replacements.")
	assert.Equal(t, expectedReplacements, replacements, "Should find the expected replacement information.")
}

func Test_FilterReplacements(t *testing.T) {
	t.Parallel()

	t.Run("OffenderEmpty", func(t *testing.T) {
		filtered := testReplacements.FilterOnOffendingModule(nil)
		assert.Equal(t, testReplacements, filtered, "Should return an identical array.")
	})
	t.Run("Offender", func(t *testing.T) {
		filtered := testReplacements.FilterOnOffendingModule([]string{"offender", "pre-offender", "offender-post"})
		assert.Equal(t, &Replacements{
			main: "test-module",
			topLevel: map[string]string{
				"originalA": "overrideA",
				"originalB": "overrideB-bis",
			},
			replacedModules: []string{
				"originalA",
			},
			originToReplace: map[string][]Replacement{
				"originalA": {replaceA},
			},
		}, filtered, "Should filter out the expected replacements.")
	})

	t.Run("OriginsEmpty", func(t *testing.T) {
		filtered := testReplacements.FilterOnReplacedModule(nil)
		assert.Equal(t, testReplacements, filtered, "Should return an identical array.")
	})
	t.Run("Origins", func(t *testing.T) {
		filtered := testReplacements.FilterOnReplacedModule([]string{"originalA", "originalC", "not-original"})
		assert.Equal(t, &Replacements{
			main: "test-module",
			topLevel: map[string]string{
				"originalA": "overrideA",
				"originalB": "overrideB-bis",
			},
			replacedModules: []string{
				"originalA",
				"originalC",
			},
			originToReplace: map[string][]Replacement{
				"originalA": {replaceA, replaceE},
				"originalC": {replaceC},
			},
		}, filtered, "Should filter out the expected replacements.")
	})
}

func Test_PrintReplacements(t *testing.T) {
	t.Parallel()
	const expectedOutput = `'originalA' is replaced:
 ✓ offender     -> overrideA     @ v1.0.0
   offender-bis -> overrideA-bis @ v2.0.0

'originalB' is replaced:
   moduleA         -> overrideB     @ v1.0.0
 ✓ offender-tertio -> overrideB-bis @ v2.0.0

'originalC' is replaced:
   moduleA -> ./overrideC

[✓] Match with a top-level replace in 'test-module'
`

	log := zap.NewNop()

	writer := &strings.Builder{}
	testReplacements.Print(log, writer, nil, nil)
	assert.Equal(t, expectedOutput, writer.String(), "Should print the expected output.")
}

func Test_FindGoModFile(t *testing.T) {
	t.Parallel()

	log := zap.NewNop()

	testcases := map[string]struct {
		module         *modules.ModuleInfo
		expectedModule *modules.ModuleInfo
		expectedPath   string
	}{
		"NoModule": {
			module:         nil,
			expectedModule: nil,
			expectedPath:   "",
		},
		"Standard": {
			module:         moduleA,
			expectedModule: moduleA,
			expectedPath:   filepath.Join("testdata", "moduleA", "go.mod"),
		},
		"NoGoMod": {
			module:         moduleB,
			expectedModule: moduleB,
			expectedPath:   filepath.Join("testdata", "moduleB", "go.mod"),
		},
		"Replaced": {
			module:         moduleC,
			expectedModule: moduleA,
			expectedPath:   filepath.Join("testdata", "moduleA", "go.mod"),
		},
		"Invalid": {
			module:         moduleD,
			expectedModule: moduleD,
			expectedPath:   "",
		},
	}

	for name := range testcases {
		testcase := testcases[name]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			module, goModPath := findGoModFile(log, testcase.module)
			assert.Equal(t, testcase.expectedModule, module, "Should have determined the used module correctly.")
			assert.Equal(t, testcase.expectedPath, goModPath, "Should have determined the correct go.mod path.")
		})
	}
}
