package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("deploypilot version %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
