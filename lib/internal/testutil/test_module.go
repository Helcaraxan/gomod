package testutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const fakeGoDriver = `#!/usr/bin/env bash
set -e -u -o pipefail

content_dir="%s"

if [[ -f error.lock ]]; then
	echo >2& "deliberate fake go driver error"
	exit 1
fi

case "$1" in
	mod)
		cat "${content_dir}/graph-output.txt"
		;;
	list)
		resource_type="pkg"
		for arg in ${@:2}; do
			if [[ ${arg} == "-m" ]]; then
				resource_type="mod"
			fi
		done

		files_to_print=()
		for arg in ${@:2}; do
			if [[ ${arg} == "all" ]]; then
				files_to_print=($(ls "${content_dir}/list-${resource_type}"-*.txt))
				break
			elif [[ ${arg:0:1} != "-" ]]; then
				files_to_print+=("${content_dir}/list-${resource_type}-${arg//\//_}.txt")
			fi
		done
		cat "${files_to_print[@]}"
		;;
	*)
		echo >2& "Unrecognised command '$1' to fake go driver"
		exit 1
		;;
esac
`

type TestDefinition interface {
	GoDriverError() bool
	GoGraphOutput() string
	GoListPkgOutput() map[string]string
	GoListModOutput() map[string]string
}

func SetupTestModule(t *testing.T, testDefinitionPath string, testDefinition TestDefinition) (string, func()) {
	testDir, testErr := ioutil.TempDir("", "gomod-testing")
	require.NoError(t, testErr)

	require.NoError(t, ioutil.WriteFile(filepath.Join(testDir, "go"), []byte(fmt.Sprintf(fakeGoDriver, testDir)), 0700))

	currentEnvPath := os.Getenv("PATH")
	os.Setenv("PATH", fmt.Sprintf("%s:%s", testDir, currentEnvPath))

	cleanup := func() {
		if !t.Failed() {
			require.NoError(t, os.RemoveAll(testDir))
		}
		require.NoError(t, os.Setenv("PATH", currentEnvPath))
	}

	raw, testErr := ioutil.ReadFile(testDefinitionPath)
	require.NoError(t, testErr)
	require.NoError(t, yaml.Unmarshal(raw, testDefinition))

	if testDefinition.GoDriverError() {
		require.NoError(t, ioutil.WriteFile(filepath.Join(testDir, "error.lock"), []byte(""), 0600))
		return testDir, cleanup
	}

	require.NoError(t, ioutil.WriteFile(filepath.Join(testDir, "graph-output.txt"), []byte(testDefinition.GoGraphOutput()), 0600))
	for mod, output := range testDefinition.GoListModOutput() {
		filename := fmt.Sprintf("list-mod-%s.txt", strings.ReplaceAll(mod, "/", "_"))
		require.NoError(t, ioutil.WriteFile(filepath.Join(testDir, filename), []byte(output), 0600))
	}
	for pkg, output := range testDefinition.GoListPkgOutput() {
		filename := fmt.Sprintf("list-pkg-%s.txt", strings.ReplaceAll(pkg, "/", "_"))
		require.NoError(t, ioutil.WriteFile(filepath.Join(testDir, filename), []byte(output), 0600))
	}
	return testDir, cleanup
}
