# Terminal UI Implementation Plan

## Problem statement

The current dashboard refresh strategy can cause scrollback growth or visible scrolling because each poll writes a full multi-line frame, the sparkline can wrap on narrow terminals, and recurring errors append lines to stderr.

The target is a stable screen that updates in place.

## Proposed architecture

```text
Prometheus endpoint
        |
        v
     fetch
        |
        v
 Application State
 value / rate / history / status / last error
        |
        v
   Frame Builder
        |
        v
 Terminal Renderer
   |           |
 interactive   non-TTY
 alternate     plain
 screen        output
```

## Suggested internal responsibilities

### Terminal capability layer

Responsible for:

- detecting whether stdout is a terminal;
- reading terminal width/height;
- entering/leaving alternate screen;
- hiding/showing cursor;
- moving cursor home;
- clearing stale content.

### UI state

Keep display state separate from direct printing where practical:

- latest metric series;
- history;
- latest rate;
- last successful update;
- current error/status;
- configured interval.

### Frame builder

Build the visible dashboard as text based on:

- current state;
- terminal width;
- terminal height.

The frame should be renderable/testable without needing a live terminal.

## Interactive lifecycle

Expected startup:

1. validate CLI arguments;
2. detect terminal mode;
3. enter alternate screen if interactive;
4. hide cursor;
5. start polling/render loop.

Expected refresh:

1. fetch metric;
2. update state;
3. derive current terminal size;
4. build bounded frame;
5. cursor home;
6. write frame;
7. clear stale remainder.

Expected shutdown:

1. stop ticker/work;
2. show cursor;
3. leave alternate screen;
4. return to shell.

## Width policy

Remove the unconditional 120-column assumption.

The sparkline width should be bounded by:

- terminal width;
- label/prefix requirements;
- a reasonable minimum;
- available history length.

If terminal size cannot be detected, use a conservative fallback such as 80 columns.

## Height policy

The first implementation may keep the dashboard compact enough to fit common terminals. However, it must avoid producing uncontrolled wrapped lines.

For extremely small terminals, degrade gracefully rather than writing an oversized frame.

## Error policy

Interactive transient error:

```text
status : ERROR
error  : connection refused
```

A later successful fetch should replace this status in the same frame.

Fatal startup error:

- print once to stderr;
- exit non-zero;
- do not enter the long-running dashboard loop.

## Dependency policy

If terminal-size/TTY detection needs a dependency, `golang.org/x/term` is acceptable.

Avoid adding a full TUI framework for this workstream unless evidence from implementation/tests shows it is necessary.

## Verification matrix

At minimum verify:

- normal 80x24 terminal;
- narrow terminal;
- resize wider/narrower while running;
- Prometheus endpoint unavailable for multiple polls;
- endpoint recovers after errors;
- Ctrl+C cleanup;
- SIGTERM cleanup;
- redirected stdout;
- piped stdout;
- SSH terminal;
- modern Windows ANSI-capable terminal where available.
