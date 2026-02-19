package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/toshism/tnotes/internal/config"
	"github.com/toshism/tnotes/internal/index"
	"github.com/toshism/tnotes/internal/note"
	"github.com/toshism/tnotes/internal/project"
)

var (
	addTags    string
	addProject string
)

var addCmd = &cobra.Command{
	Use:   "add [title]",
	Short: "Create a new note",
	Long:  `Creates a new note with a timestamp ID and the given title.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := args[0]

		// Parse tags
		var tags []string
		if addTags != "" {
			for _, t := range strings.Split(addTags, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
		}

		// Resolve project and path
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		projectName := addProject
		if projectName == "" {
			info := project.Resolve(cwd)
			projectName = info.Project
		}

		tags = append(tags, "project:"+projectName, "path:"+cwd)

		// Create new note
		n := note.NewNote(title, tags)
		filename := n.Filename()
		filePath := filepath.Join(config.NotesDir, filename)

		// Write note file
		content := n.ToMarkdown("")
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write note: %w", err)
		}

		// Update path and add to index
		n.Path = filePath
		idx, err := index.Load()
		if err != nil {
			return fmt.Errorf("failed to load index: %w", err)
		}

		idx.AddEntry(n.ToIndexEntry())
		if err := idx.Save(); err != nil {
			return fmt.Errorf("failed to save index: %w", err)
		}

		if jsonOutput {
			out := map[string]interface{}{
				"id":      n.ID,
				"title":   n.Title,
				"path":    filePath,
				"tags":    n.Tags,
				"project": projectName,
				"cwd":     cwd,
			}
			data, _ := json.MarshalIndent(out, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Printf("Created note: %s\n", filePath)
			fmt.Printf("ID: %s\n", n.ID)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&addTags, "tags", "t", "", "comma-separated list of tags")
	addCmd.Flags().StringVarP(&addProject, "project", "p", "", "project name (auto-detected if not provided)")
}
