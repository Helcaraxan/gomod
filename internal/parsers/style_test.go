package parsers

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Helcaraxan/gomod/lib/printer"
)

func TestVisualConfig(t *testing.T) {
	testcases := map[string]struct {
		optionValue    string
		expectedConfig *printer.StyleOptions
		expectedError  bool
	}{
		"Empty": {
			optionValue:    "",
			expectedConfig: &printer.StyleOptions{},
		},
		"ScaleNodesFalse": {
			optionValue:    "scale_nodes=false",
			expectedConfig: &printer.StyleOptions{ScaleNodes: false},
		},
		"ScaleNodesNo": {
			optionValue:    "scale_nodes=no",
			expectedConfig: &printer.StyleOptions{ScaleNodes: false},
		},
		"ScaleNodesOff": {
			optionValue:    "scale_nodes=off",
			expectedConfig: &printer.StyleOptions{ScaleNodes: false},
		},
		"ScaleNodesEmpty": {
			optionValue:    "scale_nodes",
			expectedConfig: &printer.StyleOptions{ScaleNodes: true},
		},
		"ScaleNodesTrue": {
			optionValue:    "scale_nodes=true",
			expectedConfig: &printer.StyleOptions{ScaleNodes: true},
		},
		"ScaleNodesYes": {
			optionValue:    "scale_nodes=yes",
			expectedConfig: &printer.StyleOptions{ScaleNodes: true},
		},
		"ScaleNodesOn": {
			optionValue:    "scale_nodes=on",
			expectedConfig: &printer.StyleOptions{ScaleNodes: true},
		},
		"ClusterFalse": {
			optionValue:    "cluster=false",
			expectedConfig: &printer.StyleOptions{Cluster: printer.Off},
		},
		"ClusterNo": {
			optionValue:    "cluster=no",
			expectedConfig: &printer.StyleOptions{Cluster: printer.Off},
		},
		"ClusterOff": {
			optionValue:    "cluster=off",
			expectedConfig: &printer.StyleOptions{Cluster: printer.Off},
		},
		"ClusterEmpty": {
			optionValue:    "cluster",
			expectedConfig: &printer.StyleOptions{Cluster: printer.Shared},
		},
		"ClusterShared": {
			optionValue:    "cluster=shared",
			expectedConfig: &printer.StyleOptions{Cluster: printer.Shared},
		},
		"ClusterTrue": {
			optionValue:    "cluster=true",
			expectedConfig: &printer.StyleOptions{Cluster: printer.Shared},
		},
		"ClusterOn": {
			optionValue:    "cluster=on",
			expectedConfig: &printer.StyleOptions{Cluster: printer.Shared},
		},
		"ClusterYes": {
			optionValue:    "cluster=yes",
			expectedConfig: &printer.StyleOptions{Cluster: printer.Shared},
		},
		"ClusterFull": {
			optionValue:    "cluster=full",
			expectedConfig: &printer.StyleOptions{Cluster: printer.Full},
		},
		"AllConfigsSimple": {
			optionValue: "cluster=true,scale_nodes=true",
			expectedConfig: &printer.StyleOptions{
				Cluster:    printer.Shared,
				ScaleNodes: true,
			},
		},
		"AllConfigsComplex": {
			optionValue: "cluster=True , scale_nodes = tRuE",
			expectedConfig: &printer.StyleOptions{
				Cluster:    printer.Shared,
				ScaleNodes: true,
			},
		},
		"UnknownConfig": {
			optionValue:   "foo",
			expectedError: true,
		},
		"UnknownScaleNodeValue": {
			optionValue:   "scale_nodes=foo",
			expectedError: true,
		},
		"UnknownClusterValue": {
			optionValue:   "cluster=foo",
			expectedError: true,
		},
	}

	for name, testcase := range testcases {
		t.Run(name, func(t *testing.T) {
			logger := logrus.New()
			config, err := ParseVisualConfig(logger, testcase.optionValue)
			if testcase.expectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, testcase.expectedConfig, config)
			}
		})
	}
}
