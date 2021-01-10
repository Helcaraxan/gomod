package main

import (
	"flag"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Helcaraxan/gomod/internal/printer"
	"github.com/Helcaraxan/gomod/internal/testutil"
)

var regenerate = flag.Bool("regenerate", false, "Instead of testing the output, use the generated output to refresh the golden images.")

func TestGraphGeneration(t *testing.T) {
	testcases := map[string]struct {
		expectedFileBase string
		dotArgs          *graphArgs
		visualArgs       *graphArgs
	}{
		"Full": {
			expectedFileBase: "full",
			dotArgs: &graphArgs{
				query: "deps(github.com/Helcaraxan/gomod:test)",
				style: &printer.StyleOptions{
					ScaleNodes: true,
					Cluster:    printer.Full,
				},
			},
		},
		"Shared": {
			expectedFileBase: "shared-dependencies",
			dotArgs: &graphArgs{
				query: "shared(deps(github.com/Helcaraxan/gomod))",
				style: &printer.StyleOptions{},
			},
		},
		"TargetDependency": {
			expectedFileBase: "dependency-chains",
			dotArgs: &graphArgs{
				annotate: true,
				query:    "rdeps(github.com/stretchr/testify)",
				style:    &printer.StyleOptions{},
			},
		},
	}

	for name := range testcases {
		testcase := testcases[name]
		t.Run(name, func(t *testing.T) {
			tempDir := t.TempDir()

			cArgs := &commonArgs{log: testutil.TestLogger(t)}

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
		})
	}
}
