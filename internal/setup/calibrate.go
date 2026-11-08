package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/aeroknight786/trialm/internal/calib"
	"github.com/aeroknight786/trialm/internal/engine"
	"github.com/aeroknight786/trialm/internal/hw"

	"github.com/aeroknight786/trialm/internal/setup/calibration"
	"github.com/aeroknight786/trialm/internal/setup/download"
)

// EnsureCalibrationEngine uses exactly the same typed, pinned install as setup.
func EnsureCalibrationEngine(ctx context.Context, root string, log func(string)) (string, error) {
	binary := engine.SharedBinary(root, "llama-bench", runtime.GOOS)
	unlock, err := lock(filepath.Join(root, "engine.lock"))
	if err != nil {
		return "", err
	}
	defer unlock()
	if engine.SharedInstalled(root, runtime.GOOS) {
		return binary, nil
	}
	log("Installing shared llama.cpp " + engine.Release() + " for calibration · about 17 MB · no model downloads")
	a, err := engine.ReleaseAsset(runtime.GOOS, runtime.GOARCH, "metal")
	if err != nil {
		return "", err
	}
	step := pinnedInstall(engine.SharedDir(root), a)
	e := Executor{Root: root}
	j := Journal{Steps: make([]Record, 1)}
	err = e.run(ctx, step, &j, 0, filepath.Join(root, "calibration-install.json"), Update{})
	_ = os.Remove(filepath.Join(root, "calibration-install.json"))
	return binary, err
}
func pinnedInstall(dest string, a engine.Asset) Step {
	return Step{ID: "engine", Kind: Install, Title: "Install llama.cpp " + engine.Release(), Path: dest, ArchiveRoot: a.Root, Request: &download.Request{URL: a.URL, SHA256: a.SHA256, Size: a.Size, Dest: dest + ".archive"}}
}

func (e *Executor) runCalibration(ctx context.Context, p Plan, u Update) error {
	if e.Calibration != nil {
		return e.Calibration(ctx, p, u)
	}

	return calibration.Run(ctx, e.Root, p.OS, p.Backend, func(kind string, progress *download.Progress, line string) {
		update := u
		update.Progress, update.Lines = progress, []string{line}
		if progress != nil && progress.Total > 0 {
			update.Fraction = float64(progress.Done) / float64(progress.Total)
		}
		e.emit(kind, update)
	})
}

// reserveCalibration includes all model files, even when fetch and calibration overlap.
func (e *Executor) reserveCalibration(ctx context.Context, p Plan) error {
	if p.OS == "darwin" && hw.MacPower(ctx).LowPower {
		return nil
	}

	required, calibrate := int64(calib.MinFreeBytes), false
	for _, s := range p.Steps {
		calibrate = calibrate || s.Kind == Calibrate
		if s.Request != nil {
			required += s.Request.Size
			if st, err := os.Stat(e.expand(s.Request.Dest)); err == nil {
				required -= min(st.Size(), s.Request.Size)
			}
		}
	}
	if free := download.AvailableBytes(e.Root); calibrate && free >= 0 && free < required {
		return fmt.Errorf("setup needs model download space plus about 6 GB for calibration temporary files")
	}

	return nil
}

func calibrationStep() Step {
	return Step{ID: "calibrate", Kind: Calibrate, Title: "Calibrate this Mac · about 5–8 min · no model downloads"}
}
