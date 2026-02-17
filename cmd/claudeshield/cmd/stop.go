package cmd

import (
	"fmt"

	"github.com/MakazhanAlpamys/claudeshield/internal/audit"
	"github.com/MakazhanAlpamys/claudeshield/internal/config"
	"github.com/MakazhanAlpamys/claudeshield/internal/sandbox"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop [session-id]",
	Short: "Stop a sandboxed session",
	Long:  "Stops and removes the Docker container for a ClaudeShield session.",
	RunE: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")

		cfg, err := config.Load(getProjectDir(cmd))
		if err != nil {
			return err
		}

		auditor, err := audit.NewLogger(cfg.Audit.LogDir)
		if err != nil {
			return err
		}
		defer auditor.Close()

		engine, err := sandbox.New(auditor, nil)
		if err != nil {
			return err
		}
		defer engine.Close()

		sessions, err := engine.ListSessions(cmd.Context())
		if err != nil {
			return err
		}

		if len(sessions) == 0 {
			fmt.Println("No active sessions")
			return nil
		}

		for _, s := range sessions {
			if !all && len(args) > 0 && s.ID != args[0] {
				continue
			}

			if err := engine.StopSession(cmd.Context(), s); err != nil {
				fmt.Printf("⚠️  Error stopping %s: %v\n", s.ID, err)
				continue
			}
			fmt.Printf("🛑 Stopped: %s (%s)\n", s.ID, s.AgentName)
		}

		return nil
	},
}

func init() {
	stopCmd.Flags().Bool("all", false, "Stop all sessions")
}
