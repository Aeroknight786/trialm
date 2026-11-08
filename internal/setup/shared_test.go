package setup

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/aeroknight786/trialm/internal/engine"
	"github.com/aeroknight786/trialm/internal/setup/download"
)

func TestMacSharedEngineAndCalibrationPlan(t *testing.T) {
	in := fixture(t)
	in.OS, in.Arch, in.Backend = "darwin", "arm64", "metal"
	in.Installed, in.Binary, in.InstalledVersion = true, "/opt/homebrew/bin/llama-server", "version: 11960"
	in.Calibrate = true
	p, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	shared := "~/.trialm/engines/llama.cpp-" + engine.Release()
	if p.Steps[0].Path != shared || p.Steps[0].Request.SHA256 == "" {
		t.Fatal(p.Steps[0])
	}
	for _, step := range p.Steps {
		if step.ID == "server" && step.Argv[0] != shared+"/llama-server" {
			t.Fatal(step)
		}
	}
	if p.Steps[len(p.Steps)-3].Kind != Calibrate {
		t.Fatal("local files must calibrate before Ready", p.Steps)
	}
	in.OwnedFileExists = func(rel string) bool {
		return strings.HasPrefix(rel, ".trialm/engines/llama.cpp-"+engine.Release()+"/")
	}
	in.Calibrate = false
	p, err = Build(in)
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(p.Steps, func(s Step) bool { return s.Kind == Install || s.Kind == Calibrate }) || !strings.Contains(strings.Join(p.Warnings, " "), "no engine download") {
		t.Fatal(p)
	}
	// Another model still uses exactly the same directory.
	in.Model.ID += "-second"
	second, _ := Build(in)
	for _, step := range second.Steps {
		if step.ID == "server" && step.Argv[0] != shared+"/llama-server" {
			t.Fatal(step)
		}
	}
}

type overlappingFetch struct {
	start, finish chan struct{}
	data          []byte
}

func (f overlappingFetch) Fetch(ctx context.Context, r download.Request) (download.Result, error) {
	select {
	case <-f.start:
	case <-time.After(3 * time.Second):
		return download.Result{}, errors.New("calibration did not overlap fetch")
	}
	if err := os.MkdirAll(filepath.Dir(r.Dest), 0700); err != nil {
		return download.Result{}, err
	}
	if err := os.WriteFile(r.Dest, f.data, 0600); err != nil {
		return download.Result{}, err
	}
	close(f.finish)
	return download.Result{}, nil
}
func TestCalibrationOverlapsFetchAndResumes(t *testing.T) {
	e := testExecutor(t)
	started, fetched := make(chan struct{}), make(chan struct{})
	data := []byte("model bytes")
	e.Downloader = overlappingFetch{started, fetched, data}
	calls := 0
	e.Calibration = func(ctx context.Context, p Plan, u Update) error {
		calls++
		if calls == 1 {
			close(started)
			<-fetched
			return context.Canceled
		}
		return nil
	}
	p := basicPlan(Step{ID: "model", Kind: Fetch, Request: &download.Request{URL: "https://huggingface.co/a/b/resolve/main/m.gguf", Dest: "~/.trialm/models/m.gguf", SHA256: fmt.Sprintf("%x", sha256.Sum256(data)), Size: int64(len(data))}}, calibrationStep())
	j, err := e.Apply(context.Background(), p, Policy{PlanID: p.ID})
	if !errors.Is(err, context.Canceled) || j.Steps[0].State != "done" || j.Steps[1].State != "failed" {
		t.Fatal(j, err)
	}
	j, err = e.Resume(context.Background(), p.Slug, "")
	if err != nil || j.State != "ready" || calls != 2 {
		t.Fatal(j, err, calls)
	}
	// A finished calibration is not repeated on subsequent resume.
	if _, err = e.Resume(context.Background(), p.Slug, ""); err != nil || calls != 2 {
		t.Fatal(err, calls)
	}
}
func TestCalibrationCancellationJoinsBeforeReturning(t *testing.T) {
	e := testExecutor(t)
	ctx, cancel := context.WithCancel(context.Background())
	started, joined := make(chan struct{}), make(chan struct{})
	e.Calibration = func(ctx context.Context, _ Plan, _ Update) error {
		close(started)
		<-ctx.Done()
		close(joined)
		return ctx.Err()
	}
	e.Downloader = cancelFetch{started: started, cancel: cancel}
	p := basicPlan(Step{ID: "model", Kind: Fetch, Request: &download.Request{URL: "https://huggingface.co/a/b/resolve/main/m.gguf", Dest: "~/.trialm/models/m.gguf", SHA256: strings.Repeat("a", 64), Size: 1}}, calibrationStep())
	j, err := e.Apply(ctx, p, Policy{PlanID: p.ID})
	if !errors.Is(err, context.Canceled) || j.State != "paused" || j.Steps[1].State != "paused" {
		t.Fatal(j, err)
	}
	select {
	case <-joined:
	default:
		t.Fatal("calibration child outlived Apply")
	}
}

