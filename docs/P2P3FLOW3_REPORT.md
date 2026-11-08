# Flow round 3 report

2026-11-07. Base `bd440c0`. Delivery uses the ordered incremental patches `/tmp/claude-501/p0b/p2p3flow3-1.patch` and `p2p3flow3-2.patch`, with matching one-line `.txt` descriptions. Patch 1 contains implementation, tests, budgets and goldens. Patch 2 contains this report. Both milestones passed the required gates. A fresh `git archive HEAD` export accepted the series with `git apply --whitespace=nowarn` and matched the final tree byte for byte, including new files.

## Interaction and rendering

- The model screen starts in list focus. Arrows browse models and preview their defaults. Enter chooses the model and focuses the grid. The chosen row gains a check, the list dims, and the panel border and selected cell use the family colour.
- In the grid, vertical arrows change files and horizontal arrows change contexts. Directional movement skips unfitting cells and stops at the edge. Cache and engine cycling preserve a valid selection and retain the existing snap behaviour when a configuration stops fitting. Escape returns to the list, preserving each chosen configuration across model navigation. The `f` action was removed.
- The file viewport shows five rows and scrolls with its selected row. A synthetic eight-file fixture covers this path in snapshots and navigation tests. Today's fitting catalog often has fewer rows.
- Every root TUI screen starts with the shared bold `✦ triaLM` green → cyan → violet wordmark. Ready uses the same gradient. Agent, machine and model-memory context shares that line where space permits. The model heading now carries only the model position. The plain theme keeps the wordmark and Unicode glyphs.
- Footers wrap complete key hints. Their ASCII expansions are composed before width calculation. The trial footer was shortened to fit 80 columns. Selected grid cells use brackets on both their speed and memory lines under NO_COLOR. The list preview also marks its default cell without a filled background.

## Calibration

The child sends `calib.Plan`'s per-step `EstimatedSeconds` as the optional `step_estimates_s` array on JSON events. The UI initialises from the same plan and updates the estimate every second. Progress is the sum of completed step estimates plus elapsed time in the current step, capped at 95% of that step, divided by the total estimate. Measurement events correct it upwards, including resumed finished steps. Repeated messages within a step retain its start time. The fraction is monotonic, and its displayed fill remains `1 − (1 − t)^2.5`. ETA uses the remaining fraction of that same total. Overruns hold their estimated position while the spinner continues. The time line includes a dim step count. NO_COLOR also keeps a moving calibration spinner.

`internal/hw/procs.go` uses the existing stdlib runner boundary for a single `ps -axo pid,rss,args` snapshot per scan on macOS and Linux. Other platforms return an empty result. It recognises llama-server, loaded Ollama runners, LM Studio model workers and mlx-lm servers. PID plus recorded argv under this TRIALM_HOME's `run/*.pid` identify owned servers, including installations that share the engine executable. Stale PID records with different argv remain visible. RSS is converted from KiB and displayed through `figure.Bytes`.

The calibrate offer scans when opened and refreshes before calibration starts. Setup planning refreshes the same warning. A yellow count appears on the calibration card and above Check speed on setup. Details list process names and resident memory. Scan failures return an empty result. The warning allows the operation to proceed. Process handling is read-only.

## Files and scope

- `internal/core/configs.go` adds directional grid movement and removes file cycling.
- `internal/ui/tui/{root,firstrun,views,components,theme,launcher,session,setup,calibration}.go` integrate focus, identity, warnings and timed progress.
- `internal/ui/tui/pick/view.go` renders the two focus states, selected-cell brackets and five-file viewport.
- `internal/ui/tui/chrome/chrome.go` holds shared identity and complete-hint footer composition. `internal/ui/tui/calprogress/progress.go` holds the deterministic clock model. These small packages keep the root package below 2,500 lines.
- `internal/hw/procs.go` and `procs_test.go` add process inspection and canned macOS/Linux snapshots. Fixtures cover each server family, owned and stale PIDs, idle Ollama and LM Studio apps, malformed rows, inspection failures and the unsupported-platform no-op. Tests assert exactly one process-list call per scan.
- `internal/calibrationcli/calibrate.go` attaches the plan estimates to progress events.
- `internal/ui/tui/{flow3,firstrun,root}_test.go` and `calprogress/progress_test.go` cover focus, retention, cell skipping, scrolling, NO_COLOR markers, footer widths, first-token counts, warning behaviour, per-second interpolation, progress corrections, resumed steps and overruns.

The small changes outside the owned implementation areas are the optional event field in `internal/calib/run.go`, the two budget ceilings and their diagnostics in `internal/budgets/budgets_test.go`, and the matching sentence in PLAN §9. This report is the requested handoff document. The downloader and Ready exported signatures retain their existing forms.

## Gates and measurements

All commands used Go 1.26.10 with `GOTOOLCHAIN=local`, scratch HOME, XDG configuration/cache, TRIALM_HOME, Hugging Face and Ollama paths. The environment file is `/tmp/claude-501/p0b/flow3/env.sh`.

| Check | Result |
|---|---|
| `gofmt -l .` | Empty |
| `go vet ./...` | Pass |
| `GOOS=windows go vet ./...` | Pass |
| `go test -race ./...` | Pass |
| `go mod verify` | Pass |
| Linux amd64 and Windows amd64 cross-builds | Pass |
| TUI snapshots | 246 frames, 41 screen variants × 60/80/120 × colour/NO_COLOR |
| Real binary PTY | 80 columns × 40 rows, frames below |
| Patch replay | Both incremental patches applied and compared against the final tree |

One intermediate race run saw six Help golden mismatches because the Help label was edited while that run was active. The final gate reran against the settled source and goldens. Gate logs, count tests, raw PTY bytes, build overlays and the driver are under `/tmp/claude-501/p0b/flow3/`.

