package modules

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

type testcase struct {
	ExpectedError   bool               `yaml:"error"`
	ExpectedMain    *Module            `yaml:"main"`
	ExpectedModules map[string]*Module `yaml:"modules"`
	Output          string             `yaml:"go_list_output"`
}

func TestModuleInformationRetrieval(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Prepend the testdata directory to the path so we use the fake "go" script.
	err = os.Setenv("PATH", filepath.Join(cwd, "testdata")+":"+os.Getenv("PATH"))
	require.NoError(t, err)

	files, err := ioutil.ReadDir(filepath.Join(cwd, "testdata"))
	require.NoError(t, err)

	for idx := range files {
		file := files[idx]
		if file.IsDir() || filepath.Ext(file.Name()) != ".yaml" {
			continue
		}

		testname := strings.TrimSuffix(file.Name(), ".yaml")
		t.Run(testname, func(t *testing.T) {
			testDir, testErr := ioutil.TempDir("", "gomod-module-loading")
			require.NoError(t, testErr)
			defer func() {
				assert.NoError(t, os.RemoveAll(testDir))
			}()

			raw, testErr := ioutil.ReadFile(filepath.Join(cwd, "testdata", file.Name()))
			require.NoError(t, testErr)

			test := &testcase{}
			require.NoError(t, yaml.Unmarshal(raw, test))

			if test.Output != "" {
				testErr = ioutil.WriteFile(filepath.Join(testDir, "test-output.txt"), []byte(test.Output), 0400)
				require.NoError(t, testErr)
			}

			main, modules, testErr := RetrieveModuleInformation(logrus.New(), testDir)
			if test.ExpectedError {
				assert.Error(t, testErr)
			} else {
				assert.Equal(t, test.ExpectedMain, main)
				assert.Equal(t, test.ExpectedModules, modules)
			}
		})
	}
}
