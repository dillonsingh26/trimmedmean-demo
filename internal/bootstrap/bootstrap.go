// Package bootstrap is a parallel Go version of the course's
// run-bootstrap-robust.R, estimating how stable the mean, median, and
// trimmed mean are when samples contain rare outliers.
//
// Each (repetition, sample size) pair is an independent job with its own
// seeded random number generator, so results are identical for any
// number of workers.
package bootstrap

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"slices"
	"sync"

	"github.com/dillonsingh26/trimmedmean"
	"github.com/seehuhn/mt19937"
)

// Mixture describes a population that is mostly a main normal distribution,
// with a small chance that each value comes from an outlier distribution.
type Mixture struct {
	MainMean, MainSD       float64
	OutlierMean, OutlierSD float64
	OutlierProbability     float64
}

// Config holds the study settings.
type Config struct {
	Population  Mixture
	SampleSizes []int
	Iterations  int     // repetitions of the whole study (100 in the R script)
	Resamples   int     // bootstrap resamples per sample (B in the R script)
	Trim        float64 // symmetric trimming proportion for the trimmed mean
	Seed        int64
	Workers     int // goroutines running jobs at the same time
}

// DefaultConfig returns the settings used in run-bootstrap-robust.R.
func DefaultConfig() Config {
	return Config{
		Population: Mixture{
			MainMean: 100, MainSD: 10,
			OutlierMean: 200, OutlierSD: 1,
			OutlierProbability: 0.01,
		},
		SampleSizes: []int{25, 100, 225, 400},
		Iterations:  100,
		Resamples:   100,
		Trim:        0.05,
		Seed:        9999,
		Workers:     1,
	}
}

// SizeResult holds the estimated standard errors for one sample size.
type SizeResult struct {
	SampleSize    int
	TheoreticalSE float64 // MainSD / sqrt(n): the Central Limit Theorem value with no outliers
	SampleMeanSE  float64 // from the original samples' means
	MeanSE        float64 // from the bootstrap means
	MedianSE      float64 // from the bootstrap medians
	TrimmedMeanSE float64 // from the bootstrap trimmed means
}

// jobResult holds one job's output: one row of studyResults in the R script.
type jobResult struct {
	sampleMean, bootMean, bootMedian, bootTrimmedMean float64
}

// Run carries out the study and returns one result per sample size, in
// the order of cfg.SampleSizes.
func Run(cfg Config) ([]SizeResult, error) {
	if err := validate(cfg); err != nil {
		return nil, err
	}

	jobCount := cfg.Iterations * len(cfg.SampleSizes)
	results := make([]jobResult, jobCount)

	// Workers take job numbers from the channel. Each job writes only its
	// own slot in results, so no locking is needed.
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range cfg.Workers {
		wg.Go(func() {
			for job := range jobs {
				results[job] = runJob(cfg, job)
			}
		})
	}
	for job := range jobCount {
		jobs <- job
	}
	close(jobs)
	wg.Wait()

	return summarize(cfg, results), nil
}

// runJob handles one (repetition, sample size) pair. Job numbers run
// through every sample size for repetition 0, then repetition 1, and so on.
func runJob(cfg Config, job int) jobResult {
	n := cfg.SampleSizes[job%len(cfg.SampleSizes)]
	rng := newGenerator(cfg.Seed + int64(job))
	sample := cfg.Population.draw(rng, n)

	resample := make([]float64, n)
	var meanTotal, medianTotal, trimmedTotal float64
	for range cfg.Resamples {
		for i := range resample {
			resample[i] = sample[rng.Intn(n)]
		}
		meanTotal += mean(resample)

		slices.Sort(resample)
		medianTotal += medianOfSorted(resample)

		// The proportion was checked in validate, so this cannot fail.
		trimmed, _ := trimmedmean.TrimmedMean(resample, cfg.Trim)
		trimmedTotal += trimmed
	}

	resamples := float64(cfg.Resamples)
	return jobResult{
		sampleMean:      mean(sample),
		bootMean:        meanTotal / resamples,
		bootMedian:      medianTotal / resamples,
		bootTrimmedMean: trimmedTotal / resamples,
	}
}

// summarize groups job results by sample size and computes the standard
// deviation of each statistic across repetitions.
func summarize(cfg Config, results []jobResult) []SizeResult {
	summaries := make([]SizeResult, len(cfg.SampleSizes))
	for sizeIndex, n := range cfg.SampleSizes {
		var sampleMeans, means, medians, trimmedMeans []float64
		for job := sizeIndex; job < len(results); job += len(cfg.SampleSizes) {
			sampleMeans = append(sampleMeans, results[job].sampleMean)
			means = append(means, results[job].bootMean)
			medians = append(medians, results[job].bootMedian)
			trimmedMeans = append(trimmedMeans, results[job].bootTrimmedMean)
		}

		summaries[sizeIndex] = SizeResult{
			SampleSize:    n,
			TheoreticalSE: cfg.Population.MainSD / math.Sqrt(float64(n)),
			SampleMeanSE:  stdDev(sampleMeans),
			MeanSE:        stdDev(means),
			MedianSE:      stdDev(medians),
			TrimmedMeanSE: stdDev(trimmedMeans),
		}
	}
	return summaries
}

// newGenerator returns a Mersenne Twister generator, the same algorithm
// family R uses by default, wrapped to provide normal and integer draws.
func newGenerator(seed int64) *rand.Rand {
	source := mt19937.New()
	source.Seed(seed)
	return rand.New(source)
}

// draw returns n values from the mixture, choosing the distribution for
// each value separately, like rmixnorm in the R script.
func (m Mixture) draw(rng *rand.Rand, n int) []float64 {
	values := make([]float64, n)
	for i := range values {
		if rng.Float64() >= m.OutlierProbability {
			values[i] = m.MainMean + m.MainSD*rng.NormFloat64()
		} else {
			values[i] = m.OutlierMean + m.OutlierSD*rng.NormFloat64()
		}
	}
	return values
}

// validate returns an error describing the first unusable setting.
func validate(cfg Config) error {
	if len(cfg.SampleSizes) == 0 {
		return errors.New("at least one sample size is required")
	}
	for _, n := range cfg.SampleSizes {
		if n < 1 {
			return fmt.Errorf("sample sizes must be at least 1, got %d", n)
		}
	}
	if cfg.Iterations < 2 {
		return fmt.Errorf("iterations must be at least 2 to estimate a standard error, got %d", cfg.Iterations)
	}
	if cfg.Resamples < 1 {
		return fmt.Errorf("resamples must be at least 1, got %d", cfg.Resamples)
	}
	if cfg.Workers < 1 {
		return fmt.Errorf("workers must be at least 1, got %d", cfg.Workers)
	}
	if p := cfg.Population.OutlierProbability; p < 0 || p > 1 || math.IsNaN(p) {
		return fmt.Errorf("outlier probability must be between 0 and 1, got %g", p)
	}
	// Reuse the package's own rules for the trimming proportion.
	if _, err := trimmedmean.TrimmedMean([]float64{0}, cfg.Trim); err != nil {
		return err
	}
	return nil
}