| Budget | Base | Final | Final ceiling |
|---|---|---|---|
| Product Go token lines | 29,946 | 30,210 | 30,210 |
| CLI and all UI token lines | 5,940 | 6,106 | 6,106 |
| Root TUI package | 2,379 | 2,430 | 2,500 |
| Hardware package | — | 2,345 | 2,500 for this brief |
| Core package | 1,891 | 1,903 | 2,500 |
| Picker package | 202 | 216 | 2,500 |
| Calibration progress package | — | 65 | 2,500 |
| Shared chrome package | — | 36 | 2,500 |
| cmd/trialm physical lines | 13 | 13 | 30 |
| Stripped macOS arm64 binary | — | 11,552,818 bytes | 14,000,000 bytes |
| Stripped Linux amd64 binary | — | 12,226,722 bytes | 14,000,000 bytes |
| Stripped Windows amd64 binary | — | 12,633,600 bytes | 14,000,000 bytes |

Before adding code, the detect brand box, duplicate launcher identity/context card and obsolete picker navigation were removed. The old event-only ETA renderer was replaced. The resulting net additions require ceiling increases of 210 product lines and 106 CLI/UI lines. The ceilings equal measured use and stay within the owner's permitted maxima. Every package touched by this work is below 2,500 lines.

## Key-count acceptance

| Path | New count |
|---|---|
| Fresh run, two detected tools, skip calibration, first trial token | 5 keys + 1 browser click |
| Fresh run, cached fitting file, local setup and automatic tool handoff | 5 keys, including the single plan approval |
| Journey A through cloud trial and local tool opening | 8 keys, or 9 with the optional rating, plus 1 browser click and 1 setup approval |
| Returning home, cached model needing setup | 2 keys, including approval |
| Returning home, Ready setup | 1 key |

The fresh trial path is Agents Enter → Later `s` → Choose Enter → Try Enter → browser/sign-in Enter. The five-key test reaches the first response from a local fake trial upstream through the real message loop and child handoff. The local path is Agents Enter → Later `s` → Choose Enter → Set up `s` → approval Enter. One detected tool skips the Agents key. Calibration remains optional in these counts. The Journey A total adds Install, approval and Open to the tested first-trial path. Tool keystrokes inside the trial are excluded, as in the round 2 count.

## Real-binary PTY verification

`trialm-prod` was built from `./cmd/trialm` and run with `find --device 'Apple M5' --ram 24 --for claude-code`. Its tty was explicitly set to 80×40. HOME, XDG, TRIALM_HOME, Hugging Face and Ollama state stayed in scratch directories. The five model frames use that production binary. They show 14 fitting Claude Code models. The snapshot fixture uses Codex, whose context and fit requirements yield 11 in this tree.

For the warning and running calibration, `trialm-fixture` was built from the same CLI entry point using `/tmp/claude-501/p0b/flow3/overlay.json`. The external Go build overlay replaces the hardware runner with canned `ps` output for two resident 9B llama-server processes. A second test-only overlay replaces calibration measurements with JSON events. The production source has no environment-based process or calibration injection hook.

The fake run announces nine step estimates `[40, 5, 5, 5, 5, 5, 5, 5, 5]`, then waits 40 seconds before its next event. The frames at approximately 2, 12 and 22 seconds therefore advance between measurement events. Their ETA crosses into Under a minute left. The offer retains the ordinary plan's ~7-minute estimate while the test child announces its accelerated plan. The model session exited 0. Interrupting the fake calibration exited 130 and drained the child.

Raw captures, per-frame raw prefixes and decoded text are in `/tmp/claude-501/p0b/flow3/pty/`. The driver `drive_pty.py` uses a Python PTY and pyte installed in the scratch tree for VT decoding. Frames below are decoded from terminal output with NO_COLOR enabled. Colour snapshots additionally cover the dim list, dim preview border, family-colour focused border, filled selected cell and gradient identity.

### List focus after ↓ selects Gemma

```text
✦ triaLM  Claude Code · Apple M5 · 16 GB for models
Models · 2 of 14
  ○ Qwen3.6 35B-A3B             up to 256K  ~48 t/s   13.2 GB █
❯ ○ Gemma 4 26B-A4B             up to 256K  ~26 t/s   13.6 GB │
  ○ gpt-oss 20B                 up to 128K  ~40 t/s   12.1 GB │
  ○ Gemma 4 12B                 up to 256K  ~14 t/s   7.12 GB │
  ○ Qwen3.5 9B                  up to 256K  ~22 t/s   5.68 GB │
  ○ Qwen3.5 4B                  up to 256K  ~38 t/s   2.74 GB │
  ○ Qwen3-Coder 30B-A3B         up to 130K  ~47 t/s   12.9 GB │

╭────────────────────────────────────────────────────────────────────────────╮
│Gemma  Gemma 4 26B-A4B  estimate                                            │
│KV cache  [f16] q8_0 q4_0                                                   │
│Engine    [llama.cpp]                                                       │
│                                                                            │
│t/s when full       32K          64K          128K         256K             │
│IQ4_XS              ~15 t/s      [~11 t/s]    ✗            ✗                │
│13.6 GB             14.3 GB      [14.9 GB]    16.1 GB      18.6 GB          │
│Q3_K_XL             ~15 t/s      ~11 t/s      ~7.2 t/s     ✗                │
│12.9 GB             13.6 GB      14.3 GB      15.5 GB      18.0 GB          │
│IQ3_XXS             ~16 t/s      ~11 t/s      ~7.4 t/s     ✗                │
│11.4 GB             12.2 GB      12.9 GB      14.1 GB      16.6 GB          │
│3 larger files don't fit                                                    │
│                                                                            │
│Speed   ~26 t/s empty → ~11 t/s at 64K    First reply  ~46 s                │
│Memory  ███████████████████░  14.9 of 16 GB                                 │
╰────────────────────────────────────────────────────────────────────────────╯

↑↓ Model   ⏎ Choose   / Search   v Details   q Quit
```

