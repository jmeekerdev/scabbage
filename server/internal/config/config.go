// Package config loads server settings from flags, falling back to SCABBAGE_* env vars.
package config

import (
	"errors"
	"flag"
	"strconv"
	"strings"
)

// Config holds server settings.
type Config struct {
	Addr       string
	WebDir     string
	DataDir    string // holds scabbage.db
	ServerName string
	TickRate   int
	Origins    []string // extra allowed WebSocket Origin hosts
}

// Load parses args (without the program name); getenv supplies defaults for unset flags.
func Load(args []string, getenv func(string) string) (Config, error) {
	or := func(key, def string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return def
	}
	tickDefault, err := strconv.Atoi(or("SCABBAGE_TICK_RATE", "20"))
	if err != nil {
		return Config{}, errors.New("SCABBAGE_TICK_RATE must be an integer")
	}

	var c Config
	var origins string
	fs := flag.NewFlagSet("scabbage-server", flag.ContinueOnError)
	fs.StringVar(&c.Addr, "addr", or("SCABBAGE_ADDR", ":8080"), "listen address")
	fs.StringVar(&c.WebDir, "web-dir", or("SCABBAGE_WEB_DIR", "web"), "directory of built client files")
	fs.StringVar(&c.DataDir, "data-dir", or("SCABBAGE_DATA_DIR", "data"), "directory for the database")
	fs.StringVar(&c.ServerName, "name", or("SCABBAGE_NAME", "Scabbage"), "server display name")
	fs.IntVar(&c.TickRate, "tick-rate", tickDefault, "simulation ticks per second")
	fs.StringVar(&origins, "origins", or("SCABBAGE_ORIGINS", ""), "comma-separated extra allowed WebSocket origins")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	if c.TickRate < 1 || c.TickRate > 120 {
		return Config{}, errors.New("tick rate must be between 1 and 120")
	}
	c.Origins = splitList(origins)
	return c, nil
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
