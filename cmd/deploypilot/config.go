package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and manage DeployPilot configuration",
	Long:  "View, get, or set DeployPilot configuration values.",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		if configPath == "" {
			configPath = defaultInstallDir + "/config/config.yaml"
		}

		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read config: %w", err)
		}

		fmt.Println(string(data))
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		if configPath == "" {
			configPath = defaultInstallDir + "/config/config.yaml"
		}

		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read config: %w", err)
		}

		var cfg map[string]interface{}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}

		key := args[0]
		parts := strings.Split(key, ".")
		val, found := getNestedValue(cfg, parts)
		if !found {
			return fmt.Errorf("key '%s' not found in config", key)
		}

		fmt.Printf("%s: %v\n", key, val)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		if configPath == "" {
			configPath = defaultInstallDir + "/config/config.yaml"
		}

		key := args[0]
		value := args[1]

		// Use sed to set the value in YAML
		parts := strings.Split(key, ".")
		if len(parts) != 2 {
			return fmt.Errorf("key must be in format 'section.field' (e.g. 'server.port')")
		}

		section, field := parts[0], parts[1]

		// Check if config file exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return fmt.Errorf("config file not found: %s", configPath)
		}

		// Escape special sed replacement characters from user input
		safeValue := strings.NewReplacer(
			`\`, `\\`,
			`/`, `\/`,
			`&`, `\&`,
			`\n`, `\\n`,
		).Replace(value)
		pattern := fmt.Sprintf("s/^  %s:\\s*.*/  %s: %s/", field, field, safeValue)
		cmdExec := exec.Command("sed", "-i", pattern, configPath)
		if output, err := cmdExec.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to set config: %s (%s)", err, strings.TrimSpace(string(output)))
		}

		fmt.Printf("Set %s.%s = %s\n", section, key, value)
		fmt.Println("Please restart the service for changes to take effect.")
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		if configPath == "" {
			configPath = defaultInstallDir + "/config/config.yaml"
		}
		fmt.Println(configPath)
		return nil
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open configuration in default editor",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		if configPath == "" {
			configPath = defaultInstallDir + "/config/config.yaml"
		}

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}

		editCmd := exec.Command(editor, configPath)
		editCmd.Stdin = os.Stdin
		editCmd.Stdout = os.Stdout
		editCmd.Stderr = os.Stderr

		if err := editCmd.Run(); err != nil {
			return fmt.Errorf("failed to open editor: %w", err)
		}
		return nil
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a default configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputPath, _ := cmd.Flags().GetString("output")
		if outputPath == "" {
			outputPath = defaultInstallDir + "/config/config.yaml"
		}

		dir := filepath.Dir(outputPath)
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		defaultConfig := `server:
  host: 0.0.0.0
  port: 8080
  mcp_port: 9090
  web_port: 3000

database:
  type: sqlite
  dsn: /opt/deploypilot/data/deploypilot.db

auth:
  jwt_secret: ""
  token_expire: 24h
  ws_ticket_expire: 30s

deploy:
  default_mode: api
  build_timeout: 10m
  health_check_interval: 30s
  health_check_retries: 3
  rollback_on_failure: true

log:
  level: info
  format: json
  file: /opt/deploypilot/logs/deploypilot.log
  max_size: 100MB
  max_backups: 10

monitor:
  enabled: true
  metrics_port: 9091

redis:
  addr: localhost:6379
  password: ""
  db: 0
`

		if err := os.WriteFile(outputPath, []byte(defaultConfig), 0600); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		fmt.Printf("Default configuration written to %s\n", outputPath)
		fmt.Println("Please edit the file and set your jwt_secret before starting the service.")
		return nil
	},
}

func getNestedValue(m map[string]interface{}, keys []string) (interface{}, bool) {
	if len(keys) == 0 {
		return nil, false
	}
	val, ok := m[keys[0]]
	if !ok {
		return nil, false
	}
	if len(keys) == 1 {
		return val, true
	}
	if sub, ok := val.(map[string]interface{}); ok {
		return getNestedValue(sub, keys[1:])
	}
	return nil, false
}

func init() {
	configShowCmd.Flags().StringP("config", "c", "", "config file path")
	configGetCmd.Flags().StringP("config", "c", "", "config file path")
	configSetCmd.Flags().StringP("config", "c", "", "config file path")
	configPathCmd.Flags().StringP("config", "c", "", "config file path")
	configEditCmd.Flags().StringP("config", "c", "", "config file path")
	configInitCmd.Flags().StringP("output", "o", "", "output file path")

	configCmd.AddCommand(configShowCmd, configGetCmd, configSetCmd, configPathCmd, configEditCmd, configInitCmd)
	rootCmd.AddCommand(configCmd)
}
