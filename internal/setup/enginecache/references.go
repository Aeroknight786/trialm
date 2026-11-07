// Package enginecache checks ownership before removing a shared engine.
package enginecache

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func Referenced(root, dir, except string) (bool, error) {
	entries, err := os.ReadDir(filepath.Join(root, "setups"))
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.Name() == except || !entry.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, "setups", entry.Name(), "journal.json"))
		type plan struct {
			Steps []struct {
				Path string
				Argv []string
			}
		}
		var j struct {
			State   string
			Plan    plan
			Pending *plan `json:"pending_plan"`
		}

		if err == nil {
			err = json.Unmarshal(b, &j)
		}
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return false, err
		}
		if j.State == "undone" {
			continue
		}
		steps := j.Plan.Steps
		if j.Pending != nil {
			steps = append(steps, j.Pending.Steps...)
		}
		for _, step := range steps {
			if within(dir, strings.ReplaceAll(step.Path, "~/.trialm", root)) || len(step.Argv) > 0 && within(dir, strings.ReplaceAll(step.Argv[0], "~/.trialm", root)) {
				return true, nil
			}
		}
	}
	return false, nil
}

func within(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && !filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Remove keeps engines referenced by another live journal, including paused plans.
func Remove(root, dir, except string, log func(string)) error {
	used, err := Referenced(root, dir, except)
	if err != nil {
		return err
	}
	if used {
		log("Keep shared llama.cpp · another setup still references it")
		return nil
	}
	for _, suffix := range []string{"", ".part", ".archive", ".archive.part", ".archive.part.json", ".install"} {
		if err = os.RemoveAll(dir + suffix); err != nil {
			return err
		}
	}
	return nil
}
