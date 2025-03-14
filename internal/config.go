package internal

import (
	"net"
	"time"
)

type EngineType string

const (
	InMemoryEngineType EngineType = "in-memory"
)

// EngineConfig представляет конфигурацию движка
type EngineConfig struct {
	Type EngineType `yaml:"type"`
}

// NetworkConfig представляет конфигурацию сети
type NetworkConfig struct {
	MaxConnections int           `yaml:"max_connections" mapstructure:"max_connections"`
	MaxMessageSize int           `yaml:"max_message_size" mapstructure:"max_message_size"`
	IdleTimeout    time.Duration `yaml:"idle_timeout" mapstructure:"idle_timeout"`
	Address        net.TCPAddr   `yaml:"address" mapstructure:"address"`
}

const ConsoleLogOutput = "console"

// LoggingConfig представляет конфигурацию логирования
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Output string `yaml:"output"`
}

type WalConfig struct {
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
	// BatchSize  предельный размер батча для записи на диск
	BatchSize int `yaml:"flushing_batch_size" mapstructure:"flushing_batch_size"`
	// BatchTimeout предельное время начала записи батча
	BatchTimeout time.Duration `yaml:"flushing_batch_timeout" mapstructure:"flushing_batch_timeout"`
	// SegmentSize размер сегмента на ЖД
	SegmentSize int `yaml:"max_segment_size" mapstructure:"max_segment_size"`
	// DataDir место куда сохранять данные на диск
	DataDir string `yaml:"data_directory" mapstructure:"data_directory"`
}

const (
	ConsoleAppMode = "console"
	TCPAppMode     = "tcp"
)

// Config представляет основную структуру конфигурации
type Config struct {
	Mode    string        `yaml:"mode"`
	Engine  EngineConfig  `yaml:"engine"`
	Network NetworkConfig `yaml:"network"`
	Logging LoggingConfig `yaml:"logging"`
	Wal     WalConfig     `yaml:"wal"`
}
