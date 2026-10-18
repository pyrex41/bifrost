package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	// Hooks export repository-local Git variables. Fixture commands must use
	// their temporary repos, even when go test runs from a commit/push hook.
	// https://git-scm.com/docs/githooks#_description
	if names, err := exec.Command("git", "rev-parse", "--local-env-vars").Output(); err == nil {
		for _, name := range strings.Fields(string(names)) {
			os.Unsetenv(name)
		}
	}
	os.Exit(m.Run())
}
