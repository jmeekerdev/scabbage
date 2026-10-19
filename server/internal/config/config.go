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
	Backups    int      // nightly copies to keep; 0 turns backups off
	BackupHour int      // local hour of day the backup runs
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
	ints := map[string]int{}
	for key, def := range map[string]string{
		"SCABBAGE_TICK_RATE": "20", "SCABBAGE_BACKUPS": "7", "SCABBAGE_BACKUP_HOUR": "4",
	} {
		n, err := strconv.Atoi(or(key, def))
		if err != nil {
			return Config{}, errors.New(key + " must be an integer")
		}
		ints[key] = n
	}

	var c Config
	var origins string
	fs := flag.NewFlagSet("scabbage-server", flag.ContinueOnError)
	fs.StringVar(&c.Addr, "addr", or("SCABBAGE_ADDR", ":8080"), "listen address")
	fs.StringVar(&c.WebDir, "web-dir", or("SCABBAGE_WEB_DIR", "web"), "directory of built client files")
	fs.StringVar(&c.DataDir, "data-dir", or("SCABBAGE_DATA_DIR", "data"), "directory for the database")
	fs.StringVar(&c.ServerName, "name", or("SCABBAGE_NAME", "Scabbage"), "server display name")
	fs.IntVar(&c.TickRate, "tick-rate", ints["SCABBAGE_TICK_RATE"], "simulation ticks per second")
	fs.IntVar(&c.Backups, "backups", ints["SCABBAGE_BACKUPS"], "nightly database copies to keep (0 = off)")
	fs.IntVar(&c.BackupHour, "backup-hour", ints["SCABBAGE_BACKUP_HOUR"], "local hour (0-23) to back up")
	fs.StringVar(&origins, "origins", or("SCABBAGE_ORIGINS", ""), "comma-separated extra allowed WebSocket origins")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	if c.TickRate < 1 || c.TickRate > 120 {
		return Config{}, errors.New("tick rate must be between 1 and 120")
	}
	if c.Backups < 0 || c.BackupHour < 0 || c.BackupHour > 23 {
		return Config{}, errors.New("backups must be 0 or more and backup hour 0 to 23")
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
