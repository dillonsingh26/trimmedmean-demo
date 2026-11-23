// Package numfile reads numbers stored one per line, as in a one-column
// CSV file. An optional text header on the first line is skipped, and
// blank lines are ignored.
package numfile

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// ErrNoValues means the input held no numbers.
var ErrNoValues = errors.New("no numbers found")

// ReadFloats reads decimal numbers, one per line.
func ReadFloats(r io.Reader) ([]float64, error) {
	return readValues(r, func(text string) (float64, error) {
		return strconv.ParseFloat(text, 64)
	})
}

// ReadInts reads whole numbers, one per line.
func ReadInts(r io.Reader) ([]int, error) {
	return readValues(r, strconv.Atoi)
}

// ReadFloatsFile opens a file and reads decimal numbers from it. Errors
// start with the file name.
func ReadFloatsFile(path string) ([]float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	values, err := ReadFloats(file)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return values, nil
}

// readValues does the line-by-line work for both number types, skipping a
// first line that starts with a letter as a header. Any other line that
// does not parse is an error naming the line.
func readValues[T any](r io.Reader, parse func(string) (T, error)) ([]T, error) {
	var values []T
	scanner := bufio.NewScanner(r)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}

		value, err := parse(text)
		if err == nil {
			values = append(values, value)
			continue
		}
		if lineNumber == 1 && isHeader(text) {
			continue
		}
		return nil, fmt.Errorf("line %d: %q is not a valid number", lineNumber, text)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(values) == 0 {
		return nil, ErrNoValues
	}
	return values, nil
}

// isHeader reports whether text looks like a column name such as "value",
// rather than a mistyped number such as "12..5". "NaN" and "Inf" parse as
// numbers, so they never reach this check.
func isHeader(text string) bool {
	first := text[0]
	return ('a' <= first && first <= 'z') || ('A' <= first && first <= 'Z')
}
