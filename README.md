# promwatch

`promwatch` is a lightweight Go CLI for watching one Prometheus metric series directly in a terminal.

It polls a Prometheus-compatible `/metrics` endpoint, selects exactly one series, keeps recent samples in memory, calculates counter rate when requested, and renders a responsive Unicode sparkline.

No Prometheus server, Grafana, browser, or external database is required.

## Features

- Prometheus text exposition parsing
- Exact metric-label selection
- Ambiguous-series detection
- Counter rate calculation with reset handling
- Up to 3600 in-memory samples
- Responsive Unicode sparkline
- Fixed-screen interactive terminal UI
- Alternate-screen rendering without poll-by-poll scrollback growth
- Terminal resize handling
- In-place error/status display
- Plain non-TTY output for pipes, files, service logs, and CI
- Graceful Ctrl+C / SIGTERM shutdown path
- Optional rate-threshold event logging to an append-only file

## Requirements

- Go 1.23 or newer
- A Prometheus-compatible HTTP metrics endpoint
- For the interactive dashboard, an ANSI/VT-capable terminal

The terminal implementation uses `golang.org/x/term` for TTY and terminal-size detection.

## Build

```bash
go mod tidy
gofmt -w main.go terminal.go main_test.go terminal_test.go runner_test.go
go test ./...
go vet ./...
go build -o promwatch .
```

## Usage

Basic metric:

```bash
./promwatch \
  --url http://localhost:9999/metrics \
  --metric 'submit_packets{name="receive",system_id="smpp_client_0"}'
```

Counter rate:

```bash
./promwatch \
  --url http://localhost:9999/metrics \
  --metric 'submit_packets{name="receive",system_id="smpp_client_0"}' \
  --rate \
  --interval 1s
```

### Options

- `--url`: Prometheus metrics endpoint. Default: `http://localhost:9999/metrics`
- `--metric`: required metric selector
- `--rate`: show per-second rate; valid only for a metric exposed as `counter`
- `--interval`: polling interval. Default: `1s`
- `--threshold`: log an event whenever the calculated rate is strictly greater than this non-negative value
- `--threshold-file`: threshold event log path. Default: `promwatch-threshold.log`

## Metric selection

A metric can be selected without labels:

```bash
--metric 'submit_packets'
```

Or with exact labels:

```bash
--metric 'submit_packets{name="receive",system_id="smpp_client_0"}'
```

The selector must match exactly one series. If multiple series match, add more labels.

Supported label matching:

```text
=
```

PromQL operators such as `!=`, `=~`, and `!~` are not currently supported.

## Interactive terminal mode

When stdout is a TTY, promwatch uses a fixed-screen dashboard similar to tools such as `top`.

Example:

```text
promwatch
metric : submit_packets{name="receive",system_id="smpp_client_0"}
status : OK
type   : counter
value  : 12540
rate   : 2540/s
updated: 2026-10-03T02:09:10+03:30

▁▂▂▃▄▅▅▆▆▇████

samples: 18    interval: 17s
```

The interactive renderer:

- enters the alternate screen;
- hides the cursor while active;
- redraws from the home position;
- clears stale frame content;
- sizes the graph from the current terminal width;
- re-reads terminal dimensions on every refresh;
- leaves one column unused to reduce terminal auto-wrap risk;
- restores the cursor and normal screen on shutdown.

The frame is also bounded by terminal height. Very small terminals degrade to a truncated but valid frame instead of continuously wrapping.

## Error handling

Transient Prometheus errors do not create a new terminal line on every poll in interactive mode.

They are shown inside the same dashboard:

```text
promwatch
metric : submit_packets{name="receive"}
status : ERROR
error  : connection refused
type   : counter
last value: 12540
last ok: 2026-10-03T02:09:10+03:30
```

When the endpoint recovers, the error is cleared on the next successful poll.

Startup/configuration errors are still printed once to stderr.

## Non-TTY / piped output

When stdout is redirected or piped, promwatch does not emit alternate-screen or cursor-control ANSI sequences.

