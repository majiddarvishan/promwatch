# Session

## Current branch

`fix/terminal-rendering`

## Current status

**TR0 through TR5 are complete.**

The interactive path now uses a fixed alternate-screen dashboard for both successful polls and transient errors. Non-interactive stdout now emits plain line-oriented records without ANSI control sequences.

## Completed through TR3

- Regression coverage for parsing/history/rate.
- Testable renderer/frame separation.
- Alternate-screen lifecycle.
- Cursor-home redraw and stale-content clearing.
- Responsive terminal width/height.
- Sparkline width derived from the terminal.
- Defensive rate handling for inconsistent history state.

## TR4 — Error and status rendering

Implemented:

- introduced `UIState` to carry selector, latest series, history, rate mode, current error, poll time, and last successful poll;
- transient fetch errors are rendered into the dashboard instead of being appended to stderr;
- rate-validation errors are also represented as dashboard state;
- interactive frames now show `status : OK` or `status : ERROR`;
- error frames include the error text and retain last known metric data when available;
- last successful update time is retained across outages;
- the next successful poll clears the error state automatically;
- fatal startup/configuration errors still use one-shot stderr output;
- renderer write failure is treated as fatal for the run and cancels the context instead of repeatedly spamming stderr.

Result: a prolonged Prometheus outage redraws the same interactive screen rather than adding a new error line per poll.

## TR5 — TTY and non-interactive fallback

Implemented:

- stdout TTY detection continues to use `golang.org/x/term`;
- alternate-screen/cursor ANSI lifecycle is only used for interactive TTY output;
- non-TTY `Start()` and `Close()` emit nothing;
- non-TTY `Render()` emits exactly one plain line per poll;
- plain success records include status, metric, type, value, optional rate, and sample count;
- plain error records include status, metric, quoted error text, and last-known value/success time when available;
- redirected/piped output contains no terminal escape sequences;
- tests cover both plain success and plain error output.

Example non-TTY success shape:

```text
2026-10-03T00:00:01Z status=ok metric="submit_packets{name=\"receive\"}" type=counter value=1250.0 rate=250/s samples=2
```

Example non-TTY error shape:

```text
2026-10-03T00:00:02Z status=error metric="submit_packets{name=\"receive\"}" error="connection refused" last_value=1250.0 last_success=2026-10-03T00:00:01Z samples=2
```

## Verification status

The implementation and regression tests for TR4/TR5 have been committed, but local execution should be run on the target checkout:

```bash
go test ./...
go vet ./...
go build ./...
```

Also manually verify one pipe/redirect example:

```bash
./promwatch --metric '<selector>' --interval 1s | head
```

The piped output must contain plain text only and no visible ANSI escape sequences.

## Next action

Proceed to **TR6 — Lifecycle hardening** after the local test/vet/build gate passes.
