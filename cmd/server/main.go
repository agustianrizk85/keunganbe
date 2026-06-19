// Command server starts the Finance (keuangan) control dashboard API.
//
// It wires the layers together — repository -> service -> HTTP transport — and
// runs an HTTP server with graceful shutdown.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"greenpark/finance/internal/auth"
	"greenpark/finance/internal/config"
	"greenpark/finance/internal/gsheets"
	"greenpark/finance/internal/repository"
	"greenpark/finance/internal/service"
	httptransport "greenpark/finance/internal/transport/http"
)

func main() {
	cfg := config.Load()

	// Dependency wiring (composition root). Use PostgreSQL when a DSN is set,
	// otherwise persist to the JSON file.
	var (
		repo repository.FinanceRepository
		err  error
	)
	if cfg.DatabaseURL != "" {
		repo, err = repository.NewPostgresRepository(cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("finance: postgres: %v", err)
		}
		log.Println("finance: using PostgreSQL store")
	} else {
		repo, err = repository.NewRepository(cfg.DataPath)
		if err != nil {
			log.Fatalf("finance: failed to open data store %q: %v", cfg.DataPath, err)
		}
		log.Println("finance: using file store")
	}
	svc := service.New(repo, service.Options{FocusYear: cfg.FocusYear, TargetAkad: cfg.TargetAkad})
	authSvc := auth.New(repo, cfg.SessionTTL)

	syncClient, err := gsheets.New(cfg.GoogleCreds)
	if err != nil {
		log.Fatalf("finance: google credentials: %v", err)
	}
	if syncClient != nil {
		log.Printf("finance: Google Sheets sync enabled (sheet %s)", cfg.SheetID)
	} else {
		log.Println("finance: Google Sheets sync disabled (set FINANCE_GOOGLE_CREDENTIALS to enable)")
	}

	handler := httptransport.NewHandler(svc, authSvc, syncClient, cfg.SheetID, cfg.SyncSec)
	router := httptransport.NewRouter(handler, cfg.AllowOrigin)

	// Realtime push + background auto-sync scheduler.
	ctx, cancelBg := context.WithCancel(context.Background())
	defer cancelBg()
	handler.StartRealtime()
	handler.StartAutoSync(ctx)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Run the server in a goroutine so main can wait for shutdown signals.
	go func() {
		log.Printf("finance API listening on http://localhost:%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("finance: server error: %v", err)
		}
	}()

	// Wait for an interrupt signal for graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("finance: shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("finance: graceful shutdown failed: %v", err)
	}
	log.Println("finance: stopped")
}
