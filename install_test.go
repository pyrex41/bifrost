package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// TestInstallGitBuildFetchesPinnedTagOnExistingClone pins the path a moved
// adapters.json ref takes on a sibling checkout that predates the tag: the
// installer must fetch before it checks out, or the pin only works on a fresh
// clone. The source repo gains tag v2 after the clone is made.
func TestInstallGitBuildFetchesPinnedTagOnExistingClone(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	gitRun(t, src, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(src, "f"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, src, "add", "f")
	gitRun(t, src, "commit", "-q", "-m", "one")
	gitRun(t, src, "tag", "v1")
	clone := filepath.Join(tmp, "clone")
	gitRun(t, tmp, "clone", "-q", src, clone)
	if err := os.WriteFile(filepath.Join(src, "f"), []byte("2"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, src, "commit", "-q", "-am", "two")
	gitRun(t, src, "tag", "v2")
	want := gitRun(t, src, "rev-parse", "v2")

	cfg := Adapter{Env: "BIFROST_T", Build: &BuildRecipe{Cwd: clone, Argv: []string{"git", "--version"}, Out: "unused"}}
	spec := &InstallSpec{Method: "git-build", Git: src, Ref: "v2"}
	if !installGitBuild("t", cfg, spec, "", "") {
		t.Fatal("installGitBuild failed on an existing clone with a pinned tag it had not fetched")
	}
	if got := gitRun(t, clone, "rev-parse", "HEAD"); got != want {
		t.Fatalf("checked out %s, want tag v2 at %s", got, want)
	}
}
