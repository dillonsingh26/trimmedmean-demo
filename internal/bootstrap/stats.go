package bootstrap

import "math"

// mean returns the arithmetic average. Callers pass non-empty slices.
func mean(values []float64) float64 {
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// medianOfSorted returns the middle value of an already sorted slice, or
// the average of the two middle values when the length is even.
func medianOfSorted(sorted []float64) float64 {
	middle := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[middle]
	}
	return (sorted[middle-1] + sorted[middle]) / 2
}

// stdDev returns the sample standard deviation (dividing by n-1), the same
// as R's sd(). Callers pass at least two values.
func stdDev(values []float64) float64 {
	average := mean(values)
	var squares float64
	for _, v := range values {
		squares += (v - average) * (v - average)
	}
	return math.Sqrt(squares / float64(len(values)-1))
}
