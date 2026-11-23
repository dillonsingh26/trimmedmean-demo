package main

import (
	"embed"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/dillonsingh26/trimmedmean"
	"github.com/dillonsingh26/trimmedmean-demo/internal/numfile"
)

// The samples and R's results are built into the program, so the
// comparison works from any folder without extra files.
//
//go:embed data/*.csv
var dataFiles embed.FS

// comparison is one row of the Go-vs-R table.
type comparison struct {
	sample     string
	size       int
	trim       float64
	goValue    float64
	rValue     float64
	difference float64 // goValue - rValue; 0 when every bit matches
}

func runCompare(args []string, stdout, stderr io.Writer) error {
	flags := newFlagSet("compare", "compare", stderr)
	if err := parseFlags(flags, args); err != nil {
		return err
	}

	comparisons, err := compareWithR()
	if err != nil {
		return err
	}
	printComparisons(stdout, comparisons)
	for _, c := range comparisons {
		if c.difference != 0 {
			return fmt.Errorf("Go and R disagree for %s at trim %.2f", c.sample, c.trim)
		}
	}

	return printAsymmetricExample(stdout)
}

// compareWithR computes Go's trimmed mean for every row of
// data/r-results.csv, which r/make-samples.R created.
func compareWithR() ([]comparison, error) {
	rows, err := readRResults()
	if err != nil {
		return nil, err
	}

	var comparisons []comparison
	for _, row := range rows {
		goValue, size, err := goTrimmedMean(row.sample, row.trim)
		if err != nil {
			return nil, err
		}
		comparisons = append(comparisons, comparison{
			sample:     row.sample,
			size:       size,
			trim:       row.trim,
			goValue:    goValue,
			rValue:     row.rValue,
			difference: goValue - row.rValue,
		})
	}
	return comparisons, nil
}

// goTrimmedMean loads a sample and computes its trimmed mean. Integer
// samples are read as []int and float samples as []float64, so both
// types go through the generic package function.
func goTrimmedMean(sample string, trim float64) (value float64, size int, err error) {
	file, err := dataFiles.Open("data/" + sample + ".csv")
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	if strings.HasPrefix(sample, "integers") {
		values, err := numfile.ReadInts(file)
		if err != nil {
			return 0, 0, fmt.Errorf("%s: %w", sample, err)
		}
		value, err = trimmedmean.TrimmedMean(values, trim)
		return value, len(values), err
	}

	values, err := numfile.ReadFloats(file)
	if err != nil {
		return 0, 0, fmt.Errorf("%s: %w", sample, err)
	}
	value, err = trimmedmean.TrimmedMean(values, trim)
	return value, len(values), err
}

type rResult struct {
	sample string
	trim   float64
	rValue float64
}

// readRResults reads data/r-results.csv, whose columns are
// sample, trim, and r_value.
func readRResults() ([]rResult, error) {
	file, err := dataFiles.Open("data/r-results.csv")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("r-results.csv: %w", err)
	}

	var results []rResult
	for _, record := range records[1:] { // skip the header row
		trim, trimErr := strconv.ParseFloat(record[1], 64)
		rValue, valueErr := strconv.ParseFloat(record[2], 64)
		if trimErr != nil || valueErr != nil {
			return nil, fmt.Errorf("r-results.csv: bad row %v", record)
		}
		results = append(results, rResult{sample: record[0], trim: trim, rValue: rValue})
	}
	return results, nil
}

func printComparisons(stdout io.Writer, comparisons []comparison) {
	fmt.Fprintln(stdout, "Go trimmed means compared with R's mean(x, trim = p)")
	fmt.Fprintln(stdout)

	table := tabwriter.NewWriter(stdout, 0, 0, 3, ' ', tabwriter.AlignRight)
	fmt.Fprintln(table, "Sample\tn\tTrim\tGo\tR\tGo − R\t")
	identical := 0
	for _, c := range comparisons {
		fmt.Fprintf(table, "%s\t%d\t%.2f\t%.17g\t%.17g\t%g\t\n",
			c.sample, c.size, c.trim, c.goValue, c.rValue, c.difference)
		if c.difference == 0 {
			identical++
		}
	}
	table.Flush()

	fmt.Fprintf(stdout, "\n%d of %d Go results are identical to R's, to all 17 significant digits.\n",
		identical, len(comparisons))
	fmt.Fprintln(stdout, "Trim 0.00 is the ordinary mean; the outliers pull it well above the trimmed means.")
}

// printAsymmetricExample shows the package's asymmetric trimming on the
// floats-100 sample, whose outliers are all on the high side.
func printAsymmetricExample(stdout io.Writer) error {
	file, err := dataFiles.Open("data/floats-100.csv")
	if err != nil {
		return err
	}
	defer file.Close()
	values, err := numfile.ReadFloats(file)
	if err != nil {
		return err
	}

	symmetric, err := trimmedmean.TrimmedMean(values, 0.05)
	if err != nil {
		return err
	}
	highOnly, err := trimmedmean.TrimmedMean(values, 0, 0.05)
	if err != nil {
		return err
	}

	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Asymmetric trimming (Go only; R's mean() trims both ends equally)")
	fmt.Fprintln(stdout, "floats-100 has 5 outliers near 200, all on the high side:")
	fmt.Fprintf(stdout, "  5%% from each end:          %.4f  (also drops the 5 lowest ordinary values)\n", symmetric)
	fmt.Fprintf(stdout, "  5%% from the high end only: %.4f  (drops just the outliers)\n", highOnly)
	return nil
}
