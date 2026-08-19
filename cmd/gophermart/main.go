package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/qutaq/gophermart/internal/accrual"
	"github.com/qutaq/gophermart/internal/config"
	"github.com/qutaq/gophermart/internal/handler"
	"github.com/qutaq/gophermart/internal/repository"
	"github.com/qutaq/gophermart/internal/storage"
	"github.com/qutaq/gophermart/internal/worker"
	"golang.org/x/sync/errgroup"
)

const shutdownTimeout = 5 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("gophermart failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Parse()

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	g, ctx := errgroup.WithContext(ctx)

	store, err := storage.New(ctx, cfg.DatabaseURI)
	if err != nil {
		return err
	}
	defer store.Close()

	if err := store.MigrateUp(ctx); err != nil {
		return err
	}

	userRepository := repository.NewUserRepository(store.Pool())
	orderRepository := repository.NewOrderRepository(store.Pool())
	withdrawalRepository := repository.NewWithdrawalRepository(store.Pool())

	accrualClient := accrual.NewClient(cfg.AccrualSystemAddress)
	poller := worker.New(orderRepository, accrualClient, worker.WithPollInterval(1*time.Second), worker.WithBatchSize(10))

	router := chi.NewRouter()
	handler.New(userRepository, orderRepository, withdrawalRepository, []byte(cfg.JWTSecret)).RegisterRoutes(router)

	server := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: router,
	}

	g.Go(func() error {
		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
	g.Go(func() error {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		return poller.Run(ctx)
	})

	slog.Info("gophermart started",
		"run_address", cfg.RunAddress,
		"accrual", cfg.AccrualSystemAddress,
	)

	if err := g.Wait(); err != nil {
		return err
	}

	slog.Info("gophermart shutdown complete")
	return nil
}
