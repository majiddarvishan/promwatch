package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type ThresholdEvent struct {
	Timestamp   time.Time
	Selector    Selector
	Rate        float64
	Threshold   float64
	MetricValue float64
}

type ThresholdLogger struct {
	path string
	file *os.File
}

func NewThresholdLogger(path string) *ThresholdLogger {
	return &ThresholdLogger{path: path}
}

func (l *ThresholdLogger) Record(event ThresholdEvent) error {
	if l == nil {
		return fmt.Errorf("threshold logger is nil")
	}

	if l.file == nil {
		file, err := os.OpenFile(
			l.path,
			os.O_CREATE|os.O_APPEND|os.O_WRONLY,
			0o644,
		)
		if err != nil {
			return err
		}
		l.file = file
	}

	_, err := fmt.Fprintln(l.file, formatThresholdEvent(event))
	return err
}

func (l *ThresholdLogger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}

	file := l.file
	l.file = nil
	return file.Close()
}

func thresholdExceeded(history *History, threshold float64) (float64, bool) {
	rate, ok := calculateRate(history)
	if !ok {
		return 0, false
	}
	return rate, rate > threshold
}

func formatThresholdEvent(event ThresholdEvent) string {
	return strings.Join(
		[]string{
			event.Timestamp.Format(time.RFC3339Nano),
			"metric=" + strconv.Quote(selectorToString(event.Selector)),
			"rate=" + formatNumber(event.Rate) + "/s",
			"threshold=" + formatNumber(event.Threshold) + "/s",
			"value=" + formatNumber(event.MetricValue),
		},
		" ",
	)
}
