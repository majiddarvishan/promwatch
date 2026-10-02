package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	maxHistory = 3600
	sparkChars = "▁▂▃▄▅▆▇█"
)

type Selector struct {
	Name   string
	Labels map[string]string
}

type MetricSeries struct {
	Name   string
	Type   string
	Labels map[string]string
	Value  float64
}

type History struct {
	Values []float64
	Times  []time.Time
}

func parseSelector(input string) (Selector, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Selector{}, errors.New("metric selector is empty")
	}

	open := strings.IndexByte(input, '{')
	if open == -1 {
		return Selector{
			Name:   input,
			Labels: map[string]string{},
		}, nil
	}

	if !strings.HasSuffix(input, "}") {
		return Selector{}, errors.New("invalid selector: missing closing }")
	}

	name := strings.TrimSpace(input[:open])
	if name == "" {
		return Selector{}, errors.New("metric name is empty")
	}

	labelText := strings.TrimSpace(input[open+1 : len(input)-1])
	labels := make(map[string]string)

	if labelText == "" {
		return Selector{
			Name:   name,
			Labels: labels,
		}, nil
	}

	parts, err := splitLabels(labelText)
	if err != nil {
		return Selector{}, err
	}

	for _, part := range parts {
		part = strings.TrimSpace(part)

		eq := strings.IndexByte(part, '=')
		if eq <= 0 {
			return Selector{}, fmt.Errorf("invalid label selector: %q", part)
		}

		key := strings.TrimSpace(part[:eq])
		value := strings.TrimSpace(part[eq+1:])

		if key == "" {
			return Selector{}, fmt.Errorf("empty label name in %q", part)
		}

		if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
			return Selector{}, fmt.Errorf("label value must be quoted: %q", part)
		}

		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return Selector{}, fmt.Errorf("invalid label value %q: %w", value, err)
		}

		if _, exists := labels[key]; exists {
			return Selector{}, fmt.Errorf("duplicate label %q", key)
		}

		labels[key] = unquoted
	}

	return Selector{
		Name:   name,
		Labels: labels,
	}, nil
}

func splitLabels(input string) ([]string, error) {
	var result []string
	var current strings.Builder

	inQuotes := false
	escaped := false

	for _, r := range input {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false

		case r == '\\' && inQuotes:
			current.WriteRune(r)
			escaped = true

		case r == '"':
			current.WriteRune(r)
			inQuotes = !inQuotes

		case r == ',' && !inQuotes:
			result = append(result, strings.TrimSpace(current.String()))
			current.Reset()

		default:
			current.WriteRune(r)
		}
	}

	if inQuotes {
		return nil, errors.New("unterminated label string")
	}

	if current.Len() > 0 {
		result = append(result, strings.TrimSpace(current.String()))
	}

	return result, nil
}

func parseTypeLine(line string) (string, string, bool) {
	fields := strings.Fields(line)

	if len(fields) != 4 {
		return "", "", false
	}

	if fields[0] != "#" || fields[1] != "TYPE" {
		return "", "", false
	}

	return fields[2], fields[3], true
}

func parseMetricLine(line string) (string, map[string]string, float64, bool) {
	line = strings.TrimSpace(line)

	if line == "" || strings.HasPrefix(line, "#") {
		return "", nil, 0, false
	}

	space := strings.IndexAny(line, " \t")
	if space == -1 {
		return "", nil, 0, false
	}

	metricPart := line[:space]
	valuePart := strings.TrimSpace(line[space:])

	if valuePart == "" {
		return "", nil, 0, false
	}

	valueFields := strings.Fields(valuePart)
	if len(valueFields) == 0 {
		return "", nil, 0, false
	}

	value, err := strconv.ParseFloat(valueFields[0], 64)
	if err != nil {
		return "", nil, 0, false
	}

	name := metricPart
	labels := make(map[string]string)

	open := strings.IndexByte(metricPart, '{')
	if open != -1 {
		if !strings.HasSuffix(metricPart, "}") {
			return "", nil, 0, false
		}

		name = metricPart[:open]
		labelText := metricPart[open+1 : len(metricPart)-1]

		if labelText != "" {
			parts, err := splitLabels(labelText)
			if err != nil {
				return "", nil, 0, false
			}

			for _, part := range parts {
				eq := strings.IndexByte(part, '=')
				if eq <= 0 {
					return "", nil, 0, false
				}

				key := strings.TrimSpace(part[:eq])
				rawValue := strings.TrimSpace(part[eq+1:])

				value, err := strconv.Unquote(rawValue)
				if err != nil {
					return "", nil, 0, false
				}

				labels[key] = value
			}
		}
	}

	return name, labels, value, true
}

func matchesLabels(series map[string]string, selector map[string]string) bool {
	for key, expected := range selector {
		actual, exists := series[key]
		if !exists || actual != expected {
			return false
		}
	}

	return true
}

