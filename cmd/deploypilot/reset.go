package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset DeployPilot system settings",
	Long:  "Reset various DeployPilot settings (password, port, security entrance, etc.).",
}

var resetPasswordCmd = &cobra.Command{
	Use:   "password",
	Short: "Reset admin password",
	RunE: func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString("username")
		if username == "" {
			username = "admin"
		}

		fmt.Printf("Enter new password for user '%s': ", username)
		password := readPassword()
		fmt.Println()
		fmt.Print("Confirm password: ")
		confirm := readPassword()
		fmt.Println()

		if password != confirm {
			return fmt.Errorf("passwords do not match")
		}
		if len(password) < 6 {
			return fmt.Errorf("password must be at least 6 characters")
		}

		// Generate a random JWT secret
		secret := generateRandomSecret(32)

		// Use the API to update password
		fmt.Printf("Updating password for user '%s'...\n", username)

		// Try calling the API
		apiURL := "http://localhost:8080/api/v1/auth/reset-password"
		curlCmd := exec.Command("curl", "-s", "-X", "POST", apiURL,
			"-H", "Content-Type: application/json",
			"-d", fmt.Sprintf(`{"username":"%s","new_password":"%s"}`, username, password))

		output, err := curlCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to update password via API: %w", err)
		}

		result := strings.TrimSpace(string(output))
		if strings.Contains(result, "success") || strings.Contains(result, "200") {
			fmt.Println("Password updated successfully ✓")
		} else {
			// API might not be running, show manual instructions
			fmt.Println("Could not update via API. You can update manually:")
			fmt.Printf("  deploypilot config set auth.jwt_secret %s\n", secret)
			fmt.Println("  Then restart the service and use the web UI to change the password.")
		}

		return nil
	},
}

var resetPortCmd = &cobra.Command{
	Use:   "port",
	Short: "Reset panel port to default (8080)",
	RunE: func(cmd *cobra.Command, args []string) error {
		newPort, _ := cmd.Flags().GetInt("port")
		if newPort == 0 {
			newPort = 8080
		}

		fmt.Printf("Setting panel port to %d...\n", newPort)

		configPath := defaultInstallDir + "/config/config.yaml"
		sedCmd := exec.Command("sed", "-i",
			fmt.Sprintf("s/^  port:\\s*.*/  port: %d/", newPort),
			configPath)
		if output, err := sedCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to update port: %s", strings.TrimSpace(string(output)))
		}

		// Update systemd service if port is in the service file
		fmt.Println("Port updated. Restart the service to apply changes:")
		fmt.Println("  deploypilot restart")
		return nil
	},
}

var resetSecretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Generate and set a new JWT secret",
	RunE: func(cmd *cobra.Command, args []string) error {
		secret := generateRandomSecret(32)
		fmt.Printf("Generated new JWT secret: %s\n", secret)

		configPath := defaultInstallDir + "/config/config.yaml"
		sedCmd := exec.Command("sed", "-i",
			fmt.Sprintf("s/^  jwt_secret:\\s*.*/  jwt_secret: %s/", secret),
			configPath)
		if output, err := sedCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to update secret: %s", strings.TrimSpace(string(output)))
		}

		fmt.Println("JWT secret updated. All existing sessions will be invalidated.")
		fmt.Println("Restart the service to apply changes:")
		fmt.Println("  deploypilot restart")
		return nil
	},
}

func readPassword() string {
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func generateRandomSecret(length int) string {
	bytes := make([]byte, length)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func init() {
	resetPasswordCmd.Flags().StringP("username", "u", "", "username to reset (default: admin)")
	resetPortCmd.Flags().IntP("port", "p", 0, "new port number (default: 8080)")

	resetCmd.AddCommand(resetPasswordCmd, resetPortCmd, resetSecretCmd)
	rootCmd.AddCommand(resetCmd)
}
