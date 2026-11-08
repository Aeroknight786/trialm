# First-run flow report — 2026-11-07

Base: `4510720` (includes the shared-engine work). No commits, model downloads, admin commands, or OpenRouter sign-ins.

## Patch 1 — picker first

`/tmp/claude-501/p0b/p2p3flow-1.patch` adds Detect → Agents → Calibrate → Pick to the real binary. The returning launcher retains its list and opens the same picker. Core owns preferred-agent persistence, default selection, predictor tables and cache snapping. The picker lives in `internal/ui/tui/pick` to preserve the 2,500-line package limit.

The picker ranks every fitting model, keeps seven visible, supports file scrolling, native context limits, runtime/cache toggles, measured-cell provenance and the existing details view. Tables use `predict.Predict` per file/context/cache and `Timing.DecodeTPS(depth)`. Immutable tables are cached in the picker session, never computed in View. Try and Setup receive the selected runtime, quant, context and KV cache.

A measured cell requires an on-disk llama.cpp file and current machine/build evidence at a depth within 10%. Legacy evidence has no cache metadata, so it establishes only f16. Other cache types and deeper extrapolation remain estimates. No predictor, engine, calibration or setup package changes were needed.

Calibration runs the existing calibrate command, which installs the shared engine as needed. Its time comes from the midpoint of `calib.Duration(calib.Plan(...))`; progress uses `1-(1-t)^2.5`; ETA uses real events. Low Power Mode refuses Enter on the card. Current calibration skips the offer.

Plain/JSON changes are limited to the requested seam repair: calibration progress retains kind/fraction/ETA and group counts, and plain setup logs now print their lines. Download payloads and all other plain/JSON output retain their contract. No CLI golden changed.

Validation: gofmt clean; native and Windows go vet; go test -race ./...; core table tests for smaller-cache-only fit, native limit, >5 files, measured flag, selection retention and argument forwarding. TUI snapshots cover 60/80/120 columns × colour/NO_COLOR, including calibration progress/Low Power and picker cursor/cache/file variants. Cached draw timing is tested against 150 ms.

Patch 1 source budgets: **29,758 product**, **5,882 CLI/UI**; ceilings rounded to **29,800 / 5,900** in internal/budgets and PLAN §9. Every package remains under its existing limit. The reason recorded in PLAN is “the first-run flow and configuration picker”.

Key path in Patch 1: detection advances automatically; Agents Enter, Calibrate Later, Pick Enter, browser Enter = **4 keys + 1 click** to the existing trial handoff. Patch 2 replaces the old sign-in screen with the approved Try card and keeps the four-key path. The full local key path is checked with the later setup patch.

## Patch 1 PTY frames

Built `./cmd/trialm` and drove that binary through a Python PTY (80×40), with scratch HOME, XDG_CONFIG_HOME, TRIALM_HOME, HF_HOME and OLLAMA_MODELS. `home --device 'Apple M5' --ram 24` supplies a reproducible Mac target; tiny executable tool fixtures supply Claude Code/Codex discovery. No weights or key exist in the scratch home. These are the real binary’s frames, not snapshot-test output. The displayed ~22 t/s empty and depth slowdown come from the real predictor, not mockup constants.

### 1 Detect

```text
╭────────────────────────────────────────────────────────────────────────────╮
│ ▲ triaLM                                                                   │
╰────────────────────────────────────────────────────────────────────────────╯

✓  Apple M5 10c · 24.0 GB memory
✓  Coding tools: Claude Code, Codex
✓  Already on your Mac: None yet
✓  Engine: None yet · triaLM brings its own

16.0 GB for models

⏎ Continue    v Details    q Quit
```

### 2 Agents

```text
Agents

❯ [✓] Claude Code  Found on this Mac
  [✓] Codex  Found on this Mac
  [ ] OpenCode  Not installed
  [ ] Your own agent  One local address for any app

⏎ Continue    space Toggle    ↑↓ Move
```

### 3 Calibrate

```text
╭────────────────────────────────────────────────────────────────────────────╮
│ ✦ Calibrate for Accurate Estimates | ~7 mins | No Downloads                │
╰────────────────────────────────────────────────────────────────────────────╯

⏎ Now   s Later (run trialm calibrate)
```

### 4 Pick first

```text
Models  Claude Code · Apple M5 10c · 16.0 GB for models · 1 of 14
❯ ○ Qwen3.6 35B-A3B             up to 256K  ~48 t/s   13.2 GB █
  ○ Gemma 4 26B-A4B             up to 256K  ~26 t/s   13.6 GB │
  ○ gpt-oss 20B                 up to 128K  ~40 t/s   12.1 GB │
  ○ Gemma 4 12B                 up to 256K  ~14 t/s   7.12 GB │
  ○ Qwen3.5 9B                  up to 256K  ~22 t/s   5.68 GB │
  ○ Qwen3.5 4B                  up to 256K  ~38 t/s   2.74 GB │
  ○ Qwen3-Coder 30B-A3B         up to 128K  ~44 t/s   13.8 GB │

╭────────────────────────────────────────────────────────────────────────────╮
│Qwen  Qwen3.6 35B-A3B  estimate                                             │
│KV cache  [f16] q8_0 q4_0                                                   │
│Engine    [llama.cpp]                                                       │
│                                                                            │
│t/s when full       32K          64K          128K         256K             │
│IQ3_XXS             ~23 t/s      ~15 t/s      ~8.7 t/s     ✗                │
│13.2 GB             13.6 GB      14.2 GB      15.5 GB      18.0 GB          │
│5 larger files don't fit                                                    │
│                                                                            │
│Speed   ~48 t/s empty → ~15 t/s at 64K                                      │
│First reply  ~32 s                                                          │
│Memory  ██████████████████░░  14.2 of 16.0 GB                               │
╰────────────────────────────────────────────────────────────────────────────╯

↑↓ Model   ←→ Context   f File   c Cache   e Engine   ⏎ Try   s Set up   /
Search   v Details   esc Back
```

