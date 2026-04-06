package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the DeployPilot MCP server",
	Long:  "Start the MCP server that exposes deployment tools to AI assistants.",
	Run: func(cmd *cobra.Command, args []string) {
		transport, _ := cmd.Flags().GetString("transport")
		port, _ := cmd.Flags().GetInt("port")

		fmt.Printf("Starting DeployPilot MCP server (v%s)\n", version)
		fmt.Printf("Transport: %s\n", transport)
		if transport == "sse" {
			fmt.Printf("Port: %d\n", port)
		}
		fmt.Println("Server ready")
	},
}

func init() {
	serveCmd.Flags().StringP("transport", "t", "stdio", "transport type: stdio, sse")
	serveCmd.Flags().IntP("port", "p", 8080, "port for SSE transport")
	rootCmd.AddCommand(serveCmd)
}
