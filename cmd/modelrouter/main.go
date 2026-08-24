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
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("modelrouter: load config: %v", err)
	}
	server, err := BuildComponents(cfg)
	if err != nil {
		log.Fatalf("modelrouter: build components: %v", err)
	}
	if err := server.Start(); err != nil {
		log.Fatalf("modelrouter: start: %v", err)
	}
	log.Printf("modelrouter listening on %s", cfg.Addr)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("modelrouter: shutdown: %v", err)
	}
	log.Printf("modelrouter stopped")
}
