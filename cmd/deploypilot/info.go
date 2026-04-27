package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var userInfoCmd = &cobra.Command{
	Use:   "user-info",
	Short: "Show current user and system information",
	Long:  "Display DeployPilot user information and system status.",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Root().PersistentFlags().GetString("format")

		// Get version
		versionInfo := map[string]string{
			"version": appversion.Version,
		}

		// Get service status
		apiRunning, _ := isServiceRunning(apiServiceName)
		mcpRunning, _ := isServiceRunning(mcpServiceName)

		// Get install info
		installDir := defaultInstallDir

		// Try to get listening port from config
		port := "8080"
		if out, err := exec.Command("grep", "-oP", `port:\s*\K\d+`, installDir+"/config/config.yaml").Output(); err == nil {
			port = strings.TrimSpace(string(out))
		}

		if format == "json" {
			info := map[string]interface{}{
				"version":     appversion.Version,
				"install_dir": installDir,
				"api_port":    port,
				"services": map[string]bool{
					"api": apiRunning,
					"mcp": mcpRunning,
				},
			}
			data, _ := json.MarshalIndent(info, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Println("DeployPilot System Information")
			fmt.Println("─────────────────────────────")
			fmt.Printf("  Version:     %s\n", versionInfo["version"])
			fmt.Printf("  Install Dir: %s\n", installDir)
			fmt.Printf("  API Port:    %s\n", port)
			fmt.Println()
			fmt.Println("Service Status:")
			fmt.Printf("  API Server:  %s\n", statusText(apiRunning))
			fmt.Printf("  MCP Server:  %s\n", statusText(mcpRunning))
			fmt.Println()
			if apiRunning {
				fmt.Printf("  Dashboard:   http://localhost:%s\n", port)
				fmt.Printf("  API Docs:    http://localhost:%s/swagger/index.html\n", port)
			}
		}
		return nil
	},
}

func statusText(running bool) string {
	if running {
		return "running ✓"
	}
	return "stopped ✗"
}

func init() {
	rootCmd.AddCommand(userInfoCmd)
}
