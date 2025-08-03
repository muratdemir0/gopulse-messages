package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type App struct {
	Name         string `mapstructure:"name"`
	Port         int    `mapstructure:"port"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	IdleTimeout  int    `mapstructure:"idle_timeout"`
	MaxHeaderMB  int    `mapstructure:"max_header_mb"`
}

type Webhook struct {
	Host string `mapstructure:"host"`
	Path string `mapstructure:"path"`
}

type Redis struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type Database struct {
	DSN string `mapstructure:"dsn"`
}

type Telemetry struct {
	ServiceName  string  `mapstructure:"service_name"`
	OTLPEndpoint string  `mapstructure:"otlp_endpoint"`
	Enabled      bool    `mapstructure:"enabled"`
	SampleRate   float64 `mapstructure:"sample_rate"`
}

type Config struct {
	App       App       `mapstructure:"app"`
	Webhook   Webhook   `mapstructure:"webhook"`
	Redis     Redis     `mapstructure:"redis"`
	Database  Database  `mapstructure:"database"`
	Telemetry Telemetry `mapstructure:"telemetry"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