### Enter chooses Gemma and focuses its default cell

```text
✦ triaLM  Claude Code · Apple M5 · 16 GB for models
Models · 2 of 14
  ○ Qwen3.6 35B-A3B             up to 256K  ~48 t/s   13.2 GB █
✓ ○ Gemma 4 26B-A4B             up to 256K  ~26 t/s   13.6 GB │
  ○ gpt-oss 20B                 up to 128K  ~40 t/s   12.1 GB │
  ○ Gemma 4 12B                 up to 256K  ~14 t/s   7.12 GB │
  ○ Qwen3.5 9B                  up to 256K  ~22 t/s   5.68 GB │
  ○ Qwen3.5 4B                  up to 256K  ~38 t/s   2.74 GB │
  ○ Qwen3-Coder 30B-A3B         up to 130K  ~47 t/s   12.9 GB │

╭────────────────────────────────────────────────────────────────────────────╮
│Gemma  Gemma 4 26B-A4B  estimate                                            │
│KV cache  [f16] q8_0 q4_0                                                   │
│Engine    [llama.cpp]                                                       │
│                                                                            │
│t/s when full       32K          64K          128K         256K             │
│IQ4_XS              ~15 t/s      [~11 t/s]    ✗            ✗                │
│13.6 GB             14.3 GB      [14.9 GB]    16.1 GB      18.6 GB          │
│Q3_K_XL             ~15 t/s      ~11 t/s      ~7.2 t/s     ✗                │
│12.9 GB             13.6 GB      14.3 GB      15.5 GB      18.0 GB          │
│IQ3_XXS             ~16 t/s      ~11 t/s      ~7.4 t/s     ✗                │
│11.4 GB             12.2 GB      12.9 GB      14.1 GB      16.6 GB          │
│3 larger files don't fit                                                    │
│                                                                            │
│Speed   ~26 t/s empty → ~11 t/s at 64K    First reply  ~46 s                │
│Memory  ███████████████████░  14.9 of 16 GB                                 │
╰────────────────────────────────────────────────────────────────────────────╯

↑↓ File   ←→ Context   c Cache   e Engine   ⏎ Try   s Set up   esc Models
```

### ↓ then → selects the next fitting file and 128K context

```text
✦ triaLM  Claude Code · Apple M5 · 16 GB for models
Models · 2 of 14
  ○ Qwen3.6 35B-A3B             up to 256K  ~48 t/s   13.2 GB █
✓ ○ Gemma 4 26B-A4B             up to 256K  ~26 t/s   13.6 GB │
  ○ gpt-oss 20B                 up to 128K  ~40 t/s   12.1 GB │
  ○ Gemma 4 12B                 up to 256K  ~14 t/s   7.12 GB │
  ○ Qwen3.5 9B                  up to 256K  ~22 t/s   5.68 GB │
  ○ Qwen3.5 4B                  up to 256K  ~38 t/s   2.74 GB │
  ○ Qwen3-Coder 30B-A3B         up to 130K  ~47 t/s   12.9 GB │

╭────────────────────────────────────────────────────────────────────────────╮
│Gemma  Gemma 4 26B-A4B  estimate                                            │
│KV cache  [f16] q8_0 q4_0                                                   │
│Engine    [llama.cpp]                                                       │
│                                                                            │
│t/s when full       32K          64K          128K         256K             │
│IQ4_XS              ~15 t/s      ~11 t/s      ✗            ✗                │
│13.6 GB             14.3 GB      14.9 GB      16.1 GB      18.6 GB          │
│Q3_K_XL             ~15 t/s      ~11 t/s      [~7.2 t/s]   ✗                │
│12.9 GB             13.6 GB      14.3 GB      [15.5 GB]    18.0 GB          │
│IQ3_XXS             ~16 t/s      ~11 t/s      ~7.4 t/s     ✗                │
│11.4 GB             12.2 GB      12.9 GB      14.1 GB      16.6 GB          │
│3 larger files don't fit                                                    │
│                                                                            │
│Speed   ~27 t/s empty → ~7.2 t/s at 128K    First reply  ~46 s              │
│Memory  ███████████████████░  15.5 of 16 GB                                 │
╰────────────────────────────────────────────────────────────────────────────╯

↑↓ File   ←→ Context   c Cache   e Engine   ⏎ Try   s Set up   esc Models
```

### c cycles to q8_0 while preserving the selected file and context

```text
✦ triaLM  Claude Code · Apple M5 · 16 GB for models
Models · 2 of 14
  ○ Qwen3.6 35B-A3B             up to 256K  ~48 t/s   13.2 GB █
✓ ○ Gemma 4 26B-A4B             up to 256K  ~26 t/s   13.6 GB │
  ○ gpt-oss 20B                 up to 128K  ~40 t/s   12.1 GB │
  ○ Gemma 4 12B                 up to 256K  ~14 t/s   7.12 GB │
  ○ Qwen3.5 9B                  up to 256K  ~22 t/s   5.68 GB │
  ○ Qwen3.5 4B                  up to 256K  ~38 t/s   2.74 GB │
  ○ Qwen3-Coder 30B-A3B         up to 130K  ~47 t/s   12.9 GB │

╭────────────────────────────────────────────────────────────────────────────╮
│Gemma  Gemma 4 26B-A4B  estimate                                            │
│KV cache  f16 [q8_0] q4_0                                                   │
│Engine    [llama.cpp]                                                       │
│                                                                            │
│t/s when full       32K          64K          128K         256K             │
│IQ4_XS              ~20 t/s      ~16 t/s      ~11 t/s      ✗                │
│13.6 GB             13.8 GB      14.1 GB      14.8 GB      16.1 GB          │
│Q3_K_XL             ~20 t/s      ~16 t/s      [~11 t/s]    ~7.2 t/s         │
│12.9 GB             13.2 GB      13.5 GB      [14.2 GB]    15.5 GB          │
│IQ3_XXS             ~22 t/s      ~17 t/s      ~12 t/s      ~7.5 t/s         │
│11.4 GB             11.8 GB      12.1 GB      12.8 GB      14.1 GB          │
│3 larger files don't fit                                                    │
│                                                                            │
│Speed   ~29 t/s empty → ~11 t/s at 128K    First reply  ~42 s               │
│Memory  ██████████████████░░  14.2 of 16 GB                                 │
╰────────────────────────────────────────────────────────────────────────────╯

↑↓ File   ←→ Context   c Cache   e Engine   ⏎ Try   s Set up   esc Models
```

