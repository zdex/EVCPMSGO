package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cpms/internal/config"
	"cpms/internal/db"
	"cpms/internal/httpapi"
	"cpms/internal/repo"
	"cpms/internal/services"
	"cpms/internal/gatewayclient"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	d, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	chargers := repo.NewChargersRepo(d.Pool)
	events := repo.NewEventsRepo(d.Pool)
	state := repo.NewStateRepo(d.Pool)
	sessions := repo.NewSessionsRepo(d.Pool)
	commands := repo.NewCommandsRepo(d.Pool)

	gw := gatewayclient.New(cfg.GatewayBaseURL, cfg.GatewayAPIKey)

	processor := services.NewEventsProcessor(events, chargers, state, sessions, cfg.MaxEventSkew)
	srv := httpapi.NewServer(cfg, chargers, state, sessions, commands, gw, processor)

	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Println("CPMS listening on", cfg.ListenAddr)
		log.Fatal(httpServer.ListenAndServe())
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	_ = httpServer.Shutdown(ctx2)
	log.Println("CPMS shutdown complete")
}
