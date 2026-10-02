package main

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
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

func testState() UIState {
	base := time.Unix(100, 0).UTC()
	return UIState{
		Selector:  Selector{Name: "submit_packets", Labels: map[string]string{"name": "receive"}},
		Series:    MetricSeries{Name: "submit_packets", Type: "counter", Value: 1250},
		HasSeries: true,
		History: &History{
			Values: []float64{1000, 1250},
			Times:  []time.Time{base, base.Add(time.Second)},
		},
		Rate:        true,
		CheckedAt:   base.Add(time.Second),
		LastSuccess: base.Add(time.Second),
	}
}

func TestBuildFrameShowsHealthyStatus(t *testing.T) {
	got := buildFrame(testState(), TerminalSize{Width: 200, Height: 24})

	for _, want := range []string{
		"status : OK",
		"type   : counter",
		"value  : 1250.0",
		"rate   : 250/s",
		"samples: 2    interval: 1s",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("frame missing %q:\n%s", want, got)
		}
	}
}

func TestBuildFrameShowsErrorAndLastGoodData(t *testing.T) {
	state := testState()
	state.Err = errors.New("connection refused")
	state.CheckedAt = state.CheckedAt.Add(time.Second)

	got := buildFrame(state, TerminalSize{Width: 200, Height: 24})

	for _, want := range []string{
		"status : ERROR",
		"error  : connection refused",
		"last value: 1250.0",
		"last ok:",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("error frame missing %q:\n%s", want, got)
		}
	}
}

func TestBuildFrameBoundsWidthAndHeight(t *testing.T) {
	state := testState()
	state.Selector = Selector{
		Name: "very_long_metric_name",
		Labels: map[string]string{
			"instance": "a-very-long-instance-name",
		},
	}

	const (
		terminalWidth  = 10
		terminalHeight = 7
	)

	frame := buildFrame(
		state,
		TerminalSize{Width: terminalWidth, Height: terminalHeight},
	)
	lines := strings.Split(frame, "\n")

	if len(lines) > terminalHeight {
		t.Fatalf("frame has %d lines, want <= %d", len(lines), terminalHeight)
	}

	maxLineWidth := drawableWidth(terminalWidth)
	for _, line := range lines {
		if got := utf8.RuneCountInString(line); got > maxLineWidth {
			t.Fatalf("line %q has width %d, want <= %d", line, got, maxLineWidth)
		}
	}
}

func TestRendererInteractiveLifecycle(t *testing.T) {
	var buf bytes.Buffer
	renderer := newTerminalRenderer(
		&buf,
		true,
		func() (int, int, error) { return 80, 24, nil },
	)

	if err := renderer.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	wantStart := enterAlternateScreenSequence +
		hideCursorSequence +
		clearScreenSequence +
		cursorHomeSequence
	if got := buf.String(); got != wantStart {
		t.Fatalf("start output = %q, want %q", got, wantStart)
	}

	buf.Reset()
	if err := renderer.Render(testState()); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !strings.HasPrefix(buf.String(), cursorHomeSequence+"promwatch\n") {
		t.Fatalf("render output does not begin at cursor home: %q", buf.String())
	}
	if !strings.HasSuffix(buf.String(), clearToEndSequence) {
		t.Fatalf("render output does not clear stale remainder: %q", buf.String())
	}
	if strings.Contains(buf.String(), clearScreenSequence) {
		t.Fatalf("render unexpectedly performs full-screen clear: %q", buf.String())
	}

	buf.Reset()
	if err := renderer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	wantClose := showCursorSequence + leaveAlternateScreenSequence
	if got := buf.String(); got != wantClose {
		t.Fatalf("close output = %q, want %q", got, wantClose)
	}
}

func TestInteractiveErrorsRedrawInPlace(t *testing.T) {
	var buf bytes.Buffer
	renderer := newTerminalRenderer(
		&buf,
		true,
		func() (int, int, error) { return 80, 24, nil },
	)
	renderer.started = true

	state := testState()
	state.Err = errors.New("endpoint unavailable")

	if err := renderer.Render(state); err != nil {
		t.Fatalf("first Render() error = %v", err)
	}
	if err := renderer.Render(state); err != nil {
		t.Fatalf("second Render() error = %v", err)
	}

	got := buf.String()
	if strings.Count(got, cursorHomeSequence) != 2 {
		t.Fatalf("cursor-home count = %d, want 2", strings.Count(got, cursorHomeSequence))
	}
	if strings.Contains(got, enterAlternateScreenSequence) || strings.Contains(got, clearScreenSequence) {
		t.Fatalf("error redraw unexpectedly re-enters/clears screen: %q", got)
	}
}

func TestRendererReadsSizeOnEveryRender(t *testing.T) {
	var buf bytes.Buffer
	width := 20
	calls := 0
	renderer := newTerminalRenderer(
		&buf,
		true,
		func() (int, int, error) {
			calls++
			return width, 24, nil
		},
	)
	renderer.started = true

	state := testState()
	if err := renderer.Render(state); err != nil {
		t.Fatalf("first Render() error = %v", err)
	}
	width = 10
	if err := renderer.Render(state); err != nil {
		t.Fatalf("second Render() error = %v", err)
	}

	if calls != 2 {
		t.Fatalf("size provider calls = %d, want 2", calls)
	}
}

