package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestThresholdExceeded(t *testing.T) {
	base := time.Unix(100, 0)
	history := &History{
		Values: []float64{100, 140},
		Times:  []time.Time{base, base.Add(2 * time.Second)},
	}

	rate, exceeded := thresholdExceeded(history, 19)
	if !exceeded {
		t.Fatal("expected threshold to be exceeded")
	}
	if rate != 20 {
		t.Fatalf("rate = %v, want 20", rate)
	}

	_, exceeded = thresholdExceeded(history, 20)
	if exceeded {
		t.Fatal("rate equal to threshold must not count as exceeded")
	}

	_, exceeded = thresholdExceeded(
		&History{Values: []float64{100}, Times: []time.Time{base}},
		1,
	)
	if exceeded {
		t.Fatal("insufficient history must not exceed threshold")
	}
}

func TestThresholdLoggerIsLazyAndAppends(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "threshold.log")
	logger := NewThresholdLogger(path)

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("threshold file exists before first event: err=%v", err)
	}

	base := time.Unix(100, 0).UTC()
	event := ThresholdEvent{
		Timestamp:   base,
		Selector:    Selector{Name: "requests_total", Labels: map[string]string{"name": "rx"}},
		Rate:        25.5,
		Threshold:   20,
		MetricValue: 1250,
	}

	if err := logger.Record(event); err != nil {
		t.Fatalf("first Record() error = %v", err)
	}
	event.Timestamp = base.Add(time.Second)
	event.Rate = 30
	if err := logger.Record(event); err != nil {
		t.Fatalf("second Record() error = %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("line count = %d, want 2: %q", len(lines), string(data))
	}

	for _, want := range []string{
		"metric=\"requests_total{name=\\\"rx\\\"}\"",
		"threshold=20/s",
		"value=1250.0",
	} {
		if !strings.Contains(lines[0], want) {
			t.Fatalf("threshold event missing %q: %q", want, lines[0])
		}
	}
	if !strings.Contains(lines[0], "rate=25.50/s") {
		t.Fatalf("first event rate missing: %q", lines[0])
	}
	if !strings.Contains(lines[1], "rate=30/s") {
		t.Fatalf("second event rate missing: %q", lines[1])
	}
}

func TestFormatThresholdEvent(t *testing.T) {
	event := ThresholdEvent{
		Timestamp:   time.Date(2026, 10, 3, 2, 15, 0, 123000000, time.FixedZone("IRST", 3*60*60+30*60)),
		Selector:    Selector{Name: "packets_total", Labels: map[string]string{}},
		Rate:        101.25,
		Threshold:   100,
		MetricValue: 5000,
	}

	got := formatThresholdEvent(event)
	for _, want := range []string{
		"2026-10-03T02:15:00.123+03:30",
		"metric=\"packets_total\"",
		"rate=101.25/s",
		"threshold=100/s",
		"value=5000.0",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatted event missing %q: %q", want, got)
		}
	}
}
