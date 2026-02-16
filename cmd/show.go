package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tosh/tnotes/internal/config"
	"github.com/tosh/tnotes/internal/index"
	"github.com/tosh/tnotes/internal/note"
)

var showCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Display a note",
	Long:  `Shows the content of a note by its ID.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		// First try to find in index for quick path lookup
		idx, _ := index.Load()
		var filePath string

		if idx != nil {
			if entry := idx.FindByID(id); entry != nil {
				filePath = entry.Path
			}
		}

		var n *note.Note
		var content string
		var err error

		if filePath != "" {
			// Read from known path
			n, content, err = note.ParseFile(filePath)
		} else {
			// Search for the note
			n, content, err = note.FindNoteByID(config.NotesDir, id)
		}

		if err != nil {
			return fmt.Errorf("note not found: %w", err)
		}

		if jsonOutput {
			out := map[string]interface{}{
				"id":       n.ID,
				"title":    n.Title,
				"tags":     n.Tags,
				"links":    n.Links,
				"created":  n.Created,
				"modified": n.Modified,
				"path":     n.Path,
				"content":  content,
			}
			data, _ := json.MarshalIndent(out, "", "  ")
			fmt.Println(string(data))
		} else {
			// Read and print the raw file
			data, err := os.ReadFile(n.Path)
			if err != nil {
				return fmt.Errorf("failed to read note: %w", err)
			}
			fmt.Print(string(data))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
