package numfile

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestReadFloats(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []float64
	}{
		{"with header", "value\n1.5\n-2\n3e2\n", []float64{1.5, -2, 300}},
		{"without header", "1.5\n-2\n", []float64{1.5, -2}},
		{"blank lines and spaces", "value\n\n  4.25  \n\n5\n", []float64{4.25, 5}},
		{"Windows line endings", "value\r\n1\r\n2\r\n", []float64{1, 2}},
		{"no final newline", "7\n8", []float64{7, 8}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadFloats(strings.NewReader(tt.input))
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadInts(t *testing.T) {
	got, err := ReadInts(strings.NewReader("value\n31\n-47\n0\n"))
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{31, -47, 0}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestReadErrors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantText string
	}{
		{"bad number", "value\n1\n12..5\n", `line 3: "12..5" is not a valid number`},
		{"header after line 1", "1\nvalue\n", `line 2: "value" is not a valid number`},
		{"bad first line that is not a word", "#1\n2\n", `line 1: "#1" is not a valid number`},
		{"decimal in integer file", "value\n2.5\n", `line 2: "2.5" is not a valid number`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ReadInts(strings.NewReader(tt.input))
			if err == nil || err.Error() != tt.wantText {
				t.Errorf("got error %v, want %q", err, tt.wantText)
			}
		})
	}
}

func TestNoValues(t *testing.T) {
	for _, input := range []string{"", "value\n", "\n\n"} {
		_, err := ReadFloats(strings.NewReader(input))
		if !errors.Is(err, ErrNoValues) {
			t.Errorf("input %q: got %v, want ErrNoValues", input, err)
		}
	}
}

func TestReadFloatsFileMissing(t *testing.T) {
	if _, err := ReadFloatsFile("does-not-exist.csv"); err == nil {
		t.Error("expected an error for a missing file")
	}
}
