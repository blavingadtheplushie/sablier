// paths decides where sablier keeps its local files.
package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

// DataDir is SABLIER_HOME if set, else the os data dir:
// mac ~/Library/Application Support/sablier, linux ~/.local/share/sablier, windows %AppData%\sablier.
func DataDir() (string, error) {
	if d := os.Getenv("SABLIER_HOME"); d != "" {
		return d, nil
	}
	if runtime.GOOS == "linux" {
		if d := os.Getenv("XDG_DATA_HOME"); d != "" {
			return filepath.Join(d, "sablier"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share", "sablier"), nil
	}
	d, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "sablier"), nil
}
