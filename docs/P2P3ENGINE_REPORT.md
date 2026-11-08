# P2/P3 shared engine handoff

This tree shares pinned llama.cpp **b12182** on macOS arm64 between calibration and every new setup. Existing journals retain their saved engine paths. No commit was made; `.git` remained read-only.

## Changes

- `internal/engine/templates.go`: shared directory/binary helpers and completed-install checks. The Mac asset remains the existing 16,698,597-byte ggml-org release with SHA256 `71cfee075e0c54852e225dafdc8dc79c32b26e8d45f052369b7d4f78d2ccf1a6`.
- `internal/setup/{plan,resolve,types,policy,executor,calibrate}.go`: new setups use `$TRIALM_HOME/engines/llama.cpp-b12182/llama-server`. An installed shared release removes the install/download row and adds an explicit reuse warning. Calibration defaults on through `prefs.json`'s `calibrate_during_setup` boolean; false disables the step. Complete, usable calibration matching the current machine, build and baseline skips it.
- `internal/setup/calibration/run.go`: the existing predictor baseline is reused by setup and the calibration command. The `calibrate` step starts alongside the first model fetch, finishes before inference begins, or runs immediately before the recipe/Ready rows for an existing local model. It emits standard step/log/progress events, including a fraction and ETA. Events are serialized; only the main execution goroutine writes the journal. Cancellation joins calibration before returning and saves its paused state; the calibration runner retains finished measurements and removes its temporary synthetic file.
- `internal/setup/undo.go`, `internal/setup/enginecache`: installation and undo share an installation lock. Undo checks other journals, including paused/failed plans and pending replacements, before deleting a shared engine. A malformed journal prevents deletion. The last setup can remove an engine originally installed by calibration or another setup. Legacy per-setup inverses retain their existing behavior. Archive extraction moved unchanged into the engine-cache helper package.
- `internal/calib`: Mac discovery prefers the shared benchmark over Homebrew/PATH. Server fingerprints use their sibling benchmark when present, so recording and calibration identify the same build. The disk check accounts for already allocated temporary probe files; duplicate disk-space probes were replaced by the downloader's platform-aware helper. CPU benchmark argv now contains exactly one `-ngl 0`: repeated `-ngl` flags made llama-bench run a sweep.
- `internal/calibrationcli/calibrate.go`: an execution/check command installs the shared release first if needed, using the same typed pinned install and one explanatory line. Status, plan and refit stay read-only. Calibration text says “no model downloads.” The legacy plain calibration offer was removed as setup takes over; its CLI hook remains a no-op. The interactive offer belongs to the parallel UI agent.

The combined disk preflight reserves remaining request bytes plus 6 GiB for calibration, and skips that reservation in Low Power Mode. Low Power Mode emits one skip reason and continues setup. Linux/Windows keep their existing engine-selection and per-setup installation behavior. Their calibration disk check also benefits from the downloader helper.

## UI seam

No file in `internal/ui/**` or `internal/cli/**` changed. No golden changed. The event contract is:

- Kind: `calibrate`.
- Title: `Calibrate this Mac · about 5–8 min · no model downloads`.
- `setup_progress`: `setup.Update.Fraction` in [0, 1], `Progress.Done/Total` counting calibration groups, and `Progress.ETA` as a duration.
- `setup_log`: standard `Update.Kind`, `Title`, and `Lines`.
- A concurrent calibration emits its terminal step event without waiting for the network transfer to finish.

**Integration needed in the parallel renderer work:** the existing plain/JSON renderer formats every progress event as download bytes, drops kind/fraction from JSON progress, and ignores plain setup-log lines. It should preserve calibration kind/fraction/ETA and render its logs, including Low Power skip and shared-engine retention reasons. The same note is in `/tmp/claude-501/p0b/p2p3engine-ui-contract.txt`. `core.Session.Models.Speed` carries measured Ready writing speed; `Reply` remains an honest calibrated extrapolation, potentially with `~` for its wide band.

## Small change outside ownership

`internal/hw/macprobe.go` and its test: when restricted macOS sessions deny `sysctl hw.memsize`, read RAM and chip from `system_profiler SPHardwareDataType -json`. This sandbox returned zero RAM before the fallback, preventing a valid fingerprint and CPU setup prediction. The fallback uses actual System Information data, never an invented machine or an environment override. Its regression test uses synthetic hardware JSON.

## Verification

Required gates: empty `gofmt -l .`, `go vet ./...`, `GOOS=windows go vet ./...`, and `go test -race ./...`. Preliminary gates and the initial real smoke used the shell default Go 1.27.3. Final checks and binary budgets use the existing local Go 1.26.10 toolchain, with GOTOOLCHAIN=local and scratch HOME/XDG/TRIALM_HOME. Logs are `/tmp/claude-501/p0b/p2p3engine-{gofmt,vet,vet-windows,race-pinned}.log`.

