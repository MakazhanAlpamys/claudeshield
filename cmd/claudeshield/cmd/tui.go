package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/MakazhanAlpamys/claudeshield/internal/audit"
	"github.com/MakazhanAlpamys/claudeshield/internal/config"
	"github.com/MakazhanAlpamys/claudeshield/internal/sandbox"
	"github.com/MakazhanAlpamys/claudeshield/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch interactive TUI dashboard",
	Long:  "Opens a terminal-based UI to monitor sessions, view audit logs, and manage agents.",
	RunE: func(cmd *cobra.Command, args []string) error {
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
			return fmt.Errorf("connecting to Docker: %w", err)
		}
		defer engine.Close()

		model := tui.NewModel(engine, cfg.Audit.LogDir)
		p := tea.NewProgram(model, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}

		return nil
	},
}
