package cmd

import (
	"context"
	"fmt"

	"github.com/MakazhanAlpamys/claudeshield/internal/audit"
	"github.com/MakazhanAlpamys/claudeshield/internal/config"
	"github.com/MakazhanAlpamys/claudeshield/internal/rollback"
	"github.com/MakazhanAlpamys/claudeshield/internal/sandbox"
	"github.com/spf13/cobra"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback [checkpoint-id]",
	Short: "Rollback to a checkpoint",
	Long:  "Restores a session to a previous Docker layer checkpoint.",
	RunE: func(cmd *cobra.Command, args []string) error {
		latest, _ := cmd.Flags().GetBool("latest")
		list, _ := cmd.Flags().GetBool("list")
		sessionID, _ := cmd.Flags().GetString("session")

		projectDir := getProjectDir(cmd)

		cfg, err := config.Load(projectDir)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		auditor, err := audit.NewLogger(cfg.Audit.LogDir)
		if err != nil {
			return fmt.Errorf("creating auditor: %w", err)
		}
		defer auditor.Close()

		engine, err := sandbox.New(auditor, nil)
		if err != nil {
			return err
		}
		defer engine.Close()

		mgr := rollback.New(engine.Client())

		ctx := context.Background()

		// Find the target session
		sessions, err := engine.ListSessions(ctx)
		if err != nil {
			return fmt.Errorf("listing sessions: %w", err)
		}

		if len(sessions) == 0 {
			return fmt.Errorf("no active sessions found")
		}

		// Pick session: use --session flag or default to first active one
		var targetSession = sessions[0]
		if sessionID != "" {
			found := false
			for _, s := range sessions {
				if s.ID == sessionID {
					targetSession = s
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("session %q not found", sessionID)
			}
		}

		if list {
			checkpoints := mgr.ListCheckpoints(targetSession.ID)
			if len(checkpoints) == 0 {
				fmt.Printf("No checkpoints for session %s\n", targetSession.ID)
				fmt.Println("Checkpoints are created automatically before risky commands.")
				return nil
			}
			fmt.Printf("Checkpoints for session %s:\n\n", targetSession.ID)
			for _, cp := range checkpoints {
				fmt.Printf("  %s  %s  %s\n", cp.ID, cp.CreatedAt.Format("2006-01-02 15:04:05"), cp.Description)
			}
			return nil
		}

		if latest {
			fmt.Printf("🔄 Rolling back session %s to latest checkpoint...\n", targetSession.ID)
			if err := mgr.RollbackToLatest(ctx, targetSession); err != nil {
				return fmt.Errorf("rollback failed: %w", err)
			}
			fmt.Println("✅ Rollback complete.")
			return nil
		}

		if len(args) == 0 {
			return fmt.Errorf("specify checkpoint ID or use --latest")
		}

		fmt.Printf("🔄 Rolling back session %s to checkpoint %s...\n", targetSession.ID, args[0])
		if err := mgr.Rollback(ctx, targetSession, args[0]); err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}
		fmt.Println("✅ Rollback complete.")
		return nil
	},
}

func init() {
	rollbackCmd.Flags().Bool("latest", false, "Rollback to latest checkpoint")
	rollbackCmd.Flags().Bool("list", false, "List available checkpoints")
	rollbackCmd.Flags().String("session", "", "Target session ID")
}
