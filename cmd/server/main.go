package main

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"key-value-storage/cmd"
	"key-value-storage/internal"
)

func main() {
	cfg, err := cmd.ReadConfig()
	if err != nil {
		fmt.Println(err)
		fmt.Println("warning: using default config")
		return
	}

	logger, err := cmd.NewLogger(cfg.Logging)
	if err != nil {
		fmt.Println(err)
		return
	}

	db, err := newDB(cfg.Engine, logger)
	if err != nil {
		fmt.Println(err)
		return
	}

	runner, err := newRunner(cfg, db, logger)
	if err != nil {
		fmt.Println(err)
		return
	}

	ctx := context.Background()
	err = runner.Run(ctx)
	if err != nil {
		fmt.Println(err)
	}
}

func newDB(cfg internal.EngineConfig, logger zerolog.Logger) (*internal.DB, error) {
	if cfg.Type == internal.InMemoryEngineType {
		return internal.NewDB(
			internal.NewParser(logger),
			internal.NewStorage(internal.NewInMemoryEngine(), logger),
			logger,
		), nil
	}

	return nil, errors.New("unknown engine type")
}

type iRunner interface {
	Run(ctx context.Context) error
}

func newRunner(cfg internal.Config, db *internal.DB, logger zerolog.Logger) (iRunner, error) {
	if cfg.Mode == internal.ConsoleAppMode {
		return internal.NewConsole(db, logger), nil
	}

	if cfg.Mode == internal.TCPAppMode {
		return internal.NewServerTCP(cfg.Network, db, logger), nil
	}

	return nil, errors.New("invalid app mode")
}
