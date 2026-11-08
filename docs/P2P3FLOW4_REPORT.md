# First-run flow round 4

2026-11-07. Base `6e3174c`. Delivery is the ordered pair `/tmp/claude-501/p0b/p2p3flow4-1.patch` and `p2p3flow4-2.patch`, with matching one-line `.txt` summaries. Patch 1 contains implementation, tests, budgets and goldens. Patch 2 contains this report. Both milestones passed the required gates. The series was replayed with `git apply --whitespace=nowarn` on a fresh `git archive HEAD` export and compared byte for byte with the final tree, including new files.

## Changes

- Ready waits for Enter to open the agent or `q` for Later. The `autoLaunch` field, assignments and sparkle-tick handoff were deleted. A saved Ready setup on the returning launcher still opens on its explicit Enter.
- Check speed starts a one-second timer when its setup step starts. The estimate uses the plan's predicted prompt-reading, near-empty writing and target-depth writing rates. It includes the near-empty sample, the prefill sample at the configured batch size, cached follow-up, spec sweep, 8K and target samples, compatibility checks and the additional 20 tool calls below 3 bpw. The duplicate 8K sample is omitted. Deep prompts follow Ready's 90% target and 0.8 token-estimation convention. Small compatibility requests are budgeted at 128 input and 32 output tokens each. Output limits budget the timed writing samples. The initial estimate rounds up to a minute, counts down in seconds, and stays at About 1 min left on overrun. Completion and retry reset the timer.
- The dim sub-check line reports Reading a 23K-token prompt, Writing, Follow-up or Checking Claude Code as that work runs. Deep prompts switch to Writing on their first streamed token. Expanded details show the same progress.
- The model list takes its maximum from fitting grid cells across the model's available engines and cache types. The separate binary search for intermediate context limits was deleted. List limits now use the same columns as the grid.
- Plan cards show Install? 17 MB for the engine-only fixture and Install? 13.2 GB for the download fixture. Trial summaries and plain plans also drop approval counts. An on-disk model gets a completed Model already on disk step. A password line appears only for a plan with an admin step, using Needs your Mac password once.

## Files and contract additions

| Files | Work |
|---|---|
| `internal/ui/tui/root.go`, `setup.go`, `trial.go` | Explicit Ready handoff, countdown, dim sub-checks and install text |
| `internal/ui/plan.go` | Plain install header and conditional password text |
| `internal/core/configs.go` | Grid-derived model limits and removal of the binary search |
| `internal/core/firstrun.go` | Ready sample-duration estimate |
| `internal/ui/tui/root_test.go`, `firstrun_test.go`, `setup_test.go` | Key counts, waiting, timer ticks, overrun, sub-checks, password text and snapshots |
| `internal/core/configs_test.go` | Fitting-column maximum and duration tests |
| `internal/verify/ready.go`, `ready_test.go`, `verify.go` | Optional progress callback and first-token transition with tests |
| `internal/setup/executor.go`, `display.go` | Forward progress and simplify the shared install-size formatter |
| `internal/budgets/budgets_test.go`, `docs/PLAN.md` | Exact measured ceilings |

The small changes outside the owned areas are the one-line `setup_check` event forwarding in `internal/setup/executor.go`, the existing `CountsLine(bool)` implementation in `internal/setup/display.go`, the optional first-token callback in the internal `verify.measure` helper in `internal/verify/verify.go`, budget ceilings, listed CLI goldens and this handoff document. `ReadyOptions` adds only `SubCheck func(string)`. Existing `Progress` log callbacks retain their results. `Ready`, downloader and other exported function signatures retain their forms. The executor forwards the callback through the existing core event stream. Plain output prints its text and JSON emits the additive `setup_check` event.

## Gates and budgets

All Go commands used Go 1.26.10 with GOTOOLCHAIN=local. Test and PTY HOME, XDG configuration/cache, TRIALM_HOME, Hugging Face and Ollama state were scratch directories. The environment and logs are under `/tmp/claude-501/p0b/flow4/`.

| Gate | Result |
|---|---|
| `gofmt -l .` | Empty |
| `go vet ./...` | Pass |
| `GOOS=windows go vet ./...` | Pass |
| `go test -race ./...` | Pass |
| `go mod verify` | Pass |
| TUI snapshots | 258 frames, 43 variants at 60, 80 and 120 columns in colour and NO_COLOR |
| Real CLI-entry binary in a PTY | 80 columns by 40 rows, frames below |
| Ordered patch replay | Final tree matched |

