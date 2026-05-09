package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	defaultInstallDir  = "/opt/deploypilot"
	apiServiceName     = "deploypilot"
	mcpServiceName     = "deploypilot-mcp"
	defaultAPIPort     = 8080
	defaultHealthPath  = "/api/v1/system/health"
	healthCheckTimeout = 3 * time.Second
)

var statusCmd = &cobra.Command{
	Use:   "status [service]",
	Short: "Check DeployPilot service status",
	Long:  "Check the status of DeployPilot services (api-server and mcp-server), including health checks.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services := []string{apiServiceName}
		if len(args) > 0 {
			services = []string{args[0]}
		}

		allRunning := true
		for _, svc := range services {
			running, err := isServiceRunning(svc)
			if err != nil {
				fmt.Printf("  %s: error (%v)\n", svc, err)
				allRunning = false
				continue
			}
			if running {
				fmt.Printf("  %s: running\n", svc)
			} else {
				fmt.Printf("  %s: stopped\n", svc)
				allRunning = false
			}
		}

		// Show MCP server status (stdio mode - not a persistent service)
		if len(args) == 0 || args[0] == mcpServiceName {
			fmt.Println()
			fmt.Println("  MCP Server:")
			fmt.Println("    Mode: stdio (on-demand)")
			fmt.Println("    Status: launched by AI IDE when needed")
			mcpPath := "/opt/deploypilot/bin/mcp-server"
			if _, err := exec.LookPath(mcpPath); err == nil {
				fmt.Println("    Binary: installed")
			} else {
				fmt.Println("    Binary: not found")
			}
		}

		// Run health check if api-server is running
		if allRunning || len(services) == 1 && services[0] == apiServiceName {
			running, _ := isServiceRunning(apiServiceName)
			if running {
				fmt.Println()
				checkAPIHealth()
			}
		}

		if allRunning {
			fmt.Println("\nAll services are running.")
		}
		return nil
	},
}

// checkAPIHealth checks the API server health endpoint and port listening.
func checkAPIHealth() {
	fmt.Println("  Health Check:")

	// Check if port is listening
	port := getAPIPort()
	listening := isPortListening(port)
	if listening {
		fmt.Printf("    Port %d: listening\n", port)
	} else {
		fmt.Printf("    Port %d: not listening\n", port)
	}

	// Check health endpoint
	client := &http.Client{Timeout: healthCheckTimeout}
	healthURL := fmt.Sprintf("http://127.0.0.1:%d%s", port, defaultHealthPath)
	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		fmt.Printf("    API: unreachable (%v)\n", err)
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("    API: unreachable (%v)\n", err)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusOK {
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			if status, ok := result["status"].(string); ok {
				fmt.Printf("    API: healthy (status: %s)\n", status)
			} else {
				fmt.Printf("    API: healthy\n")
			}
			if db, ok := result["database"].(map[string]interface{}); ok {
				if dbStatus, ok := db["status"].(string); ok {
					fmt.Printf("    Database: %s\n", dbStatus)
				}
			}
		} else {
			fmt.Printf("    API: healthy (HTTP %d)\n", resp.StatusCode)
		}
	} else {
		fmt.Printf("    API: unhealthy (HTTP %d)\n", resp.StatusCode)
	}
}

// getAPIPort reads the configured port from config.yaml.
func getAPIPort() int {
	// Try to read port from config file
	configPath := defaultInstallDir + "/config/config.yaml"
	if data, err := exec.Command("grep", "-oP", `port:\s*\K\d+`, configPath).Output(); err == nil && len(data) > 0 {
		var port int
		if _, err := fmt.Sscanf(string(data), "%d", &port); err == nil && port > 0 {
			return port
		}
	}
	return defaultAPIPort
}

// isPortListening checks if a TCP port is accepting connections.
func isPortListening(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 1*time.Second)
	if err != nil {
		return false
	}
	if cerr := conn.Close(); cerr != nil {
		return false
	}
	return true
}

var startCmd = &cobra.Command{
	Use:   "start [service]",
	Short: "Start DeployPilot services",
	Long:  "Start DeployPilot services. Use 'all' to start both api-server and mcp-server.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services := []string{mcpServiceName, apiServiceName}
		if len(args) > 0 && args[0] != "all" {
			services = []string{args[0]}
		}

		for _, svc := range services {
			fmt.Printf("Starting %s...\n", svc)
			if err := systemctl("start", svc); err != nil {
				return fmt.Errorf("failed to start %s: %w", svc, err)
			}
			fmt.Printf("  %s started\n", svc)
		}
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop [service]",
	Short: "Stop DeployPilot services",
	Long:  "Stop DeployPilot services. Use 'all' to stop both api-server and mcp-server.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services := []string{apiServiceName, mcpServiceName}
		if len(args) > 0 && args[0] != "all" {
			services = []string{args[0]}
		}

		for _, svc := range services {
			fmt.Printf("Stopping %s...\n", svc)
			if err := systemctl("stop", svc); err != nil {
				return fmt.Errorf("failed to stop %s: %w", svc, err)
			}
			fmt.Printf("  %s stopped\n", svc)
		}
		return nil
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart [service]",
	Short: "Restart DeployPilot services",
	Long:  "Restart DeployPilot services. Use 'all' to restart both api-server and mcp-server.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services := []string{mcpServiceName, apiServiceName}
		if len(args) > 0 && args[0] != "all" {
			services = []string{args[0]}
		}

		for _, svc := range services {
			fmt.Printf("Restarting %s...\n", svc)
			if err := systemctl("restart", svc); err != nil {
				return fmt.Errorf("failed to restart %s: %w", svc, err)
			}
			fmt.Printf("  %s restarted\n", svc)
		}
		return nil
	},
}

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload DeployPilot configuration",
	Long:  "Reload DeployPilot configuration without restarting (SIGHUP).",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Reloading configuration...")
		if err := systemctl("reload", apiServiceName); err != nil {
			return fmt.Errorf("failed to reload: %w", err)
		}
		fmt.Println("Configuration reloaded")
		return nil
	},
}

func isServiceRunning(name string) (bool, error) {
	cmd := exec.Command("systemctl", "is-active", "--quiet", name)
	err := cmd.Run()
	if err != nil {
		// Check if service exists
		checkCmd := exec.Command("systemctl", "status", name)
		if checkErr := checkErrString(checkCmd); checkErr != "" && strings.Contains(checkErr, "could not be found") {
			return false, fmt.Errorf("service %s not found", name)
		}
		return false, nil
	}
	return true, nil
}

func systemctl(action, service string) error {
	cmd := exec.Command("systemctl", action, service)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(output)))
	}
	return nil
}

func checkErrString(cmd *exec.Cmd) string {
	output, _ := cmd.CombinedOutput()
	return string(output)
}

func init() {
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(reloadCmd)
}
