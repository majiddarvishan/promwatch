package main

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func TestParseSelector(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      Selector
		wantError bool
	}{
		{
			name:  "metric only",
			input: "requests_total",
			want:  Selector{Name: "requests_total", Labels: map[string]string{}},
		},
		{
			name:  "labels",
			input: `requests_total{method="GET",path="/a,b"}`,
			want: Selector{
				Name: "requests_total",
				Labels: map[string]string{
					"method": "GET",
					"path":   "/a,b",
				},
			},
		},
		{name: "empty", input: "", wantError: true},
		{name: "missing close", input: `requests_total{method="GET"`, wantError: true},
		{name: "duplicate label", input: `requests_total{method="GET",method="POST"}`, wantError: true},
		{name: "unquoted label", input: `requests_total{method=GET}`, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSelector(tt.input)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseSelector() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseSelector() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseMetricLine(t *testing.T) {
	name, labels, value, ok := parseMetricLine(`requests_total{method="GET",path="/a,b"} 12.5 123456`)
	if !ok {
		t.Fatal("expected metric line to parse")
	}
	if name != "requests_total" {
		t.Fatalf("name = %q", name)
	}
	if !reflect.DeepEqual(labels, map[string]string{"method": "GET", "path": "/a,b"}) {
		t.Fatalf("labels = %#v", labels)
	}
	if value != 12.5 {
		t.Fatalf("value = %v", value)
	}

	if _, _, _, ok := parseMetricLine("# HELP requests_total test"); ok {
		t.Fatal("comment line must not parse as a metric")
	}
	if _, _, _, ok := parseMetricLine("requests_total not-a-number"); ok {
		t.Fatal("invalid numeric value must not parse")
	}
}

func TestMatchesLabels(t *testing.T) {
	series := map[string]string{"method": "GET", "code": "200"}
	if !matchesLabels(series, map[string]string{"method": "GET"}) {
		t.Fatal("expected subset selector to match")
	}
	if matchesLabels(series, map[string]string{"method": "POST"}) {
		t.Fatal("unexpected mismatch to match")
	}
	if matchesLabels(series, map[string]string{"instance": "one"}) {
		t.Fatal("missing label must not match")
	}
}

func TestCalculateRate(t *testing.T) {
	base := time.Unix(100, 0)

	tests := []struct {
		name    string
		history History
		want    float64
		ok      bool
	}{
		{
			name:    "normal",
			history: History{Values: []float64{100, 140}, Times: []time.Time{base, base.Add(2 * time.Second)}},
			want:    20,
			ok:      true,
		},
		{
			name:    "counter reset",
			history: History{Values: []float64{100, 10}, Times: []time.Time{base, base.Add(2 * time.Second)}},
			want:    5,
			ok:      true,
		},
		{
			name:    "insufficient samples",
			history: History{Values: []float64{100}, Times: []time.Time{base}},
			ok:      false,
		},
		{
			name:    "non positive elapsed",
			history: History{Values: []float64{100, 110}, Times: []time.Time{base, base}},
			ok:      false,
		},
		{
			name:    "mismatched history lengths",
			history: History{Values: []float64{100, 110, 120}, Times: []time.Time{base, base.Add(time.Second)}},
			ok:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := calculateRate(&tt.history)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if tt.ok && math.Abs(got-tt.want) > 1e-9 {
				t.Fatalf("rate = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppendHistoryCapsSamples(t *testing.T) {
	h := &History{}
	base := time.Unix(0, 0)
	for i := 0; i < maxHistory+2; i++ {
		appendHistory(h, float64(i), base.Add(time.Duration(i)*time.Second))
	}

	if len(h.Values) != maxHistory || len(h.Times) != maxHistory {
		t.Fatalf("history lengths = %d/%d, want %d", len(h.Values), len(h.Times), maxHistory)
	}
	if h.Values[0] != 2 {
		t.Fatalf("oldest retained value = %v, want 2", h.Values[0])
	}
}