The initial race run identified four plain-plan goldens that still contained approval and zero-password counts. Those four headers were updated and the full gate rerun. A separate diagnostic TUI-only run with an explicitly shortened 90-second timeout expired while repeatedly building calibration estimates for snapshots. The required full race gate uses its normal timeout.

| Budget | Base | Final | Ceiling |
|---|---|---|---|
| Product Go token lines | 30,210 | 30,282 | 30,282 |
| CLI and all UI token lines | 6,106 | 6,142 | 6,142 |
| Root TUI package | 2,430 | 2,463 | 2,500 |
| Core package | 1,903 | 1,915 | 2,500 |
| Setup package | — | 2,373 | 2,500 |
| CLI package | — | 2,304 | 2,500 |
| Picker package | 216 | 216 | 2,500 |
| cmd/trialm physical lines | 13 | 13 | 30 |
| Stripped macOS arm64 binary | 11,552,818 bytes | 11,569,378 bytes | 14,000,000 bytes |

The automatic-launch state, intermediate-context search, duplicate maximum scan and old counts formatter were removed before adding progress. Net additions require 72 product lines and 36 CLI/UI lines. The ceilings equal measured use and stay within the authorized maxima. Every package stays within its existing limit.

## Key counts

| Path | Keys |
|---|---|
| Fresh run, two detected tools, skip calibration, first trial token | 5 plus 1 browser click |
| Fresh run, fitting on-disk file, local setup through agent opening | 6 including Enter on the plan and Enter on Ready |
| Journey A through cloud trial and local agent opening | 8, or 9 with the optional rating, plus 1 browser click |
| Returning home, on-disk model needing setup | 3 |
| Returning home, saved Ready setup | 1 |

The fresh local path is Agents Enter → Later `s` → Choose Enter → Set up `s` → plan Enter → Open Enter. One detected tool skips the Agents key. Model, file, cache and context adjustments add their own navigation keys. The returning-path test waits for the sparkle to finish and confirms Ready still waits before sending Enter. The direct first-run test separately verifies Later exits Ready. The measured first-trial test still reaches its fake first response in five keys.

## Real-binary PTY

`trialm-prod` was built directly from `./cmd/trialm`. `trialm-fixture` was built from that same CLI entry point using the external `/tmp/claude-501/p0b/flow4/overlay.json`. Its overlay changes only SessionConfig dependencies. It marks the Qwen3.5 9B Q4_K_M row as on disk, builds the plan with the real `setup.Build`, and supplies scripted setup and Ready events. The model picker, prediction values, approval key, timer, rendering and Ready wait are production code. The fixture does not write model weights or run installers, services or admin commands.

The 64K selection gives a two-minute predictor estimate and visibly counts down between events. Reading holds for 12 seconds, Writing for four seconds, then Follow-up and Checking Claude Code each hold for a second. Ready was left untouched for three seconds after arrival and remained visible with the process alive. `q` exited zero. A second 32K run also waited on Ready and exercised the About 1 min left floor because its predictor estimate is below one minute. The 64K scripted Reading label is a callback fixture. The 23K label for a real 32K Ready request is separately tested through the actual fitted-prompt and streaming code.

Raw terminal bytes, decoded frames and exit status are in `flow4/pty/` and `flow4/pty-32k/`. The driver is `flow4/drive_pty.py`, using a Python PTY and the existing scratch pyte installation. NO_COLOR frames follow.

### On-disk plan

```text
✦ triaLM  Claude Code · Apple M5 · 16 GB for models
╭────────────────────────────────────────────────────────────────────────────╮
│ Setting up Qwen3.5 9B for Claude Code                                      │
│                                                                            │
│ Install? 17 MB                                                             │
│                                                                            │
│ · Install the engine (17 MB)                                               │
│ ✓ Model already on disk                                                    │
│ · Start and connect Claude Code                                            │
│ · Check speed                                                              │
╰────────────────────────────────────────────────────────────────────────────╯

⏎ Install   esc Back   v Details
```

### Check speed while reading

