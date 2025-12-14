### 2025-12-14 14:23 - Established Go parity plan with detailed beads backlog referencing Node repo. Epic mmm-1 now encodes execution order; each task notes to consult epic; P1 audit (mmm-14) blocks epic.
### 2025-12-14 15:03 - bd --no-db ignores blockers in ready output; rely on issue dependencies/epic order instead of ready list until DB bug fixed.
### 2025-12-14 15:27 - New session kicked off; need to identify next beads issue for Go port despite ready cmd bug—cross-reference epic ordering before picking work.
### 2025-12-14 15:37 - Completed mmm-14 audit; logged gaps for config schema (mmm-18), Makefile (mmm-19), locale detection (mmm-20), HTTP retry leaks (mmm-21), init TUI networking (mmm-22), and environment metadata (mmm-23). Prioritize these before jumping into parity work.
### 2025-12-14 15:49 - mmm-18 fixed: ModsJson.mods now uses []models.Mod (config entries) instead of lock-style ModInstall; default config seeded with []models.Mod. README updated to reflect per-mod allowVersionFallback (no global flag).
### 2025-12-14 16:01 - mcp beads ready/list endpoints crash when dependencies metadata incomplete; inspect .beads/issues.jsonl directly until schema bug fixed.
### 2025-12-14 16:03 - CLI bd --no-db ready works; API-only failure scoped to MCP. Use CLI for ready list until fix.
### 2025-12-14 16:21 - mmm-23 now active; release prepare script rewrites internal/environment.go placeholders (version + API keys + HELP_URL) but Go build still hardcodes REPL_* defaults—need ldflags-driven versioning + env helpers wired into CLI/help output.
### 2025-12-14 16:30 - Wired HELP_URL helper + Cobra help footer mirroring Node CLI; ended up keeping helper as placeholder-only (no runtime env lookup) because release scripts still replace REPL_* directly; tests cover footer contains REPL_HELP_URL.
### 2025-12-14 17:08 - Ready list reviewed; plan to tackle mmm-21 (retry body leak) next since it's the lone P1 bug blocking network stability work.
### 2025-12-14 17:12 - mmm-21 resolved via `drainAndClose` helper + regression test checking retry responses are drained/closed; remembered to cover nil bodies to keep coverage at 100%.