func TestRendererFallsBackWhenSizeUnavailable(t *testing.T) {
	renderer := newTerminalRenderer(
		&bytes.Buffer{},
		true,
		func() (int, int, error) { return 0, 0, errors.New("size unavailable") },
	)

	got := renderer.currentSize()
	want := TerminalSize{
		Width:  fallbackTerminalWidth,
		Height: fallbackTerminalHeight,
	}
	if got != want {
		t.Fatalf("currentSize() = %#v, want %#v", got, want)
	}
}

func TestNonInteractiveOutputHasNoANSI(t *testing.T) {
	var buf bytes.Buffer
	renderer := newTerminalRenderer(&buf, false, nil)

	if err := renderer.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := renderer.Render(testState()); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if err := renderer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	got := buf.String()
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("non-interactive output contains ANSI: %q", got)
	}
	for _, want := range []string{
		"status=ok",
		"metric=\"submit_packets{name=\\\"receive\\\"}\"",
		"type=counter",
		"value=1250.0",
		"rate=250/s",
		"samples=2",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("non-interactive output missing %q: %q", want, got)
		}
	}
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("non-interactive output must be line-oriented: %q", got)
	}
}

func TestNonInteractiveErrorIsPlainLine(t *testing.T) {
	state := testState()
	state.Err = errors.New("connection refused")

	got := buildPlainLine(state)
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("plain error contains ANSI: %q", got)
	}
	for _, want := range []string{
		"status=error",
		"error=\"connection refused\"",
		"last_value=1250.0",
		"last_success=",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("plain error missing %q: %q", want, got)
		}
	}
}

func TestNormalizeTerminalSizeUsesPerDimensionFallback(t *testing.T) {
	got := normalizeTerminalSize(0, 12)
	if got.Width != fallbackTerminalWidth || got.Height != 12 {
		t.Fatalf("normalizeTerminalSize() = %#v", got)
	}

	got = normalizeTerminalSize(50, 0)
	if got.Width != 50 || got.Height != fallbackTerminalHeight {
		t.Fatalf("normalizeTerminalSize() = %#v", got)
	}
}

type failOnceWriter struct {
	buf    bytes.Buffer
	failed bool
}

func (w *failOnceWriter) Write(p []byte) (int, error) {
	if !w.failed {
		w.failed = true
		n := len(p) / 2
		if n == 0 {
			n = 1
		}
		_, _ = w.buf.Write(p[:n])
		return n, errors.New("simulated write failure")
	}
	return w.buf.Write(p)
}

func TestRendererCloseIsIdempotent(t *testing.T) {
	var buf bytes.Buffer
	renderer := newTerminalRenderer(
		&buf,
		true,
		func() (int, int, error) { return 80, 24, nil },
	)

	if err := renderer.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	buf.Reset()

	if err := renderer.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	first := buf.String()

	if err := renderer.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if got := buf.String(); got != first {
		t.Fatalf("second Close() wrote additional output: %q", got)
	}
}

func TestRendererStartFailureAttemptsRestore(t *testing.T) {
	writer := &failOnceWriter{}
	renderer := newTerminalRenderer(
		writer,
		true,
		func() (int, int, error) { return 80, 24, nil },
	)

	if err := renderer.Start(); err == nil {
		t.Fatal("Start() error = nil, want simulated write failure")
	}
	if renderer.started {
		t.Fatal("renderer remains started after failed Start()")
	}
	if !strings.Contains(
		writer.buf.String(),
		showCursorSequence+leaveAlternateScreenSequence,
	) {
		t.Fatalf("failed Start() did not attempt terminal restore: %q", writer.buf.String())
	}
}

func TestBuildFrameHandlesTinyTerminal(t *testing.T) {
	frame := buildFrame(testState(), TerminalSize{Width: 1, Height: 1})

	if !utf8.ValidString(frame) {
		t.Fatalf("tiny frame is invalid UTF-8: %q", frame)
	}
	if utf8.RuneCountInString(frame) > 1 {
		t.Fatalf("tiny frame width = %d, want <= 1", utf8.RuneCountInString(frame))
	}
	if strings.Count(frame, "\n") != 0 {
		t.Fatalf("tiny frame exceeds one row: %q", frame)
	}
}

func TestUnicodeSparklineAndTruncationRemainValidUTF8(t *testing.T) {
	sparkline := buildSparkline(
		[]float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		6,
	)
	if !utf8.ValidString(sparkline) {
		t.Fatalf("sparkline is invalid UTF-8: %q", sparkline)
	}
	if got := utf8.RuneCountInString(sparkline); got != 6 {
		t.Fatalf("sparkline rune count = %d, want 6", got)
	}
	for _, r := range sparkline {
		if !strings.ContainsRune(sparkChars, r) {
			t.Fatalf("unexpected sparkline rune %q", r)
		}
	}

	line := truncateLine("metric : ایران🙂abcdef", 12)
	if !utf8.ValidString(line) {
		t.Fatalf("truncated line is invalid UTF-8: %q", line)
	}
	if got := utf8.RuneCountInString(line); got > 12 {
		t.Fatalf("truncated line width = %d, want <= 12", got)
	}
}
