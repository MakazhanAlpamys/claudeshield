package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MakazhanAlpamys/claudeshield/internal/audit"
	"github.com/MakazhanAlpamys/claudeshield/internal/config"
	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit [session-id]",
	Short: "View audit logs",
	Long:  "Displays audit log entries. Optionally filter by session ID.",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir := getProjectDir(cmd)
		cfg, err := config.Load(projectDir)
		if err != nil {
			return err
		}

		sessionFilter := ""
		if len(args) > 0 {
			sessionFilter = args[0]
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		last, _ := cmd.Flags().GetInt("last")

		logDir := cfg.Audit.LogDir
		if !filepath.IsAbs(logDir) {
			logDir = filepath.Join(projectDir, logDir)
		}

		entries, err := audit.ReadSession(logDir, sessionFilter)
		if err != nil {
			return fmt.Errorf("reading audit logs: %w", err)
		}

		if len(entries) == 0 {
			fmt.Println("No audit entries found")
			return nil
		}

		// Apply --last filter
		if last > 0 && last < len(entries) {
			entries = entries[len(entries)-last:]
		}

		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(entries)
		}

		fmt.Printf("%-20s %-20s %-15s %-10s %-30s %s\n",
			"TIME", "SESSION", "EVENT", "ACTION", "COMMAND", "REASON")
		fmt.Println("───────────────────────────────────────────────────────────────────────────────────────────────────────")

		for _, e := range entries {
			command := e.Command
			if len(command) > 30 {
				command = command[:27] + "..."
			}
			fmt.Printf("%-20s %-20s %-15s %-10s %-30s %s\n",
				e.Timestamp.Format("15:04:05"),
				truncate(e.SessionID, 20),
				e.EventType,
				e.Action,
				command,
				e.Reason,
			)
		}

		return nil
	},
}

func init() {
	auditCmd.Flags().Bool("json", false, "Output as JSON")
	auditCmd.Flags().Int("last", 0, "Show last N entries")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