### 4 Pick second

```text
Models  Claude Code · Apple M5 10c · 16.0 GB for models · 2 of 14
  ○ Qwen3.6 35B-A3B             up to 256K  ~48 t/s   13.2 GB █
❯ ○ Gemma 4 26B-A4B             up to 256K  ~26 t/s   13.6 GB │
  ○ gpt-oss 20B                 up to 128K  ~40 t/s   12.1 GB │
  ○ Gemma 4 12B                 up to 256K  ~14 t/s   7.12 GB │
  ○ Qwen3.5 9B                  up to 256K  ~22 t/s   5.68 GB │
  ○ Qwen3.5 4B                  up to 256K  ~38 t/s   2.74 GB │
  ○ Qwen3-Coder 30B-A3B         up to 128K  ~44 t/s   13.8 GB │

╭────────────────────────────────────────────────────────────────────────────╮
│Gemma  Gemma 4 26B-A4B  estimate                                            │
│KV cache  [f16] q8_0 q4_0                                                   │
│Engine    [llama.cpp]                                                       │
│                                                                            │
│t/s when full       32K          64K          128K         256K             │
│IQ4_XS              ~15 t/s      ~11 t/s      ✗            ✗                │
│13.6 GB             14.3 GB      14.9 GB      16.1 GB      18.6 GB          │
│Q3_K_XL             ~15 t/s      ~11 t/s      ~7.2 t/s     ✗                │
│12.9 GB             13.6 GB      14.3 GB      15.5 GB      18.0 GB          │
│IQ3_XXS             ~16 t/s      ~11 t/s      ~7.4 t/s     ✗                │
│11.4 GB             12.2 GB      12.9 GB      14.1 GB      16.6 GB          │
│3 larger files don't fit                                                    │
│                                                                            │
│Speed   ~26 t/s empty → ~11 t/s at 64K                                      │
│First reply  ~46 s                                                          │
│Memory  ███████████████████░  14.9 of 16.0 GB                               │
╰────────────────────────────────────────────────────────────────────────────╯

↑↓ Model   ←→ Context   f File   c Cache   e Engine   ⏎ Try   s Set up   /
Search   v Details   esc Back
```

### 4 Pick seventh

```text
Models  Claude Code · Apple M5 10c · 16.0 GB for models · 7 of 14
  ○ Gemma 4 12B                 up to 256K  ~14 t/s   7.12 GB │
  ○ Qwen3.5 9B                  up to 256K  ~22 t/s   5.68 GB │
  ○ Qwen3.5 4B                  up to 256K  ~38 t/s   2.74 GB │
❯ ○ Qwen3-Coder 30B-A3B         up to 128K  ~44 t/s   13.8 GB █
  ○ GLM-4.7-Flash               up to 64K   ~44 t/s   13.8 GB │
  ○ Devstral Small 2 24B        up to 64K   ~10 t/s   11.9 GB │
  ○ Devstral Small 24B          up to 32K   ~8.7 t/s  14.3 GB │

╭────────────────────────────────────────────────────────────────────────────╮
│Qwen  Qwen3-Coder 30B-A3B  estimate                                         │
│KV cache  f16 [q8_0] q4_0                                                   │
│Engine    [llama.cpp]                                                       │
│                                                                            │
│t/s when full       32K          64K          128K         256K             │
│Q3_K_XL             ~9.5 t/s     ✗            ✗            ✗                │
│13.8 GB             15.0 GB      16.6 GB      19.8 GB      26.2 GB          │
│IQ3_XXS             ~9.6 t/s     ~5.3 t/s     ✗            ✗                │
│12.9 GB             14.1 GB      15.7 GB      18.9 GB      25.3 GB          │
│4 larger files don't fit                                                    │
│                                                                            │
│Speed   ~47 t/s empty → ~5.3 t/s at 64K                                     │
│First reply  ~71 s                                                          │
│Memory  ████████████████████  15.7 of 16.0 GB                               │
╰────────────────────────────────────────────────────────────────────────────╯

↑↓ Model   ←→ Context   f File   c Cache   e Engine   ⏎ Try   s Set up   /
Search   v Details   esc Back
```

### 4 Cache toggle

```text
Models  Claude Code · Apple M5 10c · 16.0 GB for models · 7 of 14
  ○ Gemma 4 12B                 up to 256K  ~14 t/s   7.12 GB │
  ○ Qwen3.5 9B                  up to 256K  ~22 t/s   5.68 GB │
  ○ Qwen3.5 4B                  up to 256K  ~38 t/s   2.74 GB │
❯ ○ Qwen3-Coder 30B-A3B         up to 128K  ~44 t/s   13.8 GB █
  ○ GLM-4.7-Flash               up to 64K   ~44 t/s   13.8 GB │
  ○ Devstral Small 2 24B        up to 64K   ~10 t/s   11.9 GB │
  ○ Devstral Small 24B          up to 32K   ~8.7 t/s  14.3 GB │

╭────────────────────────────────────────────────────────────────────────────╮
│Qwen  Qwen3-Coder 30B-A3B  estimate                                         │
│KV cache  f16 q8_0 [q4_0]                                                   │
│Engine    [llama.cpp]                                                       │
│                                                                            │
│t/s when full       32K          64K          128K         256K             │
│Q3_K_XL             ~15 t/s      ~9.1 t/s     ✗            ✗                │
│13.8 GB             14.3 GB      15.1 GB      16.8 GB      20.2 GB          │
│IQ3_XXS             ~15 t/s      ~9.2 t/s     ~5.1 t/s     ✗                │
│12.9 GB             13.4 GB      14.2 GB      15.9 GB      19.3 GB          │
│4 larger files don't fit                                                    │
│                                                                            │
│Speed   ~48 t/s empty → ~9.2 t/s at 64K                                     │
│First reply  ~64 s                                                          │
│Memory  ██████████████████░░  14.2 of 16.0 GB                               │
╰────────────────────────────────────────────────────────────────────────────╯

↑↓ Model   ←→ Context   f File   c Cache   e Engine   ⏎ Try   s Set up   /
Search   v Details   esc Back
```

