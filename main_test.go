package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Helcaraxan/gomod/internal/logger"
	"github.com/Helcaraxan/gomod/internal/printer"
)

var regenerate = flag.Bool("regenerate", false, "Instead of testing the output, use the generated output to refresh the golden images.")

func TestGraphGeneration(t *testing.T) {
	testcases := map[string]struct {
		expectedFileBase string
		dotArgs          *graphArgs
		visualArgs       *graphArgs
	}{
		// Disabled test as the produced DOT graph crashes the .dot tool despite being valid.
		// Issue that is being experienced is: https://gitlab.com/graphviz/graphviz/-/issues/1408
		// Also relates to: https://github.com/Helcaraxan/gomod/issues/54
		// "Full": {
		// 	expectedFileBase: "full",
		// 	dotArgs:          &graphArgs{},
		// 	visualArgs: &graphArgs{
		// 		style: &printer.StyleOptions{
		// 			ScaleNodes: true,
		// 			Cluster:    printer.Full,
		// 		},
		// 	},
		// },
		"Shared": {
			expectedFileBase: "shared-dependencies",
			dotArgs:          &graphArgs{shared: true},
			visualArgs: &graphArgs{
				shared: true,
				style:  &printer.StyleOptions{},
			},
		},
		"TargetDependency": {
			expectedFileBase: "dependency-chains",
			dotArgs: &graphArgs{
				annotate:     true,
				dependencies: []string{"github.com/stretchr/testify", "golang.org/x/sys"},
			},
			visualArgs: &graphArgs{
				annotate:     true,
				dependencies: []string{"github.com/stretchr/testify", "golang.org/x/sys"},
				style:        &printer.StyleOptions{},
			},
		},
	}

	tempDir, tempErr := ioutil.TempDir("", "gomod")
	require.NoError(t, tempErr)
	defer func() {
		if !t.Failed() {
			require.NoError(t, os.RemoveAll(tempDir))
		}
	}()

	cArgs := &commonArgs{log: zap.New(zapcore.NewCore(logger.NewGoModEncoder(), os.Stdout, zapcore.DebugLevel))}

	for name := range testcases {
		testcase := testcases[name]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Test the dot generation.
			dotArgs := *testcase.dotArgs
			dotArgs.commonArgs = cArgs
			dotArgs.outputPath = filepath.Join(tempDir, testcase.expectedFileBase+".dot")

			require.NoError(t, runGraphCmd(&dotArgs))
			actual, err := ioutil.ReadFile(filepath.Join(tempDir, testcase.expectedFileBase+".dot"))
			require.NoError(t, err)
			if *regenerate {
				require.NoError(t, ioutil.WriteFile(filepath.Join("images", testcase.expectedFileBase+".dot"), actual, 0644))
			} else {
				var expected []byte
				expected, err = ioutil.ReadFile(filepath.Join("images", testcase.expectedFileBase+".dot"))
				require.NoError(t, err)
				assert.Equal(t, string(expected), string(actual))
			}

			// Test the visual generation.
			visualArgs := *testcase.visualArgs
			visualArgs.commonArgs = cArgs
			visualArgs.outputPath = filepath.Join(tempDir, testcase.expectedFileBase+".jpg")
			require.NoError(t, runGraphCmd(&visualArgs))

			actual, err = ioutil.ReadFile(filepath.Join(tempDir, testcase.expectedFileBase+".jpg"))
			require.NoError(t, err)
			if *regenerate {
				require.NoError(t, ioutil.WriteFile(filepath.Join("images", testcase.expectedFileBase+".jpg"), actual, 0644))
			}
		})
	}
}
