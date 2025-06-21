# Minecraft Mod Manager — TUI Design Document

---

## 1  Purpose

This document captures the visual wire‑frames and interaction model for the Minecraft Mod Manager (MMM) Terminal User Interface, along with the fallback CLI prompting behaviour. It is intended to guide both implementation and future maintenance.

---

## 2  Three‑Tier Execution Model

| Tier                | Detection                                                                                       | Behaviour                                                                                                     | Example                                                            |
|---------------------|-------------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------| ------------------------------------------------------------------ |
| **Non‑interactive** | • All required flags present **and** `stdin` **not** a TTY• or `--yes / --force / --json` given | • Perform task silently• Return exit‑code, machine‑readable logs                                              | `mmm add --platform curseforge --id 438050 --version 1.20.1 --yes` |
| **Prompted CLI**    | • TTY available **but** required info missing                                                   | • Ask only for missing input via simple line prompts (no Bubble Tea)• Honour `--abort-on-prompt` to fail fast | `mmm remove sodium` → disambiguation prompt                        |
| **Full TUI**        | • No flags **or** `mmm tui` **or** `--tui` supplied                                             | • Launch Bubble Tea interface described in §3                                                                 | User types `mmm` with no args                                      |

### 2.1  Flags & Environment variables

| Switch/Var               | Effect                                           |
| ------------------------ | ------------------------------------------------ |
| `-y`, `--yes`, `--force` | Skip *all* confirmations                         |
| `--pick-first`           | Auto‑select first match when multiple are found  |
| `--json`, `--quiet`      | Suppress progress lines, emit JSON               |
| `MMM_NO_PROMPT=1`        | Fail if a prompt would be required (good for CI) |

---

## 3  Wire‑frame Sketches

**Legend:** `Sidebar ▸` indicates current menu focus.

### 3.1  `add`

```text
+----------------------+-----------------------------------------------+
| ▸ Add                |  Add a new mod                                |
|   Remove             |                                               |
|   Update             |   Platform:  (←/→) CurseForge ▸ Modrinth      |
|   …                  |                                               |
|                      |   Project ID: [______________]                |
+----------------------+                                               |
| ↑/↓ Navigate • ↵ Confirm • Esc Cancel • q Quit • h Help              |
+---------------------------------------------------------------------+
```

### 3.2  `remove`

```text
+----------------------+-----------------------------------------------+
|   Add                |  Remove mods  (Space = toggle, ↵ = delete)    |
| ▸ Remove             |                                               |
|   Update             |  ☐ Fabric API             (curseforge)        |
|   …                  |  ☑ Sodium                   (modrinth)        |
|                      |  ☐ Iris Shaders             (modrinth)        |
|                      |                                               |
|                      |  3 selected • Press ↵ to remove               |
+----------------------+-----------------------------------------------+
| ↑/↓ Move • Space Toggle • ↵ Remove • a Toggle All • q Back           |
+---------------------------------------------------------------------+
```

### 3.3  `update`

```text
+----------------------+---------------------------------------------------------------+
|   Add                |  Updates available                                            |
|   Remove             |                                                               |
| ▸ Update             |  Name            Current      Latest      Status              |
|   …                  |  ───────────────────────────────────────────────────────────   |
|                      |  Fabric API      0.76.1       0.77.0      ● Ready             |
|                      |  Sodium          0.5.5        0.5.6       ● Ready             |
|                      |  Iris Shaders    1.6.2        1.6.2       ✓ Up‑to‑date        |
|                      |                                                               |
|                      |  [u] Update selected   [U] Update all   [Esc] Cancel          |
+----------------------+---------------------------------------------------------------+
| ↑/↓ Select • u One • U All • q Quit • h Help                                         |
+-------------------------------------------------------------------------------------+
```

### 3.4  `scan`

```text
+----------------------+---------------------------------------------------------------+
|   …                  |  Scan results (mods folder)                                   |
| ▸ Scan               |                                                               |
|                      |  ✓ worldedit‑7.2.12.jar  → WorldEdit (modrinth)               |
|                      |  ? custom-hud.jar        → Unknown (no match)                |
|                      |  ✓ jei-11.6.jar          → JEI (curseforge)                   |
|                      |                                                               |
|                      |  [a] Add recognised  •  [o] Open in file manager •  Esc Back  |
+----------------------+---------------------------------------------------------------+
| ↑/↓ Scroll • a Add • o Open • q Quit                                                 |
+-------------------------------------------------------------------------------------+
```

### 3.5  `list`

```text
+----------------------+---------------------------------------------------------------+
|   …                  |  Installed mods                                               |
| ▸ List               |                                                               |
|                      |  ✓ Fabric API          (curseforge)                            |
|                      |  ✓ Sodium              (modrinth)                             |
|                      |  ✗ MiniMap             (missing)                              |
|                      |                                                               |
+----------------------+---------------------------------------------------------------+
| q Quit • h Help                                                                    |
+-------------------------------------------------------------------------------------+
```

