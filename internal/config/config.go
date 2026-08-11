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
	CurrentSchemaVersion = 2
	FileName             = "config.json"
)

type NetworkType string

const (
	NetworkInternet NetworkType = "internet"
	NetworkLAN      NetworkType = "lan"
	NetworkAREDN    NetworkType = "aredn"
	NetworkRadio    NetworkType = "radio"
)

type TransportType string

const (
	TransportCMSTelnet TransportType = "cms_telnet"
	TransportDirectTCP TransportType = "direct_tcp"
)

type CMSMode string

const (
	CMSModeTest       CMSMode = "test"
	CMSModeProduction CMSMode = "production"
)

type CMSProfile struct {
	Mode CMSMode `json:"mode"`
}

type TCPProfile struct {
	Address    string `json:"address"`
	TargetCall string `json:"target_call"`
}

type ConnectionProfile struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Network   NetworkType   `json:"network"`
	Transport TransportType `json:"transport"`

	CMS *CMSProfile `json:"cms,omitempty"`
	TCP *TCPProfile `json:"tcp,omitempty"`
}

type Config struct {
	SchemaVersion      int                 `json:"schema_version"`
	Callsign           string              `json:"callsign,omitempty"`
	Locator            string              `json:"locator,omitempty"`
	ConnectionProfiles []ConnectionProfile `json:"connection_profiles"`
}

func Default() Config {
	return Config{
		SchemaVersion:      CurrentSchemaVersion,
		ConnectionProfiles: DefaultConnectionProfiles(),
	}
}

func DefaultConnectionProfiles() []ConnectionProfile {
	return []ConnectionProfile{
		{
			ID:        "cms-test",
			Name:      "Winlink Test CMS",
			Network:   NetworkInternet,
			Transport: TransportCMSTelnet,
			CMS: &CMSProfile{
				Mode: CMSModeTest,
			},
		},
		{
			ID:        "cms-production",
			Name:      "Winlink Production CMS",
			Network:   NetworkInternet,
			Transport: TransportCMSTelnet,
			CMS: &CMSProfile{
				Mode: CMSModeProduction,
			},
		},
	}
}

func (c Config) Clone() Config {
	cloned := c
	cloned.ConnectionProfiles = cloneConnectionProfiles(
		c.ConnectionProfiles,
	)
	return cloned
}

func cloneConnectionProfiles(
	profiles []ConnectionProfile,
) []ConnectionProfile {
	if profiles == nil {
		return nil
	}

	cloned := make([]ConnectionProfile, len(profiles))
	for i, profile := range profiles {
		cloned[i] = profile

		if profile.CMS != nil {
			cms := *profile.CMS
			cloned[i].CMS = &cms
		}

		if profile.TCP != nil {
			tcp := *profile.TCP
			cloned[i].TCP = &tcp
		}
	}

	return cloned
}

func (c Config) IdentityReady() bool {
	c = c.Normalized()

	return c.Callsign != "" && c.Locator != ""
}

func (c Config) Normalized() Config {
	c = c.Clone()
	c.SchemaVersion = CurrentSchemaVersion
	c.Callsign = strings.ToUpper(strings.TrimSpace(c.Callsign))
	c.Locator = strings.TrimSpace(c.Locator)

	if c.ConnectionProfiles == nil {
		c.ConnectionProfiles = DefaultConnectionProfiles()
	}

	for i := range c.ConnectionProfiles {
		c.ConnectionProfiles[i] =
			c.ConnectionProfiles[i].Normalized()
	}

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
		cfg.ConnectionProfiles = DefaultConnectionProfiles()
		cfg.SchemaVersion = CurrentSchemaVersion
	case 1:
		cfg.ConnectionProfiles = DefaultConnectionProfiles()
		cfg.SchemaVersion = CurrentSchemaVersion
	case CurrentSchemaVersion:
	default:
		return Config{}, fmt.Errorf(
			"unsupported configuration schema version %d",
			cfg.SchemaVersion,
		)
	}

	cfg = cfg.Normalized()
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf(
			"validate configuration: %w",
			err,
		)
	}

	return cfg, nil
}

func Save(path string, cfg Config) error {
	cfg = cfg.Normalized()

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf(
			"validate configuration: %w",
			err,
		)
	}

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
