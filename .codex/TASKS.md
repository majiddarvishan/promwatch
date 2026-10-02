# Tasks — Terminal Rendering

Legend:

- `[ ]` pending
- `[x]` complete

## TR0 — Baseline and regression safety

- [x] TR0.1 Document exact current rendering behavior and known failure cases.
- [x] TR0.2 Add/organize tests for selector parsing, metric parsing, label matching, history, and rate logic before refactoring.
- [x] TR0.3 Add test coverage for frame-related pure functions as they are introduced.
- [x] TR0.4 Establish baseline commands: `go test ./...`, `go vet ./...`, `go build ./...`.
- [x] TR0 Gate: baseline tests/build pass before terminal behavior changes.

## TR1 — Terminal abstraction

- [x] TR1.1 Introduce a small terminal capability/renderer abstraction.
- [x] TR1.2 Separate direct terminal control writes from metric polling logic.
- [x] TR1.3 Make frame construction testable without a real TTY.
- [x] TR1.4 Keep metric selection/rate semantics unchanged.
- [x] TR1 Gate: existing behavior remains functionally equivalent before alternate-screen activation.

## TR2 — Stable interactive screen

- [x] TR2.1 Enter alternate screen in interactive mode.
- [x] TR2.2 Hide cursor while dashboard is active.
- [x] TR2.3 Redraw from cursor home instead of appending a new dashboard frame.
- [x] TR2.4 Clear stale remainder after shorter frames.
- [x] TR2.5 Restore cursor and normal screen on shutdown.
- [x] TR2 Gate: repeated successful polling does not grow terminal scrollback.

## TR3 — Responsive dimensions and resize

- [x] TR3.1 Detect current terminal width/height.
- [x] TR3.2 Replace hard-coded 120-column sparkline behavior.
- [x] TR3.3 Bound all lines to avoid accidental wrapping where practical.
- [x] TR3.4 Recalculate layout on terminal resize/poll refresh.
- [x] TR3.5 Add safe fallback dimensions when size detection fails.
- [x] TR3 Gate: narrow/wide resizing does not create continuous scrolling or corrupt the frame.

## TR4 — Error and status rendering

- [ ] TR4.1 Represent transient fetch/validation errors as UI state.
- [ ] TR4.2 Remove recurring interactive stderr newline spam.
- [ ] TR4.3 Display connection/status information inside the fixed frame.
- [ ] TR4.4 Clear/replace prior error when the endpoint recovers.
- [ ] TR4.5 Preserve one-shot stderr output for fatal startup/config errors.
- [ ] TR4 Gate: a prolonged endpoint outage does not append one terminal line per poll.

## TR5 — TTY and non-interactive fallback

- [ ] TR5.1 Detect whether stdout is a TTY.
- [ ] TR5.2 Disable alternate-screen ANSI lifecycle for non-TTY output.
- [ ] TR5.3 Define simple plain-output behavior for pipes/files/service logs.
- [ ] TR5.4 Verify redirected and piped execution.
- [ ] TR5 Gate: non-TTY consumers receive usable output without dashboard control sequences.

## TR6 — Lifecycle hardening

- [ ] TR6.1 Make terminal cleanup idempotent.
- [ ] TR6.2 Verify Ctrl+C cleanup.
- [ ] TR6.3 Verify SIGTERM cleanup.
- [ ] TR6.4 Handle very small terminal dimensions gracefully.
- [ ] TR6.5 Check Unicode sparkline behavior and avoid partial/corrupt rendering.
- [ ] TR6 Gate: terminal is restored correctly after all supported normal shutdown paths.

## TR7 — Verification and documentation

- [ ] TR7.1 Run `gofmt` on changed Go files.
- [ ] TR7.2 Run `go test ./...`.
- [ ] TR7.3 Run `go vet ./...`.
- [ ] TR7.4 Run `go build ./...`.
- [ ] TR7.5 Manually verify normal terminal, narrow terminal, resize, outage/recovery, and Ctrl+C.
- [ ] TR7.6 Verify SSH behavior.
- [ ] TR7.7 Verify modern Windows terminal behavior where practical.
- [ ] TR7.8 Update README with terminal behavior and non-TTY notes.
- [ ] TR7.9 Update `.codex/SESSION.md` with final implementation status.
- [ ] TR7 Gate: all automated checks pass and interactive mode remains fixed-screen without continuous scrolling.

## Milestone Gate — Terminal Rendering Complete

All of the following must be true:

- [ ] Dashboard refresh does not continuously grow scrollback.
- [x] Sparkline respects available terminal width.
- [ ] Repeated transient errors do not spam new terminal lines.
- [x] Resize remains usable.
- [ ] Ctrl+C/SIGTERM restore terminal state.
- [ ] Non-TTY execution avoids alternate-screen control codes.
- [ ] Prometheus selector/rate behavior has no known regression.
- [ ] README and Codex handoff files reflect the implemented state.
