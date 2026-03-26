package main

import (
	"os"
	"os/signal"
	"syscall"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

var stdioCmd = &cobra.Command{
	Use:   "stdio",
	Short: "Run MCP server over stdio",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := buildServer()
		if err != nil {
			return err
		}
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		stdioServer := mcpserver.NewStdioServer(s)
		return stdioServer.Listen(ctx, os.Stdin, os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(stdioCmd)
}
