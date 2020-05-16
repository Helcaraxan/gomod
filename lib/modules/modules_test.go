package modules

import (
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Helcaraxan/gomod/internal/logger"
	"github.com/Helcaraxan/gomod/lib/internal/testutil"
)

type testcase struct {
	DriverError   bool              `yaml:"driver_error"`
	ListModOutput map[string]string `yaml:"go_list_mod_output"`
	ListPkgOutput map[string]string `yaml:"go_list_pkg_output"`

	ExpectedError   bool               `yaml:"error"`
	ExpectedMain    *Module            `yaml:"main"`
	ExpectedModules map[string]*Module `yaml:"modules"`
}

func (c *testcase) GoDriverError() bool                { return c.DriverError }
func (c *testcase) GoListModOutput() map[string]string { return c.ListModOutput }
func (c *testcase) GoListPkgOutput() map[string]string { return c.ListPkgOutput }
func (c *testcase) GoGraphOutput() string              { return "" }

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

			log := zap.New(zapcore.NewCore(logger.NewGoModEncoder(), os.Stdout, zap.DebugLevel))

			main, modules, testErr := GetDependencies(log, testDir)
			if testDefinition.ExpectedError {
				assert.Error(t, testErr)
			} else {
				require.NoError(t, testErr)
				assert.Equal(t, testDefinition.ExpectedMain, main)
				assert.Equal(t, testDefinition.ExpectedModules, modules)
			}

			main, modules, testErr = GetDependenciesWithUpdates(log, testDir)
			if testDefinition.ExpectedError {
				assert.Error(t, testErr)
			} else {
				require.NoError(t, testErr)
				assert.Equal(t, testDefinition.ExpectedMain, main)
				assert.Equal(t, testDefinition.ExpectedModules, modules)
			}

			if !testDefinition.ExpectedError {
				main, testErr = GetModule(log, testDir, testDefinition.ExpectedMain.Path)
				require.NoError(t, testErr)
				assert.Equal(t, testDefinition.ExpectedMain, main)

				main, testErr = GetModuleWithUpdate(log, testDir, testDefinition.ExpectedMain.Path)
				require.NoError(t, testErr)
				assert.Equal(t, testDefinition.ExpectedMain, main)
			}
		})
	}
}

func TestLackOfConnectivity(t *testing.T) {
	log := zap.NewNop()

	savedClient := httpClient
	defer func() { httpClient = savedClient }()

	for _, fakeRTT := range []http.RoundTripper{&disconnectedRTT{}, &erroneousRTT{}} {
		httpClient = &http.Client{Transport: fakeRTT}

		_, _, err := GetDependenciesWithUpdates(log, "")
		assert.Error(t, err)

		_, err = GetModuleWithUpdate(log, ".", "github.com/Helcaraxan/gomod")
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
