# TR7 Verification Matrix

TR6 implementation is complete. TR7 documentation is complete, but the executable verification items below must be run against the current branch HEAD before TR7 Gate and the final Milestone Gate are checked.

## Automated checks

Run from the repository root:

```bash
gofmt -w main.go terminal.go main_test.go terminal_test.go runner_test.go
git diff --check
go test ./...
go vet ./...
go build ./...
```

Expected result: all commands exit successfully.

After they pass, mark TR7.1 through TR7.4 complete in `.codex/TASKS.md`.

## Interactive terminal test

Use a real selector that matches exactly one series:

```bash
./promwatch \
  --url http://localhost:9999/metrics \
  --metric 'YOUR_REAL_SELECTOR' \
  --rate \
  --interval 1s
```

Verify:

- dashboard stays on one screen for at least 20 polls;
- normal scrollback does not grow once the alternate screen is active;
- cursor is hidden while running;
- graph grows within the current terminal width.

## Resize / tiny terminal

While promwatch is running:

1. make the terminal much narrower;
2. make it wider again;
3. reduce height substantially;
4. restore normal dimensions.

Verify:

- no panic;
- no continuous wrapping/scrolling;
- frame is truncated gracefully;
- graph follows the new width.

## Outage and recovery

Start promwatch against a live endpoint, then temporarily stop/block that endpoint.

Verify:

- `status : ERROR` appears in-place;
- no new error line is appended per poll;
- last known value remains visible when available.

Restore the endpoint.

Verify:

- status returns to `OK`;
- error text disappears;
- sampling resumes.

## Ctrl+C cleanup

Run interactively, then press Ctrl+C.

Verify:

- shell screen is restored;
- cursor is visible;
- prompt is usable normally;
- no alternate-screen artifacts remain.

## SIGTERM cleanup

From another shell, identify the promwatch PID and send SIGTERM:

```bash
pgrep -a promwatch
kill -TERM <pid>
```

Verify the same restoration behavior as Ctrl+C.

The unit test `TestRunMonitorCanceledContextRestoresTerminal` covers the common context-cancellation lifecycle used by both os.Interrupt and SIGTERM. This manual check confirms the operating-system signal wiring.

## Non-TTY verification

```bash
./promwatch --metric 'YOUR_REAL_SELECTOR' --interval 1s | head
```

Verify plain line-oriented output.

Optional strict ANSI check:

```bash
timeout 3s ./promwatch --metric 'YOUR_REAL_SELECTOR' > /tmp/promwatch.out || true
if grep -q $'\033' /tmp/promwatch.out; then
  echo "FAIL: ANSI escape found"
else
  echo "PASS: no ANSI escape"
fi
```

## SSH verification

On a host reached through SSH:

```bash
ssh user@host
./promwatch --url http://localhost:9999/metrics --metric 'YOUR_REAL_SELECTOR'
```

Verify fixed-screen refresh, resize behavior, and Ctrl+C restoration.

## Modern Windows terminal verification

Build/run in a current Windows Terminal / PowerShell environment if native Windows support is required.

Verify:

- interactive dashboard renders correctly;
- cursor restores on Ctrl+C;
- resizing remains usable;
- redirected output contains plain text and no ANSI control sequences.

## Completion rule

Only after all applicable checks pass:

- mark TR7.1–TR7.7 as complete;
- mark TR7 Gate complete;
- mark remaining Milestone Gate items complete.
