package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	appversion "github.com/Yogdunana/deploypilot/internal/version"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup and restore DeployPilot data",
	Long:  "Create backups of DeployPilot data (database, configs, binaries) or restore from a backup.",
}

var backupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a backup of DeployPilot data",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		includeDB, _ := cmd.Flags().GetBool("include-db")

		if name == "" {
			name = "backup-" + time.Now().Format("20060102-150405")
		}

		backupDir := filepath.Join(defaultInstallDir, "backups", name)
		if err := os.MkdirAll(backupDir, 0750); err != nil {
			return fmt.Errorf("failed to create backup directory: %w", err)
		}

		fmt.Printf("Creating backup: %s\n", name)

		// Backup config
		configSrc := defaultInstallDir + "/config"
		configDst := backupDir + "/config"
		if err := copyDir(configSrc, configDst); err != nil {
			fmt.Printf("  Warning: failed to backup config: %v\n", err)
		} else {
			fmt.Println("  ✓ Config backed up")
		}

		// Backup database
		if includeDB {
			dbSrc := defaultInstallDir + "/data"
			dbDst := backupDir + "/data"
			if err := copyDir(dbSrc, dbDst); err != nil {
				fmt.Printf("  Warning: failed to backup database: %v\n", err)
			} else {
				fmt.Println("  ✓ Database backed up")
			}
		}

		// Backup binaries
		binSrc := defaultInstallDir + "/bin"
		binDst := backupDir + "/bin"
		if err := copyDir(binSrc, binDst); err != nil {
			fmt.Printf("  Warning: failed to backup binaries: %v\n", err)
		} else {
			fmt.Println("  ✓ Binaries backed up")
		}

		// Write metadata
		meta := map[string]interface{}{
			"name":       name,
			"created_at": time.Now().Format(time.RFC3339),
			"version":    appversion.Version,
			"include_db": includeDB,
		}
		metaData, _ := json.MarshalIndent(meta, "", "  ")
		_ = os.WriteFile(backupDir+"/metadata.json", metaData, 0640)

		fmt.Printf("\nBackup created: %s\n", backupDir)
		return nil
	},
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore [backup-name]",
	Short: "Restore from a backup",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		backupName, _ := cmd.Flags().GetString("name")
		force, _ := cmd.Flags().GetBool("force")

		if len(args) > 0 {
			backupName = args[0]
		}
		if backupName == "" {
			// List available backups
			return listBackups()
		}

		backupDir := filepath.Join(defaultInstallDir, "backups", backupName)
		if _, err := os.Stat(backupDir); os.IsNotExist(err) {
			return fmt.Errorf("backup not found: %s", backupDir)
		}

		if !force {
			fmt.Printf("⚠️  Restoring from '%s' will overwrite current data.\n", backupName)
			fmt.Print("Continue? [y/N]: ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Restore cancelled.")
				return nil
			}
		}

		fmt.Println("Restoring...")

		// Stop services
		fmt.Println("  Stopping services...")
		_ = systemctl("stop", mcpServiceName)
		_ = systemctl("stop", apiServiceName)

		// Restore config
		if err := copyDir(backupDir+"/config", defaultInstallDir+"/config"); err == nil {
			fmt.Println("  ✓ Config restored")
		}

		// Restore database
		if _, err := os.Stat(backupDir + "/data"); err == nil {
			if err := copyDir(backupDir+"/data", defaultInstallDir+"/data"); err == nil {
				fmt.Println("  ✓ Database restored")
			}
		}

		// Restore binaries
		if _, err := os.Stat(backupDir + "/bin"); err == nil {
			if err := copyDir(backupDir+"/bin", defaultInstallDir+"/bin"); err == nil {
				fmt.Println("  ✓ Binaries restored")
			}
		}

		// Restart services
		fmt.Println("  Starting services...")
		_ = systemctl("start", apiServiceName)
		_ = systemctl("start", mcpServiceName)

		fmt.Printf("\nRestore from '%s' completed ✓\n", backupName)
		return nil
	},
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available backups",
	RunE: func(cmd *cobra.Command, args []string) error {
		return listBackups()
	},
}

func listBackups() error {
	backupDir := filepath.Join(defaultInstallDir, "backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No backups found.")
			return nil
		}
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No backups found.")
		return nil
	}

	fmt.Printf("Found %d backup(s):\n", len(entries))
	fmt.Println(strings.Repeat("─", 60))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, _ := entry.Info()
		fmt.Printf("  %-40s %s\n", entry.Name(), info.ModTime().Format("2006-01-02 15:04:05"))
	}
	return nil
}

func copyDir(src, dst string) error {
	return exec.Command("cp", "-r", src, dst).Run()
}

func init() {
	backupCreateCmd.Flags().StringP("name", "n", "", "backup name (default: auto-generated timestamp)")
	backupCreateCmd.Flags().Bool("include-db", true, "include database in backup")
	backupRestoreCmd.Flags().StringP("name", "n", "", "backup name to restore")
	backupRestoreCmd.Flags().BoolP("force", "f", false, "skip confirmation")

	backupCmd.AddCommand(backupCreateCmd, backupRestoreCmd, backupListCmd)
	rootCmd.AddCommand(backupCmd)
}
