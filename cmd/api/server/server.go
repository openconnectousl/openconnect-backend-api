package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/OpenConnectOUSL/backend-api-v1/cmd/api/app"
	"github.com/OpenConnectOUSL/backend-api-v1/cmd/api/routes"
)

// Serve starts the HTTP server with graceful shutdown
func Serve(appPtr *app.Application) error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", appPtr.Config.Port),
		Handler:      routes.Setup(appPtr),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	shutdownError := make(chan error)

	// Background goroutine for graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		s := <-quit

		appPtr.Logger.PrintInfo("caught signal", map[string]string{
			"signal": s.String(),
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		appPtr.Logger.PrintInfo("completing background tasks", map[string]string{
			"addr": srv.Addr,
		})

		appPtr.WG.Wait()
		shutdownError <- nil
	}()

	appPtr.Logger.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
		"env":  appPtr.Config.Env,
	})

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError
	if err != nil {
		return err
	}

	appPtr.Logger.PrintInfo("stopped server", map[string]string{
		"addr": srv.Addr,
	})

	return nil
}
