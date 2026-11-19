package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRootAndID(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repo, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := Root(sub); got != repo {
		t.Fatalf("Root = %s, want %s", got, repo)
	}

	salt := []byte("s")
	if ID(salt, sub) != ID(salt, repo) {
		t.Error("subdir and root should share an id")
	}
	if ID(salt, repo) == ID([]byte("other"), repo) {
		t.Error("salt should change the id")
	}

	plain := t.TempDir() // no .git above it
	if Root(plain) != plain {
		t.Errorf("no repo: Root should be the dir itself")
	}
}