### Escape restores list focus and retains that configuration

```text
✦ triaLM  Claude Code · Apple M5 · 16 GB for models
Models · 2 of 14
  ○ Qwen3.6 35B-A3B             up to 256K  ~48 t/s   13.2 GB █
❯ ○ Gemma 4 26B-A4B             up to 256K  ~26 t/s   13.6 GB │
  ○ gpt-oss 20B                 up to 128K  ~40 t/s   12.1 GB │
  ○ Gemma 4 12B                 up to 256K  ~14 t/s   7.12 GB │
  ○ Qwen3.5 9B                  up to 256K  ~22 t/s   5.68 GB │
  ○ Qwen3.5 4B                  up to 256K  ~38 t/s   2.74 GB │
  ○ Qwen3-Coder 30B-A3B         up to 130K  ~47 t/s   12.9 GB │

╭────────────────────────────────────────────────────────────────────────────╮
│Gemma  Gemma 4 26B-A4B  estimate                                            │
│KV cache  f16 [q8_0] q4_0                                                   │
│Engine    [llama.cpp]                                                       │
│                                                                            │
│t/s when full       32K          64K          128K         256K             │
│IQ4_XS              ~20 t/s      ~16 t/s      ~11 t/s      ✗                │
│13.6 GB             13.8 GB      14.1 GB      14.8 GB      16.1 GB          │
│Q3_K_XL             ~20 t/s      ~16 t/s      [~11 t/s]    ~7.2 t/s         │
│12.9 GB             13.2 GB      13.5 GB      [14.2 GB]    15.5 GB          │
│IQ3_XXS             ~22 t/s      ~17 t/s      ~12 t/s      ~7.5 t/s         │
│11.4 GB             11.8 GB      12.1 GB      12.8 GB      14.1 GB          │
│3 larger files don't fit                                                    │
│                                                                            │
│Speed   ~29 t/s empty → ~11 t/s at 128K    First reply  ~42 s               │
│Memory  ██████████████████░░  14.2 of 16 GB                                 │
╰────────────────────────────────────────────────────────────────────────────╯

↑↓ Model   ⏎ Choose   / Search   v Details   q Quit
```

### Calibration offer with injected process snapshot

```text
✦ triaLM  Claude Code · Apple M5 · 16 GB for models
╭────────────────────────────────────────────────────────────────────────────╮
│ ✦ Calibrate for Accurate Estimates | ~7 mins | No Downloads                │
│                                                                            │
│ 2 Other Models Loaded | Expect Slower Speeds                               │
╰────────────────────────────────────────────────────────────────────────────╯

⏎ Now   s Later (run trialm calibrate)   v Details
```

### Calibration details list both processes and memory

```text
✦ triaLM  Claude Code · Apple M5 · 16 GB for models
╭────────────────────────────────────────────────────────────────────────────╮
│ ✦ Calibrate for Accurate Estimates | ~7 mins | No Downloads                │
│                                                                            │
│ 2 Other Models Loaded | Expect Slower Speeds                               │
│ llama-server · 7.52 GB memory                                              │
│ llama-server · 6.98 GB memory                                              │
╰────────────────────────────────────────────────────────────────────────────╯

⏎ Now   s Later (run trialm calibrate)   v Details
```

### Running calibration at approximately 2 seconds

```text
✦ triaLM  Claude Code · Apple M5 · 16 GB for models
╭────────────────────────────────────────────────────────────────────────────╮
│ ✦ Calibrate for Accurate Estimates | ~7 mins | No Downloads                │
│                                                                            │
│ ⣯  Calibrating                                                             │
│ ▌▌░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░                           │
│ About 2 min left · 1 of 9                                                  │
│                                                                            │
│ 2 Other Models Loaded | Expect Slower Speeds                               │
╰────────────────────────────────────────────────────────────────────────────╯

ctrl-c Pause   v Details
```

### Running calibration at approximately 12 seconds

```text
✦ triaLM  Claude Code · Apple M5 · 16 GB for models
╭────────────────────────────────────────────────────────────────────────────╮
│ ✦ Calibrate for Accurate Estimates | ~7 mins | No Downloads                │
│                                                                            │
│ ⡿  Calibrating                                                             │
│ ▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░                           │
│ About 2 min left · 1 of 9                                                  │
│                                                                            │
│ 2 Other Models Loaded | Expect Slower Speeds                               │
╰────────────────────────────────────────────────────────────────────────────╯

ctrl-c Pause   v Details
```

### Running calibration at approximately 22 seconds

