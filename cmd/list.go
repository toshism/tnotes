package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/toshism/tnotes/internal/index"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all notes",
	Long:  `Lists all notes in the index.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		idx, err := index.Load()
		if err != nil {
			return fmt.Errorf("failed to load index: %w", err)
		}

		if len(idx.Entries) == 0 {
			if jsonOutput {
				fmt.Println("[]")
			} else {
				fmt.Println("No notes found. Run 'tnotes add' to create one.")
			}
			return nil
		}

		if jsonOutput {
			data, _ := json.MarshalIndent(idx.Entries, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Printf("%-14s  %-40s  %s\n", "ID", "TITLE", "TAGS")
			fmt.Println(strings.Repeat("-", 80))
			for _, e := range idx.Entries {
				tags := strings.Join(e.Tags, ", ")
				title := e.Title
				if len(title) > 38 {
					title = title[:35] + "..."
				}
				fmt.Printf("%-14s  %-40s  %s\n", e.ID, title, tags)
			}
			fmt.Printf("\n%d note(s)\n", len(idx.Entries))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
