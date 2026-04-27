package main

import (
	"fmt"
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs [service]",
	Short: "View DeployPilot service logs",
	Long:  "View logs from DeployPilot services using journalctl.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := apiServiceName
		if len(args) > 0 {
			svc = args[0]
		}

		tail, _ := cmd.Flags().GetInt("tail")
		follow, _ := cmd.Flags().GetBool("follow")
		since, _ := cmd.Flags().GetString("since")
		until, _ := cmd.Flags().GetString("until")

		logArgs := []string{"-u", svc, "--no-pager"}

		if tail > 0 {
			logArgs = append(logArgs, "-n", strconv.Itoa(tail))
		} else {
			logArgs = append(logArgs, "-n", "100") // default last 100 lines
		}

		if follow {
			logArgs = append(logArgs, "-f")
		}
		if since != "" {
			logArgs = append(logArgs, "--since", since)
		}
		if until != "" {
			logArgs = append(logArgs, "--until", until)
		}

		c := exec.Command("journalctl", logArgs...)
		c.Stdout = cmd.OutOrStdout()
		c.Stderr = cmd.ErrOrStderr()

		if err := c.Run(); err != nil {
			// Fallback to log file if journalctl fails
			return fallbackLogFiles(cmd, svc, tail)
		}
		return nil
	},
}

func fallbackLogFiles(cmd *cobra.Command, svc string, tail int) error {
	// Try reading from install directory log files
	logDir := defaultInstallDir + "/logs"
	var logFile string

	switch svc {
	case apiServiceName:
		logFile = logDir + "/deploypilot.log"
	case mcpServiceName:
		logFile = logDir + "/mcp-server.log"
	default:
		logFile = logDir + "/" + svc + ".log"
	}

	c := exec.Command("tail", "-n", strconv.Itoa(tail), logFile)
	c.Stdout = cmd.OutOrStdout()
	c.Stderr = cmd.ErrOrStderr()

	if err := c.Run(); err != nil {
		return fmt.Errorf("no logs found for %s (tried journalctl and %s)", svc, logFile)
	}
	return nil
}

func init() {
	logsCmd.Flags().IntP("tail", "n", 100, "number of lines to show")
	logsCmd.Flags().BoolP("follow", "f", false, "follow log output")
	logsCmd.Flags().String("since", "", "show logs since timestamp (e.g. '2024-01-01' or '1 hour ago')")
	logsCmd.Flags().String("until", "", "show logs until timestamp")
	rootCmd.AddCommand(logsCmd)
}