Instead it writes one plain line per poll:

```text
2026-10-03T02:09:10+03:30 status=ok metric="submit_packets{name=\"receive\"}" type=counter value=12540 rate=2540/s samples=18
```

Error example:

```text
2026-10-03T02:09:11+03:30 status=error metric="submit_packets{name=\"receive\"}" error="connection refused" last_value=12540 last_success=2026-10-03T02:09:10+03:30 samples=18
```

Examples:

```bash
./promwatch --metric 'submit_packets{name="receive"}' | head
./promwatch --metric 'submit_packets{name="receive"}' > promwatch.log
```

This mode is suitable for shell pipelines, files, systemd/supervisor log capture, and CI.

## Counter rate

For `--rate`, the selected metric must expose:

```text
# TYPE submit_packets counter
```

Rate is calculated from consecutive samples using the actual elapsed time:

```text
rate = (current - previous) / elapsed_seconds
```

If the counter decreases, promwatch treats it as a reset and calculates from the new counter value instead of returning a negative rate.

The first sample displays no rate because no previous sample exists.

## Rate threshold logging

Use `--threshold` to persist rate spikes to a file.

Example:

```bash
./promwatch \
  --url http://localhost:9999/metrics \
  --metric 'submit_packets{name="receive",system_id="smpp_client_0"}' \
  --threshold 2500 \
  --threshold-file ./submit-rate-threshold.log \
  --interval 1s
```

Setting `--threshold` automatically enables rate calculation, so `--rate` is optional in this mode.

The comparison is strict: `rate > threshold`. A rate equal to the threshold is not logged.

The file is created lazily: if the threshold is never exceeded, no threshold log file is created. Existing files are opened in append mode.

Each exceeded poll produces one line:

```text
2026-10-03T02:15:00.123+03:30 metric="submit_packets{name=\"receive\",system_id=\"smpp_client_0\"}" rate=2740/s threshold=2500/s value=481250
```

Each record contains the poll timestamp, selected metric, calculated rate, configured threshold, and current raw metric value.

While enabled, the interactive dashboard also shows the threshold and marks the current poll as `EXCEEDED` when applicable. Non-TTY output includes `threshold` and `threshold_exceeded`.

If a threshold event cannot be written, promwatch returns a visible runtime error instead of silently dropping the event.
## History

Up to 3600 samples are kept in memory.

At a 1-second interval this is approximately one hour.

History is not persisted; restarting promwatch starts a new history.

## SSH usage

`promwatch` is designed to work well over SSH:

```bash
ssh user@gateway

./promwatch \
  --url http://localhost:9999/metrics \
  --metric 'submit_packets{name="receive",system_id="smpp_client_0"}' \
  --rate
```

The dashboard uses the terminal dimensions reported by the SSH TTY.

## Windows

Modern Windows Terminal / PowerShell environments with ANSI/VT support can use the interactive dashboard. Redirected output automatically uses plain non-TTY mode.

## Exit and terminal restoration

Press:

```text
Ctrl+C
```

to stop promwatch.

The application also listens for SIGTERM where supported. Both paths cancel the monitoring context and run terminal cleanup before returning.

Renderer cleanup is idempotent, and a partial startup write triggers a best-effort restoration attempt.

## Supported Prometheus subset

Supported:

- metric names
- exact labels
- `# TYPE`
- counter values
- gauge values
- `NaN`
- `+Inf`
- `-Inf`

Not currently supported:

- general PromQL expressions
- `!=`, `=~`, `!~`
- aggregations
- multiple simultaneously selected series
- histogram visualization
- summary visualization
- exemplars
- persistent history
- remote storage
- multiple graph panels

## Architecture

```text
Prometheus /metrics
        |
        v
 text parser + selector
        |
        v
    UI state
 value / rate / history / error
        |
        v
 TerminalRenderer
      /   \
     /     \
  TTY       non-TTY
 fixed      plain
 screen     lines
```

## Verification

The release verification checklist is maintained in:

```text
.codex/TR7_VERIFY.md
```

## License

Internal/project-specific tool.
