package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "claudeshield",
	Short: "ClaudeShield — secure sandbox for Claude Code agents",
	Long: `ClaudeShield wraps Claude Code in a secure Docker sandbox
with policy enforcement, secret protection, audit logging,
and one-click rollback.

Run Claude Code at full speed — without risking your machine or secrets.`,
	SilenceUsage: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringP("project", "p", ".", "Project directory")
	rootCmd.PersistentFlags().StringP("config", "c", "", "Config file (default: .claudeshield.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")

	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(initConfigCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(tuiCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print ClaudeShield version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ClaudeShield v%s\n", version)
	},
}

func getProjectDir(cmd *cobra.Command) string {
	dir, _ := cmd.Flags().GetString("project")
	if dir == "." {
		dir, _ = os.Getwd()
	}
	return dir
}
