package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sablier.cfg")
	t.Setenv("SABLIER_CONFIG", p)

	c, err := Load()
	if err != nil || c.APIKey != "" || c.APIURL != DefaultAPIURL {
		t.Fatalf("missing file: %+v, %v", c, err)
	}

	if err := Save(Config{APIKey: "sab_x", APIURL: "https://example.com"}); err != nil {
		t.Fatal(err)
	}
	c, err = Load()
	if err != nil || c.APIKey != "sab_x" || c.APIURL != "https://example.com" {
		t.Fatalf("got %+v, %v", c, err)
	}

	info, _ := os.Stat(p)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("perm %v, want 600", info.Mode().Perm())
	}
}

func TestParse(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sablier.cfg")
	t.Setenv("SABLIER_CONFIG", p)
	os.WriteFile(p, []byte("# comment\n\n  api_key=sab_y  \napi_url = http://x/ \nunknown = 1\nnot a line\n"), 0o600)

	c, err := Load()
	if err != nil || c.APIKey != "sab_y" || c.APIURL != "http://x" {
		t.Fatalf("got %+v, %v", c, err)
	}
}
