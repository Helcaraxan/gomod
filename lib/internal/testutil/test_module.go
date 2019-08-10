package testutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const fakeGoDriver = `#!/usr/bin/env bash
set -e -u -o pipefail

if [[ -f error.lock ]]; then
	echo >2& "deliberate fake go driver error"
	exit 1
fi

case "$1" in
	list)
		cat "list-output.txt"
		;;
	mod)
		cat "graph-output.txt"
		;;
	*)
		echo >2& "Unrecognised command '$1' to fake go driver"
		exit 1
		;;
esac
`

type TestDefinition interface {
	GoDriverError() bool
	GoListOutput() string
	GoGraphOutput() string
}

func SetupTestModule(t *testing.T, testDefinitionPath string, testDefinition TestDefinition) (string, func()) {
	testDir, testErr := ioutil.TempDir("", "gomod-testing")
	require.NoError(t, testErr)

	require.NoError(t, ioutil.WriteFile(filepath.Join(testDir, "go"), []byte(fakeGoDriver), 0700))

	currentEnvPath := os.Getenv("PATH")
	os.Setenv("PATH", fmt.Sprintf("%s:%s", testDir, currentEnvPath))

	cleanup := func() {
		require.NoError(t, os.RemoveAll(testDir))
		require.NoError(t, os.Setenv("PATH", currentEnvPath))
	}

	raw, testErr := ioutil.ReadFile(testDefinitionPath)
	require.NoError(t, testErr)
	require.NoError(t, yaml.Unmarshal(raw, testDefinition))

	if testDefinition.GoDriverError() {
		require.NoError(t, ioutil.WriteFile(filepath.Join(testDir, "error.lock"), []byte(""), 0600))
	} else {
		require.NoError(t, ioutil.WriteFile(filepath.Join(testDir, "list-output.txt"), []byte(testDefinition.GoListOutput()), 0600))
		require.NoError(t, ioutil.WriteFile(filepath.Join(testDir, "graph-output.txt"), []byte(testDefinition.GoGraphOutput()), 0600))
	}
	return testDir, cleanup
}
