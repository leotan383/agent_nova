package paths

import (
	"os"
	"path/filepath"
)

// Layout holds global nova config/data paths.
type Layout struct {
	ConfigDir string
	DataDir   string
}

// Global returns the global nova layout (~/.config/nova, ~/.local/share/nova).
func Global() Layout {
	if home := os.Getenv("NOVA_HOME"); home != "" {
		return Layout{
			ConfigDir: filepath.Join(home, "config"),
			DataDir:   filepath.Join(home, "data"),
		}
	}
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share")
	}
	return Layout{
		ConfigDir: filepath.Join(configDir, "nova"),
		DataDir:   filepath.Join(dataDir, "nova"),
	}
}

func (l Layout) ConfigFile() string  { return filepath.Join(l.ConfigDir, "config.yaml") }
func (l Layout) CurrentProjectFile() string { return filepath.Join(l.ConfigDir, "current") }
