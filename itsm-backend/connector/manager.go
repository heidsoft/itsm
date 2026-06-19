package connector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Manager 负责"已注册连接器" + "已配置实例" 的生命周期管理
// 多个租户、每个租户可挂多个同名连接器实例（例如：飞书A区机器人 + 飞书B区机器人）
type Manager struct {
	registry *Registry
	logger   *zap.SugaredLogger

	mu        sync.RWMutex
	instances map[string]*instance // key = tenantID + "/" + connectorName + "/" + instanceID
}

type instance struct {
	cfg  Config
	conn Connector
}

// NewManager 创建管理器
func NewManager(registry *Registry, logger *zap.SugaredLogger) *Manager {
	if registry == nil {
		registry = Default()
	}
	return &Manager{
		registry:  registry,
		logger:    logger,
		instances: make(map[string]*instance),
	}
}

func instanceKey(c Config) string {
	return fmt.Sprintf("%d/%s/%s", c.TenantID, c.Name, c.Provider)
}

// Provision 根据配置创建/更新一个连接器实例
func (m *Manager) Provision(ctx context.Context, cfg Config) error {
	if !cfg.Enabled {
		m.Revoke(cfg)
		return nil
	}
	factory, ok := m.registry.Get(cfg.Name)
	if !ok {
		return fmt.Errorf("connector %q not registered", cfg.Name)
	}
	c := factory()
	if err := c.Init(ctx, cfg); err != nil {
		return fmt.Errorf("connector %q init failed: %w", cfg.Name, err)
	}
	m.mu.Lock()
	m.instances[instanceKey(cfg)] = &instance{cfg: cfg, conn: c}
	m.mu.Unlock()
	if m.logger != nil {
		m.logger.Infow("connector provisioned",
			"tenant", cfg.TenantID, "name", cfg.Name, "provider", cfg.Provider)
	}
	return nil
}

// Revoke 关闭并移除一个实例
func (m *Manager) Revoke(cfg Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := instanceKey(cfg)
	inst, ok := m.instances[k]
	if !ok {
		return
	}
	_ = inst.conn.Close()
	delete(m.instances, k)
	if m.logger != nil {
		m.logger.Infow("connector revoked", "tenant", cfg.TenantID, "name", cfg.Name)
	}
}

// Get 根据租户+名称取出连接器
func (m *Manager) Get(tenantID int, name string) (Connector, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, inst := range m.instances {
		if inst.cfg.TenantID == tenantID && inst.cfg.Name == name && inst.cfg.Enabled {
			return inst.conn, true
		}
	}
	return nil, false
}

// ListByTenant 列出某租户所有运行中的连接器
func (m *Manager) ListByTenant(tenantID int) []Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Config, 0)
	for _, inst := range m.instances {
		if inst.cfg.TenantID == tenantID {
			out = append(out, inst.cfg)
		}
	}
	return out
}

// Send 通过指定连接器发送消息
func (m *Manager) Send(ctx context.Context, tenantID int, name string, msg *Message) error {
	c, ok := m.Get(tenantID, name)
	if !ok {
		return fmt.Errorf("connector %q not provisioned for tenant %d", name, tenantID)
	}
	return c.Send(ctx, msg)
}

// HealthCheckAll 对所有运行中的连接器做健康检查
func (m *Manager) HealthCheckAll(ctx context.Context) map[string]HealthStatus {
	m.mu.RLock()
	insts := make([]*instance, 0, len(m.instances))
	for _, v := range m.instances {
		insts = append(insts, v)
	}
	m.mu.RUnlock()

	out := make(map[string]HealthStatus, len(insts))
	for _, ins := range insts {
		key := fmt.Sprintf("%d/%s/%s", ins.cfg.TenantID, ins.cfg.Name, ins.cfg.Provider)
		cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		out[key] = ins.conn.HealthCheck(cctx)
		cancel()
	}
	return out
}

// CloseAll 关闭所有连接器（用于优雅停机）
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, inst := range m.instances {
		_ = inst.conn.Close()
		delete(m.instances, k)
	}
}
