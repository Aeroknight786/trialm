package calib

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aeroknight786/trialm/internal/engine"
)

func TestSharedBenchmarkPreferenceAndServerFingerprint(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Mac engine preference")
	}
	root := t.TempDir()
	dir := engine.SharedDir(root)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	bench := engine.SharedBinary(root, "llama-bench", runtime.GOOS)
	server := engine.SharedBinary(root, "llama-server", runtime.GOOS)
	if err := os.WriteFile(bench, []byte("#!/bin/sh\necho 'version: 12182 (pinned)'\necho 'built with clang for arm64'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(server, []byte("#!/bin/sh\nexit 7\n"), 0700); err != nil {
		t.Fatal(err)
	}
	pathDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(pathDir, "llama-bench"), []byte("#!/bin/sh\necho 'version: 11960 (brew)'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", pathDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	got, err := FindEngine(root)
	if err != nil || got != bench {
		t.Fatal(got, err)
	}
	a, err := MachineFingerprint(context.Background(), bench)
	if err != nil {
		t.Fatal(err)
	}
	b, err := MachineFingerprint(context.Background(), server)
	if err != nil || a.Hash != b.Hash {
		t.Fatal(a, b, err)
	}
	if err = RecordReal(context.Background(), root, server, RealMeasurement{Model: "qwen3.5:9b", Quant: "Q4_K_M", Engine: "llamacpp", Decode: map[int]float64{0: 3}}); err != nil {
		t.Fatal(err)
	}
	c, err := Load(root)
	if err != nil || len(c.Rows) == 0 || c.Fingerprint.Hash != a.Hash {
		t.Fatal(c, err)
	}
	// Real-only records do not claim synthetic calibration is complete.
	if Current(context.Background(), root, bench) {
		t.Fatal("real-only calibration marked complete")
	}
	c.Complete = true
	c.BaselineVer = BaselineVersion
	c.Coverage = []string{"dense/Q4_K_M"}
	if err = c.Save(root); err != nil {
		t.Fatal(err)
	}
	if !Current(context.Background(), root, bench) {
		t.Fatal("completed current calibration not reused")
	}
}

func TestCPUBenchHasOneOffloadSetting(t *testing.T) {
	for _, step := range Plan(false, true) {
		args := step.Args("synthetic.gguf", true)
		count := 0
		for i, arg := range args {
			if arg == "-ngl" {
				count++
				if args[i+1] != "0" {
					t.Fatal(args)
				}
			}
		}
		if count != 1 {
			t.Fatal("llama-bench sweeps repeated flags instead of replacing them", args)
		}
	}
}
