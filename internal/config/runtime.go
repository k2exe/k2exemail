package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
)

// Runtime owns the application's current configuration and keeps updates
// synchronized with persistent storage.
type Runtime struct {
	mu   sync.RWMutex
	path string
	cfg  Config
}

func NewRuntime(path string, cfg Config) *Runtime {
	return &Runtime{
		path: path,
		cfg:  cfg.Normalized(),
	}
}

// Current returns an independent snapshot of the current configuration.
func (r *Runtime) Current() Config {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.cfg.Clone()
}

// UpdateIdentity persists a new station identity and only publishes it to
// the running application after the configuration file is safely saved.
func (r *Runtime) UpdateIdentity(
	callsign string,
	locator string,
) (Config, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	next := r.cfg.Clone()
	next.Callsign = callsign
	next.Locator = locator

	if err := r.saveLocked(next); err != nil {
		return r.cfg.Clone(), err
	}

	return r.cfg.Clone(), nil
}

// CreateConnectionProfile validates and persists a new connection profile.
// The runtime owns profile IDs; callers should leave ID empty.
func (r *Runtime) CreateConnectionProfile(
	profile ConnectionProfile,
) (ConnectionProfile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	profile = profile.Normalized()
	if profile.ID != "" {
		return ConnectionProfile{}, fmt.Errorf(
			"new connection profile ID must be empty",
		)
	}

	id, err := r.newProfileIDLocked()
	if err != nil {
		return ConnectionProfile{}, err
	}
	profile.ID = id

	if err := profile.Validate(); err != nil {
		return ConnectionProfile{}, fmt.Errorf(
			"validate connection profile: %w",
			err,
		)
	}

	next := r.cfg.Clone()
	next.ConnectionProfiles = append(
		next.ConnectionProfiles,
		profile,
	)

	if err := r.saveLocked(next); err != nil {
		return ConnectionProfile{}, err
	}

	return profile.Normalized(), nil
}

// UpdateConnectionProfile replaces an existing profile with the same ID.
func (r *Runtime) UpdateConnectionProfile(
	profile ConnectionProfile,
) (ConnectionProfile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	profile = profile.Normalized()

	if profile.ID == "" {
		return ConnectionProfile{}, fmt.Errorf(
			"connection profile ID is required",
		)
	}
	if err := profile.Validate(); err != nil {
		return ConnectionProfile{}, fmt.Errorf(
			"validate connection profile: %w",
			err,
		)
	}

	next := r.cfg.Clone()

	found := false
	for i := range next.ConnectionProfiles {
		if next.ConnectionProfiles[i].ID != profile.ID {
			continue
		}

		next.ConnectionProfiles[i] = profile
		found = true
		break
	}

	if !found {
		return ConnectionProfile{}, fmt.Errorf(
			"connection profile %q not found",
			profile.ID,
		)
	}

	if err := r.saveLocked(next); err != nil {
		return ConnectionProfile{}, err
	}

	return profile.Normalized(), nil
}

// DeleteConnectionProfile removes a persisted profile by ID.
func (r *Runtime) DeleteConnectionProfile(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("connection profile ID is required")
	}

	next := r.cfg.Clone()

	index := -1
	for i := range next.ConnectionProfiles {
		if next.ConnectionProfiles[i].ID == id {
			index = i
			break
		}
	}

	if index < 0 {
		return fmt.Errorf(
			"connection profile %q not found",
			id,
		)
	}

	next.ConnectionProfiles = append(
		next.ConnectionProfiles[:index],
		next.ConnectionProfiles[index+1:]...,
	)

	if err := r.saveLocked(next); err != nil {
		return err
	}

	return nil
}

// saveLocked persists and then publishes next. The caller must hold r.mu.
func (r *Runtime) saveLocked(next Config) error {
	next = next.Normalized()

	if err := Save(r.path, next); err != nil {
		return err
	}

	r.cfg = next
	return nil
}

func (r *Runtime) newProfileIDLocked() (string, error) {
	for {
		var raw [16]byte
		if _, err := rand.Read(raw[:]); err != nil {
			return "", fmt.Errorf(
				"generate connection profile ID: %w",
				err,
			)
		}

		id := "profile-" + hex.EncodeToString(raw[:])

		var exists bool
		for _, profile := range r.cfg.ConnectionProfiles {
			if profile.ID == id {
				exists = true
				break
			}
		}

		if !exists {
			return id, nil
		}
	}
}
