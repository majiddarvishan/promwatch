package main

import (
	"fmt"
	"io"
	"math"
	"strings"
)

const (
	clearScreenSequence   = "\033[H\033[2J"
	defaultSparklineWidth = 120
)

type TerminalRenderer struct {
	out io.Writer
}

func NewTerminalRenderer(out io.Writer) *TerminalRenderer {
	return &TerminalRenderer{out: out}
}

func (r *TerminalRenderer) Render(selector Selector, series MetricSeries, history *History, rate bool) error {
	frame := buildFrame(selector, series, history, rate)
	_, err := io.WriteString(r.out, clearScreenSequence+frame)
	return err
}

func buildFrame(selector Selector, series MetricSeries, history *History, rate bool) string {
	var output strings.Builder

	fmt.Fprintln(&output, "promwatch")
	fmt.Fprintf(&output, "metric : %s\n", selectorToString(selector))
	fmt.Fprintf(&output, "type   : %s\n", series.Type)
	fmt.Fprintf(&output, "value  : %s\n", formatNumber(series.Value))

	if rate {
		r, ok := calculateRate(history)
		if !ok {
			fmt.Fprintln(&output, "rate   : --/s")
		} else {
			fmt.Fprintf(&output, "rate   : %s/s\n", formatNumber(r))
		}
	}

	output.WriteByte('\n')
	output.WriteString(buildSparkline(history.Values, defaultSparklineWidth))
	output.WriteString("\n\n\n")
	fmt.Fprintf(&output, "samples: %d    interval: %s\n", len(history.Values), historyInterval(history))

	return output.String()
}

func buildSparkline(values []float64, maxWidth int) string {
	if len(values) == 0 {
		return "(no data)"
	}

	if maxWidth <= 0 {
		return ""
	}

	start := 0
	if len(values) > maxWidth {
		start = len(values) - maxWidth
	}

	values = values[start:]

	minValue := math.Inf(1)
	maxValue := math.Inf(-1)

	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}

		if value < minValue {
			minValue = value
		}

		if value > maxValue {
			maxValue = value
		}
	}

	if math.IsInf(minValue, 1) || math.IsInf(maxValue, -1) {
		return "(no finite data)"
	}

	levels := []rune(sparkChars)
	var output strings.Builder

	for _, value := range values {
		if math.IsNaN(value) {
			output.WriteByte('?')
			continue
		}

		if math.IsInf(value, 1) {
			output.WriteByte('+')
			continue
		}

		if math.IsInf(value, -1) {
			output.WriteByte('-')
			continue
		}

		level := 0
		if maxValue > minValue {
			ratio := (value - minValue) / (maxValue - minValue)
			level = int(math.Round(ratio * float64(len(levels)-1)))
		}

		if level < 0 {
			level = 0
		}
		if level >= len(levels) {
			level = len(levels) - 1
		}

		output.WriteRune(levels[level])
	}

	return output.String()
}
