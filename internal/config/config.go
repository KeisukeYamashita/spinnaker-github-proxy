package config

import (
	"context"
	"fmt"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	Port         int    `env:"PORT,required"`
	Log          string `env:"LOG"`
	Organization string `env:"ORGANIZATION,required"`
}

func LoadConfig(ctx context.Context) (*Config, error) {
	var cfg Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Address returns a value which can be used for HTTP server
func (cfg *Config) Address() string {
	return fmt.Sprintf(":%d", cfg.Port)
}
