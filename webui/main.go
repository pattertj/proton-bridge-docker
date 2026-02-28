package main

import (
	"context"
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pattertj/proton-bridge-docker/webui/handlers"
)

//go:embed frontend/*
var frontendFS embed.FS

func main() {
	port := flag.String("port", "3000", "Web UI server port")
	flag.Parse()

	// Initialize bridge manager
	bridge := handlers.NewBridgeManager()

	// Initialize state
	state := handlers.LoadState()

	// Create API handler
	api := handlers.NewAPI(bridge, state)

	// Setup routes
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("GET /api/health", api.Health)
	mux.HandleFunc("GET /api/status", api.Status)
	mux.HandleFunc("GET /api/accounts", api.GetAccounts)
	mux.HandleFunc("POST /api/login", api.Login)
	mux.HandleFunc("POST /api/login/2fa", api.LoginTwoFactor)
	mux.HandleFunc("POST /api/accounts/{id}/mode", api.ToggleMode)
	mux.HandleFunc("POST /api/logout/{id}", api.Logout)

	// Serve embedded frontend
	frontendSub, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		log.Fatalf("Failed to create frontend sub-filesystem: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(frontendSub)))

	server := &http.Server{
		Addr:         ":" + *port,
		Handler:      handlers.AuthMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Web UI server starting on port %s", *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-done
	log.Println("Shutting down web UI server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bridge.Close()
	server.Shutdown(ctx)
}
