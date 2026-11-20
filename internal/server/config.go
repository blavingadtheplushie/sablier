package server

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	Addr        string
	DatabaseURL string
	BaseURL     string // public url
	SiteDir     string // the website's static files, for the shared css and fonts

	HackClubClientID     string
	HackClubClientSecret string
}

func ConfigFromEnv() (Config, error) {
	c := Config{
		Addr:                 os.Getenv("SABLIER_ADDR"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		BaseURL:              strings.TrimSuffix(os.Getenv("SABLIER_BASE_URL"), "/"),
		SiteDir:              os.Getenv("SABLIER_SITE_DIR"),
		HackClubClientID:     os.Getenv("HACKCLUB_CLIENT_ID"),
		HackClubClientSecret: os.Getenv("HACKCLUB_CLIENT_SECRET"),
	}
	if c.Addr == "" {
		c.Addr = ":8080"
	}
	if c.SiteDir == "" {
		c.SiteDir = "website/public"
	}
	if c.BaseURL == "" {
		c.BaseURL = "http://localhost:8080"
	}
	if c.DatabaseURL == "" {
		return c, errors.New("DATABASE_URL is not set")
	}
	return c, nil
}
