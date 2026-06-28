# 架构清理建议

## 问题分析

### 1. 未使用的 internal/domain 目录

`/internal/domain/` 目录包含完整的 DDD 架构实现，但：
- **未被任何代码引用** - Grep 搜索显示没有任何 import
- 与根目录 `service/` 重复实现
- 占用空间约 15 个子模块

### 2. 多个入口点

存在多个 `cmd/` 入口：
- `cmd/simple` - 简单入口
- `cmd/cmdb` - CMDB 专用入口

### 3. 建议清理步骤

#### 步骤 1: 备份 internal/domain
```bash
cp -r internal/domain internal/domain.backup
```

#### 步骤 2: 验证无依赖
```bash
go list -f '{{.ImportPath}} {{.Imports}}' ./... | grep "internal/domain"
```

#### 步骤 3: 删除未使用的目录
```bash
rm -rf internal/domain
```

#### 步骤 4: 统一入口点
保留单一入口 `cmd/simple`，删除其他入口：
```bash
rm -rf cmd/cmdb
```

## 建议架构

```
itsm-backend/
├── cmd/                    # 单一入口点
│   └── simple/
│       └── main.go
├── controller/             # HTTP 处理层
├── service/                # 业务逻辑层 (当前使用)
├── ent/                    # 数据访问层 (Ent ORM)
├── middleware/             # 中间件
├── dto/                    # 数据传输对象
├── cache/                  # 缓存服务
├── config/                 # 配置
└── main.go                 # 入口
```

## 当前状态

- **internal/domain/**: 未使用，建议删除
- **cmd/cmdb**: 需要评估是否需要独立 CMDB 入口
- **service/**: 当前使用的服务层，符合简单架构
