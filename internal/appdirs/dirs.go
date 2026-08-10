package appdirs

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	appDirName   = "K2EXEmail"
	unixDirName  = "k2exemail"
	dataOverride = "K2EXEMAIL_DATA_DIR"
)

type Dirs struct {
	Data   string
	Config string
}

func Default() (Dirs, error) {
	configDir, err := defaultConfigDir()
	if err != nil {
		return Dirs{}, err
	}

	if override := strings.TrimSpace(os.Getenv(dataOverride)); override != "" {
		if !filepath.IsAbs(override) {
			return Dirs{}, fmt.Errorf("%s must be an absolute path", dataOverride)
		}

		return Dirs{
			Data:   filepath.Clean(override),
			Config: configDir,
		}, nil
	}

	dataDir, err := defaultDataDir()
	if err != nil {
		return Dirs{}, err
	}

	return Dirs{
		Data:   dataDir,
		Config: configDir,
	}, nil
}

func defaultDataDir() (string, error) {
	return resolveDataDir(runtime.GOOS, os.Getenv, os.UserHomeDir)
}

func defaultConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("determine configuration directory: %w", err)
	}

	name := unixDirName
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		name = appDirName
	}

	return filepath.Join(base, name), nil
}

func resolveDataDir(
	goos string,
	getenv func(string) string,
	homeDir func() (string, error),
) (string, error) {
	switch goos {
	case "windows":
		base := getenv("LOCALAPPDATA")
		if base == "" {
			return "", fmt.Errorf("LOCALAPPDATA is not defined")
		}
		return filepath.Join(base, appDirName), nil

	case "darwin":
		home, err := homeDir()
		if err != nil {
			return "", fmt.Errorf("determine home directory: %w", err)
		}
		return filepath.Join(home, "Library", "Application Support", appDirName), nil

	default:
		if base := getenv("XDG_DATA_HOME"); base != "" {
			if !strings.HasPrefix(base, "/") {
				return "", fmt.Errorf("XDG_DATA_HOME must be an absolute path")
			}
			return filepath.Join(base, unixDirName), nil
		}

		home, err := homeDir()
		if err != nil {
			return "", fmt.Errorf("determine home directory: %w", err)
		}

		return filepath.Join(home, ".local", "share", unixDirName), nil
	}
}
