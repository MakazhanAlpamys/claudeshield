package cmd

import (
	"fmt"

	"github.com/MakazhanAlpamys/claudeshield/internal/audit"
	"github.com/MakazhanAlpamys/claudeshield/internal/config"
	"github.com/MakazhanAlpamys/claudeshield/internal/policy"
	"github.com/MakazhanAlpamys/claudeshield/internal/sandbox"
	"github.com/MakazhanAlpamys/claudeshield/internal/secrets"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a sandboxed Claude Code session",
	Long:  "Creates a Docker container with security policies and launches Claude Code inside it.",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir := getProjectDir(cmd)
		agentName, _ := cmd.Flags().GetString("agent")

		cfg, err := config.Load(projectDir)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		auditor, err := audit.NewLogger(cfg.Audit.LogDir)
		if err != nil {
			return fmt.Errorf("creating auditor: %w", err)
		}
		defer auditor.Close()

		policyEngine := policy.New(cfg)

		engine, err := sandbox.New(auditor, policyEngine)
		if err != nil {
			return err
		}
		defer engine.Close()

		// Load secrets from configured provider
		var loadedSecrets map[string]string
		registry := secrets.NewRegistry()
		provider, err := registry.Get(cfg.Secrets.Provider)
		if err == nil && provider.Available() {
			// Load secret keys from config options if specified
			var secretKeys []string
			for _, v := range cfg.Secrets.Options {
				secretKeys = append(secretKeys, v)
			}
			if len(secretKeys) > 0 {
				loadedSecrets, err = provider.Load(secretKeys)
				if err != nil {
					fmt.Printf("   Warning: failed to load some secrets: %v\n", err)
				}
			}
		}

		fmt.Println("🛡️  ClaudeShield starting...")
		fmt.Printf("   Project:  %s\n", projectDir)
		fmt.Printf("   Agent:    %s\n", agentName)
		fmt.Printf("   Network:  %v\n", cfg.Sandbox.Network)
		fmt.Printf("   GVisor:   %v\n", cfg.Sandbox.UseGVisor)
		fmt.Printf("   Secrets:  %s\n", cfg.Secrets.Provider)
		fmt.Printf("   Policy:   %d allow rules, %d block rules\n", len(cfg.Rules.Allow), len(cfg.Rules.Block))

		session, err := engine.CreateSession(cmd.Context(), projectDir, cfg.Sandbox, agentName, loadedSecrets)
		if err != nil {
			return fmt.Errorf("creating session: %w", err)
		}

		fmt.Printf("\n✅ Session started: %s\n", session.ID)
		fmt.Printf("   Container: %s\n", session.ContainerID[:12])
		fmt.Println("\n   Use 'claudeshield stop' to end the session")
		fmt.Println("   Use 'claudeshield status' to see running sessions")
		fmt.Println("   Use 'claudeshield audit' to view the audit log")

		return nil
	},
}

func init() {
	startCmd.Flags().StringP("agent", "a", "default", "Agent name")
}
