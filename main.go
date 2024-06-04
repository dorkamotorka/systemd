package main

import (
		"os"
		"os/signal"
		"syscall"
    "log"
    "net/http"
		"context"
		"time"

    "github.com/coreos/go-systemd/activation"
)

func main() {
	// Retrieve the listening sockets provided by systemd for this process
	// Systemd know that "main.socket" belongs in pair with the "main.service" due to the matching filename
	listeners, err := activation.Listeners()
	if err != nil {
		log.Panicf("cannot retrieve listeners: %s", err)
	}
	if len(listeners) != 1 {
		log.Panicf("unexpected number of socket activation (%d != 1)", len(listeners))
	}

	// On signal, gracefully shut down the server and wait 5
	// seconds for current connections to stop.
	done := make(chan struct{})
	quit := make(chan os.Signal, 1)
	server := &http.Server{}
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Server is shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
		defer cancel()
		server.SetKeepAlivesEnabled(false)
		// Shutdown() gracefully shuts down the server without interrupting any active connections. 
		// Shutdown() works by first closing all open listeners, then closing all idle connections, and 
		// then waiting indefinitely for connections to return to idle and then shut down.
		if err := server.Shutdown(ctx); err != nil {
			log.Panicf("Cannot gracefully shut down the server: %s", err)
		}
		close(done)
	}()

	// There can be multiple listening socket, but we only use the first one in this case
	server.Serve(listeners[0]) 

	// Wait for existing connections before exiting.
	<-done
}