package http

import (
	"context"
	"github.com/heyjorgedev/deploykit/pkg/core"
	"net"
	"net/http"
	"sync"
	"time"
)

type ServeConfig struct {
	// HttpAddr is the TCP address to listen for the HTTP server (eg. `127.0.0.1:80`).
	HttpAddr string
}

func Serve(app core.App, config ServeConfig) error {
	app.Logger().Info("Starting HTTP server...")

	router, err := newRouter(app)
	if err != nil {
		return err
	}

	// base request context used for cancelling long running requests
	// like the SSE connections
	baseCtx, cancelBaseCtx := context.WithCancel(context.Background())
	defer cancelBaseCtx()

	server := &http.Server{
		ReadTimeout:       10 * time.Minute,
		ReadHeaderTimeout: 30 * time.Second,
		// WriteTimeout: 60 * time.Second, // breaks sse!
		Handler: router,
		Addr:    config.HttpAddr,
		BaseContext: func(l net.Listener) context.Context {
			return baseCtx
		},
	}

	var wg sync.WaitGroup
	app.OnTerminate().Add(func(e *core.TerminateEvent) error {
		app.Logger().Info("Stopping HTTP server...")
		cancelBaseCtx()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		wg.Add(1)
		_ = server.Shutdown(ctx)
		wg.Done()

		return nil
	})

	defer wg.Wait()

	return server.ListenAndServe()
}