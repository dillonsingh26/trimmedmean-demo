package bootstrap

import (
	"fmt"
	"math"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// smallConfig keeps tests fast while still exercising every step.
func smallConfig() Config {
	cfg := DefaultConfig()
	cfg.SampleSizes = []int{10, 25}
	cfg.Iterations = 20
	cfg.Resamples = 20
	return cfg
}

// TestSameResultsWithAnyWorkerCount shows that running jobs in parallel
// does not change the answer.
func TestSameResultsWithAnyWorkerCount(t *testing.T) {
	cfg := smallConfig()
	cfg.Workers = 1
	sequential, err := Run(cfg)
	if err != nil {
		t.Fatal(err)
	}

	cfg.Workers = 8
	parallel, err := Run(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(sequential, parallel) {
		t.Errorf("1 worker gave %+v\n8 workers gave %+v", sequential, parallel)
	}
}

func TestSeedChangesResults(t *testing.T) {
	cfg := smallConfig()
	first, _ := Run(cfg)
	cfg.Seed++
	second, _ := Run(cfg)
	if reflect.DeepEqual(first, second) {
		t.Error("different seeds gave identical results")
	}
}

// TestNoOutliersMatchesTheory checks the simulation against the Central
// Limit Theorem: with no outliers, the standard error of the mean should
// be close to SD / sqrt(n) = 10 / sqrt(100) = 1.
func TestNoOutliersMatchesTheory(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Population.OutlierProbability = 0
	cfg.SampleSizes = []int{100}
	cfg.Iterations = 400
	cfg.Resamples = 20
	cfg.Workers = runtime.NumCPU()

	results, err := Run(cfg)
	if err != nil {
		t.Fatal(err)
	}
	got := results[0]
	if math.Abs(got.SampleMeanSE-got.TheoreticalSE) > 0.1*got.TheoreticalSE {
		t.Errorf("sample mean SE %.3f is not within 10%% of the theoretical %.3f", got.SampleMeanSE, got.TheoreticalSE)
	}
}

// TestTrimmedMeanIsMoreStable checks the study's main finding: with
// outliers present, the trimmed mean varies less than the ordinary mean.
func TestTrimmedMeanIsMoreStable(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SampleSizes = []int{100}
	cfg.Resamples = 50
	cfg.Workers = runtime.NumCPU()

	results, err := Run(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := results[0]; got.TrimmedMeanSE >= got.MeanSE {
		t.Errorf("trimmed mean SE %.3f is not below mean SE %.3f", got.TrimmedMeanSE, got.MeanSE)
	}
}

func TestInvalidConfig(t *testing.T) {
	tests := []struct {
		name     string
		change   func(*Config)
		wantText string
	}{
		{"no sample sizes", func(c *Config) { c.SampleSizes = nil }, "at least one sample size"},
		{"zero sample size", func(c *Config) { c.SampleSizes = []int{10, 0} }, "sample sizes must be at least 1"},
		{"one iteration", func(c *Config) { c.Iterations = 1 }, "iterations must be at least 2"},
		{"no resamples", func(c *Config) { c.Resamples = 0 }, "resamples must be at least 1"},
		{"no workers", func(c *Config) { c.Workers = 0 }, "workers must be at least 1"},
		{"probability above 1", func(c *Config) { c.Population.OutlierProbability = 1.5 }, "outlier probability"},
		{"trim of 0.5", func(c *Config) { c.Trim = 0.5 }, "invalid trimming proportion"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := smallConfig()
			tt.change(&cfg)
			_, err := Run(cfg)
			if err == nil || !strings.Contains(err.Error(), tt.wantText) {
				t.Errorf("got error %v, want one containing %q", err, tt.wantText)
			}
		})
	}
}

func TestStats(t *testing.T) {
	values := []float64{2, 4, 4, 4, 5, 5, 7, 9}
	if got := mean(values); got != 5 {
		t.Errorf("mean: got %g, want 5", got)
	}
	// R: sd(c(2, 4, 4, 4, 5, 5, 7, 9))
	if got := stdDev(values); math.Abs(got-2.138089935299395) > 1e-15 {
		t.Errorf("stdDev: got %.16g, want 2.138089935299395", got)
	}
	if got := medianOfSorted([]float64{1, 3, 8}); got != 3 {
		t.Errorf("odd median: got %g, want 3", got)
	}
	if got := medianOfSorted([]float64{1, 3, 8, 10}); got != 5.5 {
		t.Errorf("even median: got %g, want 5.5", got)
	}
}

// BenchmarkRun times the full study from the R script (100 repetitions,
// 100 resamples, 4 sample sizes) with one worker and with every CPU core
// (run with make bench).
func BenchmarkRun(b *testing.B) {
	for _, workers := range []int{1, runtime.NumCPU()} {
		b.Run(fmt.Sprintf("workers=%d", workers), func(b *testing.B) {
			cfg := DefaultConfig()
			cfg.Workers = workers
			b.ReportAllocs()
			for b.Loop() {
				if _, err := Run(cfg); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
