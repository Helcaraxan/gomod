package analysis

import (
	"fmt"
	"io"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Helcaraxan/gomod/lib/depgraph"
	"github.com/Helcaraxan/gomod/lib/modules"
)

type DepAnalysis struct {
	Module string `yaml:"module"`

	DirectDependencyCount   int `yaml:"direct_dependencies"`
	IndirectDependencyCount int `yaml:"indirect_dependencies"`

	MeanDepAge              time.Duration `yaml:"mean_age"`
	MaxDepAge               time.Duration `yaml:"max_age"`
	DepAgeMonthDistribution []int         `yaml:"age_per_month"`

	AvailableUpdates               int           `yaml:"available_updates"`
	AvailableUpdatesDirect         int           `yaml:"available_updates_direct"`
	MeanUpdateBacklog              time.Duration `yaml:"mean_backlog"`
	MaxUpdateBacklog               time.Duration `yaml:"max_backlog"`
	UpdateBacklogMonthDistribution []int         `yaml:"backlog_per_month"`

	MeanReverseDependencyCount    float64 `yaml:"mean_reverse_deps"`
	MaxReverseDependencyCount     int     `yaml:"max_reverse_deps"`
	ReverseDependencyDistribution []int   `yaml:"reverse_deps_distribution"`
}

var testCurrentTimeInjection *time.Time

func Analyse(log *zap.Logger, g *depgraph.DepGraph) (*DepAnalysis, error) {
	_, moduleMap, err := modules.GetDependenciesWithUpdates(log, g.Path)
	if err != nil {
		return nil, err
	}

	result := &analysis{
		log:       log,
		graph:     g,
		moduleMap: moduleMap,
	}

	for _, dependency := range g.Dependencies.List() {
		result.processDependency(dependency)
	}

	meanDepAge, maxDepAge, depAgeDistribution := result.depAges.compute()
	meanBacklog, maxBacklog, backlogDistribution := result.updateBacklogs.compute()
	meanArity, maxArity, arityDistribution := result.reverseDependencies.compute()
	return &DepAnalysis{
		Module:                         g.Main.Name(),
		DirectDependencyCount:          result.directDependencies,
		IndirectDependencyCount:        result.indirectDependencies,
		MeanDepAge:                     time.Duration(meanDepAge),
		MaxDepAge:                      time.Duration(maxDepAge),
		DepAgeMonthDistribution:        depAgeDistribution,
		AvailableUpdates:               result.updateBacklogs.count(),
		AvailableUpdatesDirect:         result.updatableDirectDependencies,
		MeanUpdateBacklog:              time.Duration(meanBacklog),
		MaxUpdateBacklog:               time.Duration(maxBacklog),
		UpdateBacklogMonthDistribution: backlogDistribution,
		MeanReverseDependencyCount:     meanArity,
		MaxReverseDependencyCount:      int(maxArity),
		ReverseDependencyDistribution:  arityDistribution,
	}, nil
}

type analysis struct {
	log       *zap.Logger
	graph     *depgraph.DepGraph
	moduleMap map[string]*modules.Module

	directDependencies          int
	indirectDependencies        int
	updatableDirectDependencies int
	depAges                     meanMaxDistribution
	updateBacklogs              meanMaxDistribution
	reverseDependencies         meanMaxDistribution
}

func (r *analysis) processDependency(dependency *depgraph.DependencyReference) {
	const month = 30 * 24 * time.Hour
	var isDirect int

	if dependency.Name() == r.graph.Main.Name() {
		return
	}

	if _, ok := r.graph.Main.Successors.Get(dependency.Name()); ok {
		r.directDependencies++
		isDirect = 1
	} else {
		r.indirectDependencies++
	}
	if depArity := dependency.Predecessors.Len(); depArity > 0 {
		r.reverseDependencies.insert(int64(depArity), depArity)
	}

	if timestamp := dependency.Timestamp(); timestamp == nil {
		r.log.Warn("No associated timestamp was found.", zap.String("dependency", dependency.Name()))
		return
	}

	depAge := time.Since(*dependency.Timestamp())
	if testCurrentTimeInjection != nil { // Needed for deterministic tests.
		depAge = testCurrentTimeInjection.Sub(*dependency.Timestamp())
	}
	r.depAges.insert(int64(depAge), int(depAge.Nanoseconds()/month.Nanoseconds()))

	if module := r.moduleMap[dependency.Name()]; module != nil && module.Update != nil && module.Update.Time != nil {
		r.log.Debug("Update available.", zap.String("dependency", dependency.Name()), zap.String("version", module.Update.Version))
		if module.Update.Time.After(*dependency.Timestamp()) {
			updateBacklog := module.Update.Time.Sub(*dependency.Timestamp())
			r.updateBacklogs.insert(int64(updateBacklog), int(updateBacklog.Nanoseconds()/month.Nanoseconds()))
			r.updatableDirectDependencies += isDirect
		} else {
			r.log.Warn("Available update is older than the version currently in use.", zap.String("dependency", dependency.Name()))
		}
	}
}

