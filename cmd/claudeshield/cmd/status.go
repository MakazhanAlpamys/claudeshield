package cmd

import (
	"fmt"

	"github.com/MakazhanAlpamys/claudeshield/internal/audit"
	"github.com/MakazhanAlpamys/claudeshield/internal/config"
	"github.com/MakazhanAlpamys/claudeshield/internal/sandbox"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show active sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
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
			fmt.Println("No active ClaudeShield sessions")
			return nil
		}

		fmt.Printf("%-25s %-15s %-12s %-15s %s\n", "SESSION", "AGENT", "STATE", "CONTAINER", "PROJECT")
		fmt.Println("─────────────────────────────────────────────────────────────────────────────────────")

		for _, s := range sessions {
			containerID := s.ContainerID
			if len(containerID) > 12 {
				containerID = containerID[:12]
			}

			fmt.Printf("%-25s %-15s %-12s %-15s %s\n",
				s.ID,
				s.AgentName,
				s.State,
				containerID,
				s.ProjectDir,
			)
		}

		return nil
	},
}