### 4 File change

```text
Models  Claude Code · Apple M5 10c · 16.0 GB for models · 7 of 14
  ○ Gemma 4 12B                 up to 256K  ~14 t/s   7.12 GB │
  ○ Qwen3.5 9B                  up to 256K  ~22 t/s   5.68 GB │
  ○ Qwen3.5 4B                  up to 256K  ~38 t/s   2.74 GB │
❯ ○ Qwen3-Coder 30B-A3B         up to 128K  ~44 t/s   13.8 GB █
  ○ GLM-4.7-Flash               up to 64K   ~44 t/s   13.8 GB │
  ○ Devstral Small 2 24B        up to 64K   ~10 t/s   11.9 GB │
  ○ Devstral Small 24B          up to 32K   ~8.7 t/s  14.3 GB │

╭────────────────────────────────────────────────────────────────────────────╮
│Qwen  Qwen3-Coder 30B-A3B  estimate                                         │
│KV cache  f16 q8_0 [q4_0]                                                   │
│Engine    [llama.cpp]                                                       │
│                                                                            │
│t/s when full       32K          64K          128K         256K             │
│Q3_K_XL             ~15 t/s      ~9.1 t/s     ✗            ✗                │
│13.8 GB             14.3 GB      15.1 GB      16.8 GB      20.2 GB          │
│IQ3_XXS             ~15 t/s      ~9.2 t/s     ~5.1 t/s     ✗                │
│12.9 GB             13.4 GB      14.2 GB      15.9 GB      19.3 GB          │
│4 larger files don't fit                                                    │
│                                                                            │
│Speed   ~46 t/s empty → ~9.1 t/s at 64K                                     │
│First reply  ~65 s                                                          │
│Memory  ███████████████████░  15.1 of 16.0 GB                               │
╰────────────────────────────────────────────────────────────────────────────╯

↑↓ Model   ←→ Context   f File   c Cache   e Engine   ⏎ Try   s Set up   /
Search   v Details   esc Back
```

## Golden changes in Patch 1

Every filename below denotes one width/colour combination.

- `internal/ui/tui/testdata/agents-120-colorfalse.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/agents-120-colortrue.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/agents-60-colorfalse.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/agents-60-colortrue.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/agents-80-colorfalse.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/agents-80-colortrue.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/calibrate-120-colorfalse.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/calibrate-120-colortrue.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/calibrate-60-colorfalse.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/calibrate-60-colortrue.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/calibrate-80-colorfalse.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/calibrate-80-colortrue.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/calibrate-low-power-120-colorfalse.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/calibrate-low-power-120-colortrue.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/calibrate-low-power-60-colorfalse.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/calibrate-low-power-60-colortrue.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/calibrate-low-power-80-colorfalse.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/calibrate-low-power-80-colortrue.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/calibrating-120-colorfalse.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/calibrating-120-colortrue.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/calibrating-60-colorfalse.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/calibrating-60-colortrue.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/calibrating-80-colorfalse.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/calibrating-80-colortrue.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/details-120-colorfalse.golden` — Details now describe the selected grid cell.
- `internal/ui/tui/testdata/details-120-colortrue.golden` — Details now describe the selected grid cell.
- `internal/ui/tui/testdata/details-60-colorfalse.golden` — Details now describe the selected grid cell.
- `internal/ui/tui/testdata/details-60-colortrue.golden` — Details now describe the selected grid cell.
- `internal/ui/tui/testdata/details-80-colorfalse.golden` — Details now describe the selected grid cell.
- `internal/ui/tui/testdata/details-80-colortrue.golden` — Details now describe the selected grid cell.
- `internal/ui/tui/testdata/detect-120-colorfalse.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/detect-120-colortrue.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/detect-60-colorfalse.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/detect-60-colortrue.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/detect-80-colorfalse.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/detect-80-colortrue.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/detecting-120-colorfalse.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/detecting-120-colortrue.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/detecting-60-colorfalse.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/detecting-60-colortrue.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/detecting-80-colorfalse.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/detecting-80-colortrue.golden` — New first-run detection, agents or calibration card.
- `internal/ui/tui/testdata/launcher-calibrating-120-colorfalse.golden` — Deleted: calibration is a separate screen.
- `internal/ui/tui/testdata/launcher-calibrating-120-colortrue.golden` — Deleted: calibration is a separate screen.
- `internal/ui/tui/testdata/launcher-calibrating-60-colorfalse.golden` — Deleted: calibration is a separate screen.
- `internal/ui/tui/testdata/launcher-calibrating-60-colortrue.golden` — Deleted: calibration is a separate screen.
- `internal/ui/tui/testdata/launcher-calibrating-80-colorfalse.golden` — Deleted: calibration is a separate screen.
- `internal/ui/tui/testdata/launcher-calibrating-80-colortrue.golden` — Deleted: calibration is a separate screen.
- `internal/ui/tui/testdata/launcher-offer-120-colorfalse.golden` — Deleted: calibration is a separate screen.
- `internal/ui/tui/testdata/launcher-offer-120-colortrue.golden` — Deleted: calibration is a separate screen.
- `internal/ui/tui/testdata/launcher-offer-60-colorfalse.golden` — Deleted: calibration is a separate screen.
- `internal/ui/tui/testdata/launcher-offer-60-colortrue.golden` — Deleted: calibration is a separate screen.
- `internal/ui/tui/testdata/launcher-offer-80-colorfalse.golden` — Deleted: calibration is a separate screen.
- `internal/ui/tui/testdata/launcher-offer-80-colortrue.golden` — Deleted: calibration is a separate screen.
- `internal/ui/tui/testdata/pick-cache-120-colorfalse.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-cache-120-colortrue.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-cache-60-colorfalse.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-cache-60-colortrue.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-cache-80-colorfalse.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-cache-80-colortrue.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-file-120-colorfalse.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-file-120-colortrue.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-file-60-colorfalse.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-file-60-colortrue.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-file-80-colorfalse.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-file-80-colortrue.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-last-120-colorfalse.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-last-120-colortrue.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-last-60-colorfalse.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-last-60-colortrue.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-last-80-colorfalse.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-last-80-colortrue.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-second-120-colorfalse.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-second-120-colortrue.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-second-60-colorfalse.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-second-60-colortrue.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-second-80-colorfalse.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/pick-second-80-colortrue.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/recipes-120-colorfalse.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/recipes-120-colortrue.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/recipes-60-colorfalse.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/recipes-60-colortrue.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/recipes-80-colorfalse.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/recipes-80-colortrue.golden` — Replaced recipe table with configuration grid and navigation.
- `internal/ui/tui/testdata/why-120-colorfalse.golden` — Shared footer wrapping or removal of launcher calibration offer.
- `internal/ui/tui/testdata/why-120-colortrue.golden` — Shared footer wrapping or removal of launcher calibration offer.
- `internal/ui/tui/testdata/why-60-colorfalse.golden` — Shared footer wrapping or removal of launcher calibration offer.
- `internal/ui/tui/testdata/why-60-colortrue.golden` — Shared footer wrapping or removal of launcher calibration offer.
- `internal/ui/tui/testdata/why-80-colorfalse.golden` — Shared footer wrapping or removal of launcher calibration offer.
- `internal/ui/tui/testdata/why-80-colortrue.golden` — Shared footer wrapping or removal of launcher calibration offer.

