package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	CurrentSchemaVersion = 1
	FileName             = "config.json"
)

type Config struct {
	SchemaVersion int    `json:"schema_version"`
	Callsign      string `json:"callsign,omitempty"`
	Locator       string `json:"locator,omitempty"`
}

func Default() Config {
	return Config{
		SchemaVersion: CurrentSchemaVersion,
	}
}

func (c Config) IdentityReady() bool {
	c = c.Normalized()

	return c.Callsign != "" && c.Locator != ""
}

func (c Config) Normalized() Config {
	c.SchemaVersion = CurrentSchemaVersion
	c.Callsign = strings.ToUpper(strings.TrimSpace(c.Callsign))
	c.Locator = strings.TrimSpace(c.Locator)

	return c
}

func Path(configDir string) string {
	return filepath.Join(configDir, FileName)
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read configuration: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode configuration: %w", err)
	}

	switch cfg.SchemaVersion {
	case 0:
		// Version 0 is accepted as the initial unversioned form.
		// There has not yet been a public K2EXEmail configuration,
		// but this gives us a migration path from early development builds.
		cfg.SchemaVersion = CurrentSchemaVersion
	case CurrentSchemaVersion:
	default:
		return Config{}, fmt.Errorf(
			"unsupported configuration schema version %d",
			cfg.SchemaVersion,
		)
	}

	return cfg.Normalized(), nil
}

func Save(path string, cfg Config) error {
	cfg = cfg.Normalized()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode configuration: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}

	if err := writeSafely(path, data); err != nil {
		return fmt.Errorf("save configuration: %w", err)
	}

	return nil
}

func writeSafely(path string, data []byte) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return err
	}

	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	if err := tmp.Chmod(0o600); err != nil {
		return err
	}

	if _, err := tmp.Write(data); err != nil {
		return err
	}

	if err := tmp.Sync(); err != nil {
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpName, path); err != nil {
		return err
	}

	return nil
}
