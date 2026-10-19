package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"
	"time"

	"scabbage/internal/access"
	"scabbage/internal/art"
	"scabbage/internal/backup"
	"scabbage/internal/config"
	"scabbage/internal/gateway"
	"scabbage/internal/sim"
	"scabbage/internal/store"
	"scabbage/internal/web"
	"scabbage/internal/world"
)

// autosaveEvery bounds how much building a crash can lose while players are on.
const autosaveEvery = time.Minute

func main() {
	cfg, err := config.Load(os.Args[1:], os.Getenv)
	if err != nil {
		os.Exit(2)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(cfg, logger); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, logger *slog.Logger) error {
	db, err := openDB(cfg.DataDir)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	w, err := world.Open(world.NewStore(db), "main", world.Defaults{})
	if err != nil {
		return err
	}
	defer saveWorld(w, logger)
	logger.Info("world ready", "seed", w.Meta().Seed, "generator", w.Meta().Generator, "radius", w.Meta().Radius)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if cfg.Backups > 0 {
		go backup.Daily(ctx, cfg.BackupHour, nightlyBackup(db, w, cfg, logger))
	}

	var gw *gateway.Gateway
	saveTicks := uint64(autosaveEvery.Seconds()) * uint64(cfg.TickRate)
	hub := sim.NewHub(sim.Config{
		TickRate: cfg.TickRate,
		OnTick: func(tick uint64) {
			gw.Tick()
			if tick%saveTicks == 0 {
				go saveWorld(w, logger) // keeps disk writes off the tick loop
			}
		},
		OnIdle: unloadWorld(w, logger),
	})
	defer hub.Close()

	svc := access.New(db, access.Options{OnPendingChange: func() { gw.PendingChanged() }})
	pieces := art.New(db, art.Options{OnPendingChange: func() { gw.ArtChanged() }, Reserved: world.IsBuiltinName})
	gw, err = gateway.New(gateway.Config{
		Hub:            hub,
		Access:         svc,
		Art:            pieces,
		World:          w,
		ServerName:     cfg.ServerName,
		Mode:           access.Mode(cfg.Mode),
		OriginPatterns: cfg.Origins,
		Logger:         logger,
	})
	if err != nil {
		return err
	}
	if err := logAdminInvite(svc, logger); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/ws", gw)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok\n")) })
	mux.Handle("/", web.Static(cfg.WebDir))

	srv := &http.Server{Addr: cfg.Addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	logger.Info("listening", "addr", cfg.Addr, "web_dir", cfg.WebDir)

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func saveWorld(w *world.World, logger *slog.Logger) {
	if err := w.Flush(); err != nil {
		logger.Error("saving world", "err", err)
	}
}

// nightlyBackup saves the world, then copies the database into data/backups.
func nightlyBackup(db *sql.DB, w *world.World, cfg config.Config, logger *slog.Logger) func(time.Time) {
	b := backup.Backups{DB: db, Dir: filepath.Join(cfg.DataDir, "backups"), Keep: cfg.Backups}
	return func(now time.Time) {
		saveWorld(w, logger)
		path, err := b.Snapshot(now)
		if err != nil {
			logger.Error("backing up", "err", err)
			return
		}
		logger.Info("backed up", "path", path)
	}
}

// unloadWorld saves and drops the world, then returns freed memory to the OS.
func unloadWorld(w *world.World, logger *slog.Logger) func() {
	return func() {
		chunks := w.Loaded()
		if err := w.Unload(); err != nil {
			logger.Error("unloading world", "err", err)
			return
		}
		debug.FreeOSMemory()
		logger.Info("idle: world unloaded", "chunks", chunks)
	}
}

func openDB(dir string) (*sql.DB, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	return store.Open(filepath.Join(dir, "scabbage.db"))
}

// logAdminInvite prints a one-time admin code until the first admin has joined.
func logAdminInvite(svc *access.Service, logger *slog.Logger) error {
	code, ok, err := svc.AdminInvite()
	if err != nil || !ok {
		return err
	}
	logger.Warn("no admin yet: open the game and join with this code to become admin, "+
		"or add #join=CODE to the game URL", "code", code)
	return nil
}
