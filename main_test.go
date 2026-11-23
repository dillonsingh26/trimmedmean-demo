package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runCommand runs trimstats with the given arguments and typed input,
// returning what it printed and any error.
func runCommand(t *testing.T, stdin string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	var out, errOut bytes.Buffer
	err = run(args, strings.NewReader(stdin), &out, &errOut)
	return out.String(), errOut.String(), err
}

// TestGoMatchesR is the assignment's main check: every trimmed mean R
// computed in r/make-samples.R must be reproduced exactly by Go.
func TestGoMatchesR(t *testing.T) {
	comparisons, err := compareWithR()
	if err != nil {
		t.Fatal(err)
	}
	if len(comparisons) != 16 {
		t.Fatalf("got %d comparisons, want 16 (4 samples x 4 trims)", len(comparisons))
	}
	for _, c := range comparisons {
		if c.goValue != c.rValue {
			t.Errorf("%s trim %.2f: Go %.17g, R %.17g", c.sample, c.trim, c.goValue, c.rValue)
		}
	}
}

func TestNoArgumentsRunsCompare(t *testing.T) {
	stdout, _, err := runCommand(t, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"16 of 16 Go results are identical", "Asymmetric trimming", "trimstats help"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output is missing %q", want)
		}
	}
}

func TestMeanCommand(t *testing.T) {
	path := writeTempFile(t, "value\n1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n100\n")

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"symmetric", []string{"mean", "-trim", "0.1", path}, "Trimmed mean:  6\n"},
		{"asymmetric", []string{"mean", "-trim", "0,0.1", path}, "Trimmed mean:  5.5\n"},
		{"default trim of 5%", []string{"mean", path}, "Trimming:      5% from each end"},
		{"describes asymmetric trims", []string{"mean", "-trim", "0.07,0.1", path}, "7% from the low end, 10% from the high end"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, _, err := runCommand(t, "", tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(stdout, tt.want) {
				t.Errorf("output %q is missing %q", stdout, tt.want)
			}
		})
	}
}

func TestMeanFromStandardInput(t *testing.T) {
	stdout, _, err := runCommand(t, "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n100\n", "mean", "-trim", "0.1", "-")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "Trimmed mean:  6\n") {
		t.Errorf("unexpected output %q", stdout)
	}
}

// TestMeanRejectsBadInput checks that each kind of user mistake gets a
// specific, readable message.
func TestMeanRejectsBadInput(t *testing.T) {
	goodFile := writeTempFile(t, "1\n2\n3\n")
	badFile := writeTempFile(t, "value\n1\ntwo\n3\n")
	emptyFile := writeTempFile(t, "value\n")

	tests := []struct {
		name     string
		args     []string
		wantText string
	}{
		{"no file", []string{"mean"}, "mean needs exactly one FILE"},
		{"two files", []string{"mean", goodFile, goodFile}, "mean needs exactly one FILE"},
		{"missing file", []string{"mean", "no-such-file.csv"}, "no such file"},
		{"word in data", []string{"mean", badFile}, `line 3: "two" is not a valid number`},
		{"no numbers", []string{"mean", emptyFile}, "no numbers found"},
		{"trim not a number", []string{"mean", "-trim", "abc", goodFile}, `-trim: "abc" is not a valid number`},
		{"trim too large", []string{"mean", "-trim", "0.5", goodFile}, "invalid trimming proportion"},
		{"three trims", []string{"mean", "-trim", "0.1,0.1,0.1", goodFile}, "at most two trimming proportions"},
		{"empty typed input", []string{"mean", "-"}, "standard input: no numbers found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runCommand(t, "", tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.wantText) {
				t.Errorf("got error %v, want one containing %q", err, tt.wantText)
			}
		})
	}
}

func TestBootstrapCommand(t *testing.T) {
	stdout, stderr, err := runCommand(t, "", "bootstrap", "-iterations", "5", "-resamples", "5", "-sizes", "10,20", "-workers", "2")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "5 repetitions x 5 bootstrap resamples") {
		t.Errorf("results are missing the settings line:\n%s", stdout)
	}
	if !strings.Contains(stderr, "bootstrap study finished") {
		t.Errorf("log is missing the finish line:\n%s", stderr)
	}
}

func TestBootstrapRejectsBadInput(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantText string
	}{
		{"bad size", []string{"bootstrap", "-sizes", "10,x"}, `-sizes: "x" is not a valid number`},
		{"no workers", []string{"bootstrap", "-workers", "0"}, "workers must be at least 1"},
		{"bad trim", []string{"bootstrap", "-trim", "0.6"}, "invalid trimming proportion"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runCommand(t, "", tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.wantText) {
				t.Errorf("got error %v, want one containing %q", err, tt.wantText)
			}
		})
	}
}

func TestBootstrapProfiles(t *testing.T) {
	dir := t.TempDir()
	cpu, mem := filepath.Join(dir, "cpu.prof"), filepath.Join(dir, "mem.prof")
	_, _, err := runCommand(t, "", "bootstrap", "-iterations", "3", "-resamples", "3", "-sizes", "10",
		"-cpuprofile", cpu, "-memprofile", mem)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{cpu, mem} {
		if info, err := os.Stat(path); err != nil || info.Size() == 0 {
			t.Errorf("profile %s was not written: %v", path, err)
		}
	}
}

func TestUnknownFlagIsReported(t *testing.T) {
	_, stderr, err := runCommand(t, "", "mean", "-trimm", "0.1", "data.csv")
	if !errors.Is(err, errBadFlags) {
		t.Errorf("got %v, want errBadFlags", err)
	}
	if !strings.Contains(stderr, "flag provided but not defined: -trimm") {
		t.Errorf("stderr does not explain the problem:\n%s", stderr)
	}
}

func TestHelpAndUnknownCommand(t *testing.T) {
	stdout, _, err := runCommand(t, "", "help")
	if err != nil || !strings.Contains(stdout, "Usage:") {
		t.Errorf("help: got %v, output %q", err, stdout)
	}

	_, _, err = runCommand(t, "", "average")
	if err == nil || !strings.Contains(err.Error(), `unknown command "average"`) {
		t.Errorf("unknown command: got %v", err)
	}
}

func writeTempFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "numbers.csv")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
