package modules

import (
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Helcaraxan/gomod/lib/internal/testutil"
)

type testcase struct {
	DriverError bool   `yaml:"driver_error"`
	ListOutput  string `yaml:"go_list_output"`

	ExpectedError   bool               `yaml:"error"`
	ExpectedMain    *Module            `yaml:"main"`
	ExpectedModules map[string]*Module `yaml:"modules"`
}

func (c *testcase) GoDriverError() bool   { return c.DriverError }
func (c *testcase) GoListOutput() string  { return c.ListOutput }
func (c *testcase) GoGraphOutput() string { return "" }

func TestModuleInformationRetrieval(t *testing.T) {
	savedClient := httpClient
	defer func() { httpClient = savedClient }()
	httpClient = &http.Client{Transport: &successfullRTT{}}

	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Prepend the testdata directory to the path so we use the fake "go" script.
	err = os.Setenv("PATH", filepath.Join(cwd, "..", "internal", "testutil")+":"+os.Getenv("PATH"))
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
			testDefinition := &testcase{}
			testDir, cleanup := testutil.SetupTestModule(t, filepath.Join(cwd, "testdata", file.Name()), testDefinition)
			defer cleanup()

			main, modules, testErr := GetDependencies(logrus.New(), testDir)
			if testDefinition.ExpectedError {
				assert.Error(t, testErr)
			} else {
				require.NoError(t, testErr)
				assert.Equal(t, testDefinition.ExpectedMain, main)
				assert.Equal(t, testDefinition.ExpectedModules, modules)
			}

			main, modules, testErr = GetDependenciesWithUpdates(logrus.New(), testDir)
			if testDefinition.ExpectedError {
				assert.Error(t, testErr)
			} else {
				require.NoError(t, testErr)
				assert.Equal(t, testDefinition.ExpectedMain, main)
				assert.Equal(t, testDefinition.ExpectedModules, modules)
			}

			if !testDefinition.ExpectedError {
				main, testErr = GetModule(logrus.New(), testDir, testDefinition.ExpectedMain.Path)
				require.NoError(t, testErr)
				assert.Equal(t, testDefinition.ExpectedMain, main)

				main, testErr = GetModuleWithUpdate(logrus.New(), testDir, testDefinition.ExpectedMain.Path)
				require.NoError(t, testErr)
				assert.Equal(t, testDefinition.ExpectedMain, main)
			}
		})
	}
}

func TestLackOfConnectivity(t *testing.T) {
	savedClient := httpClient
	defer func() { httpClient = savedClient }()

	for _, fakeRTT := range []http.RoundTripper{&disconnectedRTT{}, &erroneousRTT{}} {
		httpClient = &http.Client{Transport: fakeRTT}

		_, _, err := GetDependenciesWithUpdates(logrus.New(), "")
		assert.Error(t, err)

		_, err = GetModuleWithUpdate(logrus.New(), ".", "github.com/Helcaraxan/gomod")
		assert.Error(t, err)
	}
}

type successfullRTT struct{}

func (f *successfullRTT) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK}, nil
}

type disconnectedRTT struct{}

func (f *disconnectedRTT) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusGatewayTimeout}, nil
}

type erroneousRTT struct{}

func (t *erroneousRTT) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, errors.New("broken transport")
}