## Patch 2 — Try, Install, Setup and Ready

`/tmp/claude-501/p0b/p2p3flow-2.patch` is incremental from Patch 1. It adds the approved Try card, agent cycling, a generic serve trial showing one address, the four-metric trial summary and rating, compact setup steps, and Ready tiles. The selected file/runtime/context/cache survive authorization, agent switching, trial history and setup. The real setup-argv test checks the selected context and both KV flags. The initial on-disk shortcut selects an available file without changing ranked model order; subsequent model navigation resets to the recommendation. A ready setup opens the returning launcher.

Deleted: the needs form, old recipe-table renderer, old key-screen copy, pasted-claim placeholder and calibration status formatter. The Ready title no longer changes its wording during the animation. TUI handoffs suppress the old plain prediction/connection banner while preserving the coding agent's stdout. Ordinary CLI output retains its existing path.

Low Power Mode is checked asynchronously on Enter, so turning it off permits a retry on the same card. Detection retains the discovered local files. NO_COLOR uses brackets around the selected cell. On-disk status and measured-cell tags have their own golden coverage. Cache metadata is now saved with Ready measurements; old evidence means f16, and newer evidence is reused only for its actual cache type.

### Final verification

- `gofmt -l .`: no output.
- `go vet ./...`: pass.
- `GOOS=windows go vet ./...`: pass.
- `go test -race ./...`: pass.
- Ordinary CLI characterization goldens remain unchanged; no file under `cmd/trialm/testdata` changed.
- Protected downloader and `verify.Ready` signatures remain unchanged.
- Predictor anchor: Qwen3.5 9B Q4_K_M on the M5 predicts **22.1 t/s empty, 8.3 t/s at 64K**, near the owner's 21.5 / about 9. No illustrative mockup constants were used.
- Cached picker draw measured **about 0.14 ms** in the focused draw check (50 views); the test enforces <150 ms.
- The fake trial runs the real Tea event loop and a child tool, reaches the first HTTP trial token in **4 keys**, with one browser authorization represented by the fake callback. No real OpenRouter sign-in occurred.
- Local path: **4 keys** to the launch handoff: Agents Enter, Calibrate Later, Set up, approval. The on-disk shortcut opens the agent after Ready without another key. Actual GPU coding remains an owner check.

### Final budgets

Counts are non-test Go lines containing tokens, matching `internal/budgets`.

| Scope | Measured | Ceiling |
|---|---:|---:|
| Product | 29,946 | 30,000 |
| CLI + UI | 5,940 | 6,000 |
| Core | 1,891 | 2,500 |
| TUI | 2,379 | 2,500 |
| Picker renderer | 202 | 2,500 |
| cmd/trialm physical lines | 13 | 30 |

PLAN §9 records the rounded ceilings, 2026-11-07, and “the first-run flow and configuration picker”. The global ceiling remains within the authorized 30,000; CLI/UI remains below the authorized 6,300.

Stripped native binary: **11,906,386 bytes / 14,000,000** (`CGO_ENABLED=0`, `-trimpath`, `-ldflags="-s -w"`).

### Plain and JSON changes

No removed screen had a separate plain/JSON counterpart: the needs, key and recipe flow were TTY screens. Existing command goldens were retained. The requested seam changes are: calibration setup progress carries `kind`, fraction, group done/total and ETA instead of download bytes; plain setup logs print their lines, including Low Power skip reasons. Download progress payloads retain their existing shape. The private TUI handoff switch suppresses banners only when the TUI starts a child; ordinary plain and JSON commands keep their output.

### Files and ownership

Most work is in `internal/ui/**`, `internal/cli/**`, preferences/history in `internal/store`, new `internal/core/{firstrun,configs}.go`, budgets and PLAN §9. Core owns tool persistence, grid building, selection snapping and Ready figures. UI renders those tables and forwards selected configuration.

Small changes beyond the listed core seam files:

- `internal/core/preferences.go`: selected context/cache in TrialArgs; deleted the now-unused calibration status formatter.
- `internal/core/setup.go`: record the actual cache with Ready evidence; refuse TUI setup for LM Studio when the selected cache cannot be enforced.
- `internal/calib/real.go` and its focused test: retain cache metadata and filter real-model factors by cache. RealFactor accepts an optional cache argument; existing calls remain valid.
- `internal/predict/calibration.go`: pass the resolved cache to that filter. No speed-model formula changed.

