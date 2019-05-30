package analysis

import (
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/Helcaraxan/gomod/depgraph"
)

type DepAnalysis struct {
	DirectDependencyCount   int
	IndirectDependencyCount int

	MeanDepAge              time.Duration
	MaxDepAge               time.Duration
	DepAgeMonthDistribution []int

	MeanInboundArity         float64
	MaxInboundArity          int
	InboundArityDistribution []int
}

func Analyse(g *depgraph.DepGraph) *DepAnalysis {
	const month = 30 * 24 * time.Hour

	var (
		directDependencyCount int

		maxDepAge          time.Duration
		totalDepAge        float64
		countDepAge        float64
		distributionDepAge []int

		maxDepArity          int
		totalDepArity        float64
		countDepArity        float64
		distributionDepArity []int
	)

	nodes := g.Nodes()
	for _, node := range nodes {
		if node.Name() == g.Module() {
			directDependencyCount = len(node.Successors())
		}
		if timestamp := node.Timestamp(); timestamp != nil {
			depAge := time.Since(*timestamp)
			totalDepAge += float64(depAge.Nanoseconds())
			countDepAge += 1
			if depAge > maxDepAge {
				maxDepAge = depAge
			}
			ageInMonths := int(time.Since(*timestamp).Nanoseconds() / month.Nanoseconds())
			distributionDepAge = insertIntoAgeDistribution(ageInMonths, distributionDepAge)
		}
		depArity := len(node.Predecessors())
		if depArity > 0 {
			totalDepArity += float64(depArity)
			countDepArity += 1
			if depArity > maxDepArity {
				maxDepArity = depArity
			}
			distributionDepArity = insertIntoAgeDistribution(depArity, distributionDepArity)
		}
	}

	return &DepAnalysis{
		DirectDependencyCount:    directDependencyCount,
		IndirectDependencyCount:  len(nodes) - directDependencyCount - 1,
		MeanDepAge:               time.Duration(int64(totalDepAge / countDepAge)),
		MaxDepAge:                maxDepAge,
		DepAgeMonthDistribution:  distributionDepAge,
		MeanInboundArity:         totalDepArity / countDepArity,
		MaxInboundArity:          maxDepArity,
		InboundArityDistribution: distributionDepArity,
	}
}

func (a *DepAnalysis) Print(f io.Writer) error {
	_, err := fmt.Fprintf(
		f,
		`
Dependency counts:
- Direct dependencies:   %d
- Indirect dependencies: %d

Age statistics:
- Mean age of dependencies: %s
- Maximum dependency age:   %s
- Age distribution per month:

%s

Inbound arity statistics:
- Mean inbound arity of dependencies: %.2f
- Maximum dependency inbound arity:   %d
- Arity distribution:

%s

`,
		a.DirectDependencyCount,
		a.IndirectDependencyCount,
		humanDuration(a.MeanDepAge),
		humanDuration(a.MaxDepAge),
		printedDistribution(a.DepAgeMonthDistribution),
		a.MeanInboundArity,
		a.MaxInboundArity,
		printedDistribution(a.InboundArityDistribution),
	)
	return err
}

func humanDuration(d time.Duration) string {
	totalDays := d.Nanoseconds() / (24 * time.Hour.Nanoseconds())
	months := totalDays / 30
	days := totalDays % 30
	return fmt.Sprintf("%d month(s) %d day(s)", months, days)
}

func insertIntoAgeDistribution(idx int, v []int) []int {
	if idx+1 > len(v) {
		if idx+1 < cap(v) {
			v = v[:idx+1]
		} else {
			newV := make([]int, idx+1, 2*(idx+1))
			copy(newV, v)
			v = newV
		}
	}
	v[idx]++
	return v
}

func distributionCountToPercentage(d []int, groupingFactor int) []float64 {
	var totalCount int

	// Preallocate percentage distribution.
	columns := len(d) / groupingFactor
	if len(d)%groupingFactor > 0 {
		columns++
	}
	p := make([]float64, columns)

	// Group input columns.
	for i := range p {
		for j := 0; j < groupingFactor && i*groupingFactor+j < len(d); j++ {
			totalCount += d[i*groupingFactor+j]
			p[i] += float64(d[i*groupingFactor+j])
		}
	}

	// Normalise results.
	for i := range p {
		p[i] /= float64(totalCount)
	}
	return p
}

func distributionToLines(distribution []float64, displayHeight int) []string {
	var maxColumnValue float64
	for _, columnValue := range distribution {
		if columnValue > maxColumnValue {
			maxColumnValue = columnValue
		}
	}

	step := maxColumnValue / float64(displayHeight)
	lines := make([]string, 2*len(distribution))

	lines[0] = strings.Repeat("|", displayHeight+1)
	for idx, value := range distribution {
		line := "_" + strings.Repeat("#", int(value/step))
		if math.Mod(value, step) > step/2 {
			line += "_"
		}
		lines[idx*2+1] = line
		if idx*2+2 < len(lines) {
			lines[idx*2+2] = "_"
		}
	}
	return lines
}

func rotateDistributionLines(lines []string, displayHeight int) []string {
	rows := make([]string, displayHeight+1)
	for idx := 0; idx < displayHeight+1; idx++ {
		for l := range lines {
			if len(lines[l]) >= displayHeight+1-idx {
				rows[idx] += string(lines[l][displayHeight-idx])
			} else {
				rows[idx] += " "
			}
		}
	}
	return rows
}

func annotateDistributionPrintout(lines []string, distribution []float64, groupingFactor int) []string {
	if len(lines) == 0 {
		return lines
	}

	var maxColumnValue float64
	for _, columnValue := range distribution {
		if columnValue > maxColumnValue {
			maxColumnValue = columnValue
		}
	}

	lineLength := len(lines[0])

	lines[0] = fmt.Sprintf(" %6.2f %% ", maxColumnValue*100) + lines[0]
	for idx := 1; idx < len(lines)-1; idx++ {
		lines[idx] = "          " + lines[idx]
	}
	lines[len(lines)-1] = fmt.Sprintf(" %6.2f %% ", 0.0) + lines[len(lines)-1]

	topValue := groupingFactor * len(distribution)
	bottomLineTemplate := fmt.Sprintf(" 0 %%%dd", lineLength-3)
	bottomLine := "          " + fmt.Sprintf(bottomLineTemplate, topValue)
	return append(lines, bottomLine)
}

func printedDistribution(distribution []int) string {
	const (
		displayHeight = 20
		maxColumns    = 50 // completely arbitrary value that should fit with most terminal widths.
		pStep         = 1 / 2 * float64(displayHeight)
	)

	groupingFactor := len(distribution) / maxColumns
	if len(distribution)%maxColumns > 0 {
		groupingFactor++
	}

	pDistribution := distributionCountToPercentage(distribution, groupingFactor)
	lines := distributionToLines(pDistribution, displayHeight)
	rows := rotateDistributionLines(lines, displayHeight)
	rows = annotateDistributionPrintout(rows, pDistribution, groupingFactor)
	return strings.Join(rows, "\n")
}
