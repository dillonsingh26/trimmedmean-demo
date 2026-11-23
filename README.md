# trimmedmean-demo

[![CI](https://github.com/dillonsingh26/trimmedmean-demo/actions/workflows/ci.yml/badge.svg)](https://github.com/dillonsingh26/trimmedmean-demo/actions/workflows/ci.yml)

A command-line program, `trimstats`, that uses the [trimmedmean](https://github.com/dillonsingh26/trimmedmean) Go package to:

1. **Compare Go with R.** It computes trimmed means of 100 and 1,001 integers and floats, then checks them against R's `mean(x, trim = p)`. All 16 results are identical to R's, to all 17 significant digits.
2. **Compute a trimmed mean** of any file of numbers, with symmetric or asymmetric trimming.
3. **Run a parallel bootstrap study.** This is the course's `run-bootstrap-robust.R` rewritten in Go. It shows the trimmed mean is a robust estimate of the center when data has outliers, and it runs about 180 times faster than the R script.

## Contents

- [Quick start](#quick-start)
- [Using the trimmedmean package in your own Go program](#using-the-trimmedmean-package-in-your-own-go-program)
- [Commands](#commands)
- [Results: Go compared with R](#results-go-compared-with-r)
- [Results: bootstrap study](#results-bootstrap-study)
- [Testing](#testing)
- [Building executables](#building-executables)
- [Project layout](#project-layout)
- [GenAI Tools](#genai-tools)

## Quick start

You need [Go 1.25 or newer](https://go.dev/dl/). R is only needed to recreate the R results.

```bash
git clone https://github.com/dillonsingh26/trimmedmean-demo.git
cd trimmedmean-demo
go run .                # Go-vs-R comparison
go run . bootstrap      # bootstrap study
go run . help           # every command
```

`go run .` downloads the trimmedmean package automatically the first time.

## Using the trimmedmean package in your own Go program

These steps start from an empty folder.

**1. Create a module.** A module is Go's unit of code with its own list of dependencies, like an R project with an `renv` lockfile.

```bash
mkdir myproject && cd myproject
go mod init example.com/myproject
```

**2. Download the package.** This adds it to `go.mod`, similar to `install.packages()` plus recording the version.

```bash
go get github.com/dillonsingh26/trimmedmean@latest
```

**3. Import and call it.** Save this as `main.go`:

```go
package main

import (
	"fmt"
	"log"

	"github.com/dillonsingh26/trimmedmean"
)

func main() {
	data := []float64{1.2, 0.8, 1.1, 0.9, 1.0, 1.3, 0.7, 1.0, 1.2, 0.9, 30.5}

	// Symmetric: drop 10% from each end, like mean(x, trim = 0.1) in R.
	symmetric, err := trimmedmean.TrimmedMean(data, 0.1)
	if err != nil {
		log.Fatal(err)
	}

	// Asymmetric: drop nothing from the low end and 10% from the high end.
	highOnly, err := trimmedmean.TrimmedMean(data, 0, 0.1)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("symmetric: %.3f, high end only: %.3f\n", symmetric, highOnly)
}
```

**4. Run it:**

```bash
go run .
# symmetric: 1.044, high end only: 1.010
```

**What the function accepts:**
- **Data:** any slice of built-in numbers, such as `[]int`, `[]int64`, or `[]float64`.
- **Trimming:** zero, one, or two proportions. Zero means no trimming, one means the same proportion from each end, and two means the low end, then the high end.
- **Errors:** invalid input returns an error and never a misleading number. Examples are an empty slice, a `NaN` value, or proportions that would trim everything.

The [package README](https://github.com/dillonsingh26/trimmedmean) has the full details.

## Commands

| Command | What it does |
|---|---|
| `trimstats` | Same as `trimstats compare` |
| `trimstats compare` | Go-vs-R comparison table, plus an asymmetric trimming example |
| `trimstats mean [-trim P] FILE` | Trimmed mean of a file with one number per line (a header line is allowed). Use `-` as FILE to type or pipe in numbers |
| `trimstats bootstrap [options]` | Bootstrap study. Run `trimstats bootstrap -h` for the options: `-resamples`, `-iterations`, `-sizes`, `-trim`, `-outliers`, `-seed`, `-workers`, `-cpuprofile`, `-memprofile` |
| `trimstats help` | List the commands |

**Examples:**

```bash
trimstats mean data/floats-100.csv                 # 5% from each end (the default)
trimstats mean -trim 0.1 data/floats-100.csv       # 10% from each end
trimstats mean -trim 0,0.05 data/floats-100.csv    # 5% from the high end only
printf '3\n1\n4\n1\n5\n9\n2\n6\n' | trimstats mean -trim 0.25 -
trimstats bootstrap -resamples 1000 -iterations 500
```

**Input checking.** Each mistake gets a specific message and exit code 1:

| Mistake | Message |
|---|---|
| A word in the data | `line 3: "two" is not a valid number` |
| A misspelled option | `flag provided but not defined: -trimm`, then the list of valid options (exit code 2) |
| An empty file | `no numbers found` |
| An impossible trim | `invalid trimming proportion: lower=0.6, upper=0.6 (each must be at least 0, and their sum below 1)` |

## Results: Go compared with R

[`r/make-samples.R`](r/make-samples.R) creates four samples with a fixed random seed and computes `mean(x, trim = p)` in R for four trimming proportions. Both R and Go read the same files from [`data/`](data).

| Sample | Contents |
|---|---|
| `integers-100` | 94 whole numbers around 50, plus 4 high and 2 low outliers |
| `integers-1001` | 951 whole numbers around 50, plus 35 high and 15 low outliers |
| `floats-100` | 95 decimals around 100, plus 5 high outliers near 200 |
| `floats-1001` | the course script's mixture distribution, with a 5% chance of an outlier near 200 |

The 1,001-value samples check R's rounding rule. Here 1001 × 0.05 = 50.05, and R drops `floor(50.05) = 50` values from each end. Go must do the same.

Output of `trimstats compare` (also in [docs/compare-output.txt](docs/compare-output.txt)):

```
          Sample      n   Trim                   Go                    R   Go − R
    integers-100    100   0.00   54.640000000000001   54.640000000000001        0
    integers-100    100   0.05   48.555555555555557   48.555555555555557        0
    integers-100    100   0.10   48.424999999999997   48.424999999999997        0
    integers-100    100   0.20   48.299999999999997   48.299999999999997        0
   integers-1001   1001   0.00   54.885114885114888   54.885114885114888        0
   integers-1001   1001   0.05   50.521642619311876   50.521642619311876        0
   integers-1001   1001   0.10    50.40449438202247    50.40449438202247        0
   integers-1001   1001   0.20   50.277870216306155   50.277870216306155        0
      floats-100    100   0.00   106.97873763185945   106.97873763185945        0
      floats-100    100   0.05   103.58210324871303   103.58210324871303        0
      floats-100    100   0.10   103.82164037504052   103.82164037504052        0
      floats-100    100   0.20   104.05109774883805   104.05109774883805        0
     floats-1001   1001   0.00   105.39079996477666   105.39079996477666        0
     floats-1001   1001   0.05   101.59338533240729   101.59338533240729        0
     floats-1001   1001   0.10   101.37984050540919   101.37984050540919        0
     floats-1001   1001   0.20   101.26163347713361   101.26163347713361        0

16 of 16 Go results are identical to R's, to all 17 significant digits.
```

**Summary:**

- **The required test matches exactly.** Go's 5% symmetric trimmed mean equals R's `mean(x, trim = 0.05)` bit for bit, for 100 integers and 100 floats, and for the 1,001-value samples too.
- **It wasn't exact at first.** The first Go version differed from R in the last one or two digits, by up to 3e-13.
  - Checking against exact fraction arithmetic showed that **R was exactly right and Go was not.** R adds numbers using 80-bit precision, while Go's `float64` is 64-bit.
  - The package now uses compensated summation, a method that tracks and adds back rounding error. With it, Go gives the exactly rounded answer too. The [package README](https://github.com/dillonsingh26/trimmedmean#how-it-works) has the details.
- **Trimming removes most of the outliers' pull.** On every sample, the plain mean (trim 0.00) is 3.4 to 6.1 points higher than the 5% trimmed mean, because the outliers pull it up.
- **Asymmetric trimming fixes a bias that symmetric trimming can't.** In `floats-100`, all 5 outliers are on the high side.
  - Trimming 5% from each end also removes the 5 *lowest ordinary* values, which leaves the result too high at 103.58.
  - Trimming 5% from the high end only (`TrimmedMean(x, 0, 0.05)`) removes just the outliers. That gives 102.08, exactly the mean of the 95 ordinary values.
  - R's `mean()` can only trim both ends equally.

To recreate R's side: run `make r-samples`, which runs `Rscript r/make-samples.R`. This was produced with R 4.5.1.

## Results: bootstrap study

**The question:** when about 1% of values are extreme outliers, which estimate of the center varies least from sample to sample: the mean, the median, or the 5% trimmed mean?

**The setup.** The study follows the course's [`run-bootstrap-robust.R`](r/run-bootstrap-robust.R):
- **Population:** values come from Normal(100, 10), with a 1% chance of an outlier from Normal(200, 1).
- **Sample sizes:** n = 25, 100, 225, and 400.
- **Repeats:** 100. Each repeat draws a fresh sample and takes 100 bootstrap resamples of it.
- **Result:** the standard error (SE) of each estimate is its standard deviation across the 100 repeats. Lower means more stable.

### Standard errors: R and Go side by side

| n | Theory, no outliers | R mean | R median | R trimmed | Go mean | Go median | Go trimmed |
|---|---|---|---|---|---|---|---|
| 25 | 2.00 | 2.65 | **2.17** | 2.25 | 3.09 | **2.29** | 2.57 |
| 100 | 1.00 | 1.42 | 1.11 | **1.02** | 1.30 | 1.22 | **1.04** |
| 225 | 0.67 | 0.91 | 0.72 | **0.63** | 0.96 | 0.76 | **0.68** |
| 400 | 0.50 | 0.75 | 0.68 | **0.57** | 0.70 | 0.64 | **0.54** |

Bold marks the most stable estimate in each row for each language. The R column is from the course's listing (`listing-from-100-run-bootstrap-robust.txt`), which this machine reproduces exactly. The Go column is in [docs/bootstrap-output.txt](docs/bootstrap-output.txt).

**What the study shows:**

- **R and Go reach the same conclusion.**
  - Even 1% outliers make the mean 30–55% less stable than theory predicts.
  - From n = 100 on, the 5% trimmed mean is the most stable estimate, close to what the mean would achieve with no outliers at all.
  - At n = 25 the median wins. Here 5% of 25 rounds down to 1 value per end, so a sample with 2 outliers still keeps one.
- **Why the trimmed mean beats the median.** The median relies on the middle one or two values. The trimmed mean drops only 10% of the data and averages the other 90%, so it resists outliers while still using most of the information.
- **The numbers differ but the pattern doesn't.** R and Go use different random numbers, so their figures can't match the way the trimmed means above do.
  - Go uses the Mersenne Twister, the generator the assignment suggested, through the [seehuhn/mt19937](https://github.com/seehuhn/mt19937) package. This is the 64-bit version; R uses the 32-bit version, and R also converts uniform random numbers to normal ones by a different method.
  - With only 100 repeats, an SE estimate itself varies by roughly ±7%, and more at n = 25 where outliers are rare and lumpy.
  - A 50 times larger Go run (`-iterations 500 -resamples 1000`, in [docs/bootstrap-large-output.txt](docs/bootstrap-large-output.txt)) gives SEs of 2.83 / 2.32 / 2.37 for mean / median / trimmed at n = 25, and 1.35 / 1.22 / 1.04 at n = 100. That's the same pattern, with less noise.

### Speed: Go compared with R

| Version | Time for the full study (400 samples × 100 resamples) |
|---|---|
| R, `run-bootstrap-robust.R` | 8.2 s (average of 3 runs) |
| Go, 1 core (`-workers 1`) | 0.27 s |
| Go, all 10 cores (default) | **0.045 s** |

These were measured on an Apple M4 with `make r-time`. The program's own timing log is in [docs/bootstrap-log.txt](docs/bootstrap-log.txt).

**How the parallel version works.** Each pair of (repeat, sample size) is an independent job, so the 400 jobs are spread across all CPU cores using goroutines. Goroutines are Go's lightweight threads, comparable to `parallel::mclapply` in R.
- Every job gets its own random number generator, seeded from its job number.
- So the results are **identical whether 1 or 10 workers run**. A unit test checks this, and the race detector confirms the workers never write to the same memory.

**Two caveats keep the speed comparison fair:**
- The installed R is the Intel (x86_64) build running through Apple's Rosetta translator.
- The R script grows its results table with `rbind` inside the loop, which is slow in R.

A vectorized R version would close part of the gap. Go would still be well ahead, especially with more cores.

**Profiling** (`make profile`) shows most CPU time goes to sorting the resamples. 93% of allocated memory is the trimmedmean package's sorted copy of each resample. The package copies the data so it never changes the caller's slice, which is a deliberate safety choice.

## Testing

| Command | What it does |
|---|---|
| `make test` | All unit tests, including `TestGoMatchesR`, which fails if any Go result differs from R by even one bit |
| `make race` | The same tests with Go's data race detector, which checks the parallel bootstrap workers |
| `make bench` | Benchmarks the full bootstrap study with 1 worker and with every core |
| `make profile` | CPU and memory profiles of the bootstrap study |
| `make cover` | Coverage report in the browser |

**What the tests cover:**
- **`main_test.go`:**
  - The exact Go-vs-R check.
  - Every command, including typed input.
  - Every input mistake listed above, each with its expected message.
  - Profile file output.
- **`internal/bootstrap`:**
  - Results are identical for any number of workers.
  - With no outliers, the SE matches the Central Limit Theorem within 10%.
  - The trimmed mean beats the mean when outliers are present.
  - Invalid settings are rejected, and the statistics helpers agree with R's `sd()`.
- **`internal/numfile`:** headers, blank lines, Windows line endings, and line-numbered error messages.

On GitHub, every push runs formatting, vet, tests with the race detector, the Go-vs-R comparison, the bootstrap study, and the Mac and Windows builds. The steps are in [`.github/workflows/ci.yml`](.github/workflows/ci.yml).

## Building executables

```bash
make build    # ./trimstats for this computer
make dist     # dist/trimstats-mac-arm64, dist/trimstats-mac-intel, dist/trimstats.exe
```

Without `make`, use: `go build -o trimstats .`. For Windows, from any computer: `GOOS=windows GOARCH=amd64 go build -o trimstats.exe .`

Go builds a single file with nothing else to install: no Go, no R, and no data files. The sample data and R results are built into the program with Go's `embed` feature, so `trimstats compare` works from any folder.

**Running the executables:**

- **macOS:**
  - **Terminal:** run `./trimstats-mac-arm64` (Apple Silicon) or `./trimstats-mac-intel` (Intel).
  - **Double-click:** the file opens Terminal and runs the comparison.
  - **If macOS blocks a downloaded copy:** Control-click the file and choose **Open**, or run `xattr -d com.apple.quarantine trimstats-mac-arm64`.
- **Windows:** open Command Prompt or PowerShell in the folder and run `trimstats.exe`. Double-clicking also works, but the window closes as soon as the program finishes.

## Project layout

| Path | Role |
|---|---|
| `main.go` | Entry point: picks the command, prints usage, sets exit codes |
| `compare.go` | `compare` command: the Go-vs-R table, built-in data files, asymmetric example |
| `mean.go` | `mean` command: trimmed mean of a user's file or typed numbers |
| `bootstrap.go` | `bootstrap` command: options, timing and memory log, profiles, results table |
| `internal/bootstrap/` | The parallel bootstrap study, written for reuse |
| `internal/numfile/` | Reads one-number-per-line files with clear error messages |
| `data/` | The four samples and `r-results.csv`, all created by `r/make-samples.R` |
| `r/make-samples.R` | Creates the samples and R's trimmed means |
| `r/run-bootstrap-robust.R` | The course's bootstrap script (by Thomas W. Miller), included unchanged for the timing comparison |
| `docs/` | Saved program output and timing log |
| `Makefile` | Shortcuts for testing, building, profiling, and the R steps |

## GenAI Tools

This project was built with help from Claude Code (Anthropic), an AI coding assistant running in the terminal. I used it to:

- plan the two repositories and the extra features (parallel bootstrap, fuzz testing, automatic GitHub testing),
- write the Go code, R scripts, tests, and documentation, which I reviewed,
- run the R scripts and Go programs to produce the results and timings above.

It also investigated the last-digit differences between Go and R using exact arithmetic. That investigation led to the package's improved summation method.

## License

[MIT](LICENSE)