### 3.6  `prune`

```text
+----------------------+---------------------------------------------------------------+
|   …                  |  Unmanaged files                                              |
| ▸ Prune              |                                                               |
|                      |  ☐ old-backup.zip                                             |
|                      |  ☐ debug-2023-08-14.log                                       |
|                      |                                                               |
|                      |  Delete 2 files?  y / n                                       |
+----------------------+---------------------------------------------------------------+
| ↑/↓ Move • Space Toggle • y Yes • n No • q Quit                                      |
+-------------------------------------------------------------------------------------+
```

### 3.7  `test`

```text
+----------------------+---------------------------------------------------------------+
|   …                  |  Testing against 1.20.1 …                                     |
| ▸ Test               |                                                               |
|                      |  Fabric API        ✓                                          |
|                      |  Sodium            ✗  (no build for 1.20+)                    |
|                      |  Iris Shaders      ✓                                          |
|                      |                                                               |
|                      |  2 / 3 mods compatible • Exit code 1                          |
+----------------------+---------------------------------------------------------------+
| q Quit • r Retry                                                                 |
+-------------------------------------------------------------------------------------+
```

### 3.8  `change`

```text
+----------------------+---------------------------------------------------------------+
|   …                  |  Change Minecraft version                                     |
| ▸ Change             |                                                               |
|                      |  Target version: [1.20.1_____]                                |
|                      |                                                               |
|                      |  [Enter] Test & Apply   •  Esc Cancel                         |
+----------------------+---------------------------------------------------------------+
| ↑/↓ Move • ↵ Apply • q Quit                                                          |
+-------------------------------------------------------------------------------------+
```

### 3.9  `install`

```text
+----------------------+---------------------------------------------------------------+
|   …                  |  Installing 12 mods                                           |
| ▸ Install            |                                                               |
|                      |   ██████████░░  75 %                                           |
|                      |   Downloading: Sodium-0.5.6.jar                               |
+----------------------+---------------------------------------------------------------+
| Ctrl+C Abort                                                                                |
+-------------------------------------------------------------------------------------+
```

### 3.10  `init` (wizard)

```text
+----------------------+---------------------------------------------------------------+
|   …                  |  Initialise configuration  (Step 2/3)                         |
| ▸ Init               |                                                               |
|                      |  Mods folder → [ mods ]   (Tab to edit)                       |
|                      |  Loader      → [ fabric | quilt | forge ]                     |
|                      |  MC version  → [ 1.19.2 ] (auto-complete)                     |
|                      |                                                               |
|                      |  [← Back]   [Next →]                                          |
+----------------------+---------------------------------------------------------------+
| Tab/Shift+Tab Move • ↑/↓ Select • q Quit                                             |
+-------------------------------------------------------------------------------------+
```

---

## 4  Prompted CLI Layer

```go
func ensureTTYOrAbort() {
    if !isatty.IsTerminal(os.Stdin.Fd()) {
        fmt.Fprintln(os.Stderr, "Missing flag --yes in non‑interactive mode")
        os.Exit(4)
    }
}
```

* Use **AlecAivazis/survey** for prompts.
* Wrap every prompt with `ensureTTYOrAbort` to avoid hanging in scripts.

### 4.1  Disambiguation Example

```
Multiple hits for “sodium”:
  1) Sodium (modrinth)
  2) Sodium (curseforge)
Choose [1-2]: _
```

---

## 5  Escalation Controls

* `--tui` flag converts any command run into its Bubble Tea progress screen.
* `mmm tui --cmd "install --file mods.txt"` launches straight into that screen.

---

## 6  Testing Strategy & Coverage

| Area           | Tests                                | Tooling           |
| -------------- | ------------------------------------ |-------------------|
| Prompt helpers | Unit tests with fake `survey.Stdio`  | Go test + testify |
| Flag matrix    | Table‑driven, asserting exit‑codes   | Go test           |
| TUI screens    | Snapshot tests via `tea.ProgramTest` | Bubble Tea        |

All new code must keep **100 % coverage** per project policy.

---

## 7  Shared Implementation Notes

* **Key bindings** live in a single map; footer reads from it to stay DRY.
* Listen for `tea.WindowSizeMsg` to recompute widths → responsive layout.
* Long‑running jobs issue `tea.Batch` with a `spinner` (quiet mode outputs simple log lines).
* **Colour palette:**

  * Background `#262626` (charcoal)
  * Text `#dadada` (off‑white)
  * Accent `#5af3ff` (cyan) — keep accents ≤2 per view.

---

## 8  Glossary

**MMM** – Minecraft Mod Manager, the CLI/TUI being designed.

**Bubble Tea** – Go TUI framework by Charmbracelet.

**TTY** – Teletype; a terminal capable of interactive input.
