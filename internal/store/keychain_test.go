package store

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// A stand-in for Apple's security tool: the real Keychain is never touched.
func TestKeychainKeepsTheKeyOutOfFilesAndArgv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs /bin/sh")
	}
	t.Setenv("OPENROUTER_API_KEY", "")
	dir := t.TempDir()
	vault, argv, tool := filepath.Join(dir, "vault"), filepath.Join(dir, "argv"), filepath.Join(dir, "security")
	script := "#!/bin/sh\necho \"$@\" >> " + argv + "\ncase \"$1\" in\n" +
		"-i) read -r line; set -- $line; while [ $# -gt 0 ]; do [ \"$1\" = -w ] && printf %s \"$2\" > " + vault + "; shift; done ;;\n" +
		"find-generic-password) [ -s " + vault + " ] && cat " + vault + " && echo || exit 44 ;;\n" +
		"delete-generic-password) rm -f " + vault + " ;;\nesac\n"
	if err := os.WriteFile(tool, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	old := securityTool
	securityTool = tool
	defer func() { securityTool = old }()

	s := Store{Dir: filepath.Join(dir, "home"), Keychain: true}
	const key = "sk-or-v1-0123abcd"
	if err := s.SaveKey(key); err != nil || s.Key() != key {
		t.Fatal("not kept in the Keychain", err)
	}
	if _, err := os.Stat(filepath.Join(s.Dir, "credentials")); !os.IsNotExist(err) {
		t.Fatal("the key was also written to a file")
	}
	if b, _ := os.ReadFile(argv); strings.Contains(string(b), key) {
		t.Fatal("the key appeared in argv")
	}
	if err := s.RemoveKey(); err != nil || s.Key() != "" {
		t.Fatal("not removed", err)
	}
	// A key the security tool cannot take verbatim stays in the 0600 file.
	if err := s.SaveKey("odd key"); err != nil || s.Key() != "odd key" {
		t.Fatal("no file fallback", err)
	}
}
