package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/dillonsingh26/trimmedmean"
	"github.com/dillonsingh26/trimmedmean-demo/internal/numfile"
)

// runMean computes the trimmed mean of a file of numbers given by the user.
func runMean(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := newFlagSet("mean", "mean [-trim P] FILE", stderr)
	trimText := flags.String("trim", "0.05",
		"proportion to trim: one value (0.05) trims both ends equally,\ntwo values (0.05,0.10) trim the low and high ends separately")
	if err := parseFlags(flags, args); err != nil {
		return err
	}

	if flags.NArg() != 1 {
		flags.Usage()
		return errors.New(`mean needs exactly one FILE (or "-" to type or pipe in numbers)`)
	}
	path := flags.Arg(0)

	trims, err := parseList("trim", *trimText, parseFloat)
	if err != nil {
		return err
	}

	values, err := readNumbers(path, stdin)
	if err != nil {
		return err
	}

	ordinary, err := trimmedmean.TrimmedMean(values)
	if err != nil {
		return err
	}
	trimmed, err := trimmedmean.TrimmedMean(values, trims...)
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Values:        %d\n", len(values))
	fmt.Fprintf(stdout, "Trimming:      %s\n", describeTrims(trims))
	fmt.Fprintf(stdout, "Mean:          %.10g\n", ordinary)
	fmt.Fprintf(stdout, "Trimmed mean:  %.10g\n", trimmed)
	return nil
}

// readNumbers reads from standard input when path is "-", and from the
// named file otherwise.
func readNumbers(path string, stdin io.Reader) ([]float64, error) {
	if path != "-" {
		return numfile.ReadFloatsFile(path)
	}
	values, err := numfile.ReadFloats(stdin)
	if err != nil {
		return nil, fmt.Errorf("standard input: %w", err)
	}
	return values, nil
}

// describeTrims puts the trimming proportions into words for the output.
func describeTrims(trims []float64) string {
	if len(trims) == 1 {
		return fmt.Sprintf("%.4g%% from each end", 100*trims[0])
	}
	// Two or more: the package rejects more than two before this is printed.
	return fmt.Sprintf("%.4g%% from the low end, %.4g%% from the high end", 100*trims[0], 100*trims[1])
}
