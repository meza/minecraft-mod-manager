# Discovery harness

This folder holds the black box discovery harness for MMM interactions.

It is designed for repeatable runs against the shipped binary.
It does not replace behavior authority in `docs/interactions/interaction-guidelines.md` and `docs/interactions/flows/`.

## What is here

- `docs/interactions/black-box-discovery.md` defines the scenarios and capture rules.
- `docs/interactions/discovery/expect/tty_capture.exp` runs a command in a pseudo TTY and captures output.
- `docs/interactions/discovery/expect/tty_capture_q.exp` runs a command in a pseudo TTY, captures output, and sends `q` to quit (useful for list-style TUIs).
- `docs/interactions/discovery/expect/tty_capture_n.exp` runs a command in a pseudo TTY, captures output, and sends `n` plus Enter (useful for y/N confirmations).
- `docs/interactions/discovery/expect/tty_capture_y.exp` runs a command in a pseudo TTY, captures output, and sends `y` plus Enter (useful for y/N confirmations).

## How to run

Create a fresh run folder under `/tmp/mmm-interactions` and then execute MMM through the expect wrapper.

Example:

```sh
mkdir -p /tmp/mmm-interactions/example/tty
expect docs/interactions/discovery/expect/tty_capture.exp \
  /tmp/mmm-interactions/example/tty \
  /tmp/mmm-interactions/example/tty/stdout.txt \
  10 \
  28 \
  100 \
  /work/build/linux/amd64/mmm init \
  >/tmp/mmm-interactions/example/tty/expect_stdout.txt 2>&1
```

Notes:
- The capture file includes ANSI escape sequences.
- The wrapper replies to common terminal capability probes used by Bubble Tea.
