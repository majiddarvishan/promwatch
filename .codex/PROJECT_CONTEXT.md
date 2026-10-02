# Project Context

## Overview

`promwatch` is a small Go CLI that polls a Prometheus-compatible `/metrics` endpoint, selects exactly one metric series, stores recent samples in memory, and renders the current value plus a Unicode sparkline in a terminal.

Current source layout is intentionally small:

- `main.go` — selector parsing, Prometheus text parsing, HTTP polling, history/rate calculation, and terminal rendering.
- `README.md` — usage and feature documentation.
- `go.mod` — Go module metadata.

## Existing behavior

The application currently:

- polls a metrics endpoint at a configurable interval;
- parses Prometheus text exposition;
- matches a metric name plus exact labels;
- rejects ambiguous selectors that match multiple series;
- tracks up to 3600 samples in memory;
- calculates per-second rate for counters;
- renders a Unicode sparkline;
- refreshes continuously until interrupted.

## Current terminal problem

The live renderer currently clears and redraws using ANSI sequences and prints complete multi-line output every refresh. Periodic errors are printed to stderr with a newline.

Important current characteristics:

- renderer starts with a full-screen clear sequence;
- sparkline width is hard-coded to 120 columns;
- terminal width/height are not detected;
- wrapped lines can increase visible scrolling on narrow terminals;
- repeated polling errors append new lines;
- there is no alternate-screen lifecycle;
- there is no explicit TTY/non-TTY behavior;
- terminal resize is not handled.

## Scope of this branch

Branch `fix/terminal-rendering` is dedicated to making the terminal UI stable and production-friendly.

In scope:

- terminal abstraction;
- alternate screen for interactive mode;
- in-place frame rendering;
- terminal size detection;
- resize-aware sparkline;
- error/status rendered inside the frame;
- TTY detection and non-interactive fallback;
- cleanup and signal behavior;
- tests and documentation.

Out of scope unless required by tests:

- PromQL support;
- aggregation;
- multiple metrics/panels;
- persistent storage;
- histogram visualization;
- redesigning Prometheus parsing;
- changing counter-rate semantics.

## Compatibility goals

Primary target:

- Linux/macOS terminals and SSH sessions with ANSI/VT support.

Secondary target:

- modern Windows Terminal / PowerShell environments that support ANSI escape sequences.

Non-interactive output must remain usable when stdout is redirected or piped.
