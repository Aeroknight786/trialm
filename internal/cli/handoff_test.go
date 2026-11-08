package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTUIHandoffKeepsAgentOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "codex"), []byte("#!/bin/sh\nprintf 'AGENT_READY\\n'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	args := []string{"try", "qwen3:8b", "--upstream", "mock", "--harness", "codex", "--runtime", "llamacpp", "--device", "Apple M5", "--ram", "24", "--ctx", "32768", "--port", "0", "--here", "--plain", "--tui-handoff"}
	raw, _ := json.Marshal(args)
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), goldenChildEnv+"="+string(raw), "HOME="+filepath.Join(root, "home"), "XDG_CONFIG_HOME="+filepath.Join(root, "config"), "TRIALM_HOME="+filepath.Join(root, "trialm"), "TRIALM_OFFLINE=1", "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err, string(b))
	}
	if strings.TrimSpace(string(b)) != "AGENT_READY" {
		t.Fatal("handoff printed a banner or hid the agent", string(b))
	}
}
