# START HERE

Project: **promwatch**
Repository: `majiddarvishan/promwatch`
Working branch: `fix/terminal-rendering`

## Current goal

Fix the live terminal dashboard so it behaves like a stable TUI instead of causing the terminal/scrollback to grow while Prometheus metrics are refreshed.

Do not start by changing metric parsing or Prometheus semantics. The current workstream is focused on terminal rendering, resize behavior, error/status presentation, and safe non-TTY behavior.

## Read in this order

1. `.codex/PROJECT_CONTEXT.md`
2. `.codex/DECISIONS.md`
3. `.codex/TERMINAL_UI_PLAN.md`
4. `.codex/TASKS.md`
5. `.codex/SESSION.md`

## Working rules

- Execute tasks phase by phase.
- Mark completed tasks with `[x]` in `.codex/TASKS.md`.
- Keep `.codex/SESSION.md` updated after meaningful implementation steps.
- Preserve existing metric selector, history, and rate behavior unless a task explicitly requires changing it.
- Prefer small, testable changes over a terminal UI framework unless the simple renderer proves insufficient.
- Restore terminal state on every normal shutdown path.
- Interactive rendering must not append one new dashboard frame to scrollback on every poll.

## Initial implementation target

The desired interactive behavior is similar to `top`:

- fixed screen;
- in-place redraw;
- no periodic newline spam;
- responsive sparkline width;
- clean error/status area;
- graceful Ctrl+C/SIGTERM;
- correct terminal restoration.
