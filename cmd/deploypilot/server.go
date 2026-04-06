package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage servers",
	Long:  "Register, list, and test server connections.",
}

var serverListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered servers",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Listing servers...")
	},
}

var serverAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Register a new server",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetInt("port")

		if name == "" {
			fmt.Println("Error: --name is required")
			return
		}
		if host == "" {
			fmt.Println("Error: --host is required")
			return
		}

		fmt.Printf("Adding server: %s\n", name)
		fmt.Printf("  Host: %s:%d\n", host, port)
		fmt.Println("Server added successfully")
	},
}

var serverTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test server connectivity",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")

		if name == "" {
			fmt.Println("Error: --name is required")
			return
		}

		fmt.Printf("Testing server: %s\n", name)
		fmt.Println("Server is reachable")
	},
}

func init() {
	serverAddCmd.Flags().StringP("name", "n", "", "server name (required)")
	serverAddCmd.Flags().StringP("host", "H", "", "server hostname/IP (required)")
	serverAddCmd.Flags().IntP("port", "p", 22, "SSH port")
	serverAddCmd.Flags().String("user", "root", "SSH username")

	serverTestCmd.Flags().StringP("name", "n", "", "server name (required)")

	serverCmd.AddCommand(serverListCmd)
	serverCmd.AddCommand(serverAddCmd)
	serverCmd.AddCommand(serverTestCmd)
	rootCmd.AddCommand(serverCmd)
}
