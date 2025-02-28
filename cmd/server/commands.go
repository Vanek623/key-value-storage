package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"key-value-storage/internal"
	"net"
	"os"
	"time"
)

var rootCmd = &cobra.Command{
	Use:   "key-value-storage",
	Short: "kv storage by Balun HW",
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
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
)

var cfgFilePath string

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFilePath, "config", "c", "",
		"config file (default is"+defaultConfigFilename+")",
	)
	rootCmd.PersistentFlags().IPP("address", "a", net.ParseIP(defaultAddress),
		"server address (default"+defaultAddress+")",
	)
	rootCmd.PersistentFlags().IntP("port", "p", 0,
		fmt.Sprintf("server port (default %d)", defaultPort),
	)
	rootCmd.PersistentFlags().Int("max-connections", 0,
		fmt.Sprintf("maximum connections at one time (default %d)", defaultMaxConnections),
	)
	rootCmd.PersistentFlags().Int("max-message-size", 0,
		fmt.Sprintf("maximum message size (default %d)", defaultMaxMessageSize),
	)
	rootCmd.PersistentFlags().Duration("idle-timeout", 0,
		"close tcp connection if has no activity in (default"+defaultIdleTimeout.String()+")",
	)

	viper.BindPFlag("address", rootCmd.PersistentFlags().Lookup("address"))
	viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("max-connections", rootCmd.PersistentFlags().Lookup("max-connections"))
	viper.BindPFlag("max-message-size", rootCmd.PersistentFlags().Lookup("max-message-size"))
	viper.BindPFlag("idle-timeout", rootCmd.PersistentFlags().Lookup("idle-timeout"))

	viper.SetDefault("address", net.ParseIP(defaultAddress))
	viper.SetDefault("port", 3333)

	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(initCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

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
		viper.SetConfigType("yaml")
		viper.SetConfigName(".cobra")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
