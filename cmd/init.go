package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tosh/tnotes/internal/config"
	"github.com/tosh/tnotes/internal/index"
	"github.com/tosh/tnotes/internal/note"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a notes directory",
	Long:  `Creates the .tnotes directory and an empty index.json file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		notesDir := config.NotesDir
		indexDir := config.IndexDir()

		// Create notes directory if it doesn't exist
		if err := os.MkdirAll(notesDir, 0755); err != nil {
			return fmt.Errorf("failed to create notes directory: %w", err)
		}

		// Create .tnotes directory
		if err := os.MkdirAll(indexDir, 0755); err != nil {
			return fmt.Errorf("failed to create .tnotes directory: %w", err)
		}

		// Create empty index
		idx := &index.Index{Entries: []note.IndexEntry{}}
		if err := idx.Save(); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
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
