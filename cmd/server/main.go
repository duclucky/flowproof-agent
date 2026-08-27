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

	"github.com/ducky/flowproof-agent/internal/browser"
	"github.com/ducky/flowproof-agent/internal/config"
	"github.com/ducky/flowproof-agent/internal/httpapi"
	"github.com/ducky/flowproof-agent/internal/orchestrator"
	"github.com/ducky/flowproof-agent/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	st, err := store.NewFileStore(cfg.StatePath)
	if err != nil {
		log.Fatalf("open state store: %v", err)
	}
	driver := browser.NewDriver(cfg.ChromePath, cfg.StepTimeout)
	svc := orchestrator.New(st, driver)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.New(cfg, svc),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	log.Printf("FlowProof listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve: %v", err)
	}
}
