// Package alert provides AlertSource pluggable integration for ITSM.
package alert

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

// Factory creates an AlertSource instance.
type Factory func() AlertSource

// Registry manages AlertSource registrations.
type Registry struct {
	mu    sync.RWMutex
	items map[string]Factory
}

var defaultRegistry = &Registry{items: make(map[string]Factory)}

// Default returns the global AlertSource registry.
func Default() *Registry { return defaultRegistry }

// Register adds an AlertSource factory to the registry.
// It validates that the factory's manifest has name, version, and required_permissions.
// Panics on duplicate registration or invalid manifest.
func (r *Registry) Register(f Factory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	src := f()
	mf := src.Manifest()
	if mf.Name == "" || mf.Version == "" || len(mf.RequiredPermissions) == 0 {
		panic("alert source: manifest must have name, version, and required_permissions: " + mf.Name)
	}
	if _, exists := r.items[mf.Name]; exists {
		panic(fmt.Sprintf("alert source: duplicate registration for %q", mf.Name))
	}
	r.items[mf.Name] = f
}

// MustRegister is for use in package init().
func MustRegister(f Factory) { Default().Register(f) }

// Get returns the factory for a registered alert source, or false if not found.
func (r *Registry) Get(name string) (Factory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.items[name]
	return f, ok
}

// List returns all registered alert source manifests, sorted by name.
func (r *Registry) List() []AlertSourceManifest {
	r.mu.RLock()
	items := make([]AlertSourceManifest, 0, len(r.items))
	for _, f := range r.items {
		items = append(items, f().Manifest())
	}
	r.mu.RUnlock()
	// Sort outside the lock for stability
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

// ErrNotFound is returned when an alert source is not registered.
var ErrNotFound = errors.New("alert source: not found")
