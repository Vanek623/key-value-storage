package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"key-value-storage/cmd"
	"key-value-storage/internal"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "key-value-storage",
	Short: "kv storage by Balun HW",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("use 'help' for more info")
		fmt.Println("use 'run' to run db")
	},
}

var helpCmd = &cobra.Command{
	Use:   "help",
	Short: "show command info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use: GET [KEY] [VALUE], SET [key] [value], DEL [key]")
	},
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run server",
	Run: func(cmd *cobra.Command, args []string) {
		StartServer(cfg)
	},
}

const (
	defaultConfigFilename = "$HOME/key-value-storage.yaml"
	defaultAppMode        = internal.ConsoleAppMode
	defaultEngineType     = internal.InMemoryEngineType
	defaultAddress        = "127.0.0.1"
	defaultPort           = 3333
	defaultMaxConnections = 10
	defaultMaxMessageSize = 4096
	defaultIdleTimeout    = 5 * time.Minute
	defaultLogLevel       = "info"
	defaultLogOutput      = "console"
)

var cfgFilePath string

func init() {
	cobra.OnInitialize(initConfig)

	runCmd.PersistentFlags().StringVarP(&cfgFilePath, "config", "c", "",
		"config file (default is"+defaultConfigFilename+")",
	)
	runCmd.PersistentFlags().IPP("address", "a", net.ParseIP(defaultAddress),
		"server address (default"+defaultAddress+")",
	)
	runCmd.PersistentFlags().IntP("port", "p", 0,
		fmt.Sprintf("server port (default %d)", defaultPort),
	)
	runCmd.PersistentFlags().Int("max-connections", 0,
		fmt.Sprintf("maximum connections at one time (default %d)", defaultMaxConnections),
	)
	runCmd.PersistentFlags().Int("max-message-size", 0,
		fmt.Sprintf("maximum message size (default %d)", defaultMaxMessageSize),
	)
	runCmd.PersistentFlags().Duration("idle-timeout", 0,
		"close tcp connection if has no activity in (default"+defaultIdleTimeout.String()+")",
	)
	runCmd.PersistentFlags().StringP("engine", "", "",
		"engine type (default "+string(defaultEngineType)+")",
	)
	runCmd.PersistentFlags().StringP("mode", "m", defaultAppMode,
		"application mode 'console' or tcp (default "+defaultAppMode+")",
	)
	runCmd.PersistentFlags().StringP("log-level", "l", "",
		"log level (default "+defaultLogLevel+")",
	)
	runCmd.PersistentFlags().StringP("log-output", "o", "",
		"log output 'console' or 'file' (default "+defaultLogOutput+")",
	)
	
	if err := viper.BindPFlag("network.address.ip", runCmd.PersistentFlags().Lookup("address")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("network.address.port", runCmd.PersistentFlags().Lookup("port")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("network.max_connections", runCmd.PersistentFlags().Lookup("max-connections")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("network.max_message_size", runCmd.PersistentFlags().Lookup("max-message-size")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("network.idle_timeout", runCmd.PersistentFlags().Lookup("idle-timeout")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("engine.type", runCmd.PersistentFlags().Lookup("engine")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("mode", runCmd.PersistentFlags().Lookup("mode")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("logging.level", runCmd.PersistentFlags().Lookup("log-level")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("logging.output", runCmd.PersistentFlags().Lookup("log-output")); err != nil {
		panic(err)
	}

	viper.SetDefault("mode", defaultAppMode)
	viper.SetDefault("network.address.ip", net.ParseIP(defaultAddress))
	viper.SetDefault("network.address.port", defaultPort)
	viper.SetDefault("network.max_connections", defaultMaxConnections)
	viper.SetDefault("network.max_message_size", defaultMaxMessageSize)
	viper.SetDefault("network.idle_timeout", defaultIdleTimeout.String())
	viper.SetDefault("engine.type", defaultEngineType)
	viper.SetDefault("logging.level", defaultLogLevel)
	viper.SetDefault("logging.output", defaultLogOutput)

	rootCmd.AddCommand(helpCmd)
	rootCmd.AddCommand(runCmd)
}

var cfg internal.Config

func initConfig() {
	if cfgFilePath != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFilePath)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".cobra" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("kv-storage-config")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return
	}

	fmt.Println("Using config file:", viper.ConfigFileUsed())
	if err := viper.Unmarshal(&cfg); err != nil {
		panic("unable to decode config: " + err.Error())
	}

	fmt.Println(cfg)
}

func StartServer(cfg internal.Config) {
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
