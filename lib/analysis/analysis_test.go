package analysis

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Helcaraxan/gomod/internal/logger"
	"github.com/Helcaraxan/gomod/lib/depgraph"
	"github.com/Helcaraxan/gomod/lib/internal/testutil"
)

func Test_DistributionCountToPercentage(t *testing.T) {
	inputDistribution := []int{1, 10, 5, 4}
	expectedOutputNoGrouping := []float64{0.05, 0.5, 0.25, 0.20}
	expectedOutputThreeGroup := []float64{0.8, 0.2}
	assert.Equal(t, expectedOutputNoGrouping, distributionCountToPercentage(inputDistribution, 1))
	assert.Equal(t, expectedOutputThreeGroup, distributionCountToPercentage(inputDistribution, 3))
}

func Test_DistributionToLines(t *testing.T) {
	inputDistribution := []float64{0.05, 0.48, 0.35, 0.12}
	expectedOutput := []string{
		"||||||",
		"__",
		"_",
		"_#####",
		"_",
		"_###_",
		"_",
		"_#",
	}
	assert.Equal(t, expectedOutput, distributionToLines(inputDistribution, 5))
}

func Test_RotateDistributionLines(t *testing.T) {
	input := []string{
		"||||||",
		"_##_",
		"_",
		"_#####",
		"__",
	}
	expected := []string{
		"|  # ",
		"|  # ",
		"|_ # ",
		"|# # ",
		"|# #_",
		"|____",
	}
	assert.Equal(t, expected, rotateDistributionLines(input, 5), "Should have gotten the expected output")
}

type testcase struct {
	CurrentTime   *time.Time        `yaml:"now"`
	ListModOutput map[string]string `yaml:"go_list_mod_output"`
	ListPkgOutput map[string]string `yaml:"go_list_pkg_output"`
	GraphOutput   string            `yaml:"go_graph_output"`

	ExpectedDepAnalysis *DepAnalysis `yaml:"dep_analysis"`
	ExpectedPrintOutput string       `yaml:"print_output"`
}

func (c *testcase) GoDriverError() bool                { return false }
func (c *testcase) GoListModOutput() map[string]string { return c.ListModOutput }
func (c *testcase) GoListPkgOutput() map[string]string { return c.ListPkgOutput }
func (c *testcase) GoGraphOutput() string              { return c.GraphOutput }

func TestAnalysis(t *testing.T) {
	cwd, setupErr := os.Getwd()
	require.NoError(t, setupErr)

	// Prepend the testdata directory to the path so we use the fake "go" script.
	setupErr = os.Setenv("PATH", filepath.Join(cwd, "..", "internal", "testutil")+":"+os.Getenv("PATH"))
	require.NoError(t, setupErr)

	files, setupErr := ioutil.ReadDir(filepath.Join(cwd, "testdata"))
	require.NoError(t, setupErr)

	for idx := range files {
		file := files[idx]
		if file.IsDir() || filepath.Ext(file.Name()) != ".yaml" {
			continue
		}

		testname := strings.TrimSuffix(file.Name(), ".yaml")
		t.Run(testname, func(t *testing.T) {
			testDefinition := &testcase{}
			testDir, cleanup := testutil.SetupTestModule(t, filepath.Join(cwd, "testdata", file.Name()), testDefinition)
			defer cleanup()

			if testDefinition.CurrentTime != nil {
				testCurrentTimeInjection = testDefinition.CurrentTime
				defer func() { testCurrentTimeInjection = nil }()
			}

			log := zap.New(zapcore.NewCore(logger.NewGoModEncoder(), os.Stdout, zap.DebugLevel))
			graph, err := depgraph.GetGraph(log, testDir)
			require.NoError(t, err)

			analysis, err := Analyse(log, graph)
			require.NoError(t, err)
			assert.Equal(t, testDefinition.ExpectedDepAnalysis, analysis)

			output := &strings.Builder{}
			err = analysis.Print(output)
			require.NoError(t, err)
			assert.Equal(t, testDefinition.ExpectedPrintOutput, output.String())
		})
	}
}
