package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

const (
	defaultInstallDir = "/opt/deploypilot"
	apiServiceName    = "deploypilot"
	mcpServiceName    = "deploypilot-mcp"
)

var statusCmd = &cobra.Command{
	Use:   "status [service]",
	Short: "Check DeployPilot service status",
	Long:  "Check the status of DeployPilot services (api-server and mcp-server).",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		services := []string{apiServiceName, mcpServiceName}
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
				fmt.Printf("  %s: running ✓\n", svc)
			} else {
				fmt.Printf("  %s: stopped ✗\n", svc)
				allRunning = false
			}
		}

		if !allRunning {
			return nil
		}
		fmt.Println("\nAll services are running.")
		return nil
	},
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
			fmt.Printf("  %s started ✓\n", svc)
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
			fmt.Printf("  %s stopped ✓\n", svc)
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
			fmt.Printf("  %s restarted ✓\n", svc)
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
		fmt.Println("Configuration reloaded ✓")
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
