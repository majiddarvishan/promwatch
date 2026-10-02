# TR0 Baseline

## Scope

This baseline captures the behavior of `promwatch` before terminal rendering is changed.

## Current rendering behavior

On each successful poll the application:

1. fetches one selected Prometheus series;
2. appends the value and timestamp to in-memory history;
3. writes `ESC[H ESC[2J` to stdout;
4. prints the complete dashboard again;
5. renders at most the latest 120 samples as a Unicode sparkline.

When `--rate` is enabled, rate is calculated only for metrics reported as `counter`.

Transient fetch/rate-validation errors are currently written to stderr as:

```text
\rerror: <message>\n
```

This means each repeated error creates another terminal line.

## Known rendering failure cases

- The fixed 120-column sparkline can wrap in terminals narrower than 120 columns.
- Wrapped dashboard content can create visible vertical scrolling.
- Repeated errors append one line per poll and therefore definitely grow terminal output.
- The application does not use the alternate screen buffer.
- Terminal width/height are not detected.
- Terminal resize is not handled.
- TTY and redirected/piped output are not distinguished.

These behaviors are intentionally preserved through TR1. The actual terminal behavior changes begin in TR2 and later phases.

## Regression coverage added in TR0

Tests cover:

- selector parsing, including labels and invalid selectors;
- Prometheus metric-line parsing;
- label matching;
- counter rate calculation and reset handling;
- history size retention;
- pure sparkline generation;
- complete frame generation;
- legacy clear-screen prefix through the renderer abstraction.

## Baseline verification

Before the TR1 refactor, the current source was reconstructed without behavioral changes and the following commands passed:

```bash
go test ./...
go vet ./...
go build ./...
```

At that point the repository had no test files, so `go test ./...` reported `[no test files]`.

After TR0 tests and the TR1 refactor, the same three commands also passed.

## TR0 conclusion

The existing metric-selection, parsing, history, rate, and display-layout semantics now have regression protection sufficient to begin terminal-rendering changes.
