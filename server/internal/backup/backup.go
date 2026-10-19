// Package backup writes dated copies of the database and prunes old ones.
package backup

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	prefix = "scabbage-"
	suffix = ".db"
	stamp  = "20060102-150405"
)

// Backups snapshots DB into Dir, keeping the newest Keep copies.
type Backups struct {
	DB   *sql.DB
	Dir  string
	Keep int
}

// Snapshot writes a consistent copy named for now, then removes the oldest beyond Keep.
func (b Backups) Snapshot(now time.Time) (string, error) {
	if err := os.MkdirAll(b.Dir, 0o750); err != nil {
		return "", err
	}
	path := filepath.Join(b.Dir, prefix+now.Format(stamp)+suffix)
	_ = os.Remove(path) // VACUUM INTO refuses to overwrite
	if _, err := b.DB.Exec(`VACUUM INTO ?`, path); err != nil {
		return "", err
	}
	return path, b.prune()
}

func (b Backups) prune() error {
	entries, err := os.ReadDir(b.Dir)
	if err != nil {
		return err
	}
	var snaps []string
	for _, e := range entries {
		if n := e.Name(); strings.HasPrefix(n, prefix) && strings.HasSuffix(n, suffix) {
			snaps = append(snaps, n)
		}
	}
	slices.Sort(snaps) // timestamps sort oldest first
	for len(snaps) > b.Keep {
		if err := os.Remove(filepath.Join(b.Dir, snaps[0])); err != nil {
			return err
		}
		snaps = snaps[1:]
	}
	return nil
}

// NextRun returns the first time after now that the clock reads hour:00.
func NextRun(now time.Time, hour int) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

// Daily calls run at hour:00 local time each day until ctx ends; it sleeps in between.
func Daily(ctx context.Context, hour int, run func(now time.Time)) {
	for {
		timer := time.NewTimer(time.Until(NextRun(time.Now(), hour)))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case now := <-timer.C:
			run(now)
		}
	}
}
