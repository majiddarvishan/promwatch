package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunMonitorCanceledContextRestoresTerminal(t *testing.T) {
	var buf bytes.Buffer
	renderer := newTerminalRenderer(
		&buf,
		true,
		func() (int, int, error) { return 80, 24, nil },
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runMonitor(
		ctx,
		nil,
		monitorConfig{Interval: time.Second},
		renderer,
	)
	if err != nil {
		t.Fatalf("runMonitor() error = %v", err)
	}

	got := buf.String()
	if strings.Count(got, enterAlternateScreenSequence) != 1 {
		t.Fatalf("alternate-screen enter count = %d, want 1", strings.Count(got, enterAlternateScreenSequence))
	}
	if strings.Count(got, hideCursorSequence) != 1 {
		t.Fatalf("hide-cursor count = %d, want 1", strings.Count(got, hideCursorSequence))
	}
	if strings.Count(got, showCursorSequence) != 1 {
		t.Fatalf("show-cursor count = %d, want 1", strings.Count(got, showCursorSequence))
	}
	if strings.Count(got, leaveAlternateScreenSequence) != 1 {
		t.Fatalf("alternate-screen leave count = %d, want 1", strings.Count(got, leaveAlternateScreenSequence))
	}
}

func TestRunMonitorRejectsInvalidInterval(t *testing.T) {
	renderer := newTerminalRenderer(&bytes.Buffer{}, false, nil)

	err := runMonitor(
		context.Background(),
		nil,
		monitorConfig{Interval: 0},
		renderer,
	)
	if err == nil {
		t.Fatal("runMonitor() error = nil, want invalid interval error")
	}
}
