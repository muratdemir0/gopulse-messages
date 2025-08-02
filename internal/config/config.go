package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type App struct {
	Name string `mapstructure:"name"`
	Port int    `mapstructure:"port"`
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

type Config struct {
	App     App     `mapstructure:"app"`
	Webhook Webhook `mapstructure:"webhook"`
	Redis   Redis   `mapstructure:"redis"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
