package main

import (
	"log/slog"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var httpCmd = &cobra.Command{
	Use:   "http",
	Short: "Run MCP server over Streamable HTTP",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := buildServer()
		if err != nil {
			return err
		}
		addr := viper.GetString("addr")
		slog.Info("starting Streamable HTTP server", "addr", addr)
		httpServer := mcpserver.NewStreamableHTTPServer(s)
		return httpServer.Start(addr)
	},
}

func init() {
	httpCmd.Flags().String("addr", ":8080", "Listen address")
	_ = viper.BindPFlag("addr", httpCmd.Flags().Lookup("addr"))
	rootCmd.AddCommand(httpCmd)
}
