package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/espcaa/sablier/internal/config"
)

func runLogin(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetIn(strings.NewReader(stdin))
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"login"}, args...))
	err := root.Execute()
	return out.String(), err
}

func TestLogin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sab_good" {
			w.WriteHeader(401)
			return
		}
		w.Write([]byte(`{"id":"ident!a1","name":"Alice B"}`))
	}))
	defer srv.Close()
	cfgPath := filepath.Join(t.TempDir(), "sablier.cfg")
	t.Setenv("SABLIER_CONFIG", cfgPath)

	if _, err := runLogin(t, "sab_bad\n", "--api-url", srv.URL); err == nil || !strings.Contains(err.Error(), "invalid api key") {
		t.Fatalf("bad key: %v", err)
	}
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Fatal("config saved for a bad key")
	}

	out, err := runLogin(t, "  sab_good \n", "--api-url", srv.URL+"/")
	if err != nil || !strings.Contains(out, "signed in as Alice B") {
		t.Fatalf("got %q, %v", out, err)
	}
	c, _ := config.Load()
	if c.APIKey != "sab_good" || c.APIURL != srv.URL {
		t.Fatalf("saved %+v", c)
	}

	// the url is remembered
	if _, err := runLogin(t, "sab_good\n"); err != nil {
		t.Fatalf("second login: %v", err)
	}
}

func TestLoginEmptyKey(t *testing.T) {
	t.Setenv("SABLIER_CONFIG", filepath.Join(t.TempDir(), "sablier.cfg"))
	if _, err := runLogin(t, "\n"); err == nil || !strings.Contains(err.Error(), "no api key") {
		t.Fatalf("got %v", err)
	}
}
