package analysis

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
		"_",
		"_",
		"_##",
		"_",
		"_#_",
		"_",
		"__",
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
