If you only care about **output** (no input in non-TTY), Bubble Tea already has the knobs you want:

* `tea.WithInput(nil)` **disables input entirely** ([Go Packages][1])
* `tea.WithoutRenderer()` **disables the renderer/redraw logic** so Bubble Tea won’t do cursor movement / line rewriting; printing/logging behaves like a normal CLI tool ([Go Packages][1])

That combination is the intended “non-TUI mode” switch.

## The pattern

### 1) Decide “interactive TTY mode” vs “plain output mode”

Use `golang.org/x/term` or `go-isatty` to check stdout.

### 2) In plain mode, emit output via Bubble Tea commands (`tea.Println` / `tea.Printf`)

`tea.Println(...)` and `tea.Printf(...)` are Bubble Tea commands that print “above the program” and persist across renders ([Go Packages][1]). When you also set `WithoutRenderer()`, those prints become your primary output mechanism, and there’s no screen manipulation. ([Go Packages][1])

### 3) Make the program self-driving and self-terminating

Since there’s no input, your `Init` + subsequent `Cmd`s must drive progress, and you must `return tea.Quit` when done (otherwise `Program.Run()` will block). ([Go Packages][1])

## Minimal example

```go
package main

import (
	"fmt"
	"os"

	"golang.org/x/term"

	tea "github.com/charmbracelet/bubbletea"
)

type mode int

const (
	modeTUI mode = iota
	modePlain
)

type model struct {
	mode   mode
	step   int
	done   bool
	lastLn string
}

func (m model) Init() tea.Cmd {
	// Kick off work (replace with your real async command chain).
	return tea.Sequence(
		tea.Println(m.renderLine()), // initial line
		nextStepCmd(),
	)
}

type stepMsg struct{}

func nextStepCmd() tea.Cmd {
	return func() tea.Msg { return stepMsg{} }
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case stepMsg:
		m.step++
		if m.step >= 3 {
			m.done = true
		}

		// In both modes, drive ALL output through Bubble Tea.
		ln := m.renderLine()
		var out tea.Cmd
		if ln != m.lastLn {
			out = tea.Println(ln)
			m.lastLn = ln
		}

		if m.done {
			return m, tea.Sequence(out, tea.Quit)
		}
		return m, tea.Sequence(out, nextStepCmd())
	}
	return m, nil
}

func (m model) View() string {
	// TUI view. In modePlain it won’t be used (renderer is disabled).
	return fmt.Sprintf("Step %d/3\n", m.step)
}

func (m model) renderLine() string {
	// Plaintext “snapshot” renderer. Keep it free of ANSI.
	return fmt.Sprintf("Step %d/3", m.step)
}

func main() {
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))

	runMode := modePlain
	opts := []tea.ProgramOption{tea.WithOutput(os.Stdout)}

	if isTTY {
		runMode = modeTUI
	} else {
		// Non-TTY: no input, no renderer (no cursor movement / redrawing).
		opts = append(opts, tea.WithInput(nil), tea.WithoutRenderer())
	}

	p := tea.NewProgram(model{mode: runMode}, opts...)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

## About ANSI styling (Lip Gloss / termenv)

If you’re using Lip Gloss for styling, it will automatically remove color output when output isn’t a TTY ([GitHub][2]), and termenv’s `Ascii` profile represents “ASCII only / no ANSI support detected.” ([GitHub][3])
So: `WithoutRenderer()` stops cursor/line manipulation, and Lip Gloss typically prevents color codes in non-TTY output—provided your “plain” render path doesn’t manually inject ANSI.

## Practical recommendation

For “Bubble Tea renders everything” *and* “no line rewriting in non-TTY”:

* **TTY**: normal Bubble Tea `View()` rendering
* **non-TTY**: `WithInput(nil)` + `WithoutRenderer()` and emit output via **`tea.Println/Printf`** when the *semantic* output changes (steps, events, progress milestones), not every frame.

[1]: https://pkg.go.dev/github.com/charmbracelet/bubbletea "tea package - github.com/charmbracelet/bubbletea - Go Packages"
[2]: https://github.com/charmbracelet/lipgloss?utm_source=chatgpt.com "charmbracelet/lipgloss: Style definitions for nice terminal ..."
[3]: https://github.com/muesli/termenv?utm_source=chatgpt.com "muesli/termenv: Advanced ANSI style & color support for ..."


---

Yep — your `change` flow already nails the contract MMM needs:

* In **non-tty mode**: no prompts, no control sequences, printed as a plain transcript.
* And per the guidelines: when stdout isn’t a TTY, MMM must degrade in an *idiomatic Bubble Tea way* (no custom “strip ANSI / don’t move cursor” hacks).

## The Bubble Tea way to do this (output-only)

### 1) In non-tty, run Bubble Tea with the renderer disabled

Bubble Tea provides this explicitly:

* `tea.WithoutRenderer()` disables rendering/redraw logic so output/logging becomes “normal CLI output” (no redraws/cursor movement) and is specifically called out as useful when output is not a TTY. ([Go Packages][1])

Important detail: `WithoutRenderer` uses a “nil renderer” (its `write` is a no-op), so you **cannot rely on `View()` to print anything** in this mode. ([sources.debian.org][2])

So your non-tty path should emit output via Bubble Tea *commands* (next section).

### 2) Emit transcript lines with `tea.Println` / `tea.Printf` cmds

Bubble Tea’s `tea.Println` / `tea.Printf` are designed for “unmanaged” output that persists and prints as normal lines. ([Go Packages][1])
The docs also explicitly say: “For non-interactive i/o you should use a Cmd.” ([Go Packages][1])

That maps perfectly to your “frames are printed as a plain transcript” requirement.

### 3) Disable input in non-tty (since you said “output only”)

Since there won’t be input, set `tea.WithInput(nil)` (and don’t prompt). This aligns with your interaction rules (“MUST NOT block on input when stdin or stdout is not a TTY”).

## How this maps to your `change` flow

Your spec shows:

* TTY mode: update icons in-place across sections.
* Non-TTY success: just print final per-mod outcomes + summary.
* Non-TTY errors: print a *snapshot transcript* + actionable summary.

Implementation-wise, that becomes:

* **Interactive (tty)**: normal Bubble Tea renderer; `View()` returns the full multi-section frame.
* **Non-tty**: `WithoutRenderer`; `Update()` emits *lines* (via `tea.Println/Printf`) only at meaningful moments:

  * initial header lines
  * stage headers (`Compatibility:`, `Downloading:`, `Switching:`)
  * final per-mod results (✅/❌/skipped) — ideally once, at the end or on failure snapshot
  * final command summary line(s)

## One practical pitfall to avoid

If your program depends on the initial window-size message to “start”, redirected/non-tty runs can hang. This is a known gotcha; a Bubble Tea maintainer answer notes that, when output is redirected, the usual init message may not arrive, so you must kick work off from `Init()` and ensure you eventually quit. ([GitHub][3])

## Minimal wiring sketch

```go
opts := []tea.ProgramOption{tea.WithOutput(os.Stdout)}

if nonTTY { // stdin or stdout not a tty
    opts = append(opts,
        tea.WithInput(nil),       // no input
        tea.WithoutRenderer(),    // no redraw/cursor control sequences
    )
}

p := tea.NewProgram(model, opts...)
_, err := p.Run()
```

And then: in nonTTY mode your model should *not* rely on `View()`. Instead, return `tea.Println(renderedLine)` commands from `Init()`/`Update()` using the same shared rendering helpers that your TTY `View()` uses.

* keeps one state machine,
* shares the same render helpers,
* and produces *exactly* the non-tty frames you sketched in `change.md`.

[1]: https://pkg.go.dev/github.com/charmbracelet/bubbletea "tea package - github.com/charmbracelet/bubbletea - Go Packages"
[2]: https://sources.debian.org/src/golang-github-charmbracelet-bubbletea/0.27.0-1/nil_renderer.go "File: nil_renderer.go
\| Debian Sources
"
[3]: https://github.com/charmbracelet/bubbletea/discussions/663 "Can't redirect output of non-interactive application · charmbracelet bubbletea · Discussion #663 · GitHub"
