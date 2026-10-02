# Session

## Current branch

`fix/terminal-rendering`

## Session objective

Prepare a phased implementation plan for fixing terminal scrolling and establishing a stable TUI-style renderer for promwatch.

## Completed in this session

- Reviewed current `main.go`, `go.mod`, and `README.md`.
- Identified the primary terminal issues:
  - full multi-line redraw behavior;
  - no alternate-screen lifecycle;
  - hard-coded 120-column sparkline;
  - no terminal size detection;
  - repeated transient errors printed with newline;
  - no explicit TTY/non-TTY mode.
- Created the dedicated working branch.
- Added Codex project context, decisions, implementation plan, and phased tasks.

## Product code status

No product-code implementation changes have been made as part of the planning step.

## Next action

Start with **TR0 — Baseline and regression safety** in `.codex/TASKS.md`.

Before implementing terminal behavior:

1. add/confirm regression tests for the existing pure logic;
2. run baseline test/vet/build commands;
3. record any pre-existing failures;
4. only then proceed to TR1.

## Key decision reminder

Do not introduce a full TUI framework by default. Prefer a small testable terminal renderer, with `golang.org/x/term` acceptable for TTY/size detection if needed.
