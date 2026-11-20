package server

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"path/filepath"
)

func siteHandler(dir, proxyURL string) http.Handler {
	if proxyURL != "" {
		u, _ := url.Parse(proxyURL) // checked in ConfigFromEnv
		return httputil.NewSingleHostReverseProxy(u)
	}
	if dir == "" {
		return http.NotFoundHandler()
	}

	files := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if exists(dir, r.URL.Path) {
			files.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		if b, err := os.ReadFile(filepath.Join(dir, "404.html")); err == nil {
			w.Write(b)
		}
	})
}

// a file, or a folder with an index.html
func exists(dir, p string) bool {
	full := filepath.Join(dir, filepath.FromSlash(path.Clean("/"+p)))
	info, err := os.Stat(full)
	if err != nil {
		return false
	}
	if info.IsDir() {
		_, err = os.Stat(filepath.Join(full, "index.html"))
		return err == nil
	}
	return true
}