func fetchMetric(ctx context.Context, client *http.Client, url string, selector Selector) (MetricSeries, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return MetricSeries{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return MetricSeries{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return MetricSeries{}, fmt.Errorf("HTTP status: %s", resp.Status)
	}

	metricTypes := make(map[string]string)
	var matches []MetricSeries

	scanner := bufio.NewScanner(resp.Body)

	// Increase the scanner limit for long label sets.
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") {
			if name, metricType, ok := parseTypeLine(line); ok {
				metricTypes[name] = metricType
			}
			continue
		}

		name, labels, value, ok := parseMetricLine(line)
		if !ok {
			continue
		}

		if name != selector.Name {
			continue
		}

		if !matchesLabels(labels, selector.Labels) {
			continue
		}

		matches = append(matches, MetricSeries{
			Name:   name,
			Type:   metricTypes[name],
			Labels: labels,
			Value:  value,
		})
	}

	if err := scanner.Err(); err != nil {
		return MetricSeries{}, fmt.Errorf("reading metrics: %w", err)
	}

	switch len(matches) {
	case 0:
		return MetricSeries{}, fmt.Errorf(
			"no series matched selector %q",
			selectorToString(selector),
		)

	case 1:
		return matches[0], nil

	default:
		return MetricSeries{}, fmt.Errorf(
			"selector %q matched %d series; add more labels to select exactly one series",
			selectorToString(selector),
			len(matches),
		)
	}
}

func selectorToString(selector Selector) string {
	if len(selector.Labels) == 0 {
		return selector.Name
	}

	var parts []string

	for key, value := range selector.Labels {
		parts = append(parts, fmt.Sprintf(`%s=%q`, key, value))
	}

	return selector.Name + "{" + strings.Join(parts, ",") + "}"
}

func calculateRate(history *History) (float64, bool) {
	if len(history.Values) < 2 {
		return 0, false
	}

	n := len(history.Values)

	current := history.Values[n-1]
	previous := history.Values[n-2]

	currentTime := history.Times[n-1]
	previousTime := history.Times[n-2]

	elapsed := currentTime.Sub(previousTime).Seconds()

	if elapsed <= 0 {
		return 0, false
	}

	// Counter reset.
	if current < previous {
		return current / elapsed, true
	}

	return (current - previous) / elapsed, true
}

func appendHistory(history *History, value float64, timestamp time.Time) {
	history.Values = append(history.Values, value)
	history.Times = append(history.Times, timestamp)

	if len(history.Values) > maxHistory {
		excess := len(history.Values) - maxHistory

		history.Values = history.Values[excess:]
		history.Times = history.Times[excess:]
	}
}

func historyInterval(history *History) string {
	if len(history.Times) < 2 {
		return "--"
	}

	d := history.Times[len(history.Times)-1].
		Sub(history.Times[0])

	return d.Round(time.Second).String()
}

func formatNumber(value float64) string {
	if math.IsNaN(value) {
		return "NaN"
	}

	if math.IsInf(value, 1) {
		return "+Inf"
	}

	if math.IsInf(value, -1) {
		return "-Inf"
	}

	if math.Abs(value) >= 1000000 {
		return fmt.Sprintf("%.2f", value)
	}

	if math.Abs(value) >= 1000 {
		return fmt.Sprintf("%.1f", value)
	}

	if value == math.Trunc(value) {
		return strconv.FormatFloat(value, 'f', 0, 64)
	}

	return strconv.FormatFloat(value, 'f', 2, 64)
}

func validateRateMetric(series MetricSeries) error {
	if series.Type == "" {
		return fmt.Errorf(
			"metric %q has no # TYPE information; cannot use --rate safely",
			series.Name,
		)
	}

	if series.Type != "counter" {
		return fmt.Errorf(
			"metric %q is type %q; --rate requires a counter",
			series.Name,
			series.Type,
		)
	}

	return nil
}

func main() {
	url := flag.String(
		"url",
		"http://localhost:9999/metrics",
		"Prometheus metrics URL",
	)

	metric := flag.String(
		"metric",
		"",
		"metric selector, e.g. submit_packets{name=\"receive\",system_id=\"smpp_client_0\"}",
	)

	rate := flag.Bool(
		"rate",
		false,
		"show per-second rate for a counter",
	)

	interval := flag.Duration(
		"interval",
		time.Second,
		"poll interval",
	)

	flag.Parse()

	if *metric == "" {
		fmt.Fprintln(os.Stderr, "error: --metric is required")
		os.Exit(1)
	}

	if *interval <= 0 {
		fmt.Fprintln(os.Stderr, "error: --interval must be greater than zero")
		os.Exit(1)
	}

	selector, err := parseSelector(*metric)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	history := &History{}
	renderer := NewTerminalRenderer(os.Stdout)
	if err := renderer.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: initializing terminal: %v\n", err)
		return
	}
	defer func() {
		if err := renderer.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error: restoring terminal: %v\n", err)
		}
	}()

	fetch := func() {
		series, err := fetchMetric(ctx, client, *url, selector)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\rerror: %v\n", err)
			return
		}

		if *rate {
			if err := validateRateMetric(series); err != nil {
				fmt.Fprintf(os.Stderr, "\rerror: %v\n", err)
				return
			}
		}

		now := time.Now()

		appendHistory(history, series.Value, now)

		if err := renderer.Render(selector, series, history, *rate); err != nil {
			fmt.Fprintf(os.Stderr, "\rerror: rendering terminal: %v\n", err)
		}
	}

	fetch()

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			fetch()
		}
	}
}