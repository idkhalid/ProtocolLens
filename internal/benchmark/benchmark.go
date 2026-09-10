package benchmark

import (
	"sort"
	"time"
)

type Stats struct {
	Min    time.Duration
	Median time.Duration
	Mean   time.Duration
	Max    time.Duration
}

func Durations(values []time.Duration) (Stats, bool) {
	if len(values) == 0 {
		return Stats{}, false
	}
	sorted := append([]time.Duration(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	var total time.Duration
	for _, value := range sorted {
		total += value
	}
	mid := len(sorted) / 2
	median := sorted[mid]
	if len(sorted)%2 == 0 {
		median = (sorted[mid-1] + sorted[mid]) / 2
	}
	return Stats{Min: sorted[0], Median: median, Mean: total / time.Duration(len(sorted)), Max: sorted[len(sorted)-1]}, true
}

func Comparison(browser, httpMedian time.Duration) (speedup, reduction float64, ok bool) {
	if browser <= 0 || httpMedian <= 0 {
		return 0, 0, false
	}
	speedup = float64(browser) / float64(httpMedian)
	reduction = (1 - float64(httpMedian)/float64(browser)) * 100
	return speedup, reduction, true
}
