package benchmark

import (
	"testing"
	"time"
)

func TestDurations(t *testing.T) {
	input := []time.Duration{30 * time.Millisecond, 10 * time.Millisecond, 20 * time.Millisecond}
	stats, ok := Durations(input)
	if !ok || stats.Min != 10*time.Millisecond || stats.Median != 20*time.Millisecond || stats.Mean != 20*time.Millisecond || stats.Max != 30*time.Millisecond {
		t.Fatalf("stats = %#v ok=%v", stats, ok)
	}
	if input[0] != 30*time.Millisecond || input[1] != 10*time.Millisecond || input[2] != 20*time.Millisecond {
		t.Fatalf("input mutated: %#v", input)
	}

	stats, ok = Durations([]time.Duration{10 * time.Millisecond})
	if !ok || stats.Min != 10*time.Millisecond || stats.Median != 10*time.Millisecond || stats.Mean != 10*time.Millisecond || stats.Max != 10*time.Millisecond {
		t.Fatalf("single stats = %#v ok=%v", stats, ok)
	}

	stats, ok = Durations([]time.Duration{40 * time.Millisecond, 10 * time.Millisecond, 30 * time.Millisecond, 20 * time.Millisecond})
	if !ok || stats.Min != 10*time.Millisecond || stats.Median != 25*time.Millisecond || stats.Mean != 25*time.Millisecond || stats.Max != 40*time.Millisecond {
		t.Fatalf("even stats = %#v ok=%v", stats, ok)
	}
	if _, ok := Durations(nil); ok {
		t.Fatal("empty durations should be unavailable")
	}
}

func TestComparison(t *testing.T) {
	speedup, reduction, ok := Comparison(200*time.Millisecond, 50*time.Millisecond)
	if !ok || speedup != 4 || reduction != 75 {
		t.Fatalf("comparison = %v %v %v", speedup, reduction, ok)
	}
	speedup, reduction, ok = Comparison(50*time.Millisecond, 100*time.Millisecond)
	if !ok || speedup != 0.5 || reduction != -100 {
		t.Fatalf("slower comparison = %v %v %v", speedup, reduction, ok)
	}
	if _, _, ok := Comparison(0, 50*time.Millisecond); ok {
		t.Fatal("zero browser baseline should be unavailable")
	}
	if _, _, ok := Comparison(200*time.Millisecond, 0); ok {
		t.Fatal("zero http median should be unavailable")
	}
}
