package enginecache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPendingAndMalformedJournalsKeepEngine(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "engines", "llama.cpp-b12182")
	journal := filepath.Join(root, "setups", "other", "journal.json")
	if err := os.MkdirAll(filepath.Dir(journal), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(journal, []byte(`{"state":"needs_approval","pending_plan":{"steps":[{"argv":["~/.trialm/engines/llama.cpp-b12182/llama-server"]}]}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if used, err := Referenced(root, dir, "undoing"); err != nil || !used {
		t.Fatal(used, err)
	}
	if err := os.WriteFile(journal, []byte(`broken`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Referenced(root, dir, "undoing"); err == nil {
		t.Fatal("ignored unreadable reference")
	}
}
