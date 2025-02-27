package main

import (
	"context"
	"fmt"
	"github.com/rs/zerolog"
	"key-value-storage/cmd"
	"key-value-storage/internal"
	"key-value-storage/internal/client"
	"time"
)

func main() {
	cfg, err := cmd.ReadConfig()
	if err != nil {
		fmt.Println(err)
		return
	}

	logger, err := cmd.NewLogger(internal.LoggingConfig{
		Level:  zerolog.LevelDebugValue,
		Output: "./client-output.log",
	})

	c, cl, err := client.NewClientTCP(cfg.Network.Address, logger, time.Second)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer cl()

	runner := internal.NewConsole(c, logger)
	if err := runner.Run(context.Background()); err != nil {
		fmt.Println(err)
	}
}
