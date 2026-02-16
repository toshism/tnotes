package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tosh/tnotes/internal/config"
)

var (
	cfgFile    string
	notesDir   string
	jsonOutput bool
)

var rootCmd = &cobra.Command{
	Use:   "tnotes",
	Short: "AI-native note-taking system",
	Long: `tnotes is a minimal CLI tool for managing markdown notes,
designed explicitly for AI interaction.

Plain markdown files with structured frontmatter, plus tooling
for fast search and metadata management.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/tnotes/config.toml)")
	rootCmd.PersistentFlags().StringVar(&notesDir, "dir", "", "notes directory (default is ~/tnotes or $TNOTES_DIR)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
}

func initConfig() {
	config.Init(cfgFile, notesDir)
}