const (
	noBacklog = `Update backlog statistics:
- No available updates. Congratulations you are entirely up-to-date!`

	backlogTemplate = `Update backlog statistics:
- Number of dependencies with an update:  %d (of which %s direct)
- Mean update backlog of dependencies:    %s
- Maximum update backlog of dependencies: %s
- Update backlog distribution per month:

%s`

	reportTemplate = `-- Analysis for '%s' --
Dependency counts:
- Direct dependencies:   %d
- Indirect dependencies: %d

Age statistics:
- Mean age of dependencies: %s
- Maximum dependency age:   %s
- Age distribution per month:

%s

%s

Reverse dependency statistics:
- Mean number of reverse dependencies:    %.2f
- Maximum number of reverse dependencies: %d
- Reverse dependency count distribution:

%s

`
)

func (a *DepAnalysis) Print(f io.Writer) error {
	updateContent := noBacklog
	if a.AvailableUpdates > 0 {
		directUpdates := "1 is"
		if a.AvailableUpdatesDirect == 0 || a.AvailableUpdatesDirect > 1 {
			directUpdates = fmt.Sprintf("%d are", a.AvailableUpdatesDirect)
		}
		updateContent = fmt.Sprintf(
			backlogTemplate,
			a.AvailableUpdates,
			directUpdates,
			humanDuration(a.MeanUpdateBacklog),
			humanDuration(a.MaxUpdateBacklog),
			printedDistribution(a.UpdateBacklogMonthDistribution, 10),
		)
	}

	_, err := fmt.Fprintf(
		f,
		reportTemplate,
		a.Module,
		a.DirectDependencyCount,
		a.IndirectDependencyCount,
		humanDuration(a.MeanDepAge),
		humanDuration(a.MaxDepAge),
		printedDistribution(a.DepAgeMonthDistribution, 10),
		updateContent,
		a.MeanReverseDependencyCount,
		a.MaxReverseDependencyCount,
		printedDistribution(a.ReverseDependencyDistribution, 10),
	)
	return err
}

type meanMaxDistribution struct {
	mean         float64
	max          int64
	distribution []int
	valCount     int
}

func (d *meanMaxDistribution) insert(val int64, distributionIdx int) {
	d.mean += float64(val)
	d.valCount++

	if val > d.max {
		d.max = val
	}
	d.distribution = insertIntoDistribution(distributionIdx, d.distribution)
}

func (d *meanMaxDistribution) count() int {
	return d.valCount
}

func (d *meanMaxDistribution) compute() (float64, int64, []int) {
	mean := 0.0
	if d.valCount > 0 && d.mean/float64(d.valCount) > 0 {
		mean = d.mean / float64(d.valCount)
	}
	return mean, d.max, d.distribution
}

func humanDuration(d time.Duration) string {
	totalDays := d.Nanoseconds() / (24 * time.Hour.Nanoseconds())
	months := totalDays / 30
	days := totalDays % 30
	return fmt.Sprintf("%d month(s) %d day(s)", months, days)
}

func insertIntoDistribution(idx int, v []int) []int {
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
	if len(distribution) == 0 {
		return []string{""}
	}

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
		stepCount := int(value / step)
		line := "_" + strings.Repeat("#", stepCount)
		if value-float64(stepCount)*step > step/2 { // We can't use 'math.Mod()' as that can lead to rounding issues.
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

func printedDistribution(distribution []int, displayHeight int) string {
	const maxColumns = 50 // completely arbitrary value that should fit with most terminal widths.

	groupingFactor := len(distribution) / maxColumns
	if len(distribution)%maxColumns > 0 {
		groupingFactor++
	} else if groupingFactor == 0 {
		groupingFactor = 1
	}

	pDistribution := distributionCountToPercentage(distribution, groupingFactor)
	lines := distributionToLines(pDistribution, displayHeight)
	rows := rotateDistributionLines(lines, displayHeight)
	rows = annotateDistributionPrintout(rows, pDistribution, groupingFactor)
	return strings.Join(rows, "\n")
}
