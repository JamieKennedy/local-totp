package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	apispec "github.com/JamieKennedy/local-totp/api"
	"github.com/JamieKennedy/local-totp/internal/application"
	"github.com/JamieKennedy/local-totp/internal/cli"
	"github.com/JamieKennedy/local-totp/internal/httpapi"
	"github.com/JamieKennedy/local-totp/internal/vault"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		args = []string{"help"}
	}
	if args[0] == "version" {
		fmt.Printf("local-totp %s (%s)\n", version, commit)
		return nil
	}
	if args[0] != "serve" {
		return cli.Run(context.Background(), args, os.Stdout, os.Stderr)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	dataDirectory := envDefault("LOCAL_TOTP_DATA_DIR", "./data")
	store, err := vault.Open(context.Background(), filepath.Join(dataDirectory, "local-totp.db"))
	if err != nil {
		return err
	}
	defer store.Close()
	app := application.New(store)
	status, err := app.Status(context.Background())
	if err != nil {
		return err
	}
	if status.Setup && status.Locked {
		if passwordFile := os.Getenv("LOCAL_TOTP_MASTER_PASSWORD_FILE"); passwordFile != "" {
			value, readErr := os.ReadFile(passwordFile)
			if readErr != nil {
				logger.Warn("automatic unlock file could not be read")
			} else if unlockErr := app.Unlock(context.Background(), strings.TrimSpace(string(value))); unlockErr != nil {
				logger.Warn("automatic unlock failed")
			} else {
				logger.Info("vault automatically unlocked")
			}
		}
	}
	server := httpapi.New(app, version, logger, apispec.Document())
	httpServer := &http.Server{
		Addr: listenAddress(), Handler: server.Handler(),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()
	logger.Info("Local TOTP listening", "address", httpServer.Addr, "version", version)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func listenAddress() string {
	return envDefault("LOCAL_TOTP_LISTEN_ADDR", "127.0.0.1:8080")
}

func envDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
