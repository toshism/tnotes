package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tosh/tnotes/internal/index"
)

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Rebuild the index",
	Long:  `Scans the notes directory and rebuilds the index from scratch.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		idx, err := index.Rebuild()
		if err != nil {
			return fmt.Errorf("failed to rebuild index: %w", err)
		}

		if err := idx.Save(); err != nil {
			return fmt.Errorf("failed to save index: %w", err)
		}

		if jsonOutput {
			out := map[string]interface{}{
				"status": "rebuilt",
				"count":  len(idx.Entries),
			}
			data, _ := json.MarshalIndent(out, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Printf("Index rebuilt: %d note(s) indexed\n", len(idx.Entries))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(indexCmd)
}