type cancelFetch struct {
	started chan struct{}
	cancel  context.CancelFunc
}

func (f cancelFetch) Fetch(ctx context.Context, _ download.Request) (download.Result, error) {
	<-f.started
	f.cancel()
	return download.Result{}, ctx.Err()
}

func TestSharedUndoProtectsPausedJournalAndRemovesLastReference(t *testing.T) {
	e := testExecutor(t)
	dir := engine.SharedDir(e.Root)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "llama-server")
	if err := os.WriteFile(binary, []byte("engine"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, slug := range []string{"first", "second"} {
		p := basicPlan(Step{ID: "server", Kind: Start, Argv: []string{binary}})
		p.Slug = slug
		p.Seal()
		j := Journal{Plan: p, State: "paused", Steps: make([]Record, len(p.Steps))}
		j.Steps[0] = Record{State: "done", Inverse: Inverse{Category: Start}}
		path, _ := e.journalPath(slug)
		if err := save(path, &j); err != nil {
			t.Fatal(err)
		}
	}
	kept := false
	e.Event = func(kind string, v any) {
		if kind == "setup_log" {
			kept = strings.Contains(strings.Join(v.(Update).Lines, " "), "another setup")
		}
	}
	if _, err := e.Undo(context.Background(), "first", UndoOptions{Engines: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(binary); err != nil || !kept {
		t.Fatal("shared engine not protected", err, kept)
	}
	if _, err := e.Undo(context.Background(), "second", UndoOptions{Engines: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(binary); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("last reference kept engine", err)
	}
}
func TestSetupDiskReservationIncludesCalibration(t *testing.T) {
	e := testExecutor(t)
	if err := os.MkdirAll(e.Root, 0700); err != nil {
		t.Fatal(err)
	}
	p := basicPlan(calibrationStep())
	p.Steps = append(p.Steps, Step{Request: &download.Request{Size: 1 << 62, Dest: "~/.trialm/missing"}})
	if err := e.reserveCalibration(context.Background(), p); err == nil {
		t.Fatal("combined disk reservation missing")
	}
}
func TestSetupCalibrationPreference(t *testing.T) {
	e := testExecutor(t)
	if err := os.MkdirAll(e.Root, 0700); err != nil {
		t.Fatal(err)
	}
	in := fixture(t)
	in.OS = "darwin"
	resolved, err := Resolve(context.Background(), in, "")
	if err != nil || resolved.Calibrate {
		t.Fatal("setup calibrates only on opt-in", resolved.Calibrate, err)
	}
	if err = os.WriteFile(filepath.Join(e.Root, "prefs.json"), []byte(`{"calibrate_during_setup":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	resolved, err = Resolve(context.Background(), in, "")
	if err != nil || !resolved.Calibrate {
		t.Fatal("opt-in", resolved.Calibrate, err)
	}
}

func TestCalibrationEngineDoesNotConflictWithMacAdapters(t *testing.T) {
	for _, id := range []string{"ollama", "lmstudio", "mlx-lm"} {
		t.Run(id, func(t *testing.T) {
			in := fixture(t)
			in.Profile, _ = engine.Get(id)
			in.OS, in.Arch, in.Backend = "darwin", "arm64", "metal"
			in.Calibrate = true
			if id == "mlx-lm" {
				in.Quant.Format = "mlx"
				in.Backend = "mlx"
			}
			p, err := Build(in)
			if err != nil {
				t.Fatal(err)
			}
			seen := map[string]bool{}
			for _, step := range p.Steps {
				if seen[step.ID] {
					t.Fatal("duplicate step", step.ID)
				}
				seen[step.ID] = true
			}
			if !seen["calibration-engine"] || !seen["engine"] {
				t.Fatal(seen)
			}
		})
	}
}

func TestApprovedInstallReusesEngineInstalledByCalibration(t *testing.T) {
	e := testExecutor(t)
	dir := engine.SharedDir(e.Root)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"llama-bench", "llama-server"} {
		if err := os.WriteFile(engine.SharedBinary(e.Root, name, runtime.GOOS), []byte("installed release"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	a, _ := engine.ReleaseAsset("darwin", "arm64", "metal")
	p := basicPlan(pinnedInstall(engine.SharedDir(e.Root), a))
	p.OS = runtime.GOOS
	p.Seal()
	f := &resumeDownloader{}
	e.Downloader = f
	if _, err := e.Apply(context.Background(), p, Policy{PlanID: p.ID}); err != nil {
		t.Fatal(err)
	}
	if len(f.requests) != 0 {
		t.Fatal("downloaded a second engine", f.requests)
	}
	if _, err := e.Undo(context.Background(), p.Slug, UndoOptions{Engines: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("last setup could not remove an engine installed by calibration", err)
	}
}
