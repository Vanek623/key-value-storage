package cmd

import (
	"net"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"

	"key-value-storage/internal"
)

func ReadConfig() (internal.Config, error) {
	// Создание структуры для хранения данных
	config := internal.Config{
		Mode: internal.ConsoleAppMode,
		Engine: internal.EngineConfig{
			Type: internal.InMemoryEngineType,
		},
		Network: internal.NetworkConfig{
			Address: net.TCPAddr{
				IP:   net.ParseIP("127.0.0.1"),
				Port: 3333,
			},
			MaxConnections: 100,
			MaxMessageSize: 4096,
			IdleTimeout:    5 * time.Minute,
		},
		Logging: internal.LoggingConfig{
			Level:  "info",
			Output: "console",
		},
	}

	data, err := os.ReadFile("kv-storage-config.yaml")
	if err != nil {
		return internal.Config{}, errors.Wrapf(err, "failed to read kv-storage-config.yaml")
	}

	if err = yaml.Unmarshal(data, &config); err != nil {
		return internal.Config{}, errors.Wrapf(err, "failed to unmarshal kv-storage-config.yaml")
	}

	return config, nil
}

func NewLogger(cfg internal.LoggingConfig) (zerolog.Logger, error) {
	if cfg.Output == internal.ConsoleLogOutput {
		return zerolog.New(os.Stdout), nil
	}

	file, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return zerolog.Logger{}, errors.Wrapf(err, "can't open file %s", cfg.Output)
	}

	if err = file.Truncate(0); err != nil {
		return zerolog.Logger{}, errors.Wrapf(err, "can't truncate file %s", cfg.Output)
	}

	return zerolog.New(file).With().Timestamp().Logger(), nil
}
