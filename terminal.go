package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"strings"
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

func (r *TerminalRenderer) Render(selector Selector, series MetricSeries, history *History, rate bool) error {
	size := r.currentSize()
	frame := buildFrame(selector, series, history, rate, size)

	if !r.interactive {
		// Non-TTY behavior is finalized in TR5. Preserve the legacy clear/redraw
		// behavior here while TR2 focuses on the interactive dashboard.
		_, err := io.WriteString(r.out, cursorHomeSequence+clearScreenSequence+frame)
		return err
	}

	if !r.started {
		if err := r.Start(); err != nil {
			return err
		}
	}

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

func buildFrame(
	selector Selector,
	series MetricSeries,
	history *History,
	rate bool,
	size TerminalSize,
) string {
	size = normalizeTerminalSize(size.Width, size.Height)
	width := drawableWidth(size.Width)

	lines := []string{
		"promwatch",
		fmt.Sprintf("metric : %s", selectorToString(selector)),
		fmt.Sprintf("type   : %s", series.Type),
		fmt.Sprintf("value  : %s", formatNumber(series.Value)),
	}

	if rate {
		r, ok := calculateRate(history)
		if !ok {
			lines = append(lines, "rate   : --/s")
		} else {
			lines = append(lines, fmt.Sprintf("rate   : %s/s", formatNumber(r)))
		}
	}

	lines = append(
		lines,
		"",
		buildSparkline(history.Values, width),
		"",
		"",
		fmt.Sprintf(
			"samples: %d    interval: %s",
			len(history.Values),
			historyInterval(history),
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

func drawableWidth(terminalWidth int) int {
	if terminalWidth <= 1 {
		return 1
	}

	// Avoid writing into the final column. Many terminals defer wrapping until
	// the next printable character, which can make a subsequent newline look
	// like an extra wrapped line.
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
