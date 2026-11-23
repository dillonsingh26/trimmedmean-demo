package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/dillonsingh26/trimmedmean-demo/internal/bootstrap"
)

// runBootstrap runs the bootstrap study. Results go to stdout; the log of
// timing and memory use goes to stderr.
func runBootstrap(args []string, stdout, stderr io.Writer) error {
	cfg := bootstrap.DefaultConfig()
	flags := newFlagSet("bootstrap", "bootstrap [options]", stderr)
	flags.IntVar(&cfg.Resamples, "resamples", cfg.Resamples, "bootstrap resamples per sample (B in the R script)")
	flags.IntVar(&cfg.Iterations, "iterations", cfg.Iterations, "repetitions of the whole study")
	sizesText := flags.String("sizes", "25,100,225,400", "comma-separated sample sizes")
	flags.Float64Var(&cfg.Trim, "trim", cfg.Trim, "trimmed mean: proportion trimmed from each end")
	flags.Float64Var(&cfg.Population.OutlierProbability, "outliers", cfg.Population.OutlierProbability, "probability that a value is an outlier")
	flags.Int64Var(&cfg.Seed, "seed", cfg.Seed, "random number seed")
	flags.IntVar(&cfg.Workers, "workers", runtime.NumCPU(), "jobs run at the same time (1 runs them one by one)")
	cpuProfile := flags.String("cpuprofile", "", "write a CPU profile to this file")
	memProfile := flags.String("memprofile", "", "write a memory allocation profile to this file")
	if err := parseFlags(flags, args); err != nil {
		return err
	}

	sizes, err := parseList("sizes", *sizesText, strconv.Atoi)
	if err != nil {
		return err
	}
	cfg.SampleSizes = sizes

	if *cpuProfile != "" {
		stop, err := startCPUProfile(*cpuProfile)
		if err != nil {
			return err
		}
		defer stop()
	}

	logger := slog.New(slog.NewTextHandler(stderr, nil))
	results, err := runTimed(cfg, logger)
	if err != nil {
		return err
	}

	if *memProfile != "" {
		if err := writeAllocProfile(*memProfile); err != nil {
			return err
		}
	}

	printStudy(stdout, cfg, results)
	return nil
}

// runTimed runs the study and logs how long it took and how much memory
// it allocated.
func runTimed(cfg bootstrap.Config, logger *slog.Logger) ([]bootstrap.SizeResult, error) {
	logger.Info("bootstrap study starting",
		"sizes", cfg.SampleSizes, "iterations", cfg.Iterations, "resamples", cfg.Resamples,
		"workers", cfg.Workers, "seed", cfg.Seed)

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()

	results, err := bootstrap.Run(cfg)
	if err != nil {
		return nil, err
	}

	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)
	logger.Info("bootstrap study finished",
		"elapsed", elapsed.Round(100*time.Microsecond),
		"jobs", cfg.Iterations*len(cfg.SampleSizes),
		"allocated_mb", fmt.Sprintf("%.1f", float64(after.TotalAlloc-before.TotalAlloc)/1e6),
		"gc_cycles", after.NumGC-before.NumGC)
	return results, nil
}

func printStudy(stdout io.Writer, cfg bootstrap.Config, results []bootstrap.SizeResult) {
	population := cfg.Population
	fmt.Fprintln(stdout, "Bootstrap study: standard errors of three estimates of the center")
	fmt.Fprintf(stdout, "Population: Normal(mean %g, SD %g), with probability %g of an outlier from Normal(mean %g, SD %g)\n",
		population.MainMean, population.MainSD, population.OutlierProbability, population.OutlierMean, population.OutlierSD)
	fmt.Fprintf(stdout, "%d repetitions x %d bootstrap resamples; trimmed mean drops %.4g%% from each end\n\n",
		cfg.Iterations, cfg.Resamples, 100*cfg.Trim)

	table := tabwriter.NewWriter(stdout, 0, 0, 3, ' ', tabwriter.AlignRight)
	fmt.Fprintln(table, "n\tTheory (no outliers)\tSample mean\tBootstrap mean\tBootstrap median\tBootstrap trimmed mean\tMost stable\t")
	for _, r := range results {
		fmt.Fprintf(table, "%d\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f\t%s\t\n",
			r.SampleSize, r.TheoreticalSE, r.SampleMeanSE, r.MeanSE, r.MedianSE, r.TrimmedMeanSE, mostStable(r))
	}
	table.Flush()

	fmt.Fprintln(stdout, "\nLower is more stable. Theory is SD/sqrt(n): the mean's standard error if there were no outliers.")
}

// mostStable names the bootstrap estimate with the smallest standard error.
func mostStable(r bootstrap.SizeResult) string {
	switch {
	case r.TrimmedMeanSE <= r.MeanSE && r.TrimmedMeanSE <= r.MedianSE:
		return "trimmed mean"
	case r.MedianSE <= r.MeanSE:
		return "median"
	default:
		return "mean"
	}
}

// startCPUProfile begins recording a CPU profile and returns the function
// that stops it. View the result with: go tool pprof -top FILE
func startCPUProfile(path string) (stop func(), err error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	if err := pprof.StartCPUProfile(file); err != nil {
		file.Close()
		return nil, err
	}
	return func() {
		pprof.StopCPUProfile()
		file.Close()
	}, nil
}

// writeAllocProfile saves a record of every memory allocation so far.
// View it with: go tool pprof -sample_index=alloc_space -top FILE
func writeAllocProfile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return pprof.Lookup("allocs").WriteTo(file, 0)
}
