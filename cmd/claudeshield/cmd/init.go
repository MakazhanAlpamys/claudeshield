package cmd

import (
	"fmt"

	"github.com/MakazhanAlpamys/claudeshield/internal/config"
	"github.com/spf13/cobra"
)

var initConfigCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize ClaudeShield in current project",
	Long:  "Creates a .claudeshield.yaml config file with sensible defaults.",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir := getProjectDir(cmd)
		force, _ := cmd.Flags().GetBool("force")

		cfg := config.DefaultConfig()

		configPath := projectDir + "/" + config.ConfigFileName
		if !force {
			if config.Exists(projectDir) {
				return fmt.Errorf("config already exists at %s (use --force to overwrite)", configPath)
			}
		}

		if err := config.Save(projectDir, cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Printf("🛡️  ClaudeShield initialized!\n")
		fmt.Printf("   Config: %s\n", configPath)
		fmt.Println()
		fmt.Println("   Next steps:")
		fmt.Println("   1. Edit .claudeshield.yaml to customize rules")
		fmt.Println("   2. Run 'claudeshield start' to begin")

		return nil
	},
}

func init() {
	initConfigCmd.Flags().Bool("force", false, "Overwrite existing config")
}
