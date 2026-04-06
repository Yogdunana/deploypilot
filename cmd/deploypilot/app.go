package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Manage applications",
	Long:  "Create, list, deploy, and delete applications.",
}

var appListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all applications",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Listing applications...")
		// In production: query database and display
	},
}

var appCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new application",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		repo, _ := cmd.Flags().GetString("repo")
		stack, _ := cmd.Flags().GetString("stack")

		if name == "" {
			fmt.Println("Error: --name is required")
			return
		}
		if repo == "" {
			fmt.Println("Error: --repo is required")
			return
		}

		fmt.Printf("Creating app: %s\n", name)
		fmt.Printf("  Repo: %s\n", repo)
		fmt.Printf("  Stack: %s\n", stack)
		fmt.Println("App created successfully")
	},
}

var appDeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy an application",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		image, _ := cmd.Flags().GetString("image")
		server, _ := cmd.Flags().GetString("server")

		if name == "" {
			fmt.Println("Error: --name is required")
			return
		}

		fmt.Printf("Deploying app: %s\n", name)
		fmt.Printf("  Image: %s\n", image)
		fmt.Printf("  Server: %s\n", server)
		fmt.Println("Deployment started")
	},
}

var appDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an application",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		force, _ := cmd.Flags().GetBool("force")

		if name == "" {
			fmt.Println("Error: --name is required")
			return
		}

		fmt.Printf("Deleting app: %s (force=%v)\n", name, force)
		fmt.Println("App deleted successfully")
	},
}

func init() {
	appCreateCmd.Flags().StringP("name", "n", "", "application name (required)")
	appCreateCmd.Flags().StringP("repo", "r", "", "git repository URL (required)")
	appCreateCmd.Flags().StringP("stack", "s", "docker", "tech stack: docker, node, python, go, java")
	appCreateCmd.Flags().String("branch", "main", "git branch")

	appDeployCmd.Flags().StringP("name", "n", "", "application name (required)")
	appDeployCmd.Flags().StringP("image", "i", "", "docker image")
	appDeployCmd.Flags().StringP("server", "s", "", "target server")

	appDeleteCmd.Flags().StringP("name", "n", "", "application name (required)")
	appDeleteCmd.Flags().BoolP("force", "f", false, "force delete without confirmation")

	appCmd.AddCommand(appListCmd)
	appCmd.AddCommand(appCreateCmd)
	appCmd.AddCommand(appDeployCmd)
	appCmd.AddCommand(appDeleteCmd)
	rootCmd.AddCommand(appCmd)
}
