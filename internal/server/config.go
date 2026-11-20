package server

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	Addr        string
	DatabaseURL string
	BaseURL     string // public url
	SiteDir     string // the built website, served for routes the server doesn't handle
	SiteProxy   string // in dev, astro's dev server instead of SiteDir

	HackClubClientID     string
	HackClubClientSecret string
}

func ConfigFromEnv() (Config, error) {
	c := Config{
		Addr:                 os.Getenv("SABLIER_ADDR"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		BaseURL:              strings.TrimSuffix(os.Getenv("SABLIER_BASE_URL"), "/"),
		SiteDir:              os.Getenv("SABLIER_SITE_DIR"),
		SiteProxy:            os.Getenv("SABLIER_SITE_PROXY"),
		HackClubClientID:     os.Getenv("HACKCLUB_CLIENT_ID"),
		HackClubClientSecret: os.Getenv("HACKCLUB_CLIENT_SECRET"),
	}
	if c.Addr == "" {
		c.Addr = ":8080"
	}
	if c.SiteDir == "" {
		c.SiteDir = "website/dist"
	}
	if c.BaseURL == "" {
		c.BaseURL = "http://localhost:8080"
	}
	if c.SiteProxy != "" {
		if u, err := url.Parse(c.SiteProxy); err != nil || u.Host == "" {
			return c, fmt.Errorf("SABLIER_SITE_PROXY %q is not a url", c.SiteProxy)
		}
	}
	if c.DatabaseURL == "" {
		return c, errors.New("DATABASE_URL is not set")
	}
	return c, nil
}