`internal/core/session.go` adds shared-engine detection for the UI, excluded from JSON. No engine or setup-package implementation was changed.

### Remaining contract limits

1. **LM Studio setup is blocked from the picker.** Its CLI exposes context length, GPU offload and TTL but no KV-cache selection. The existing setup adapter cannot guarantee the selected cache, so this flow returns an actionable error before planning/executing it. Simulation remains available. llama.cpp, Ollama and mlx-lm forward the chosen cache through their existing adapters. Re-enabling LM Studio setup needs an engine-owned load/configuration seam. See the official [lms load reference](https://lmstudio.ai/docs/cli/local-models/load).
2. **Ready reply durations are projections from measured rates.** The current Ready report has decode, prefill and follow-up timing, not complete workflow reply latencies. The tiles therefore carry `~`, with `est.` comparisons. They do not claim directly observed whole replies. The brief's strictly measured whole-reply tiles need richer verify-report data.
3. No real OpenRouter credit/sign-in, GPU calibration, model download, live GPU setup or live coding-agent session was run here. Windows was vetted, not interactively driven. The Mac's hardware detection and the native binary's screens were driven in a PTY; later setup/Ready frames use fake callbacks.

### Environment and owner checks

All final gates and every PTY run used scratch HOME, XDG_CONFIG_HOME and TRIALM_HOME. HF/OLLAMA paths were isolated in the PTY drivers. An early exploratory unit invocation inherited HOME and read HF-cache filenames; that was corrected before the final goldens and gates. Application writes went to scratch stores. No model weights were downloaded and no admin command ran.

To try the picker on the owner's own cached models, in the owner's terminal:

```sh
TRIALM_HOME=$(mktemp -d) /tmp/claude-501/p0b/p2p3flow-trialm
```

Check: the 9B appears on disk; left/right, f/c/e and search retain sensible choices; the Low Power warning clears after turning it off; an existing key removes sign-in/paste hints; a chosen on-disk configuration reaches a GPU Ready report with the same context/cache; then open the selected agent. Confirm the derived Ready times are acceptable, or extend verify to measure complete workflow replies. LM Studio's cache-control seam is a separate follow-up before enabling its setup action.

### Final PTY frames

The following fresh capture uses the actual Mac detection (Apple M5, 24 GB, 16 GB GPU budget), the built `cmd/trialm` binary and scratch homes. Tool discovery uses tiny Claude Code/Codex executable fixtures. Existing engine discovery is read-only. Screen 5 is captured before its sign-in action. Screens 6–8 use the same Root in the compiled `TestFlowPTYHelper` with fake Plan/Apply/Ready callbacks; no engine or model is installed. Drivers and raw terminal logs are under `/tmp/flow-pty/`.

#### 1 Detect

```text
╭────────────────────────────────────────────────────────────────────────────╮
│ ▲ triaLM                                                                   │
╰────────────────────────────────────────────────────────────────────────────╯

✓  Apple M5 · 10-core GPU · 24 GB memory
✓  Coding tools: Claude Code, Codex
✓  Already on your Mac: None yet
✓  Engine: Ollama, llama.cpp

16 GB for models

⏎ Continue    v Details    q Quit
```

#### 2 Agents

```text
Agents

❯ [✓] Claude Code  Found on this Mac
  [✓] Codex  Found on this Mac
  [ ] OpenCode  Not installed
  [ ] Your own agent  One local address for any app

⏎ Continue    space Toggle    ↑↓ Move
```

#### 3 Calibrate

```text
╭────────────────────────────────────────────────────────────────────────────╮
│ ✦ Calibrate for Accurate Estimates | ~7 mins | No Downloads                │
╰────────────────────────────────────────────────────────────────────────────╯

⏎ Now   s Later (run trialm calibrate)
```

#### 4 Pick first

```text
Models  Claude Code · Apple M5 10c · 16 GB for models · 1 of 14
❯ ○ Qwen3.6 35B-A3B             up to 256K  ~48 t/s   13.2 GB █
  ○ Gemma 4 26B-A4B             up to 256K  ~26 t/s   13.6 GB │
  ○ gpt-oss 20B                 up to 128K  ~40 t/s   12.1 GB │
  ○ Gemma 4 12B                 up to 256K  ~14 t/s   7.12 GB │
  ○ Qwen3.5 9B                  up to 256K  ~22 t/s   5.68 GB │
  ○ Qwen3.5 4B                  up to 256K  ~38 t/s   2.74 GB │
  ○ Qwen3-Coder 30B-A3B         up to 130K  ~47 t/s   12.9 GB │

╭────────────────────────────────────────────────────────────────────────────╮
│Qwen  Qwen3.6 35B-A3B  estimate                                             │
│KV cache  [f16] q8_0 q4_0                                                   │
│Engine    [llama.cpp] Ollama                                                │
│                                                                            │
│t/s when full       32K          64K          128K         256K             │
│IQ3_XXS             ~23 t/s      [~15 t/s]    ~8.7 t/s     ✗                │
│13.2 GB             13.6 GB      [14.2 GB]    15.5 GB      18.0 GB          │
│5 larger files don't fit                                                    │
│                                                                            │
│Speed   ~48 t/s empty → ~15 t/s at 64K    First reply  ~32 s                │
│Memory  ██████████████████░░  14.2 of 16 GB                                 │
╰────────────────────────────────────────────────────────────────────────────╯

↑↓ Model   ←→ Context   f File   c Cache   e Engine   ⏎ Try   s Set up   /
Search   v Details   esc Back
```

#### 4 Pick second

```text
Models  Claude Code · Apple M5 10c · 16 GB for models · 2 of 14
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
│Engine    [llama.cpp] Ollama                                                │
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

↑↓ Model   ←→ Context   f File   c Cache   e Engine   ⏎ Try   s Set up   /
Search   v Details   esc Back
```

#### 4 Pick seventh

```text
Models  Claude Code · Apple M5 10c · 16 GB for models · 7 of 14
  ○ Gemma 4 12B                 up to 256K  ~14 t/s   7.12 GB │
  ○ Qwen3.5 9B                  up to 256K  ~22 t/s   5.68 GB │
  ○ Qwen3.5 4B                  up to 256K  ~38 t/s   2.74 GB │
❯ ○ Qwen3-Coder 30B-A3B         up to 130K  ~47 t/s   12.9 GB █
  ○ GLM-4.7-Flash               up to 123K  ~45 t/s   12.9 GB │
  ○ Devstral Small 2 24B        up to 94K   ~10 t/s   11.9 GB │
  ○ Devstral Small 24B          up to 41K   ~8.7 t/s  14.3 GB │

╭────────────────────────────────────────────────────────────────────────────╮
│Qwen  Qwen3-Coder 30B-A3B  estimate                                         │
│KV cache  f16 [q8_0] q4_0                                                   │
│Engine    [llama.cpp] Ollama                                                │
│                                                                            │
│t/s when full       32K          64K          128K         256K             │
│Q3_K_XL             ~9.5 t/s     ✗            ✗            ✗                │
│13.8 GB             15.0 GB      16.6 GB      19.8 GB      26.2 GB          │
│IQ3_XXS             ~9.6 t/s     [~5.3 t/s]   ✗            ✗                │
│12.9 GB             14.1 GB      [15.7 GB]    18.9 GB      25.3 GB          │
│4 larger files don't fit                                                    │
│                                                                            │
│Speed   ~47 t/s empty → ~5.3 t/s at 64K    First reply  ~71 s               │
│Memory  ████████████████████  15.7 of 16 GB                                 │
╰────────────────────────────────────────────────────────────────────────────╯

↑↓ Model   ←→ Context   f File   c Cache   e Engine   ⏎ Try   s Set up   /
Search   v Details   esc Back
```

#### 4 Cache toggle

```text
Models  Claude Code · Apple M5 10c · 16 GB for models · 7 of 14
  ○ Gemma 4 12B                 up to 256K  ~14 t/s   7.12 GB │
  ○ Qwen3.5 9B                  up to 256K  ~22 t/s   5.68 GB │
  ○ Qwen3.5 4B                  up to 256K  ~38 t/s   2.74 GB │
❯ ○ Qwen3-Coder 30B-A3B         up to 130K  ~47 t/s   12.9 GB █
  ○ GLM-4.7-Flash               up to 123K  ~45 t/s   12.9 GB │
  ○ Devstral Small 2 24B        up to 94K   ~10 t/s   11.9 GB │
  ○ Devstral Small 24B          up to 41K   ~8.7 t/s  14.3 GB │

╭────────────────────────────────────────────────────────────────────────────╮
│Qwen  Qwen3-Coder 30B-A3B  estimate                                         │
│KV cache  f16 q8_0 [q4_0]                                                   │
│Engine    [llama.cpp] Ollama                                                │
│                                                                            │
│t/s when full       32K          64K          128K         256K             │
│Q3_K_XL             ~15 t/s      ~9.1 t/s     ✗            ✗                │
│13.8 GB             14.3 GB      15.1 GB      16.8 GB      20.2 GB          │
│IQ3_XXS             ~15 t/s      [~9.2 t/s]   ~5.1 t/s     ✗                │
│12.9 GB             13.4 GB      [14.2 GB]    15.9 GB      19.3 GB          │
│4 larger files don't fit                                                    │
│                                                                            │
│Speed   ~48 t/s empty → ~9.2 t/s at 64K    First reply  ~64 s               │
│Memory  ██████████████████░░  14.2 of 16 GB                                 │
╰────────────────────────────────────────────────────────────────────────────╯

↑↓ Model   ←→ Context   f File   c Cache   e Engine   ⏎ Try   s Set up   /
Search   v Details   esc Back
```

#### 4 File change

```text
Models  Claude Code · Apple M5 10c · 16 GB for models · 7 of 14
  ○ Gemma 4 12B                 up to 256K  ~14 t/s   7.12 GB │
  ○ Qwen3.5 9B                  up to 256K  ~22 t/s   5.68 GB │
  ○ Qwen3.5 4B                  up to 256K  ~38 t/s   2.74 GB │
❯ ○ Qwen3-Coder 30B-A3B         up to 130K  ~47 t/s   12.9 GB █
  ○ GLM-4.7-Flash               up to 123K  ~45 t/s   12.9 GB │
  ○ Devstral Small 2 24B        up to 94K   ~10 t/s   11.9 GB │
  ○ Devstral Small 24B          up to 41K   ~8.7 t/s  14.3 GB │

╭────────────────────────────────────────────────────────────────────────────╮
│Qwen  Qwen3-Coder 30B-A3B  estimate                                         │
│KV cache  f16 q8_0 [q4_0]                                                   │
│Engine    [llama.cpp] Ollama                                                │
│                                                                            │
│t/s when full       32K          64K          128K         256K             │
│Q3_K_XL             ~15 t/s      [~9.1 t/s]   ✗            ✗                │
│13.8 GB             14.3 GB      [15.1 GB]    16.8 GB      20.2 GB          │
│IQ3_XXS             ~15 t/s      ~9.2 t/s     ~5.1 t/s     ✗                │
│12.9 GB             13.4 GB      14.2 GB      15.9 GB      19.3 GB          │
│4 larger files don't fit                                                    │
│                                                                            │
│Speed   ~46 t/s empty → ~9.1 t/s at 64K    First reply  ~65 s               │
│Memory  ███████████████████░  15.1 of 16 GB                                 │
╰────────────────────────────────────────────────────────────────────────────╯

↑↓ Model   ←→ Context   f File   c Cache   e Engine   ⏎ Try   s Set up   /
Search   v Details   esc Back
```

#### 5 Try

```text
Try Qwen3-Coder 30B-A3B before you download it

╭────────────────────────────────────────────────────────────────────────────╮
│ API Simulation to get a feel for the model, speeds and performance on your │
│ workflows.                                                                 │
│ $0.20–2 of OpenRouter credit · No download                                 │
╰────────────────────────────────────────────────────────────────────────────╯

Run it in
[Claude Code]  Codex  Your own agent

⏎ Try in Claude Code (OpenRouter sign-in)    p Paste key    tab Agent    esc
Back
```

#### 6 Install

```text
Your trial of Qwen3.6 35B-A3B

╭─────────────────╮╭─────────────────╮╭─────────────────╮╭─────────────────╮
│Replies          ││Typical reply    ││Slowest          ││Cost             │
│38               ││~9.8 s           ││~41 s            ││$0.21            │
╰─────────────────╯╰─────────────────╯╰─────────────────╯╰─────────────────╯

How did it feel?
1 Too slow    2 Fine    3 Great

╭────────────────────────────────────────────────────────────────────────────╮
│ Install? 13.2 GB · 1 approval                                              │
╰────────────────────────────────────────────────────────────────────────────╯

⏎ Install    m Other models    q Later
```

#### 7 Approval

```text
╭────────────────────────────────────────────────────────────────────────────╮
│ Setting up Qwen3.6 35B-A3B for Codex                                       │
│                                                                            │
│ Install? 13.2 GB download · 1 approval · no password                       │
│                                                                            │
│ · Install the engine (17 MB)                                               │
│ · Download the model                                                       │
│ · Start and connect Codex                                                  │
│ · Check speed                                                              │
╰────────────────────────────────────────────────────────────────────────────╯

⏎ Install    esc Back    v Details
```

#### 7 Setting up

```text
╭────────────────────────────────────────────────────────────────────────────╮
│ Setting up Qwen3.6 35B-A3B for Codex                                       │
│                                                                            │
│ ✓ Install the engine (17 MB)                                               │
│ ... Download the model                                                     │
│   ▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌▌░░░░░░░░░░░░░░░░░░░                         │
│   8.10 of 13.2 GB · About 2 min left                                       │
│ · Start and connect Codex                                                  │
│ · Check speed                                                              │
╰────────────────────────────────────────────────────────────────────────────╯

ctrl-c Pause    v Details
```

#### 8 Ready

```text
✦ Ready

Qwen3.6 35B-A3B · Codex · 64K context

╭────────────────────────╮╭────────────────────────╮╭────────────────────────╮
│Per reply               ││First reply             ││Codex                   │
│~9.6 s                  ││~21 s                   ││Works                   │
│est. 8.3                ││est. 18                 ││Chat · Tools · Follow-up│
╰────────────────────────╯╰────────────────────────╯╰────────────────────────╯

Next time: trialm

⏎ Open Codex now    q Later    v Details
```

### Every changed golden in Patch 2

Each width/colour filename is listed separately.

- `internal/ui/tui/testdata/details-120-colorfalse.golden` — Selected configuration, bands, sources, argv and concise time split.
- `internal/ui/tui/testdata/details-120-colortrue.golden` — Selected configuration, bands, sources, argv and concise time split.
- `internal/ui/tui/testdata/details-60-colorfalse.golden` — Selected configuration, bands, sources, argv and concise time split.
- `internal/ui/tui/testdata/details-60-colortrue.golden` — Selected configuration, bands, sources, argv and concise time split.
- `internal/ui/tui/testdata/details-80-colorfalse.golden` — Selected configuration, bands, sources, argv and concise time split.
- `internal/ui/tui/testdata/details-80-colortrue.golden` — Selected configuration, bands, sources, argv and concise time split.
- `internal/ui/tui/testdata/detect-120-colorfalse.golden` — Hardware/core budget and detected on-disk models.
- `internal/ui/tui/testdata/detect-120-colortrue.golden` — Hardware/core budget and detected on-disk models.
- `internal/ui/tui/testdata/detect-60-colorfalse.golden` — Hardware/core budget and detected on-disk models.
- `internal/ui/tui/testdata/detect-60-colortrue.golden` — Hardware/core budget and detected on-disk models.
- `internal/ui/tui/testdata/detect-80-colorfalse.golden` — Hardware/core budget and detected on-disk models.
- `internal/ui/tui/testdata/detect-80-colortrue.golden` — Hardware/core budget and detected on-disk models.
- `internal/ui/tui/testdata/error-120-colorfalse.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/error-120-colortrue.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/error-60-colorfalse.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/error-60-colortrue.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/error-80-colorfalse.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/error-80-colortrue.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/error-details-120-colorfalse.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/error-details-120-colortrue.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/error-details-60-colorfalse.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/error-details-60-colortrue.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/error-details-80-colorfalse.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/error-details-80-colortrue.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/help-120-colorfalse.golden` — Remove needs/table keys and document picker keys.
- `internal/ui/tui/testdata/help-120-colortrue.golden` — Remove needs/table keys and document picker keys.
- `internal/ui/tui/testdata/help-60-colorfalse.golden` — Remove needs/table keys and document picker keys.
- `internal/ui/tui/testdata/help-60-colortrue.golden` — Remove needs/table keys and document picker keys.
- `internal/ui/tui/testdata/help-80-colorfalse.golden` — Remove needs/table keys and document picker keys.
- `internal/ui/tui/testdata/help-80-colortrue.golden` — Remove needs/table keys and document picker keys.
- `internal/ui/tui/testdata/key-120-colorfalse.golden` — Deleted: superseded needs, key or animated Ready variant.
- `internal/ui/tui/testdata/key-120-colortrue.golden` — Deleted: superseded needs, key or animated Ready variant.
- `internal/ui/tui/testdata/key-60-colorfalse.golden` — Deleted: superseded needs, key or animated Ready variant.
- `internal/ui/tui/testdata/key-60-colortrue.golden` — Deleted: superseded needs, key or animated Ready variant.
- `internal/ui/tui/testdata/key-80-colorfalse.golden` — Deleted: superseded needs, key or animated Ready variant.
- `internal/ui/tui/testdata/key-80-colortrue.golden` — Deleted: superseded needs, key or animated Ready variant.
- `internal/ui/tui/testdata/launcher-120-colorfalse.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/launcher-120-colortrue.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/launcher-60-colorfalse.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/launcher-60-colortrue.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/launcher-80-colorfalse.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/launcher-80-colortrue.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/needs-120-colorfalse.golden` — Deleted: superseded needs, key or animated Ready variant.
- `internal/ui/tui/testdata/needs-120-colortrue.golden` — Deleted: superseded needs, key or animated Ready variant.
- `internal/ui/tui/testdata/needs-60-colorfalse.golden` — Deleted: superseded needs, key or animated Ready variant.
- `internal/ui/tui/testdata/needs-60-colortrue.golden` — Deleted: superseded needs, key or animated Ready variant.
- `internal/ui/tui/testdata/needs-80-colorfalse.golden` — Deleted: superseded needs, key or animated Ready variant.
- `internal/ui/tui/testdata/needs-80-colortrue.golden` — Deleted: superseded needs, key or animated Ready variant.
- `internal/ui/tui/testdata/paste-120-colorfalse.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/paste-120-colortrue.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/paste-60-colorfalse.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/paste-60-colortrue.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/paste-80-colorfalse.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/paste-80-colortrue.golden` — Shared capitalized/wrapped footer and removal of obsolete copy.
- `internal/ui/tui/testdata/pick-cache-120-colorfalse.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-cache-120-colortrue.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-cache-60-colorfalse.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-cache-60-colortrue.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-cache-80-colorfalse.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-cache-80-colortrue.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-disk-120-colorfalse.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-disk-120-colortrue.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-disk-60-colorfalse.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-disk-60-colortrue.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-disk-80-colorfalse.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-disk-80-colortrue.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-file-120-colorfalse.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-file-120-colortrue.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-file-60-colorfalse.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-file-60-colortrue.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-file-80-colorfalse.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-file-80-colortrue.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-last-120-colorfalse.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-last-120-colortrue.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-last-60-colorfalse.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-last-60-colortrue.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-last-80-colorfalse.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-last-80-colortrue.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-second-120-colorfalse.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-second-120-colortrue.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-second-60-colorfalse.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-second-60-colortrue.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-second-80-colorfalse.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/pick-second-80-colortrue.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/ready-120-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/ready-120-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/ready-60-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/ready-60-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/ready-80-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/ready-80-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/ready-details-120-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/ready-details-120-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/ready-details-60-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/ready-details-60-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/ready-details-80-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/ready-details-80-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/ready-sparkle-120-colorfalse.golden` — Deleted: superseded needs, key or animated Ready variant.
- `internal/ui/tui/testdata/ready-sparkle-120-colortrue.golden` — Deleted: superseded needs, key or animated Ready variant.
- `internal/ui/tui/testdata/ready-sparkle-60-colorfalse.golden` — Deleted: superseded needs, key or animated Ready variant.
- `internal/ui/tui/testdata/ready-sparkle-60-colortrue.golden` — Deleted: superseded needs, key or animated Ready variant.
- `internal/ui/tui/testdata/ready-sparkle-80-colorfalse.golden` — Deleted: superseded needs, key or animated Ready variant.
- `internal/ui/tui/testdata/ready-sparkle-80-colortrue.golden` — Deleted: superseded needs, key or animated Ready variant.
- `internal/ui/tui/testdata/recipes-120-colorfalse.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/recipes-120-colortrue.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/recipes-60-colorfalse.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/recipes-60-colortrue.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/recipes-80-colorfalse.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/recipes-80-colortrue.golden` — Configuration grid, selected-cell brackets, local/measured status and predictor defaults.
- `internal/ui/tui/testdata/setup-120-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-120-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-60-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-60-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-80-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-80-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-details-120-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-details-120-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-details-60-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-details-60-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-details-80-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-details-80-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-disk-120-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-disk-120-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-disk-60-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-disk-60-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-disk-80-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-disk-80-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-paused-120-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-paused-120-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-paused-60-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-paused-60-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-paused-80-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-paused-80-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-progress-120-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-progress-120-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-progress-60-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-progress-60-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-progress-80-colorfalse.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/setup-progress-80-colortrue.golden` — Compact setup steps, progress, selected context and Ready tiles/details.
- `internal/ui/tui/testdata/summary-120-colorfalse.golden` — Approved summary metrics, rating and Install action.
- `internal/ui/tui/testdata/summary-120-colortrue.golden` — Approved summary metrics, rating and Install action.
- `internal/ui/tui/testdata/summary-60-colorfalse.golden` — Approved summary metrics, rating and Install action.
- `internal/ui/tui/testdata/summary-60-colortrue.golden` — Approved summary metrics, rating and Install action.
- `internal/ui/tui/testdata/summary-80-colorfalse.golden` — Approved summary metrics, rating and Install action.
- `internal/ui/tui/testdata/summary-80-colortrue.golden` — Approved summary metrics, rating and Install action.
- `internal/ui/tui/testdata/try-first-120-colorfalse.golden` — Approved Try card replaces the key screen.
- `internal/ui/tui/testdata/try-first-120-colortrue.golden` — Approved Try card replaces the key screen.
- `internal/ui/tui/testdata/try-first-60-colorfalse.golden` — Approved Try card replaces the key screen.
- `internal/ui/tui/testdata/try-first-60-colortrue.golden` — Approved Try card replaces the key screen.
- `internal/ui/tui/testdata/try-first-80-colorfalse.golden` — Approved Try card replaces the key screen.
- `internal/ui/tui/testdata/try-first-80-colortrue.golden` — Approved Try card replaces the key screen.

### Patch replay

The two incremental patches replay with `git apply --whitespace=nowarn` on a fresh `git archive HEAD` export. Every tracked/new source, test, golden and report file is compared byte-for-byte with the working tree. No commit was made.
