package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/soyvural/mcp-server-openapi/executor"
	"github.com/soyvural/mcp-server-openapi/server"
	"github.com/soyvural/mcp-server-openapi/toolgen"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

var version = "dev"

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "mcp-server-openapi",
	Short: "Serve OpenAPI endpoints as MCP tools",
	Long:  "Automatically converts OpenAPI-tagged endpoints into MCP tools for LLM integration.",
}

func init() {
	rootCmd.PersistentFlags().String("spec", "", "OpenAPI spec file path or URL (required)")
	rootCmd.PersistentFlags().String("tag", "mcp", "Tag to filter operations")
	rootCmd.PersistentFlags().String("server-url", "", "Override base URL from spec")
	rootCmd.PersistentFlags().Duration("timeout", 30*time.Second, "HTTP request timeout")
	rootCmd.PersistentFlags().String("auth-type", "", "Auth type: bearer or api-key")
	rootCmd.PersistentFlags().String("auth-token-env", "", "Env var name for bearer token")
	rootCmd.PersistentFlags().String("auth-key-env", "", "Env var name for API key")
	rootCmd.PersistentFlags().String("auth-key-name", "", "Header/query param name for API key")
	rootCmd.PersistentFlags().String("auth-key-in", "", "Where to send API key: header or query")
	rootCmd.PersistentFlags().String("log-level", "info", "Log level: debug, info, warn, error")
	rootCmd.PersistentFlags().String("log-file", "", "Log file path (default: stderr)")

	viper.SetEnvPrefix("OPENAPI_MCP")
	viper.AutomaticEnv()

	for _, name := range []string{"spec", "tag", "server-url", "timeout", "auth-type", "auth-token-env", "auth-key-env", "auth-key-name", "auth-key-in", "log-level", "log-file"} {
		if err := viper.BindPFlag(name, rootCmd.PersistentFlags().Lookup(name)); err != nil {
			panic(fmt.Sprintf("failed to bind flag %q: %v", name, err))
		}
	}

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	})
}

func buildServer() (*mcpserver.MCPServer, error) {
	spec := viper.GetString("spec")
	if spec == "" {
		return nil, fmt.Errorf("--spec is required")
	}

	if err := setupLogging(); err != nil {
		return nil, fmt.Errorf("failed to setup logging: %w", err)
	}

	tools, err := toolgen.Generate(context.Background(), toolgen.GenerateOptions{
		SpecSource: spec,
		Tag:        viper.GetString("tag"),
		ServerURL:  viper.GetString("server-url"),
	})
	if err != nil {
		return nil, fmt.Errorf("tool generation failed: %w", err)
	}

	auth := executor.NewAuthenticator(
		viper.GetString("auth-type"),
		viper.GetString("auth-token-env"),
		viper.GetString("auth-key-env"),
		viper.GetString("auth-key-name"),
		viper.GetString("auth-key-in"),
	)

	timeout := viper.GetDuration("timeout")
	exec := executor.New(&http.Client{}, auth, timeout)

	return server.New(tools, exec, version)
}

func setupLogging() error {
	var level slog.Level
	switch viper.GetString("log-level") {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var w io.Writer = os.Stderr
	if logFile := viper.GetString("log-file"); logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("failed to open log file %q: %w", logFile, err)
		}
		w = f
	}

	handler := slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
	return nil
}