```text
✦ triaLM  Claude Code · Apple M5 · 16 GB for models
╭────────────────────────────────────────────────────────────────────────────╮
│ ✦ Calibrate for Accurate Estimates | ~7 mins | No Downloads                │
│                                                                            │
│ ⡿  Calibrating                                                             │
│ ▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌░░░░░░░░░░░░░░░░░░░░░░                           │
│ Under a minute left · 1 of 9                                               │
│                                                                            │
│ 2 Other Models Loaded | Expect Slower Speeds                               │
╰────────────────────────────────────────────────────────────────────────────╯

ctrl-c Pause   v Details
```

## Golden inventory

All 204 existing TUI goldens changed for shared identity and key-hint spacing. The seven new variants add 42 frames. Each group below lists all six exact filenames and its additional reason. `colorfalse` means NO_COLOR. The CLI characterization goldens retain their existing contents.

### agents

Shared identity and complete-hint footer composition.

- `internal/ui/tui/testdata/agents-120-colorfalse.golden`
- `internal/ui/tui/testdata/agents-120-colortrue.golden`
- `internal/ui/tui/testdata/agents-60-colorfalse.golden`
- `internal/ui/tui/testdata/agents-60-colortrue.golden`
- `internal/ui/tui/testdata/agents-80-colorfalse.golden`
- `internal/ui/tui/testdata/agents-80-colortrue.golden`

### calibrate

Shared identity and Details hint.

- `internal/ui/tui/testdata/calibrate-120-colorfalse.golden`
- `internal/ui/tui/testdata/calibrate-120-colortrue.golden`
- `internal/ui/tui/testdata/calibrate-60-colorfalse.golden`
- `internal/ui/tui/testdata/calibrate-60-colortrue.golden`
- `internal/ui/tui/testdata/calibrate-80-colorfalse.golden`
- `internal/ui/tui/testdata/calibrate-80-colortrue.golden`

### calibrate-low-power

Shared identity and Details hint.

- `internal/ui/tui/testdata/calibrate-low-power-120-colorfalse.golden`
- `internal/ui/tui/testdata/calibrate-low-power-120-colortrue.golden`
- `internal/ui/tui/testdata/calibrate-low-power-60-colorfalse.golden`
- `internal/ui/tui/testdata/calibrate-low-power-60-colortrue.golden`
- `internal/ui/tui/testdata/calibrate-low-power-80-colorfalse.golden`
- `internal/ui/tui/testdata/calibrate-low-power-80-colortrue.golden`

### calibrate-models

Loaded-model warning and optional process details.

- `internal/ui/tui/testdata/calibrate-models-120-colorfalse.golden`
- `internal/ui/tui/testdata/calibrate-models-120-colortrue.golden`
- `internal/ui/tui/testdata/calibrate-models-60-colorfalse.golden`
- `internal/ui/tui/testdata/calibrate-models-60-colortrue.golden`
- `internal/ui/tui/testdata/calibrate-models-80-colorfalse.golden`
- `internal/ui/tui/testdata/calibrate-models-80-colortrue.golden`

### calibrate-models-details

Loaded-model warning and optional process details.

- `internal/ui/tui/testdata/calibrate-models-details-120-colorfalse.golden`
- `internal/ui/tui/testdata/calibrate-models-details-120-colortrue.golden`
- `internal/ui/tui/testdata/calibrate-models-details-60-colorfalse.golden`
- `internal/ui/tui/testdata/calibrate-models-details-60-colortrue.golden`
- `internal/ui/tui/testdata/calibrate-models-details-80-colorfalse.golden`
- `internal/ui/tui/testdata/calibrate-models-details-80-colortrue.golden`

### calibrating

Moving-spinner line, eased bar and step counter.

- `internal/ui/tui/testdata/calibrating-120-colorfalse.golden`
- `internal/ui/tui/testdata/calibrating-120-colortrue.golden`
- `internal/ui/tui/testdata/calibrating-60-colorfalse.golden`
- `internal/ui/tui/testdata/calibrating-60-colortrue.golden`
- `internal/ui/tui/testdata/calibrating-80-colorfalse.golden`
- `internal/ui/tui/testdata/calibrating-80-colortrue.golden`

### chat

Shared identity and complete-hint footer composition.

- `internal/ui/tui/testdata/chat-120-colorfalse.golden`
- `internal/ui/tui/testdata/chat-120-colortrue.golden`
- `internal/ui/tui/testdata/chat-60-colorfalse.golden`
- `internal/ui/tui/testdata/chat-60-colortrue.golden`
- `internal/ui/tui/testdata/chat-80-colorfalse.golden`
- `internal/ui/tui/testdata/chat-80-colortrue.golden`

### claim

Shared identity and complete-hint footer composition.

- `internal/ui/tui/testdata/claim-120-colorfalse.golden`
- `internal/ui/tui/testdata/claim-120-colortrue.golden`
- `internal/ui/tui/testdata/claim-60-colorfalse.golden`
- `internal/ui/tui/testdata/claim-60-colortrue.golden`
- `internal/ui/tui/testdata/claim-80-colorfalse.golden`
- `internal/ui/tui/testdata/claim-80-colortrue.golden`

### details

Shared identity and complete-hint footer composition.

- `internal/ui/tui/testdata/details-120-colorfalse.golden`
- `internal/ui/tui/testdata/details-120-colortrue.golden`
- `internal/ui/tui/testdata/details-60-colorfalse.golden`
- `internal/ui/tui/testdata/details-60-colortrue.golden`
- `internal/ui/tui/testdata/details-80-colorfalse.golden`
- `internal/ui/tui/testdata/details-80-colortrue.golden`

### detect

Shared identity replaces the separate brand/context box.

- `internal/ui/tui/testdata/detect-120-colorfalse.golden`
- `internal/ui/tui/testdata/detect-120-colortrue.golden`
- `internal/ui/tui/testdata/detect-60-colorfalse.golden`
- `internal/ui/tui/testdata/detect-60-colortrue.golden`
- `internal/ui/tui/testdata/detect-80-colorfalse.golden`
- `internal/ui/tui/testdata/detect-80-colortrue.golden`

