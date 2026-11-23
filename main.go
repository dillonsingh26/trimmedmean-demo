// Command trimstats shows the trimmedmean package in use: it compares Go's
// trimmed means with R's, computes a trimmed mean for any file of numbers,
// and runs a parallel bootstrap study.
// Run "trimstats help" for the commands.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const usage = `trimstats: trimmed means in Go, checked against R

Usage:
  trimstats                       compare Go and R (same as "trimstats compare")
  trimstats compare               compare Go's trimmed means with R's on the included samples
  trimstats mean [-trim P] FILE   trimmed mean of the numbers in FILE, one per line (- reads typed input)
  trimstats bootstrap [flags]     bootstrap study: how stable are the mean, median, and trimmed mean?
  trimstats help                  show this message

Run "trimstats COMMAND -h" to see a command's options.
`

// errBadFlags means the flag package has already printed the problem and
// the command's options, so main should not print it again.
var errBadFlags = errors.New("invalid command-line options")

func main() {
	err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	switch {
	case err == nil, errors.Is(err, flag.ErrHelp):
		// Success, or the user asked for a command's options with -h.
	case errors.Is(err, errBadFlags):
		os.Exit(2)
	default:
		fmt.Fprintln(os.Stderr, "trimstats:", err)
		os.Exit(1)
	}
}

// run picks the command named by the first argument. Taking the input
// and outputs as arguments lets tests run commands without a terminal.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		if err := runCompare(nil, stdout, stderr); err != nil {
			return err
		}
		fmt.Fprint(stdout, "\nRun \"trimstats help\" to see the other commands.\n")
		return nil
	}

	command, commandArgs := args[0], args[1:]
	switch command {
	case "compare":
		return runCompare(commandArgs, stdout, stderr)
	case "mean":
		return runMean(commandArgs, stdin, stdout, stderr)
	case "bootstrap":
		return runBootstrap(commandArgs, stdout, stderr)
	case "help", "-h", "-help", "--help":
		fmt.Fprint(stdout, usage)
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\n%s", command, usage)
	}
}

// newFlagSet creates a command's flag set with a usage line describing it.
func newFlagSet(name, synopsis string, stderr io.Writer) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprintf(stderr, "Usage: trimstats %s\n\nOptions:\n", synopsis)
		flags.PrintDefaults()
	}
	return flags
}

// parseFlags parses a command's options, marking errors the flag package
// has already reported.
func parseFlags(flags *flag.FlagSet, args []string) error {
	err := flags.Parse(args)
	if err != nil && !errors.Is(err, flag.ErrHelp) {
		return errBadFlags
	}
	return err
}

// parseList splits a comma-separated option such as "0.05,0.10" and
// parses each part, naming the option in any error.
func parseList[T any](option, text string, parse func(string) (T, error)) ([]T, error) {
	var values []T
	for part := range strings.SplitSeq(text, ",") {
		value, err := parse(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("-%s: %q is not a valid number", option, part)
		}
		values = append(values, value)
	}
	return values, nil
}

func parseFloat(text string) (float64, error) { return strconv.ParseFloat(text, 64) }
