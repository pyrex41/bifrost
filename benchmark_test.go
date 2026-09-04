package main

import "testing"

func TestGoBenchmarkParser(t *testing.T) {
	out := "BenchmarkVMSumMid-10\t  123\t 4567.5 ns/op\t 0 B/op\t 0 allocs/op\n"
	m := goBenchLine.FindStringSubmatch(out)
	if len(m) != 2 || m[1] != "4567.5" {
		t.Fatalf("parsed %#v", m)
	}
}

func TestMedian(t *testing.T) {
	if got := median([]float64{9, 1, 5}); got != 5 {
		t.Fatalf("odd median = %v", got)
	}
	if got := median([]float64{9, 1, 5, 3}); got != 4 {
		t.Fatalf("even median = %v", got)
	}
}