### detecting

Shared identity replaces the separate brand/context box.

- `internal/ui/tui/testdata/detecting-120-colorfalse.golden`
- `internal/ui/tui/testdata/detecting-120-colortrue.golden`
- `internal/ui/tui/testdata/detecting-60-colorfalse.golden`
- `internal/ui/tui/testdata/detecting-60-colortrue.golden`
- `internal/ui/tui/testdata/detecting-80-colorfalse.golden`
- `internal/ui/tui/testdata/detecting-80-colortrue.golden`

### error

Shared identity replaces the separate brand/context box.

- `internal/ui/tui/testdata/error-120-colorfalse.golden`
- `internal/ui/tui/testdata/error-120-colortrue.golden`
- `internal/ui/tui/testdata/error-60-colorfalse.golden`
- `internal/ui/tui/testdata/error-60-colortrue.golden`
- `internal/ui/tui/testdata/error-80-colorfalse.golden`
- `internal/ui/tui/testdata/error-80-colortrue.golden`

### error-details

Shared identity replaces the separate brand/context box.

- `internal/ui/tui/testdata/error-details-120-colorfalse.golden`
- `internal/ui/tui/testdata/error-details-120-colortrue.golden`
- `internal/ui/tui/testdata/error-details-60-colorfalse.golden`
- `internal/ui/tui/testdata/error-details-60-colortrue.golden`
- `internal/ui/tui/testdata/error-details-80-colorfalse.golden`
- `internal/ui/tui/testdata/error-details-80-colortrue.golden`

### gallery

Shared identity and complete-hint footer composition.

- `internal/ui/tui/testdata/gallery-120-colorfalse.golden`
- `internal/ui/tui/testdata/gallery-120-colortrue.golden`
- `internal/ui/tui/testdata/gallery-60-colorfalse.golden`
- `internal/ui/tui/testdata/gallery-60-colortrue.golden`
- `internal/ui/tui/testdata/gallery-80-colorfalse.golden`
- `internal/ui/tui/testdata/gallery-80-colortrue.golden`

### help

Two-level navigation guidance and removal of the f action.

- `internal/ui/tui/testdata/help-120-colorfalse.golden`
- `internal/ui/tui/testdata/help-120-colortrue.golden`
- `internal/ui/tui/testdata/help-60-colorfalse.golden`
- `internal/ui/tui/testdata/help-60-colortrue.golden`
- `internal/ui/tui/testdata/help-80-colorfalse.golden`
- `internal/ui/tui/testdata/help-80-colortrue.golden`

### launch

Shared identity and complete-hint footer composition.

- `internal/ui/tui/testdata/launch-120-colorfalse.golden`
- `internal/ui/tui/testdata/launch-120-colortrue.golden`
- `internal/ui/tui/testdata/launch-60-colorfalse.golden`
- `internal/ui/tui/testdata/launch-60-colortrue.golden`
- `internal/ui/tui/testdata/launch-80-colorfalse.golden`
- `internal/ui/tui/testdata/launch-80-colortrue.golden`

### launcher

Shared identity replaces the separate brand/context box.

- `internal/ui/tui/testdata/launcher-120-colorfalse.golden`
- `internal/ui/tui/testdata/launcher-120-colortrue.golden`
- `internal/ui/tui/testdata/launcher-60-colorfalse.golden`
- `internal/ui/tui/testdata/launcher-60-colortrue.golden`
- `internal/ui/tui/testdata/launcher-80-colorfalse.golden`
- `internal/ui/tui/testdata/launcher-80-colortrue.golden`

### paste

Shared identity and complete-hint footer composition.

- `internal/ui/tui/testdata/paste-120-colorfalse.golden`
- `internal/ui/tui/testdata/paste-120-colortrue.golden`
- `internal/ui/tui/testdata/paste-60-colorfalse.golden`
- `internal/ui/tui/testdata/paste-60-colortrue.golden`
- `internal/ui/tui/testdata/paste-80-colorfalse.golden`
- `internal/ui/tui/testdata/paste-80-colortrue.golden`

### pick-cache

Two focus states, cell marking and model-position heading.

- `internal/ui/tui/testdata/pick-cache-120-colorfalse.golden`
- `internal/ui/tui/testdata/pick-cache-120-colortrue.golden`
- `internal/ui/tui/testdata/pick-cache-60-colorfalse.golden`
- `internal/ui/tui/testdata/pick-cache-60-colortrue.golden`
- `internal/ui/tui/testdata/pick-cache-80-colorfalse.golden`
- `internal/ui/tui/testdata/pick-cache-80-colortrue.golden`

### pick-disk

Two focus states, cell marking and model-position heading.

- `internal/ui/tui/testdata/pick-disk-120-colorfalse.golden`
- `internal/ui/tui/testdata/pick-disk-120-colortrue.golden`
- `internal/ui/tui/testdata/pick-disk-60-colorfalse.golden`
- `internal/ui/tui/testdata/pick-disk-60-colortrue.golden`
- `internal/ui/tui/testdata/pick-disk-80-colorfalse.golden`
- `internal/ui/tui/testdata/pick-disk-80-colortrue.golden`

### pick-file

Two focus states, cell marking and model-position heading.

- `internal/ui/tui/testdata/pick-file-120-colorfalse.golden`
- `internal/ui/tui/testdata/pick-file-120-colortrue.golden`
- `internal/ui/tui/testdata/pick-file-60-colorfalse.golden`
- `internal/ui/tui/testdata/pick-file-60-colortrue.golden`
- `internal/ui/tui/testdata/pick-file-80-colorfalse.golden`
- `internal/ui/tui/testdata/pick-file-80-colortrue.golden`

