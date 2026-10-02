package main

import (
	"bytes"
	"math"
	"strings"
	"testing"
	"time"
)

func TestBuildSparkline(t *testing.T) {
	if got := buildSparkline([]float64{0, 1}, 120); got != "▁█" {
		t.Fatalf("sparkline = %q, want %q", got, "▁█")
	}
	if got := buildSparkline([]float64{1, 2, 3}, 2); got != "▁█" {
		t.Fatalf("truncated sparkline = %q, want %q", got, "▁█")
	}
	if got := buildSparkline([]float64{math.NaN(), math.Inf(1), math.Inf(-1)}, 120); got != "(no finite data)" {
		t.Fatalf("non-finite sparkline = %q", got)
	}
	if got := buildSparkline(nil, 120); got != "(no data)" {
		t.Fatalf("empty sparkline = %q", got)
	}
}

func TestBuildFramePreservesLegacyLayout(t *testing.T) {
	base := time.Unix(100, 0)
	selector := Selector{Name: "submit_packets", Labels: map[string]string{"name": "receive"}}
	series := MetricSeries{Name: "submit_packets", Type: "counter", Value: 1250}
	history := &History{
		Values: []float64{1000, 1250},
		Times:  []time.Time{base, base.Add(time.Second)},
	}

	got := buildFrame(selector, series, history, true)
	want := "promwatch\n" +
		"metric : submit_packets{name=\"receive\"}\n" +
		"type   : counter\n" +
		"value  : 1250.0\n" +
		"rate   : 250/s\n" +
		"\n" +
		"▁█\n\n\n" +
		"samples: 2    interval: 1s\n"

	if got != want {
		t.Fatalf("frame mismatch\n--- got ---\n%s--- want ---\n%s", got, want)
	}
}

func TestTerminalRendererWritesLegacyClearAndFrame(t *testing.T) {
	var buf bytes.Buffer
	renderer := NewTerminalRenderer(&buf)
	selector := Selector{Name: "gauge", Labels: map[string]string{}}
	series := MetricSeries{Name: "gauge", Type: "gauge", Value: 3}
	history := &History{Values: []float64{3}, Times: []time.Time{time.Unix(0, 0)}}

	if err := renderer.Render(selector, series, history, false); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if !strings.HasPrefix(buf.String(), clearScreenSequence+"promwatch\n") {
		t.Fatalf("rendered output does not preserve legacy clear sequence: %q", buf.String())
	}
}