```text
✦ triaLM  Claude Code · Apple M5 · 16 GB for models
╭────────────────────────────────────────────────────────────────────────────╮
│ Setting up Qwen3.5 9B for Claude Code                                      │
│                                                                            │
│ ✓ Install the engine (17 MB)                                               │
│ ✓ Model already on disk                                                    │
│ ✓ Start and connect Claude Code                                            │
│ ... Check speed · About 1 min 59 s left                                    │
│   Reading a 23K-token prompt                                               │
╰────────────────────────────────────────────────────────────────────────────╯

ctrl-c Pause   v Details
```

### Check speed while writing

```text
✦ triaLM  Claude Code · Apple M5 · 16 GB for models
╭────────────────────────────────────────────────────────────────────────────╮
│ Setting up Qwen3.5 9B for Claude Code                                      │
│                                                                            │
│ ✓ Install the engine (17 MB)                                               │
│ ✓ Model already on disk                                                    │
│ ✓ Start and connect Claude Code                                            │
│ ... Check speed · About 1 min 47 s left                                    │
│   Writing                                                                  │
╰────────────────────────────────────────────────────────────────────────────╯

ctrl-c Pause   v Details
```

### Ready after three seconds without a key

```text
✦ triaLM  Claude Code · Apple M5 · 16 GB for models
✦ Ready

Qwen3.5 9B · Claude Code · 64K context

╭────────────────────────╮╭────────────────────────╮╭────────────────────────╮
│Per reply               ││First reply             ││Claude Code             │
│~20 s                   ││~66 s                   ││Works                   │
│est. 25                 ││est. 42                 ││Chat · Tools · Follow-up│
╰────────────────────────╯╰────────────────────────╯╰────────────────────────╯

Next time: trialm

⏎ Open Claude Code now   q Later   v Details
```

## Changed goldens

Each filename below is listed with its reason. There are 84 updated TUI frames, 12 new TUI progress frames and four updated CLI plan frames.

