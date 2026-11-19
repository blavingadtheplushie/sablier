package server

import (
	"errors"
	"os"
)

type Config struct {
	Addr        string
	DatabaseURL string
}

func ConfigFromEnv() (Config, error) {
	c := Config{
		Addr:        os.Getenv("SABLIER_ADDR"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
	if c.Addr == "" {
		c.Addr = ":8080"
	}
	if c.DatabaseURL == "" {
		return c, errors.New("DATABASE_URL is not set")
	}
	return c, nil
}
