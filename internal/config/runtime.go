package config

import "sync"

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

// Current returns a snapshot of the current configuration.
func (r *Runtime) Current() Config {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.cfg
}

// UpdateIdentity persists a new station identity and only publishes it to
// the running application after the configuration file is safely saved.
func (r *Runtime) UpdateIdentity(
	callsign string,
	locator string,
) (Config, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	next := r.cfg
	next.Callsign = callsign
	next.Locator = locator
	next = next.Normalized()

	if err := Save(r.path, next); err != nil {
		return r.cfg, err
	}

	r.cfg = next
	return next, nil
}
