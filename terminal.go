package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/term"
)

const (
	enterAlternateScreenSequence = "\x1b[?1049h"
	leaveAlternateScreenSequence = "\x1b[?1049l"
	hideCursorSequence           = "\x1b[?25l"
	showCursorSequence           = "\x1b[?25h"
	cursorHomeSequence           = "\x1b[H"
	clearScreenSequence          = "\x1b[2J"
	clearToEndSequence           = "\x1b[J"

	fallbackTerminalWidth  = 80
	fallbackTerminalHeight = 24
)

type TerminalSize struct {
	Width  int
	Height int
}

type UIState struct {
	Selector    Selector
	Series      MetricSeries
	HasSeries   bool
	History     *History
	Rate        bool
	Err         error
	CheckedAt   time.Time
	LastSuccess time.Time
}

type terminalSizeProvider func() (int, int, error)

type TerminalRenderer struct {
	out         io.Writer
	interactive bool
	size        terminalSizeProvider
	started     bool
}

func NewTerminalRenderer(out *os.File) *TerminalRenderer {
	fd := int(out.Fd())

	return newTerminalRenderer(
		out,
		term.IsTerminal(fd),
		func() (int, int, error) {
			return term.GetSize(fd)
		},
	)
}

func newTerminalRenderer(out io.Writer, interactive bool, size terminalSizeProvider) *TerminalRenderer {
	return &TerminalRenderer{
		out:         out,
		interactive: interactive,
		size:        size,
	}
}

func (r *TerminalRenderer) Start() error {
	if !r.interactive || r.started {
		return nil
	}

	_, err := io.WriteString(
		r.out,
		enterAlternateScreenSequence+
			hideCursorSequence+
			clearScreenSequence+
			cursorHomeSequence,
	)
	if err != nil {
		return err
	}

	r.started = true
	return nil
}

func (r *TerminalRenderer) Close() error {
	if !r.interactive || !r.started {
		return nil
	}

	_, err := io.WriteString(
		r.out,
		showCursorSequence+leaveAlternateScreenSequence,
	)
	if err != nil {
		return err
	}

	r.started = false
	return nil
}

func (r *TerminalRenderer) Render(state UIState) error {
	if !r.interactive {
		_, err := io.WriteString(r.out, buildPlainLine(state)+"\n")
		return err
	}

	if !r.started {
		if err := r.Start(); err != nil {
			return err
		}
	}

	frame := buildFrame(state, r.currentSize())
	_, err := io.WriteString(
		r.out,
		cursorHomeSequence+frame+clearToEndSequence,
	)
	return err
}

func (r *TerminalRenderer) currentSize() TerminalSize {
	if r.size == nil {
		return TerminalSize{
			Width:  fallbackTerminalWidth,
			Height: fallbackTerminalHeight,
		}
	}

	width, height, err := r.size()
	if err != nil {
		return TerminalSize{
			Width:  fallbackTerminalWidth,
			Height: fallbackTerminalHeight,
		}
	}

	return normalizeTerminalSize(width, height)
}

func normalizeTerminalSize(width, height int) TerminalSize {
	if width <= 0 {
		width = fallbackTerminalWidth
	}
	if height <= 0 {
		height = fallbackTerminalHeight
	}

	return TerminalSize{
		Width:  width,
		Height: height,
	}
}

func buildFrame(state UIState, size TerminalSize) string {
	size = normalizeTerminalSize(size.Width, size.Height)
	width := drawableWidth(size.Width)

	status := "OK"
	if state.Err != nil {
		status = "ERROR"
	}

	lines := []string{
		"promwatch",
		fmt.Sprintf("metric : %s", selectorToString(state.Selector)),
		fmt.Sprintf("status : %s", status),
	}

	if state.Err != nil {
		lines = append(lines, fmt.Sprintf("error  : %s", state.Err))

		if state.HasSeries {
			lines = append(
				lines,
				fmt.Sprintf("type   : %s", state.Series.Type),
				fmt.Sprintf("last value: %s", formatNumber(state.Series.Value)),
			)
		}

		if !state.LastSuccess.IsZero() {
			lines = append(lines, fmt.Sprintf("last ok: %s", state.LastSuccess.Format(time.RFC3339)))
		}
	} else if state.HasSeries {
		lines = append(
			lines,
			fmt.Sprintf("type   : %s", state.Series.Type),
			fmt.Sprintf("value  : %s", formatNumber(state.Series.Value)),
		)

		if state.Rate {
			r, ok := calculateRate(state.History)
			if !ok {
				lines = append(lines, "rate   : --/s")
			} else {
				lines = append(lines, fmt.Sprintf("rate   : %s/s", formatNumber(r)))
			}
		}

		if !state.LastSuccess.IsZero() {
			lines = append(lines, fmt.Sprintf("updated: %s", state.LastSuccess.Format(time.RFC3339)))
		}
	}

	lines = append(
		lines,
		"",
		buildSparkline(historyValues(state.History), width),
		"",
		fmt.Sprintf(
			"samples: %d    interval: %s",
			historySampleCount(state.History),
			historyInterval(state.History),
		),
	)

	for i := range lines {
		lines[i] = truncateLine(lines[i], width)
	}

	if len(lines) > size.Height {
		lines = lines[:size.Height]
	}

	return strings.Join(lines, "\n")
}

func buildPlainLine(state UIState) string {
	parts := make([]string, 0, 10)

	if !state.CheckedAt.IsZero() {
		parts = append(parts, state.CheckedAt.Format(time.RFC3339))
	}

	status := "ok"
	if state.Err != nil {
		status = "error"
	}

	parts = append(
		parts,
		"status="+status,
		fmt.Sprintf("metric=%q", selectorToString(state.Selector)),
	)

	if state.Err != nil {
		parts = append(parts, fmt.Sprintf("error=%q", state.Err.Error()))

		if state.HasSeries {
			parts = append(parts, "last_value="+formatNumber(state.Series.Value))
		}
		if !state.LastSuccess.IsZero() {
			parts = append(parts, "last_success="+state.LastSuccess.Format(time.RFC3339))
		}
	} else if state.HasSeries {
		parts = append(
			parts,
			"type="+state.Series.Type,
			"value="+formatNumber(state.Series.Value),
		)

		if state.Rate {
			if r, ok := calculateRate(state.History); ok {
				parts = append(parts, "rate="+formatNumber(r)+"/s")
			} else {
				parts = append(parts, "rate=--/s")
			}
		}
	}

	parts = append(parts, fmt.Sprintf("samples=%d", historySampleCount(state.History)))
	return strings.Join(parts, " ")
}

func historyValues(history *History) []float64 {
	if history == nil {
		return nil
	}
	return history.Values
}

func historySampleCount(history *History) int {
	if history == nil {
		return 0
	}
	return len(history.Values)
}

func drawableWidth(terminalWidth int) int {
	if terminalWidth <= 1 {
		return 1
	}

	return terminalWidth - 1
}

func truncateLine(line string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if utf8.RuneCountInString(line) <= maxWidth {
		return line
	}

	runes := []rune(line)
	return string(runes[:maxWidth])
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
