package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	URL      string            `mapstructure:"url"`
	APIKey   string            `mapstructure:"api_key"`
	Headers  map[string]string `mapstructure:"headers"`
	Timeout  time.Duration     `mapstructure:"timeout"`
	LogLevel string            `mapstructure:"log_level"`
}

func Load(path string) (Config, error) {
	v := viper.New()
	v.SetEnvPrefix("APM")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("timeout", "30s")
	v.SetDefault("log_level", "info")

	for _, k := range []string{"url", "api_key", "timeout", "log_level"} {
		_ = v.BindEnv(k)
	}

	if path != "" {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			var nf viper.ConfigFileNotFoundError
			if !errors.As(err, &nf) && !os.IsNotExist(err) {
				return Config{}, fmt.Errorf("config: read: %w", err)
			}
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("config: unmarshal: %w", err)
	}

	if cfg.URL == "" {
		return Config{}, errors.New("config: url is required")
	}

	if cfg.APIKey == "" {
		return Config{}, errors.New("config: api_key is required")
	}

	return cfg, nil
}
