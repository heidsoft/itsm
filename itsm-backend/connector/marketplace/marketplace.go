// Package marketplace 提供"插件/技能/连接器市场"的本地展示层
// 远端市场会通过 Marketplace API 拉取，运行时与本地 Registry 合并
package marketplace

import (
	"sort"
	"sync"

	"itsm-backend/connector"
)

// Entry 市场中的一项
type Entry struct {
	Manifest  connector.Manifest `json:"manifest"`
	Local     bool               `json:"local"`     // 是否本地内置
	Installed bool               `json:"installed"` // 是否已为此租户安装
	Category  string             `json:"category"`  // 分类：im / webhook / database / ...
}

// Source 远程市场数据源（可由外部实现 HTTP 拉取、文件加载等）
type Source interface {
	Fetch() ([]connector.Manifest, error)
}

// Market 本地市场视图
type Market struct {
	mu      sync.RWMutex
	entries map[string]Entry
}

func New() *Market { return &Market{entries: make(map[string]Entry)} }

// Refresh 用 registry + remote source 重新构建视图
func (m *Market) Refresh(reg *connector.Registry, srcs ...Source) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = make(map[string]Entry)
	// 1. 本地注册表
	if reg != nil {
		for _, mf := range reg.List() {
			m.entries[mf.Name] = Entry{Manifest: mf, Local: true, Category: string(mf.Type)}
		}
	}
	// 2. 远程源（覆盖本地同 name 的元信息）
	for _, src := range srcs {
		manifests, err := src.Fetch()
		if err != nil {
			return err
		}
		for _, mf := range manifests {
			if local, ok := m.entries[mf.Name]; ok {
				local.Manifest = mf
				m.entries[mf.Name] = local
			} else {
				m.entries[mf.Name] = Entry{Manifest: mf, Category: string(mf.Type)}
			}
		}
	}
	return nil
}

// Search 按关键字过滤（title / description / tags）
func (m *Market) Search(q string, cat string) []Entry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Entry, 0, len(m.entries))
	for _, e := range m.entries {
		if cat != "" && e.Category != cat {
			continue
		}
		if q != "" && !matchAny(e.Manifest, q) {
			continue
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Manifest.Name < out[j].Manifest.Name })
	return out
}

func matchAny(m connector.Manifest, q string) bool {
	if contains(m.Title, q) || contains(m.Name, q) || contains(m.Description, q) || contains(m.Provider, q) {
		return true
	}
	for _, t := range m.Tags {
		if contains(t, q) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	if sub == "" {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if equalsFold(s[i:i+len(sub)], sub) {
			return true
		}
	}
	return false
}

func equalsFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