### pick-grid

Two focus states, cell marking and model-position heading.

- `internal/ui/tui/testdata/pick-grid-120-colorfalse.golden`
- `internal/ui/tui/testdata/pick-grid-120-colortrue.golden`
- `internal/ui/tui/testdata/pick-grid-60-colorfalse.golden`
- `internal/ui/tui/testdata/pick-grid-60-colortrue.golden`
- `internal/ui/tui/testdata/pick-grid-80-colorfalse.golden`
- `internal/ui/tui/testdata/pick-grid-80-colortrue.golden`

### pick-last

Two focus states, cell marking and model-position heading.

- `internal/ui/tui/testdata/pick-last-120-colorfalse.golden`
- `internal/ui/tui/testdata/pick-last-120-colortrue.golden`
- `internal/ui/tui/testdata/pick-last-60-colorfalse.golden`
- `internal/ui/tui/testdata/pick-last-60-colortrue.golden`
- `internal/ui/tui/testdata/pick-last-80-colorfalse.golden`
- `internal/ui/tui/testdata/pick-last-80-colortrue.golden`

### pick-moved

Two focus states, cell marking and model-position heading.

- `internal/ui/tui/testdata/pick-moved-120-colorfalse.golden`
- `internal/ui/tui/testdata/pick-moved-120-colortrue.golden`
- `internal/ui/tui/testdata/pick-moved-60-colorfalse.golden`
- `internal/ui/tui/testdata/pick-moved-60-colortrue.golden`
- `internal/ui/tui/testdata/pick-moved-80-colorfalse.golden`
- `internal/ui/tui/testdata/pick-moved-80-colortrue.golden`

### pick-scrolled

Two focus states, cell marking and model-position heading. Eight synthetic fitting files exercise the scrolling viewport.

- `internal/ui/tui/testdata/pick-scrolled-120-colorfalse.golden`
- `internal/ui/tui/testdata/pick-scrolled-120-colortrue.golden`
- `internal/ui/tui/testdata/pick-scrolled-60-colorfalse.golden`
- `internal/ui/tui/testdata/pick-scrolled-60-colortrue.golden`
- `internal/ui/tui/testdata/pick-scrolled-80-colorfalse.golden`
- `internal/ui/tui/testdata/pick-scrolled-80-colortrue.golden`

### pick-second

Two focus states, cell marking and model-position heading.

- `internal/ui/tui/testdata/pick-second-120-colorfalse.golden`
- `internal/ui/tui/testdata/pick-second-120-colortrue.golden`
- `internal/ui/tui/testdata/pick-second-60-colorfalse.golden`
- `internal/ui/tui/testdata/pick-second-60-colortrue.golden`
- `internal/ui/tui/testdata/pick-second-80-colorfalse.golden`
- `internal/ui/tui/testdata/pick-second-80-colortrue.golden`

### ready

Shared identity and matching Ready gradient.

- `internal/ui/tui/testdata/ready-120-colorfalse.golden`
- `internal/ui/tui/testdata/ready-120-colortrue.golden`
- `internal/ui/tui/testdata/ready-60-colorfalse.golden`
- `internal/ui/tui/testdata/ready-60-colortrue.golden`
- `internal/ui/tui/testdata/ready-80-colorfalse.golden`
- `internal/ui/tui/testdata/ready-80-colortrue.golden`

### ready-details

Shared identity and matching Ready gradient.

- `internal/ui/tui/testdata/ready-details-120-colorfalse.golden`
- `internal/ui/tui/testdata/ready-details-120-colortrue.golden`
- `internal/ui/tui/testdata/ready-details-60-colorfalse.golden`
- `internal/ui/tui/testdata/ready-details-60-colortrue.golden`
- `internal/ui/tui/testdata/ready-details-80-colorfalse.golden`
- `internal/ui/tui/testdata/ready-details-80-colortrue.golden`

### recipes

Two focus states, cell marking and model-position heading.

- `internal/ui/tui/testdata/recipes-120-colorfalse.golden`
- `internal/ui/tui/testdata/recipes-120-colortrue.golden`
- `internal/ui/tui/testdata/recipes-60-colorfalse.golden`
- `internal/ui/tui/testdata/recipes-60-colortrue.golden`
- `internal/ui/tui/testdata/recipes-80-colorfalse.golden`
- `internal/ui/tui/testdata/recipes-80-colortrue.golden`

### search

Shared identity and complete-hint footer composition.

- `internal/ui/tui/testdata/search-120-colorfalse.golden`
- `internal/ui/tui/testdata/search-120-colortrue.golden`
- `internal/ui/tui/testdata/search-60-colorfalse.golden`
- `internal/ui/tui/testdata/search-60-colortrue.golden`
- `internal/ui/tui/testdata/search-80-colorfalse.golden`
- `internal/ui/tui/testdata/search-80-colortrue.golden`

### setup

Shared identity and complete-hint footer composition.

- `internal/ui/tui/testdata/setup-120-colorfalse.golden`
- `internal/ui/tui/testdata/setup-120-colortrue.golden`
- `internal/ui/tui/testdata/setup-60-colorfalse.golden`
- `internal/ui/tui/testdata/setup-60-colortrue.golden`
- `internal/ui/tui/testdata/setup-80-colorfalse.golden`
- `internal/ui/tui/testdata/setup-80-colortrue.golden`

### setup-details

Shared identity and complete-hint footer composition.

- `internal/ui/tui/testdata/setup-details-120-colorfalse.golden`
- `internal/ui/tui/testdata/setup-details-120-colortrue.golden`
- `internal/ui/tui/testdata/setup-details-60-colorfalse.golden`
- `internal/ui/tui/testdata/setup-details-60-colortrue.golden`
- `internal/ui/tui/testdata/setup-details-80-colorfalse.golden`
- `internal/ui/tui/testdata/setup-details-80-colortrue.golden`

