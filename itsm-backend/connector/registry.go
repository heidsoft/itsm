package connector

import (
	"fmt"
	"sort"
	"sync"
)

// Factory 创建一个 Connector 实例
// 之所以用 Factory 而不是直接 Register(Connector)：
//   - 连接器需要"按配置初始化"（一个 feishu 飞书机器人 = N 个 Connector 实例）
//   - Factory 把"类"注册到 Registry，实例化交给上层
type Factory func() Connector

// Registry 全局注册表：保存"连接器类型 -> 工厂"
// 内置连接器在 init() 中注册；第三方插件/技能也可通过 import 触发
type Registry struct {
	mu        sync.RWMutex
	factories map[string]Factory
	manifests map[string]Manifest
}

var defaultRegistry = NewRegistry()

// NewRegistry 创建注册表
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]Factory),
		manifests: make(map[string]Manifest),
	}
}

// Default 返回全局注册表
func Default() *Registry { return defaultRegistry }

// Register 注册一个连接器工厂
func (r *Registry) Register(f Factory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := f()
	m := c.Manifest()
	if m.Name == "" {
		panic("connector: Manifest().Name must not be empty")
	}
	if _, exists := r.factories[m.Name]; exists {
		panic(fmt.Sprintf("connector: duplicate registration for %q", m.Name))
	}
	r.factories[m.Name] = f
	r.manifests[m.Name] = m
}

// MustRegister 在包 init() 中使用：失败即 panic
func MustRegister(f Factory) { Default().Register(f) }

// Get 取出指定连接器工厂
func (r *Registry) Get(name string) (Factory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.factories[name]
	return f, ok
}

// GetManifest 取元信息
func (r *Registry) GetManifest(name string) (Manifest, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.manifests[name]
	return m, ok
}

// List 列出所有已注册连接器的元信息（按 Name 排序）
// 用于：插件市场展示、API 列表、配置表单
func (r *Registry) List() []Manifest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Manifest, 0, len(r.manifests))
	for _, m := range r.manifests {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ListByType 按连接器类型过滤
func (r *Registry) ListByType(t ConnectorType) []Manifest {
	out := make([]Manifest, 0)
	for _, m := range r.List() {
		if m.Type == t {
			out = append(out, m)
		}
	}
	return out
}

// Names 返回所有已注册连接器 Name
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.factories))
	for n := range r.factories {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
