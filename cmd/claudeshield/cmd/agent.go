package cmd

import (
	"fmt"

	"github.com/MakazhanAlpamys/claudeshield/internal/audit"
	"github.com/MakazhanAlpamys/claudeshield/internal/config"
	"github.com/MakazhanAlpamys/claudeshield/internal/orchestrator"
	"github.com/MakazhanAlpamys/claudeshield/internal/policy"
	"github.com/MakazhanAlpamys/claudeshield/internal/sandbox"
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage parallel agents",
}

var agentSpawnCmd = &cobra.Command{
	Use:   "spawn <name>",
	Short: "Spawn a new parallel agent",
	Long:  "Creates a new agent with its own git worktree and Docker sandbox.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]
		projectDir := getProjectDir(cmd)

		cfg, err := config.Load(projectDir)
		if err != nil {
			return err
		}

		auditor, err := audit.NewLogger(cfg.Audit.LogDir)
		if err != nil {
			return err
		}
		defer auditor.Close()

		policyEngine := policy.New(cfg)

		engine, err := sandbox.New(auditor, policyEngine)
		if err != nil {
			return err
		}
		defer engine.Close()

		orch := orchestrator.New(engine, auditor)

		fmt.Printf("🚀 Spawning agent %q...\n", agentName)
		session, err := orch.SpawnAgent(cmd.Context(), projectDir, agentName, cfg.Sandbox)
		if err != nil {
			return err
		}

		fmt.Printf("✅ Agent %q started\n", agentName)
		fmt.Printf("   Session:  %s\n", session.ID)
		fmt.Printf("   Worktree: %s\n", session.WorktreeDir)
		fmt.Printf("   Container: %s\n", session.ContainerID[:12])

		return nil
	},
}

var agentStopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop a parallel agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]
		merge, _ := cmd.Flags().GetBool("merge")
		projectDir := getProjectDir(cmd)

		cfg, err := config.Load(projectDir)
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

		orch := orchestrator.New(engine, auditor)

		if merge {
			fmt.Printf("🔀 Stopping and merging agent %q...\n", agentName)
		} else {
			fmt.Printf("🛑 Stopping agent %q...\n", agentName)
		}

		if err := orch.StopAgent(cmd.Context(), agentName, merge); err != nil {
			return err
		}

		fmt.Printf("✅ Agent %q stopped\n", agentName)
		return nil
	},
}

var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active agents",
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
			fmt.Println("No active agents")
			return nil
		}

		fmt.Printf("%-15s %-25s %-12s %s\n", "AGENT", "SESSION", "STATE", "WORKTREE")
		fmt.Println("─────────────────────────────────────────────────────────────────────")

		for _, s := range sessions {
			fmt.Printf("%-15s %-25s %-12s %s\n",
				s.AgentName, s.ID, s.State, s.WorktreeDir)
		}

		return nil
	},
}

func init() {
	agentStopCmd.Flags().Bool("merge", true, "Merge worktree changes before stopping")

	agentCmd.AddCommand(agentSpawnCmd)
	agentCmd.AddCommand(agentStopCmd)
	agentCmd.AddCommand(agentListCmd)
}
