package store

import (
	"os/exec"
	"regexp"
	"strings"
)

// The macOS Keychain holds the key for the real home (Store.Keychain), so it
// is not a file anyone can copy. It goes through Apple's security tool, the
// key on its stdin, never in argv where other processes could read it.
var (
	securityTool = "/usr/bin/security"
	keychainArgs = []string{"-s", "com.mortiphi.trialm.openrouter", "-a", "openrouter"}
	plainKey     = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

func keychainGet() string {
	out, err := exec.Command(securityTool, append([]string{"find-generic-password", "-w"}, keychainArgs...)...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func keychainSet(key string) bool {
	if !plainKey.MatchString(key) {
		return false
	}
	cmd := exec.Command(securityTool, "-i")
	cmd.Stdin = strings.NewReader("add-generic-password -U " + strings.Join(keychainArgs, " ") + " -w " + key + "\n")
	return cmd.Run() == nil && keychainGet() == key
}

func keychainDelete() {
	_ = exec.Command(securityTool, append([]string{"delete-generic-password"}, keychainArgs...)...).Run()
}
