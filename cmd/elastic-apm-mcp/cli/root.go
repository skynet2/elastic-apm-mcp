package cli

import (
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/skynet2/elastic-apm-mcp/cmd/elastic-apm-mcp/mcp"
	"github.com/skynet2/elastic-apm-mcp/internal/config"
	"github.com/skynet2/elastic-apm-mcp/pkg/apm"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

var configPath string

var rootCmd = &cobra.Command{
	Use:   "elastic-apm-mcp",
	Short: "MCP stdio server for Elastic APM",
	RunE:  run,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "config.yaml", "path to config file")
}

func Execute() error {
	return rootCmd.Execute()
}

func run(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().Timestamp().Logger().
		Level(parseLevel(cfg.LogLevel))

	client := apm.New(apm.Config{
		BaseURL:      cfg.URL,
		APIKey:       cfg.APIKey,
		Headers:      cfg.Headers,
		AppLogsIndex: cfg.AppLogsIndex,
		HTTPClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	})

	srv := mcp.NewServer(client, log, time.Now, "0.1.0")

	log.Info().Str("url", cfg.URL).Msg("starting elastic-apm-mcp")

	return mcpserver.ServeStdio(srv)
}

func parseLevel(s string) zerolog.Level {
	lvl, err := zerolog.ParseLevel(s)
	if err != nil {
		return zerolog.InfoLevel
	}
	return lvl
}
