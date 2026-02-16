package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

var (
	NotesDir string
)

func Init(cfgFile, notesDir string) {
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

	// Priority: explicit dir > env var > config file default
	if notesDir != "" {
		NotesDir = notesDir
	} else if envDir := os.Getenv("TNOTES_DIR"); envDir != "" {
		NotesDir = envDir
	} else {
		NotesDir = viper.GetString("notes_dir")
	}

	NotesDir = expandPath(NotesDir)
}

func defaultNotesDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./tnotes"
	}
	return filepath.Join(home, "tnotes")
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
