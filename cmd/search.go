package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/toshism/tnotes/internal/index"
	"github.com/toshism/tnotes/internal/search"
)

var (
	searchTag     string
	searchProject string
	searchLimit   int
	searchSnippet bool
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search notes",
	Long:  `Searches notes by text (title + content) and/or tags.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		idx, err := index.Load()
		if err != nil {
			return fmt.Errorf("failed to load index: %w", err)
		}

		// Build query
		q := search.Query{Limit: searchLimit, Snippets: jsonOutput || searchSnippet}

		if len(args) > 0 {
			q.Text = args[0]
		}

		if searchTag != "" {
			for _, t := range strings.Split(searchTag, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					q.Tags = append(q.Tags, t)
				}
			}
		}
		if project := strings.TrimSpace(searchProject); project != "" {
			q.Tags = append(q.Tags, "project:"+project)
		}

		results, err := search.Search(idx, q)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		if len(results) == 0 {
			if jsonOutput {
				fmt.Println("[]")
			} else {
				fmt.Println("No matching notes found.")
			}
			return nil
		}

		if jsonOutput {
			data, _ := json.MarshalIndent(results, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Printf("%-14s  %-40s  %s\n", "ID", "TITLE", "MATCHES")
			fmt.Println(strings.Repeat("-", 80))
			for _, r := range results {
				title := r.Entry.Title
				if len(title) > 38 {
					title = title[:35] + "..."
				}
				matches := strings.Join(r.Matches, ", ")
				fmt.Printf("%-14s  %-40s  %s\n", r.Entry.ID, title, matches)
				if searchSnippet && r.Snippet != "" {
					fmt.Printf("%-14s  %s\n", "", r.Snippet)
				}
			}
			fmt.Printf("\n%d result(s)\n", len(results))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().StringVar(&searchTag, "tag", "", "filter by tag (comma-separated for multiple)")
	searchCmd.Flags().StringVar(&searchProject, "project", "", "filter by project name")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 20, "maximum number of results (0 means unlimited)")
	searchCmd.Flags().BoolVar(&searchSnippet, "snippet", false, "show matching snippets in human-readable output")
}