- `cmd/trialm/testdata/golden/3090-128/plan.txt` — Plain install header drops approval and zero-password counts.
- `cmd/trialm/testdata/golden/4090-64/plan.txt` — Plain install header drops approval and zero-password counts.
- `cmd/trialm/testdata/golden/5070-64/plan.txt` — Plain install header drops approval and zero-password counts.
- `cmd/trialm/testdata/golden/7900xt-64/plan.txt` — Plain install header drops approval and zero-password counts.
- `internal/ui/tui/testdata/pick-cache-120-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-cache-120-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-cache-60-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-cache-60-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-cache-80-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-cache-80-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-disk-120-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-disk-120-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-disk-60-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-disk-60-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-disk-80-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-disk-80-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-file-120-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-file-120-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-file-60-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-file-60-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-file-80-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-file-80-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-grid-120-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-grid-120-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-grid-60-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-grid-60-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-grid-80-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-grid-80-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-last-120-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-last-120-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-last-60-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-last-60-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-last-80-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-last-80-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-moved-120-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-moved-120-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-moved-60-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-moved-60-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-moved-80-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-moved-80-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-scrolled-120-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-scrolled-120-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-scrolled-60-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-scrolled-60-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-scrolled-80-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-scrolled-80-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-second-120-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-second-120-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-second-60-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-second-60-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-second-80-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/pick-second-80-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/recipes-120-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/recipes-120-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/recipes-60-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/recipes-60-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/recipes-80-colorfalse.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/recipes-80-colortrue.golden` — List context limits now match fitting grid columns.
- `internal/ui/tui/testdata/setup-120-colorfalse.golden` — Install line drops approval and zero-password counts.
- `internal/ui/tui/testdata/setup-120-colortrue.golden` — Install line drops approval and zero-password counts.
- `internal/ui/tui/testdata/setup-60-colorfalse.golden` — Install line drops approval and zero-password counts.
- `internal/ui/tui/testdata/setup-60-colortrue.golden` — Install line drops approval and zero-password counts.
- `internal/ui/tui/testdata/setup-80-colorfalse.golden` — Install line drops approval and zero-password counts.
- `internal/ui/tui/testdata/setup-80-colortrue.golden` — Install line drops approval and zero-password counts.
- `internal/ui/tui/testdata/setup-check-120-colorfalse.golden` — New countdown and current sub-check frame.
- `internal/ui/tui/testdata/setup-check-120-colortrue.golden` — New countdown and current sub-check frame.
- `internal/ui/tui/testdata/setup-check-60-colorfalse.golden` — New countdown and current sub-check frame.
- `internal/ui/tui/testdata/setup-check-60-colortrue.golden` — New countdown and current sub-check frame.
- `internal/ui/tui/testdata/setup-check-80-colorfalse.golden` — New countdown and current sub-check frame.
- `internal/ui/tui/testdata/setup-check-80-colortrue.golden` — New countdown and current sub-check frame.
- `internal/ui/tui/testdata/setup-check-later-120-colorfalse.golden` — New countdown and current sub-check frame.
- `internal/ui/tui/testdata/setup-check-later-120-colortrue.golden` — New countdown and current sub-check frame.
- `internal/ui/tui/testdata/setup-check-later-60-colorfalse.golden` — New countdown and current sub-check frame.
- `internal/ui/tui/testdata/setup-check-later-60-colortrue.golden` — New countdown and current sub-check frame.
- `internal/ui/tui/testdata/setup-check-later-80-colorfalse.golden` — New countdown and current sub-check frame.
- `internal/ui/tui/testdata/setup-check-later-80-colortrue.golden` — New countdown and current sub-check frame.
- `internal/ui/tui/testdata/setup-details-120-colorfalse.golden` — Install line drops approval and zero-password counts.
- `internal/ui/tui/testdata/setup-details-120-colortrue.golden` — Install line drops approval and zero-password counts.
- `internal/ui/tui/testdata/setup-details-60-colorfalse.golden` — Install line drops approval and zero-password counts.
- `internal/ui/tui/testdata/setup-details-60-colortrue.golden` — Install line drops approval and zero-password counts.
- `internal/ui/tui/testdata/setup-details-80-colorfalse.golden` — Install line drops approval and zero-password counts.
- `internal/ui/tui/testdata/setup-details-80-colortrue.golden` — Install line drops approval and zero-password counts.
- `internal/ui/tui/testdata/setup-disk-120-colorfalse.golden` — Engine-only 17 MB install line and completed on-disk model step.
- `internal/ui/tui/testdata/setup-disk-120-colortrue.golden` — Engine-only 17 MB install line and completed on-disk model step.
- `internal/ui/tui/testdata/setup-disk-60-colorfalse.golden` — Engine-only 17 MB install line and completed on-disk model step.
- `internal/ui/tui/testdata/setup-disk-60-colortrue.golden` — Engine-only 17 MB install line and completed on-disk model step.
- `internal/ui/tui/testdata/setup-disk-80-colorfalse.golden` — Engine-only 17 MB install line and completed on-disk model step.
- `internal/ui/tui/testdata/setup-disk-80-colortrue.golden` — Engine-only 17 MB install line and completed on-disk model step.
- `internal/ui/tui/testdata/setup-paused-120-colorfalse.golden` — Install line drops approval and zero-password counts.
- `internal/ui/tui/testdata/setup-paused-120-colortrue.golden` — Install line drops approval and zero-password counts.
- `internal/ui/tui/testdata/setup-paused-60-colorfalse.golden` — Install line drops approval and zero-password counts.
- `internal/ui/tui/testdata/setup-paused-60-colortrue.golden` — Install line drops approval and zero-password counts.
- `internal/ui/tui/testdata/setup-paused-80-colorfalse.golden` — Install line drops approval and zero-password counts.
- `internal/ui/tui/testdata/setup-paused-80-colortrue.golden` — Install line drops approval and zero-password counts.
- `internal/ui/tui/testdata/summary-120-colorfalse.golden` — Install offer drops the approval count.
- `internal/ui/tui/testdata/summary-120-colortrue.golden` — Install offer drops the approval count.
- `internal/ui/tui/testdata/summary-60-colorfalse.golden` — Install offer drops the approval count.
- `internal/ui/tui/testdata/summary-60-colortrue.golden` — Install offer drops the approval count.
- `internal/ui/tui/testdata/summary-80-colorfalse.golden` — Install offer drops the approval count.
- `internal/ui/tui/testdata/summary-80-colortrue.golden` — Install offer drops the approval count.

## Verification limits and owner checks

The PTY uses fake setup and Ready work. The actual progress callback is covered against a local HTTP streaming fake, including Reading to Writing, and the event forwarding is exercised by the real TUI loop. This round did not repeat the owner's GPU measurement, install an engine, download weights, or open a live Claude Code session. The sandbox blocks Metal. The owner should repeat the 9B at 32K path, watch the actual sub-check transitions through the long read, leave Ready untouched to read measured versus estimated times, then choose Enter or Later. Review the password line on a wired-limit plan without executing its admin step during sandbox checks.
