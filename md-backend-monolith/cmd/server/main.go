package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marketing-digest/pkg/logger"

	"github.com/marketing-digest/md-backend-monolith/internal/app/article"
	"github.com/marketing-digest/md-backend-monolith/internal/app/blog"
	"github.com/marketing-digest/md-backend-monolith/internal/config"
	"github.com/marketing-digest/md-backend-monolith/internal/infrastructure/database"
	httpserver "github.com/marketing-digest/md-backend-monolith/internal/transport/http"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "md-backend-monolith fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	log := logger.New(cfg.Service)
	log.Info("starting", "env", cfg.AppEnv, "http_port", cfg.HTTPPort)

	db, err := database.Open(cfg)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer func() {
		if cerr := database.Close(db); cerr != nil {
			log.Error("database close", "error", cerr)
		}
	}()

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := database.Ping(pingCtx, db); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}

	blogStore := blog.NewGORMStore(db)
	articleStore := article.NewGORMStore(db)
	blogSvc := blog.NewService(blogStore)
	articleSvc := article.NewService(articleStore, blogStore)

	addr := fmt.Sprintf(":%d", cfg.HTTPPort)
	srv := httpserver.New(addr, blogSvc, articleSvc, log, cfg.CORSAllowedOrigins, cfg.OwnerStudioSecret)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-sigCh:
		log.Info("shutdown signal", "signal", sig.String())
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	log.Info("shutdown complete")
	return nil
}