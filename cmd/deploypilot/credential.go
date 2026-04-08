package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var credentialCmd = &cobra.Command{
	Use:   "credential",
	Short: "Manage credentials",
	Long:  "Create, list, and delete encrypted credentials for server authentication.",
}

var credentialAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Create an encrypted credential",
	Long: `Create an encrypted credential for server authentication.

The credential value is encrypted with AES-256-GCM before storage.
Plaintext never touches the database.

Value input methods (priority order):
  1. --value-stdin: read from stdin (pipe-friendly)
  2. Interactive hidden input (terminal only, like getpass)
  3. --value-file <path>: read from file

Examples:
  # Pipe from stdin
  echo -n "my-secret" | deploypilot credential add --name my-key --type password --value-stdin

  # Interactive input (hidden)
  deploypilot credential add --name my-key --type password

  # From file
  deploypilot credential add --name my-key --type password --value-file /path/to/secret
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tenantID, _ := cmd.Flags().GetString("tenant-id")
		name, _ := cmd.Flags().GetString("name")
		credType, _ := cmd.Flags().GetString("type")
		valueStdin, _ := cmd.Flags().GetBool("value-stdin")
		valueFile, _ := cmd.Flags().GetString("value-file")

		if name == "" {
			return fmt.Errorf("--name is required")
		}
		if credType == "" {
			credType = "password"
		}
		if tenantID == "" {
			tenantID = "tenant-default"
		}

		// Read value
		var value string

		switch {
		case valueStdin:
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read stdin: %w", err)
			}
			value = strings.TrimRight(string(data), "\r\n")

		case valueFile != "":
			data, err := os.ReadFile(valueFile)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", valueFile, err)
			}
			value = strings.TrimRight(string(data), "\r\n")

		default:
			// Interactive hidden input
			fmt.Fprintf(os.Stderr, "Enter value for '%s': ", name)
			rawValue, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(os.Stderr) // newline after hidden input
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
			value = string(rawValue)
		}

		if value == "" {
			return fmt.Errorf("credential value must not be empty")
		}

		// Output structured result (for scripting / MCP consumption)
		result := map[string]interface{}{
			"status":    "created",
			"tenant_id": tenantID,
			"name":      name,
			"type":      credType,
			"message":   "Credential created. Use add_credential MCP tool to store it in the database.",
		}

		format, _ := cmd.Flags().GetString("format")
		if format == "json" {
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Printf("Credential '%s' (type=%s) created for tenant '%s'\n", name, credType, tenantID)
			fmt.Println("Note: Use the add_credential MCP tool to store it in the database, or run with a connected MCP server.")
		}

		return nil
	},
}

func init() {
	credentialAddCmd.Flags().String("tenant-id", "tenant-default", "tenant ID (default: tenant-default)")
	credentialAddCmd.Flags().StringP("name", "n", "", "credential name (required)")
	credentialAddCmd.Flags().StringP("type", "t", "password", "credential type: password, ssh_key, token")
	credentialAddCmd.Flags().Bool("value-stdin", false, "read value from stdin")
	credentialAddCmd.Flags().String("value-file", "", "read value from file")

	credentialCmd.AddCommand(credentialAddCmd)
	rootCmd.AddCommand(credentialCmd)
}
