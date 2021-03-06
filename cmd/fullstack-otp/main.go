package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/lauslim12/fullstack-otp/internal/application"
)

// Get port from environment variable. If it does not exist, use '8080'.
func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		return ":8080"
	}

	return fmt.Sprintf(":%s", port)
}

// Starting point, initialize server.
func main() {
	// Add dependency: Redis.
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	// HTTP server initialization with dependency injection.
	server := &http.Server{Addr: getPort(), Handler: application.Configure(rdb)}

	// Prepare context for graceful shutdown.
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	// Listen for syscall signals for process to interrupt or quit.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig

		// Shutdown signal with grace period of 30 seconds.
		shutdownCtx, shutdownCtxCancel := context.WithTimeout(serverCtx, 30*time.Second)
		defer shutdownCtxCancel()
		log.Println("Server starting to shutdown in 30 seconds...")

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Fatal("Graceful shutdown timeout, forcing exit.")
			}
		}()

		// Trigger graceful shutdown here.
		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		serverStopCtx()
	}()

	// Run our server and print out starting message.
	log.Printf("Server has started on port %s!", getPort())
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

	// Wait for server context to be stopped.
	<-serverCtx.Done()
	log.Println("Server shut down successfully!")
}
