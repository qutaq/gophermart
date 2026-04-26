package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"

	"github.com/qutaq/gophermart/internal/config"
	"github.com/qutaq/gophermart/internal/storage/postgres"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("gophermart: %v", err)
	}
}

func run() error {
	cfg := config.Parse()

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	store, err := postgres.New(ctx, cfg.DatabaseURI)
	if err != nil {
		return err
	}
	defer store.Close()

	log.Printf("gophermart: started run_address=%q accrual=%q",
		cfg.RunAddress, cfg.AccrualSystemAddress)

	<-ctx.Done()
	if err := ctx.Err(); err != nil &&
		!errors.Is(err, context.Canceled) {
		return err
	}

	log.Println("gophermart: shutdown complete")
	return nil
}
