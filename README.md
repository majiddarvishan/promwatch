# promwatch

`promwatch` is a lightweight CLI tool for monitoring Prometheus metrics
directly from a terminal.

It polls a Prometheus-compatible `/metrics` HTTP endpoint, selects a single
metric series, keeps a short in-memory history, and renders the values as an
ANSI/Unicode sparkline.

No Prometheus server, Grafana, browser, or external database is required.

## Features

- Terminal-only monitoring
- Reads Prometheus text exposition format
- Selects metrics using a PromQL-like selector
- Supports metric labels with `=`
- Detects `# TYPE` information
- Supports counter rate calculation
- Detects ambiguous selectors
- In-memory history
- Unicode sparkline visualization
- Configurable polling interval
- No external Go dependencies

## Requirements

- Go 1.23 or newer
- A Prometheus-compatible HTTP endpoint

For example:

```text
http://localhost:9999/metrics
```

## Build

Clone or copy the project and build:

```bash
gofmt -w main.go
go build -o promwatch .
```

The project intentionally has no external dependencies.

go.mod:

module promwatch

go 1.23
Usage

Basic metric:

./promwatch \
  --url http://localhost:9999/metrics \
  --metric 'submit_packets{name="receive",system_id="smpp_client_0"}'

Monitor a counter and display its rate:

./promwatch \
  --url http://localhost:9999/metrics \
  --metric 'submit_packets{name="receive",system_id="smpp_client_0"}' \
  --rate \
  --interval 1s

Short form:

./promwatch \
  --metric 'submit_packets{name="receive",system_id="smpp_client_0"}' \
  --rate
Command-Line Options
--url

Prometheus metrics endpoint.

Default:

http://localhost:9999/metrics

Example:

--url http://10.10.10.20:9999/metrics
--metric

Metric selector.

A metric can be selected without labels:

--metric 'submit_packets'

Or with labels:

--metric 'submit_packets{name="receive"}'

Multiple labels can be specified:

--metric 'submit_packets{name="receive",system_id="smpp_client_0"}'

Label matching currently supports:

=

For example:

name="receive"
system_id="smpp_client_0"
instance="smpp_gateway"

Regular expressions and other PromQL operators such as !=, =~,
and !~ are not currently supported.

--rate

Calculate and display the per-second rate.

Example:

--rate

--rate is intended for Prometheus counter metrics.

The metric must expose its type:

# TYPE submit_packets counter

The rate is calculated from consecutive samples:

rate = (current_value - previous_value) / elapsed_time

If the counter decreases, promwatch treats it as a counter reset.

For example:

previous = 1000
current  = 1200
elapsed  = 1s

rate = 200 packets/s
--interval

Polling interval.

Default:

1s

Example:

--interval 500ms

or:

--interval 5s
Metric Selection

promwatch requires the selector to identify exactly one time series.

For example, suppose /metrics contains:

submit_packets{instance="gw1",name="receive",system_id="smpp_client_0"} 1000
submit_packets{instance="gw2",name="receive",system_id="smpp_client_0"} 2000

This selector:

--metric 'submit_packets{name="receive",system_id="smpp_client_0"}'

matches two series.

promwatch will therefore return an error:

selector "submit_packets{name=\"receive\",system_id=\"smpp_client_0\"}"
matched 2 series; add more labels to select exactly one series

Use:

--metric 'submit_packets{instance="gw1",name="receive",system_id="smpp_client_0"}'

to select exactly one series.

This behavior is intentional. promwatch does not silently aggregate
multiple series because that could produce an incorrect monitoring result.

Example Prometheus Metrics

Given:

# HELP submit_packets Number of submitted packets
# TYPE submit_packets counter

submit_packets{instance="smpp_gateway",ip="127.0.0.1",name="receive",system_id="smpp_client_0"} 10000
submit_packets{instance="smpp_gateway",ip="127.0.0.1",name="receive",system_id="smpp_client_1"} 15000

Monitor smpp_client_0:

./promwatch \
  --url http://localhost:9999/metrics \
  --metric 'submit_packets{name="receive",system_id="smpp_client_0"}' \
  --rate \
  --interval 1s

Possible output:

```bash
promwatch
metric : submit_packets{name="receive",system_id="smpp_client_0"}
type   : counter
value  : 12540
rate   : 2540/s

▁▂▂▃▄▅▅▆▆▇████

samples: 18    interval: 17s
```

## Counter Rate

The first sample does not have a previous value, so a rate cannot yet
be calculated:

rate   : --/s

After the second sample, the rate becomes available.

For example:

t=0s   counter=1000
t=1s   counter=1250
t=2s   counter=1510

The displayed rates will approximately be:

t=0s   --
t=1s   250/s
t=2s   260/s

The calculation uses the actual elapsed time between samples rather than
assuming that the polling interval was exact.

Counter Reset

Counters can reset after a process restart or other events.

Example:

previous = 100000
current  = 100

promwatch interprets this as a counter reset and calculates:

rate = 100 / elapsed_time

rather than producing a large negative rate.

History

promwatch stores recent samples in memory.

The current implementation keeps up to:

3600 samples

For a 1-second polling interval, this represents approximately:

1 hour

The history is not persisted to disk.

Restarting promwatch starts a new history.

Architecture

The application is intentionally small:

             HTTP
              │
              ▼
       /metrics endpoint
              │
              ▼
      Prometheus text parser
              │
              ▼
       Metric selector
              │
              ▼
       Single time series
              │
              ▼
       In-memory history
              │
        ┌─────┴─────┐
        │           │
        ▼           ▼
     Current       Rate
      value      calculation
        │           │
        └─────┬─────┘
              ▼
        Terminal UI
         Sparkline
Why No Prometheus Server?

promwatch is intended for situations where running a complete monitoring
stack is unnecessary.

For example:

Debugging a telecom gateway over SSH
Monitoring a production process temporarily
Working on a secured server without a browser
Checking TPS during a load test
Inspecting a single metric during troubleshooting
Monitoring a server where Prometheus/Grafana is not installed

Instead of:

Application
    │
    ▼
Prometheus
    │
    ▼
Grafana
    │
    ▼
Browser

you can use:

Application
    │
    ▼
promwatch
    │
    ▼
Terminal
SSH Usage

promwatch can be especially useful over SSH.

For example:

ssh user@gateway

Then:

./promwatch \
  --url http://localhost:9999/metrics \
  --metric 'submit_packets{name="receive",system_id="smpp_client_0"}' \
  --rate

No browser or port forwarding is required.

Current Limitations

The current version intentionally implements only a small subset of the
Prometheus ecosystem.

Supported:

Prometheus text exposition format
Metric names
Labels
# TYPE
Counter values
Gauge values
NaN
+Inf
-Inf
Exact label matching using =

Not currently supported:

PromQL expressions
!=
=~
!~
Aggregations
Multiple selected series
Histograms
Summaries
Exemplars
Persistent history
Prometheus remote storage
Multiple graphs at the same time
Exit

Press:

Ctrl+C

to stop promwatch.

Future Improvements

Possible future features:

Multiple metrics on the same screen
--width and --height
Better terminal UI
!=, =~, and !~ label selectors
Configurable history size
Min/max/average statistics
TPS and percentile display
Gauge support
Histogram visualization
CSV export
Snapshot mode
Auto-discovery of metric names
Metric aliases
Threshold and alert indicators
Multiple panels similar to a lightweight Grafana dashboard
License

Internal/project-specific tool.