New tests cover shared Mac paths and reuse despite compatible Homebrew; distinct IDs for calibration and other Mac adapters; preference/default handling; calibration/fetch overlap; cancellation, joining and resume; current calibration detection; sibling-server fingerprint consistency; combined disk reservation; paused/pending/malformed journal references; last-reference deletion; and exactly one CPU offload flag. The opt-in `TestSharedRecordingE2E` reads an already Ready scratch setup, checks the real pinned fingerprint and saved rows, and checks the core model listing's measured speed. It downloads nothing.

## Real binary verification

Pending final CPU integration result. Artifacts are under `/tmp/claude-501/p0b/p2p3engine-real` and `/tmp/claude-501/p0b/p2p3engine-cpu`; the owner HOME is used only for read-only HF-cache reuse. All engine/model/config/calibration writes are inside the corresponding scratch TRIALM_HOME. No model weights were downloaded. The cached Qwen3.5 9B Q4_K_M digest was independently checked as `03b74727a860a56338e042c4420bb3f04b2fec5734175f4cb9fa853daf52b7e8`.

The first real run installed the release and reused the model, skipped calibration while Low Power Mode was on, and reached Ready. Low Power Mode was turned off during that run, so its CPU measurements are mixed-power smoke evidence, not GPU performance evidence. A fresh scratch run was then started with calibration inside setup. Both runs use `--backend cpu` (device none, no operator/KV offload); the first uses context 2048 and the fresh calibration run uses context 512 to keep the CPU Ready check bounded. No admin step ran. Both model files came from a verified local copy of the HF cache (the sandbox did not permit a hard link); transfer progress reports zero network model bytes.

The fresh process began before the CPU argv correction. Its benchmark had both requested `-ngl 99` and `-ngl 0`, creating two CPU configurations with devices disabled; `gpu_info` is empty in the saved raw rows. A scratch-only guard now normalizes remaining benchmark invocations to one `-ngl 0` and execs `llama-bench.real`, the unchanged pinned executable. Its SHA256 matches the original release executable; evidence is `p2p3engine-cpu/cpu-guard-proof.json`. This wrapper is a test artifact only. Production uses the corrected argv directly.

The first setup reached Ready and appended three real samples. A subsequent real `trialm launch --check --json` exited 0 and changed calib.json mtime from **1794061361586966902** to **1794062139172759357** ns, with six real rows afterward. The opt-in core listing check passed: writing **1.4 tok/s, measured** on CPU; the calibrated reply remains approximately **270 s**. Launch-check CPU reading was about **14 tok/s**. These are contended CPU smoke numbers, with mixed power conditions in the earlier samples, and say nothing about Metal performance. Evidence: `p2p3engine-real/launch-proof.json` and `/tmp/claude-501/p0b/p2p3engine-recording-proof.log`. No fingerprint error occurred. This first setup skipped synthetic calibration in Low Power Mode; the fresh setup is the separate full calibration-inside-setup check.

The sandbox rejected stopping the first detached scratch server (including a direct signal): PID **89463**, journal `/tmp/claude-501/p0b/p2p3engine-real/setups/qwen3-5-9b-q4-k-m-llamacpp/journal.json`. Coordinator cleanup outside the sandbox is required. The user's servers were never signalled.

## Budgets

| Metric | Measured | Limit |
|---|---:|---:|
| Product Go token lines | 28,998 | 29,000 |
| CLI + UI | 5,500 | 5,500 |
| Largest owned package, setup | 2,376 | 2,500 |
| calib | 1,724 | 2,500 |
| calibrationcli | 348 | 2,500 |
| engine | 435 | 2,500 |
| cmd/trialm physical lines | 13 | 30 |
| Stripped macOS arm64 binary | 11,743,010 bytes | 14,000,000 |

The same-compiler HEAD binary is 11,725,986 bytes: +17,024 bytes. The operative ceiling is the brief's 29,000 and the current budget test/PLAN §9; the copied ground-rule 28,500 figure predates this branch. Only two product lines remain for integration; the parallel UI work must preserve the total ceiling.

## Patch series and owner checks

Patch 1 contains the shared engine/setup implementation and regression tests. The CPU argv correction and this final report follow as incremental patches. The complete series will be replayed with `git apply --whitespace=nowarn` on a fresh `git archive HEAD` export and compared byte-for-byte with this tree before handoff.

The coordinator must rerun the full setup/calibration/launch-check flow with Metal and the owner's normal context, and verify GPU speeds and launcher rendering. Actual Windows/Linux execution was not rerun here; their existing tests and Windows vet remain the portability gates. The renderer integration above is owned by the parallel UI agent.
