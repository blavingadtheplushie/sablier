package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DefaultAPIURL = "https://sablier.hackclub.cc"

type Config struct {
	APIKey string
	APIURL string
}

// SABLIER_CONFIG overrides the path
func Path() (string, error) {
	if p := os.Getenv("SABLIER_CONFIG"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".sablier.cfg"), nil
}

// a missing file is an empty config
func Load() (Config, error) {
	c := Config{APIURL: DefaultAPIURL}
	p, err := Path()
	if err != nil {
		return c, err
	}
	f, err := os.Open(p)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "api_key":
			c.APIKey = strings.TrimSpace(v)
		case "api_url":
			c.APIURL = strings.TrimSuffix(strings.TrimSpace(v), "/")
		}
	}
	return c, sc.Err()
}

// 0600, it holds the api key, to tmp file first so its never half written
func Save(c Config) error {
	p, err := Path()
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	body := fmt.Sprintf("api_key = %s\napi_url = %s\n", c.APIKey, c.APIURL)
	if err := os.WriteFile(tmp, []byte(body), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
