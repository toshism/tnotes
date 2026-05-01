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

// ResolvedNotesDirFor returns an absolute, symlink-resolved notes directory when possible.
func ResolvedNotesDirFor(notesDir string) string {
	absDir, err := filepath.Abs(notesDir)
	if err != nil {
		return notesDir
	}
	resolved, err := filepath.EvalSymlinks(absDir)
	if err != nil {
		return absDir
	}
	return resolved
}

// IndexDirFor returns the path to the .tnotes directory for a notes directory.
func IndexDirFor(notesDir string) string {
	return filepath.Join(notesDir, ".tnotes")
}

// IndexFileFor returns the path to the index.json file for a notes directory.
func IndexFileFor(notesDir string) string {
	return filepath.Join(IndexDirFor(notesDir), "index.json")
}

// BleveIndexDirFor returns the path to the derived bleve index for a notes directory.
func BleveIndexDirFor(notesDir string) string {
	return filepath.Join(IndexDirFor(notesDir), "bleve")
}

// IndexDir returns the path to the .tnotes directory
func IndexDir() string {
	return IndexDirFor(NotesDir)
}

// IndexFile returns the path to the index.json file
func IndexFile() string {
	return IndexFileFor(NotesDir)
}

// BleveIndexDir returns the path to the bleve index directory.
func BleveIndexDir() string {
	return BleveIndexDirFor(NotesDir)
}