### setup-disk

Shared identity and complete-hint footer composition.

- `internal/ui/tui/testdata/setup-disk-120-colorfalse.golden`
- `internal/ui/tui/testdata/setup-disk-120-colortrue.golden`
- `internal/ui/tui/testdata/setup-disk-60-colorfalse.golden`
- `internal/ui/tui/testdata/setup-disk-60-colortrue.golden`
- `internal/ui/tui/testdata/setup-disk-80-colorfalse.golden`
- `internal/ui/tui/testdata/setup-disk-80-colortrue.golden`

### setup-models

Loaded-model warning above the speed check and optional process details.

- `internal/ui/tui/testdata/setup-models-120-colorfalse.golden`
- `internal/ui/tui/testdata/setup-models-120-colortrue.golden`
- `internal/ui/tui/testdata/setup-models-60-colorfalse.golden`
- `internal/ui/tui/testdata/setup-models-60-colortrue.golden`
- `internal/ui/tui/testdata/setup-models-80-colorfalse.golden`
- `internal/ui/tui/testdata/setup-models-80-colortrue.golden`

### setup-models-details

Loaded-model warning above the speed check and optional process details.

- `internal/ui/tui/testdata/setup-models-details-120-colorfalse.golden`
- `internal/ui/tui/testdata/setup-models-details-120-colortrue.golden`
- `internal/ui/tui/testdata/setup-models-details-60-colorfalse.golden`
- `internal/ui/tui/testdata/setup-models-details-60-colortrue.golden`
- `internal/ui/tui/testdata/setup-models-details-80-colorfalse.golden`
- `internal/ui/tui/testdata/setup-models-details-80-colortrue.golden`

### setup-paused

Shared identity and complete-hint footer composition.

- `internal/ui/tui/testdata/setup-paused-120-colorfalse.golden`
- `internal/ui/tui/testdata/setup-paused-120-colortrue.golden`
- `internal/ui/tui/testdata/setup-paused-60-colorfalse.golden`
- `internal/ui/tui/testdata/setup-paused-60-colortrue.golden`
- `internal/ui/tui/testdata/setup-paused-80-colorfalse.golden`
- `internal/ui/tui/testdata/setup-paused-80-colortrue.golden`

### setup-progress

Shared identity and complete-hint footer composition.

- `internal/ui/tui/testdata/setup-progress-120-colorfalse.golden`
- `internal/ui/tui/testdata/setup-progress-120-colortrue.golden`
- `internal/ui/tui/testdata/setup-progress-60-colorfalse.golden`
- `internal/ui/tui/testdata/setup-progress-60-colortrue.golden`
- `internal/ui/tui/testdata/setup-progress-80-colorfalse.golden`
- `internal/ui/tui/testdata/setup-progress-80-colortrue.golden`

### summary

Shared identity and complete-hint footer composition.

- `internal/ui/tui/testdata/summary-120-colorfalse.golden`
- `internal/ui/tui/testdata/summary-120-colortrue.golden`
- `internal/ui/tui/testdata/summary-60-colorfalse.golden`
- `internal/ui/tui/testdata/summary-60-colortrue.golden`
- `internal/ui/tui/testdata/summary-80-colorfalse.golden`
- `internal/ui/tui/testdata/summary-80-colortrue.golden`

### trial

Shared identity and complete-hint footer composition.

- `internal/ui/tui/testdata/trial-120-colorfalse.golden`
- `internal/ui/tui/testdata/trial-120-colortrue.golden`
- `internal/ui/tui/testdata/trial-60-colorfalse.golden`
- `internal/ui/tui/testdata/trial-60-colortrue.golden`
- `internal/ui/tui/testdata/trial-80-colorfalse.golden`
- `internal/ui/tui/testdata/trial-80-colortrue.golden`

### try-first

Shared identity and compact trial footer.

- `internal/ui/tui/testdata/try-first-120-colorfalse.golden`
- `internal/ui/tui/testdata/try-first-120-colortrue.golden`
- `internal/ui/tui/testdata/try-first-60-colorfalse.golden`
- `internal/ui/tui/testdata/try-first-60-colortrue.golden`
- `internal/ui/tui/testdata/try-first-80-colorfalse.golden`
- `internal/ui/tui/testdata/try-first-80-colortrue.golden`

### why

Shared identity and complete-hint footer composition.

- `internal/ui/tui/testdata/why-120-colorfalse.golden`
- `internal/ui/tui/testdata/why-120-colortrue.golden`
- `internal/ui/tui/testdata/why-60-colorfalse.golden`
- `internal/ui/tui/testdata/why-60-colortrue.golden`
- `internal/ui/tui/testdata/why-80-colorfalse.golden`
- `internal/ui/tui/testdata/why-80-colortrue.golden`

## Owner checks and verification limits

The sandbox rejects native `ps` with Operation not permitted and blocks Metal. macOS/Linux canned process fixtures, fail-open behaviour, and the external-overlay PTY are verified. Real GPU calibration, the warning against the owner's resident servers, and Linux process enumeration remain owner checks. The runtime-performance effect of freeing resident models requires that real hardware run.

1. On the M5, leave the two other 9B servers loaded and open a fresh scratch TRIALM_HOME. Confirm the yellow count, names and RSS under `v`, and the same warning above Check speed. Start calibration and confirm that it proceeds.
2. Run calibration with the machine in its intended normal load. Watch the bar and spinner between groups and through a long group. Check the stored measurements against the owner's full-power baseline. Browse a model, choose it, move within its grid, cycle cache, then Escape and return to it in the owner's 80-column terminal.
