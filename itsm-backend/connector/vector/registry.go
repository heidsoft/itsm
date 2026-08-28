package vector

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Factory func(collection string, config map[string]interface{}) (VectorStore, error)

var registry = struct {
	sync.RWMutex
	factories map[string]Factory
}{factories: make(map[string]Factory)}

func Register(name string, factory Factory) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" || factory == nil {
		panic("vector: backend name and factory are required")
	}
	registry.Lock()
	defer registry.Unlock()
	if _, exists := registry.factories[name]; exists {
		panic("vector: backend already registered: " + name)
	}
	registry.factories[name] = factory
}

func New(name, collection string, config map[string]interface{}) (VectorStore, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	registry.RLock()
	factory, ok := registry.factories[name]
	registry.RUnlock()
	if !ok {
		return nil, fmt.Errorf("vector: unknown backend %q (available: %s)", name, strings.Join(List(), ", "))
	}
	store, err := factory(collection, config)
	if err != nil {
		return nil, fmt.Errorf("vector: initialize %s backend: %w", name, err)
	}
	return store, nil
}

func List() []string {
	registry.RLock()
	defer registry.RUnlock()
	names := make([]string, 0, len(registry.factories))
	for name := range registry.factories {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
