package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/toshism/tnotes/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a notes directory",
	Long:  `Creates the notes directory. The index is built on first use.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		notesDir := config.NotesDir

		// Create notes directory if it doesn't exist
		if err := os.MkdirAll(notesDir, 0755); err != nil {
			return fmt.Errorf("failed to create notes directory: %w", err)
		}

		if jsonOutput {
			fmt.Printf(`{"status": "initialized", "path": %q}`, notesDir)
			fmt.Println()
		} else {
			fmt.Printf("Initialized tnotes in %s\n", notesDir)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
