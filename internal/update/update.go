package update

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/espcaa/sablier/internal/version"
)

const repo = "https://github.com/espcaa/sablier"

func AssetName() string {
	name := "sablier_" + runtime.GOOS + "_" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

func Latest(ctx context.Context, c *http.Client) (string, error) {
	noFollow := *c
	noFollow.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	req, err := http.NewRequestWithContext(ctx, "HEAD", repo+"/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", version.UserAgent())
	resp, err := noFollow.Do(req)
	if err != nil {
		return "", fmt.Errorf("can't reach github: %w", err)
	}
	resp.Body.Close()
	tag := path.Base(resp.Header.Get("Location"))
	if !strings.HasPrefix(tag, "v") {
		return "", errors.New("no release found")
	}
	return tag, nil
}

func ReleaseURL(tag string) string {
	return repo + "/releases/download/" + tag
}

// download & check checksum
func Download(ctx context.Context, c *http.Client, base string) ([]byte, error) {
	name := AssetName()
	bin, err := get(ctx, c, base+"/"+name)
	if err != nil {
		return nil, err
	}
	sums, err := get(ctx, c, base+"/checksums.txt")
	if err != nil {
		return nil, err
	}
	want, ok := checksum(sums, name)
	if !ok {
		return nil, fmt.Errorf("no checksum for %s", name)
	}
	got := sha256.Sum256(bin)
	if hex.EncodeToString(got[:]) != want {
		return nil, fmt.Errorf("checksum mismatch for %s, not installing it", name)
	}
	return bin, nil
}

func Replace(exe string, bin []byte) error {
	tmp := exe + ".new"
	if err := os.WriteFile(tmp, bin, 0o755); err != nil {
		return writable(err, exe)
	}
	if runtime.GOOS == "windows" {
		old := exe + ".old"
		os.Remove(old)
		if err := os.Rename(exe, old); err != nil {
			os.Remove(tmp)
			return writable(err, exe)
		}
	}
	if err := os.Rename(tmp, exe); err != nil {
		os.Remove(tmp)
		return writable(err, exe)
	}
	return nil
}

func Cleanup(exe string) {
	if runtime.GOOS == "windows" {
		os.Remove(exe + ".old")
	}
}

func Newer(a, b string) bool {
	pa, okA := parse(a)
	pb, okB := parse(b)
	if !okA || !okB {
		return false
	}
	for i := range pa {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func parse(v string) ([3]int, bool) {
	var out [3]int
	v, _, _ = strings.Cut(strings.TrimPrefix(v, "v"), "-")
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

func get(ctx context.Context, c *http.Client, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", version.UserAgent())
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("can't download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("can't download %s: %s", url, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 200<<20))
}

func checksum(sums []byte, name string) (string, bool) {
	sc := bufio.NewScanner(bytes.NewReader(sums))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == name {
			return f[0], true
		}
	}
	return "", false
}

func writable(err error, exe string) error {
	if errors.Is(err, os.ErrPermission) {
		return fmt.Errorf("can't write to %s, reinstall it or rerun with permission to write there", filepath.Dir(exe))
	}
	return err
}
