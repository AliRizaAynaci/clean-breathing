package main

import (
	"context"
	"errors"
	"log"
	"nasa-app/internal/app"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
)

func gracefulShutdown(app *fiber.App, done chan<- bool) {
	// Listen for interrupt signals
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done() // block until signal

	// Allow second Ctrl+C to force exit
	stop()

	// Give active connections 5 s to finish
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(timeoutCtx); err != nil {
		log.Fatalf("<UNK> Shutdown: %v", err)
	}

	done <- true // notify main() that shutdown is complete
}

func main() {

	/* ------------ build Fiber app ------------ */
	app := app.New() // all wiring (DB, routes, etc.) inside

	/* ------------ graceful shutdown ------------ */
	done := make(chan bool, 1)
	go gracefulShutdown(app, done)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // local fallback
	}

	if err := app.Listen(":" + port); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("<UNK> Server listen: %v", err)
	}

	<-done

}
