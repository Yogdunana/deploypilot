package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade [version]",
	Short: "Upgrade DeployPilot to the latest or specified version",
	Long:  "Upgrade DeployPilot by downloading and installing the latest version from GitHub Releases.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetVersion := "latest"
		if len(args) > 0 {
			targetVersion = args[0]
		}

		force, _ := cmd.Flags().GetBool("force")

		// Check if currently installed
		installDir := defaultInstallDir
		if _, err := exec.LookPath("deploypilot"); err == nil {
			// Binary is in PATH, try to find install dir from binary location
			if out, err := exec.Command("which", "deploypilot").Output(); err == nil {
				binPath := strings.TrimSpace(string(out))
				// Walk up to find install dir
				parentDir := binPath
				for i := 0; i < 3; i++ {
					parentDir = parentDir[:strings.LastIndex(parentDir, "/")]
					if parentDir == "" {
						break
					}
				}
				if parentDir != "" {
					installDir = parentDir
				}
			}
		}

		// Check if install.sh exists
		installScript := installDir + "/../scripts/install.sh"
		if err := exec.Command("test", "-f", installScript).Run(); err != nil {
			// Try downloading install.sh from GitHub
			fmt.Println("Local install.sh not found, downloading from GitHub...")
			downloadCmd := exec.Command("curl", "-fsSL",
				"https://raw.githubusercontent.com/Yogdunana/deploypilot/main/scripts/install.sh",
				"-o", "/tmp/deploypilot-install.sh")
			if output, err := downloadCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to download install.sh: %s", strings.TrimSpace(string(output)))
			}
			installScript = "/tmp/deploypilot-install.sh"
		}

		fmt.Printf("Upgrading DeployPilot to %s...\n", targetVersion)
		fmt.Println("This will:")
		fmt.Println("  1. Back up current binaries")
		fmt.Println("  2. Download new version")
		fmt.Println("  3. Replace binaries")
		fmt.Println("  4. Restart services")

		if !force {
			fmt.Print("\nContinue? [y/N]: ")
		var confirm string
		_, _ = fmt.Scanln(&confirm) //nolint:errcheck // user input, ignore error
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Upgrade cancelled.")
				return nil
			}
		}

		// Execute install.sh --upgrade
		upgradeArgs := []string{"bash", installScript, "--upgrade"}
		if targetVersion != "latest" {
			upgradeArgs = append(upgradeArgs, targetVersion)
		}

		c := exec.Command(upgradeArgs[0], upgradeArgs[1:]...)
		c.Stdout = cmd.OutOrStdout()
		c.Stderr = cmd.ErrOrStderr()
		c.Stdin = cmd.InOrStdin()

		if err := c.Run(); err != nil {
			return fmt.Errorf("upgrade failed: %w", err)
		}

		return nil
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall DeployPilot",
	Long:  "Stop services and remove DeployPilot from the system.",
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		purge, _ := cmd.Flags().GetBool("purge")

		if !force {
			fmt.Println("⚠️  This will remove DeployPilot from your system.")
			if purge {
				fmt.Println("⚠️  ALL DATA (database, configs, logs, backups) will be permanently deleted!")
			}
			fmt.Print("\nAre you sure? [y/N]: ")
			var confirm string
			_, _ = fmt.Scanln(&confirm) //nolint:errcheck // user input, ignore error
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Uninstall cancelled.")
				return nil
			}
		}

		// Stop services
		fmt.Println("Stopping services...")
		_ = systemctl("stop", mcpServiceName)
		_ = systemctl("stop", apiServiceName)

		// Disable services
		_ = systemctl("disable", mcpServiceName)
		_ = systemctl("disable", apiServiceName)

		// Remove service files
		fmt.Println("Removing service files...")
		for _, svc := range []string{apiServiceName + ".service", mcpServiceName + ".service"} {
			_ = exec.Command("rm", "-f", "/etc/systemd/system/"+svc).Run()
		}
		_ = exec.Command("systemctl", "daemon-reload").Run()

		if purge {
			fmt.Println("Removing all data...")
			_ = exec.Command("rm", "-rf", defaultInstallDir).Run()
		}

		fmt.Println("DeployPilot has been uninstalled.")
		return nil
	},
}

func init() {
	upgradeCmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")
	uninstallCmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")
	uninstallCmd.Flags().Bool("purge", false, "remove all data (database, configs, logs, backups)")

	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(uninstallCmd)
}
