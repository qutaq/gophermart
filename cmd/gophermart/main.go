package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/qutaq/gophermart/internal/config"
	"github.com/qutaq/gophermart/internal/handler"
	"github.com/qutaq/gophermart/internal/repository"
	"github.com/qutaq/gophermart/internal/storage"
)

const shutdownTimeout = 5 * time.Second

func main() {
	if err := run(); err != nil {
		log.Fatalf("gophermart: %v", err)
	}
}

func run() error {
	cfg := config.Parse()
	if cfg.JWTSecret == "" {
		return errors.New("config: пустой JWT_SECRET")
	}

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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

	router := chi.NewRouter()
	handler.New(userRepository, orderRepository, withdrawalRepository, []byte(cfg.JWTSecret)).RegisterRoutes(router)

	server := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: router,
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	log.Printf("gophermart: started run_address=%q accrual=%q",
		cfg.RunAddress, cfg.AccrualSystemAddress)

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		if err := <-serverErr; err != nil {
			return err
		}
	}

	log.Println("gophermart: shutdown complete")
	return nil
}
