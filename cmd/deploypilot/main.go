package main

import (
	"fmt"
	"os"

	appversion "github.com/Yogdunana/deploypilot/internal/version"
	"github.com/spf13/cobra"
)

// version is set via -ldflags at build time.
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "deploypilot",
	Short: "DeployPilot - AI-powered deployment automation",
	Long:  "DeployPilot is an MCP-based deployment tool that automates container deployment, health checking, and rollback.",
}

func init() {
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file path")
	rootCmd.PersistentFlags().String("format", "text", "output format: text, json")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
