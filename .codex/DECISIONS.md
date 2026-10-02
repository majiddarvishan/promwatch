# Decisions

## D-001 — Keep the implementation lightweight

Use a small internal terminal abstraction rather than adopting a full TUI framework in the first implementation.

Reasoning:

- the application has one compact screen;
- input handling is minimal;
- the main requirement is reliable rendering and cleanup;
- a framework would add dependency and lifecycle complexity before it is justified.

Revisit only if later requirements include interactive navigation, multiple panels, keyboard commands, or complex layouts.

## D-002 — Use alternate screen in interactive mode

Interactive dashboard mode should enter the terminal alternate-screen buffer and restore the normal screen on exit.

Goals:

- avoid poll-by-poll scrollback growth;
- preserve the user's original shell contents;
- provide behavior similar to `top`/lightweight terminal dashboards.

## D-003 — Redraw from cursor home, then clear stale remainder

Each refresh should:

1. build a complete frame;
2. move the cursor to the home position;
3. write the frame;
4. clear leftover content below/after the new frame.

Do not rely on appending lines followed by a full-screen clear every poll.

## D-004 — Errors are application state, not a log stream

Transient fetch/validation errors during interactive execution must be rendered in a fixed status/error area.

Do not append one stderr line per failed poll in interactive mode.

Fatal startup/configuration errors may still use stderr and exit.

## D-005 — Layout follows terminal dimensions

The renderer must read current terminal width/height.

Sparkline width must never assume 120 columns unconditionally. It should be derived from the available width with a safe fallback when terminal size detection fails.

## D-006 — Support non-TTY execution explicitly

When stdout is not an interactive terminal, do not emit alternate-screen control sequences.

Provide a plain output path suitable for:

- pipes;
- files;
- systemd/supervisor capture;
- CI;
- shell scripting.

Exact non-TTY formatting can be finalized during TR5, but ANSI dashboard control codes are prohibited in that mode.

## D-007 — Terminal cleanup must be idempotent

Cursor visibility and screen restoration should be safe if cleanup is called more than once.

Normal Ctrl+C and SIGTERM paths must restore terminal state.

## D-008 — Preserve monitoring semantics

Terminal work must not silently alter:

- selector matching;
- rate validation;
- counter reset handling;
- history semantics;
- metric value formatting,

except where a separate task explicitly documents and tests the change.
