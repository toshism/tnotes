package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

var (
	NotesDir string
)

func Init(cfgFile, notesDir string, global bool) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(filepath.Join(home, ".config", "tnotes"))
		}
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
	}

	viper.SetEnvPrefix("TNOTES")
	viper.AutomaticEnv()

	viper.SetDefault("notes_dir", defaultNotesDir())

	_ = viper.ReadInConfig()

	// Priority: CLI flag > global flag > env var > auto-detect > default
	if notesDir != "" {
		NotesDir = notesDir
	} else if global {
		NotesDir = defaultNotesDir()
	} else if envDir := os.Getenv("TNOTES_DIR"); envDir != "" {
		NotesDir = envDir
	} else if projectDir := detectProjectNotes(); projectDir != "" {
		NotesDir = projectDir
	} else {
		NotesDir = viper.GetString("notes_dir")
	}

	// Expand ~ in path
	NotesDir = expandPath(NotesDir)
}

func defaultNotesDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./notes"
	}
	return filepath.Join(home, "notes")
}

// detectProjectNotes checks for project-local notes directory
// Returns absolute path if found, empty string otherwise
func detectProjectNotes() string {
	// Check for ./tnotes/.tnotes (tnotes subdirectory)
	if info, err := os.Stat("./tnotes/.tnotes"); err == nil && info.IsDir() {
		if abs, err := filepath.Abs("./tnotes"); err == nil {
			return abs
		}
	}

	// Check for ./.tnotes (notes in current directory)
	if info, err := os.Stat("./.tnotes"); err == nil && info.IsDir() {
		if abs, err := filepath.Abs("."); err == nil {
			return abs
		}
	}

	return ""
}

func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

// IndexDir returns the path to the .tnotes directory
func IndexDir() string {
	return filepath.Join(NotesDir, ".tnotes")
}

// IndexFile returns the path to the index.json file
func IndexFile() string {
	return filepath.Join(IndexDir(), "index.json")
}
