package project

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
)

// nearest dir with a .git (a file in worktrees), else dir
func Root(dir string) string {
	dir = filepath.Clean(dir)
	for d := dir; ; {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return dir
		}
		d = parent
	}
}

// todo: per-account salt and git remote so ids match across machines
func ID(salt []byte, cwd string) string {
	if cwd == "" {
		return ""
	}
	m := hmac.New(sha256.New, salt)
	m.Write([]byte(Root(cwd)))
	return hex.EncodeToString(m.Sum(nil)[:16])
}
