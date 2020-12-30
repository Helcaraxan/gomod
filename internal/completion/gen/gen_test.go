package main

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Helcaraxan/gomod/internal/logger"
)

func Test_GetFilename(t *testing.T) {
	oldLog := log
	defer func() { log = oldLog }()
	log = zap.NewNop()

	testcases := map[string]struct {
		outputDir string
		inputPath string
		expected  string
	}{
		"output-here": {
			outputDir: "",
			inputPath: "test_func.sh",
			expected:  "test_func.go",
		},
		"output-here-dot": {
			outputDir: ".",
			inputPath: "test_func.sh",
			expected:  "test_func.go",
		},
		"output-elsewhere": {
			outputDir: "foo/bar",
			inputPath: "test_func.sh",
			expected:  "foo/bar/test_func.go",
		},
		"upper-case": {
			outputDir: "",
			inputPath: "Test_FUnc.sh",
			expected:  "test_func.go",
		},
	}

	for name := range testcases {
		testcase := testcases[name]
		t.Run(name, func(t *testing.T) {
			actual := getFilename(testcase.outputDir, testcase.inputPath)
			assert.Equal(t, testcase.expected, actual, "Should have returned the expected output path.")
		})
	}
}

func Test_GetVariable(t *testing.T) {
	oldLog := log
	defer func() { log = oldLog }()
	log = zap.NewNop()

	testcases := map[string]struct {
		inputPath string
		expected  string
	}{
		"no-casing": {
			inputPath: "test_func.sh",
			expected:  "TestFunc",
		},
		"strange-casing": {
			inputPath: "Test_FUnc.sh",
			expected:  "TestFunc",
		},
		"unicode": {
			inputPath: "T✓esTFunc.sh",
			expected:  "T✓estfunc",
		},
	}

	for name := range testcases {
		testcase := testcases[name]
		t.Run(name, func(t *testing.T) {
			actual := getVariableName(testcase.inputPath)
			assert.Equal(t, testcase.expected, actual, "Should have returned the expected output path.")
		})
	}
}

func TestGen_SkipsNonShell(t *testing.T) {
	oldLog := log
	defer func() { log = oldLog }()
	logWriter := &syncStringBuffer{}
	log = zap.New(zapcore.NewCore(logger.NewGoModEncoder(), logWriter, zap.DebugLevel))

	err := processFile("foo", "bar", "my-go-file.go")
	assert.NoError(t, err, "Should not error when processing file with no .sh extension.")
	assert.Contains(t, logWriter.String(), "'.sh' extension")
}

func TestGen(t *testing.T) {
	tempDir := t.TempDir()

	oldLog := log
	defer func() { log = oldLog }()
	log = zap.NewNop()

	inputPath := filepath.Join("testdata", "test_FUnc.sh")
	err := processFile(tempDir, "test_gen", inputPath)
	require.NoErrorf(t, err, "Must be able to generate Go file based on %q.", inputPath)

	expectedPath := filepath.Join("testdata", "test_func.go")
	expected, err := ioutil.ReadFile(expectedPath)
	require.NoErrorf(t, err, "Must be able to read expected file at %q.", expectedPath)

	actualPath := filepath.Join(tempDir, "test_func.go")
	require.FileExists(t, actualPath)

	actual, err := ioutil.ReadFile(actualPath)
	require.NoErrorf(t, err, "Must be able to read the generated file at %q.", actualPath)
	assert.Equal(t, string(expected), string(actual), "Should have generated the expected file at %q.", actualPath)
}

type syncStringBuffer struct {
	strings.Builder
}

func (b *syncStringBuffer) Sync() error { return nil }
