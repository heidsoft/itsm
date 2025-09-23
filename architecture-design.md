# ITSM系统架构设计与代码落地方案

## 1. 系统架构概览

### 1.1 整体架构
```
┌─────────────────────────────────────────────────────────────┐
│                    前端层 (Next.js)                         │
├─────────────────────────────────────────────────────────────┤
│                    API网关层 (Gin)                          │
├─────────────────────────────────────────────────────────────┤
│                    业务服务层                                │
│  ┌─────────────┬─────────────┬─────────────┬─────────────┐  │
│  │  工单服务   │  事件服务   │  AI服务     │  知识库服务 │  │
│  └─────────────┴─────────────┴─────────────┴─────────────┘  │
├─────────────────────────────────────────────────────────────┤
│                    数据访问层 (Ent ORM)                     │
├─────────────────────────────────────────────────────────────┤
│                    数据存储层                                │
│  ┌─────────────┬─────────────┬─────────────┬─────────────┐  │
│  │ PostgreSQL  │   Redis     │  Vector DB  │  File Store │  │
│  └─────────────┴─────────────┴─────────────┴─────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 技术栈选型
- **后端**: Go + Gin + Ent ORM + PostgreSQL
- **前端**: Next.js 14 + TypeScript + Tailwind CSS + Ant Design
- **缓存**: Redis
- **向量数据库**: pgvector (PostgreSQL扩展)
- **消息队列**: Redis Streams
- **监控**: Prometheus + Grafana
- **日志**: Zap + ELK Stack

## 2. 后端架构设计

### 2.1 领域驱动设计 (DDD) 架构

#### 2.1.1 领域划分
```
itsm-backend/
├── domain/                    # 领域层
│   ├── ticket/               # 工单领域
│   │   ├── entity/          # 实体
│   │   ├── repository/      # 仓储接口
│   │   ├── service/         # 领域服务
│   │   └── valueobject/     # 值对象
│   ├── incident/            # 事件领域
│   ├── problem/             # 问题领域
│   ├── change/              # 变更领域
│   ├── knowledge/           # 知识库领域
│   ├── ai/                  # AI服务领域
│   └── user/                # 用户领域
├── infrastructure/           # 基础设施层
│   ├── database/            # 数据库实现
│   ├── cache/               # 缓存实现
│   ├── message/             # 消息队列
│   ├── ai/                  # AI服务集成
│   └── external/            # 外部服务
├── application/             # 应用层
│   ├── service/             # 应用服务
│   ├── dto/                 # 数据传输对象
│   └── command/             # 命令处理
├── interface/               # 接口层
│   ├── controller/          # 控制器
│   ├── middleware/          # 中间件
│   └── router/              # 路由
└── shared/                  # 共享层
    ├── config/              # 配置
    ├── utils/               # 工具类
    └── constants/           # 常量
```

#### 2.1.2 核心实体设计

**工单实体 (Ticket Entity)**
```go
// domain/ticket/entity/ticket.go
package entity

import (
    "time"
    "itsm-backend/shared/valueobject"
)

type Ticket struct {
    ID          string
    Title       string
    Description string
    Priority    Priority
    Status      Status
    Category    Category
    Requester   UserID
    Assignee    *UserID
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DueDate     *time.Time
    
    // 领域行为
    events []DomainEvent
}

// 领域方法
func (t *Ticket) Assign(assigneeID UserID) error {
    if t.Status == StatusClosed {
        return ErrCannotAssignClosedTicket
    }
    
    t.Assignee = &assigneeID
    t.UpdatedAt = time.Now()
    
    // 发布领域事件
    t.AddEvent(NewTicketAssignedEvent(t.ID, assigneeID))
    return nil
}

func (t *Ticket) Escalate(reason string) error {
    if t.Priority == PriorityUrgent {
        return ErrAlreadyUrgentPriority
    }
    
    t.Priority = t.Priority.Escalate()
    t.UpdatedAt = time.Now()
    
    t.AddEvent(NewTicketEscalatedEvent(t.ID, t.Priority, reason))
    return nil
}
```

### 2.2 服务层架构

#### 2.2.1 应用服务
```go
// application/service/ticket_service.go
package service

type TicketApplicationService struct {
    ticketRepo     domain.TicketRepository
    userRepo       domain.UserRepository
    eventPublisher EventPublisher
    aiService      AIService
}

func (s *TicketApplicationService) CreateTicket(cmd CreateTicketCommand) (*TicketDTO, error) {
    // 1. 验证输入
    if err := cmd.Validate(); err != nil {
        return nil, err
    }
    
    // 2. 创建领域对象
    ticket := domain.NewTicket(
        cmd.Title,
        cmd.Description,
        cmd.Priority,
        cmd.RequesterID,
    )
    
    // 3. AI智能分类
    category, err := s.aiService.ClassifyTicket(ticket)
    if err == nil {
        ticket.SetCategory(category)
    }
    
    // 4. 智能分配
    assignee, err := s.aiService.SuggestAssignee(ticket)
    if err == nil {
        ticket.Assign(assignee)
    }
    
    // 5. 保存到仓储
    if err := s.ticketRepo.Save(ticket); err != nil {
        return nil, err
    }
    
    // 6. 发布事件
    s.eventPublisher.PublishEvents(ticket.GetEvents())
    
    return s.toDTO(ticket), nil
}
```

#### 2.2.2 领域服务
```go
// domain/ticket/service/assignment_service.go
package service

type TicketAssignmentService struct {
    userRepo   UserRepository
    aiService  AIAnalysisService
}

func (s *TicketAssignmentService) SuggestBestAssignee(ticket *Ticket) (*UserID, error) {
    // 1. 获取可用的技术人员
    availableUsers, err := s.userRepo.FindAvailableTechnicians()
    if err != nil {
        return nil, err
    }
    
    // 2. AI分析最佳匹配
    suggestion, err := s.aiService.AnalyzeBestMatch(ticket, availableUsers)
    if err != nil {
        return nil, err
    }
    
    return &suggestion.UserID, nil
}
```

### 2.3 数据访问层设计

#### 2.3.1 仓储模式实现
```go
// infrastructure/database/ticket_repository.go
package database

type TicketRepository struct {
    client *ent.Client
    cache  cache.Cache
}

func (r *TicketRepository) Save(ticket *domain.Ticket) error {
    // 使用事务确保数据一致性
    return r.client.Tx(func(tx *ent.Tx) error {
        // 保存主实体
        entTicket, err := tx.Ticket.Create().
            SetTitle(ticket.Title).
            SetDescription(ticket.Description).
            SetPriority(string(ticket.Priority)).
            SetStatus(string(ticket.Status)).
            Save(context.Background())
        if err != nil {
            return err
        }
        
        // 保存关联实体
        for _, comment := range ticket.Comments {
            _, err := tx.TicketComment.Create().
                SetTicketID(entTicket.ID).
                SetContent(comment.Content).
                SetAuthorID(comment.AuthorID).
                Save(context.Background())
            if err != nil {
                return err
            }
        }
        
        // 更新缓存
        r.cache.Set(fmt.Sprintf("ticket:%s", ticket.ID), ticket, time.Hour)
        
        return nil
    })
}

func (r *TicketRepository) FindByID(id string) (*domain.Ticket, error) {
    // 先查缓存
    if cached, found := r.cache.Get(fmt.Sprintf("ticket:%s", id)); found {
        return cached.(*domain.Ticket), nil
    }
    
    // 查数据库
    entTicket, err := r.client.Ticket.
        Query().
        Where(ticket.IDEQ(id)).
        WithComments().
        WithAssignee().
        Only(context.Background())
    if err != nil {
        return nil, err
    }
    
    // 转换为领域对象
    domainTicket := r.toDomain(entTicket)
    
    // 更新缓存
    r.cache.Set(fmt.Sprintf("ticket:%s", id), domainTicket, time.Hour)
    
    return domainTicket, nil
}
```

## 3. 前端架构设计

### 3.1 组件架构

#### 3.1.1 目录结构优化
```
src/
├── app/                      # Next.js 13+ App Router
│   ├── (auth)/              # 认证路由组
│   │   ├── login/
│   │   └── register/
│   ├── (dashboard)/         # 仪表板路由组
│   │   ├── tickets/
│   │   ├── incidents/
│   │   └── reports/
│   └── api/                 # API路由
├── components/              # 可复用组件
│   ├── ui/                  # 基础UI组件
│   │   ├── Button/
│   │   ├── Input/
│   │   └── Modal/
│   ├── business/            # 业务组件
│   │   ├── TicketCard/
│   │   ├── IncidentForm/
│   │   └── AIAssistant/
│   └── layout/              # 布局组件
│       ├── Header/
│       ├── Sidebar/
│       └── Footer/
├── hooks/                   # 自定义Hooks
│   ├── useTickets.ts
│   ├── useAuth.ts
│   └── useAI.ts
├── lib/                     # 工具库
│   ├── api/                 # API客户端
│   ├── utils/               # 工具函数
│   └── store/               # 状态管理
├── types/                   # TypeScript类型定义
└── styles/                  # 样式文件
```

#### 3.1.2 状态管理架构
```typescript
// lib/store/ticket-store.ts
import { create } from 'zustand'
import { devtools, persist } from 'zustand/middleware'

interface TicketState {
  tickets: Ticket[]
  selectedTicket: Ticket | null
  filters: TicketFilters
  loading: boolean
  error: string | null
}

interface TicketActions {
  fetchTickets: (filters?: TicketFilters) => Promise<void>
  createTicket: (data: CreateTicketData) => Promise<void>
  updateTicket: (id: string, data: UpdateTicketData) => Promise<void>
  selectTicket: (ticket: Ticket | null) => void
  setFilters: (filters: TicketFilters) => void
}

export const useTicketStore = create<TicketState & TicketActions>()(
  devtools(
    persist(
      (set, get) => ({
        // State
        tickets: [],
        selectedTicket: null,
        filters: {},
        loading: false,
        error: null,

        // Actions
        fetchTickets: async (filters) => {
          set({ loading: true, error: null })
          try {
            const tickets = await ticketAPI.getTickets(filters)
            set({ tickets, loading: false })
          } catch (error) {
            set({ error: error.message, loading: false })
          }
        },

        createTicket: async (data) => {
          set({ loading: true })
          try {
            const newTicket = await ticketAPI.createTicket(data)
            set(state => ({
              tickets: [newTicket, ...state.tickets],
              loading: false
            }))
          } catch (error) {
            set({ error: error.message, loading: false })
          }
        },

        selectTicket: (ticket) => set({ selectedTicket: ticket }),
        setFilters: (filters) => set({ filters })
      }),
      {
        name: 'ticket-store',
        partialize: (state) => ({ filters: state.filters })
      }
    )
  )
)
```

#### 3.1.3 组件设计模式
```typescript
// components/business/TicketCard/TicketCard.tsx
import React from 'react'
import { Card, Tag, Avatar, Tooltip } from 'antd'
import { Ticket, TicketPriority, TicketStatus } from '@/types/ticket'
import { formatDistanceToNow } from 'date-fns'
import { zhCN } from 'date-fns/locale'

interface TicketCardProps {
  ticket: Ticket
  onClick?: (ticket: Ticket) => void
  onAssign?: (ticketId: string, assigneeId: string) => void
  className?: string
}

export const TicketCard: React.FC<TicketCardProps> = ({
  ticket,
  onClick,
  onAssign,
  className
}) => {
  const getPriorityColor = (priority: TicketPriority) => {
    const colors = {
      low: 'green',
      medium: 'orange',
      high: 'red',
      urgent: 'purple'
    }
    return colors[priority] || 'default'
  }

  const getStatusColor = (status: TicketStatus) => {
    const colors = {
      open: 'blue',
      in_progress: 'orange',
      resolved: 'green',
      closed: 'gray'
    }
    return colors[status] || 'default'
  }

  return (
    <Card
      className={`ticket-card ${className}`}
      hoverable
      onClick={() => onClick?.(ticket)}
      actions={[
        <Tooltip title="查看详情" key="view">
          <EyeOutlined />
        </Tooltip>,
        <Tooltip title="编辑" key="edit">
          <EditOutlined />
        </Tooltip>,
        <Tooltip title="分配" key="assign">
          <UserAddOutlined />
        </Tooltip>
      ]}
    >
      <Card.Meta
        title={
          <div className="flex justify-between items-start">
            <span className="text-sm font-medium truncate">
              #{ticket.id} {ticket.title}
            </span>
            <div className="flex gap-1 ml-2">
              <Tag color={getPriorityColor(ticket.priority)} size="small">
                {ticket.priority}
              </Tag>
              <Tag color={getStatusColor(ticket.status)} size="small">
                {ticket.status}
              </Tag>
            </div>
          </div>
        }
        description={
          <div className="space-y-2">
            <p className="text-xs text-gray-600 line-clamp-2">
              {ticket.description}
            </p>
            <div className="flex justify-between items-center text-xs">
              <div className="flex items-center gap-2">
                {ticket.assignee && (
                  <div className="flex items-center gap-1">
                    <Avatar size="small" src={ticket.assignee.avatar} />
                    <span>{ticket.assignee.name}</span>
                  </div>
                )}
              </div>
              <span className="text-gray-500">
                {formatDistanceToNow(new Date(ticket.createdAt), {
                  addSuffix: true,
                  locale: zhCN
                })}
              </span>
            </div>
          </div>
        }
      />
    </Card>
  )
}
```

### 3.2 API客户端设计

#### 3.2.1 HTTP客户端封装
```typescript
// lib/api/http-client.ts
import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios'
import { message } from 'antd'

class HTTPClient {
  private instance: AxiosInstance

  constructor(baseURL: string) {
    this.instance = axios.create({
      baseURL,
      timeout: 10000,
      headers: {
        'Content-Type': 'application/json'
      }
    })

    this.setupInterceptors()
  }

  private setupInterceptors() {
    // 请求拦截器
    this.instance.interceptors.request.use(
      (config) => {
        const token = localStorage.getItem('access_token')
        if (token) {
          config.headers.Authorization = `Bearer ${token}`
        }
        
        // 添加租户信息
        const tenantId = localStorage.getItem('tenant_id')
        if (tenantId) {
          config.headers['X-Tenant-ID'] = tenantId
        }

        return config
      },
      (error) => Promise.reject(error)
    )

    // 响应拦截器
    this.instance.interceptors.response.use(
      (response: AxiosResponse) => {
        return response.data
      },
      (error) => {
        if (error.response?.status === 401) {
          // Token过期，跳转到登录页
          localStorage.removeItem('access_token')
          window.location.href = '/login'
        } else if (error.response?.status >= 500) {
          message.error('服务器错误，请稍后重试')
        } else if (error.response?.data?.message) {
          message.error(error.response.data.message)
        }
        
        return Promise.reject(error)
      }
    )
  }

  async get<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
    return this.instance.get(url, config)
  }

  async post<T>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
    return this.instance.post(url, data, config)
  }

  async put<T>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
    return this.instance.put(url, data, config)
  }

  async delete<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
    return this.instance.delete(url, config)
  }
}

export const httpClient = new HTTPClient(process.env.NEXT_PUBLIC_API_BASE_URL!)
```

## 4. AI服务集成架构

### 4.1 AI服务抽象层
```go
// domain/ai/service/ai_service.go
package service

type AIService interface {
    ClassifyTicket(ticket *Ticket) (Category, error)
    SuggestAssignee(ticket *Ticket) (*UserID, error)
    PredictResolutionTime(ticket *Ticket) (time.Duration, error)
    GenerateSolution(ticket *Ticket) (*Solution, error)
    AnalyzeSentiment(text string) (*SentimentAnalysis, error)
}

type OpenAIService struct {
    client   *openai.Client
    embedder EmbeddingService
    vector   VectorStore
}

func (s *OpenAIService) ClassifyTicket(ticket *Ticket) (Category, error) {
    prompt := fmt.Sprintf(`
    请分析以下工单并分类：
    标题：%s
    描述：%s
    
    请从以下类别中选择最合适的：
    - 硬件问题
    - 软件问题  
    - 网络问题
    - 权限问题
    - 其他
    
    只返回类别名称。
    `, ticket.Title, ticket.Description)
    
    response, err := s.client.CreateChatCompletion(
        context.Background(),
        openai.ChatCompletionRequest{
            Model: openai.GPT3Dot5Turbo,
            Messages: []openai.ChatCompletionMessage{
                {
                    Role:    openai.ChatMessageRoleUser,
                    Content: prompt,
                },
            },
        },
    )
    
    if err != nil {
        return "", err
    }
    
    category := strings.TrimSpace(response.Choices[0].Message.Content)
    return Category(category), nil
}
```

### 4.2 向量数据库集成
```go
// infrastructure/ai/vector_store.go
package ai

type VectorStore struct {
    db *sql.DB
}

func (vs *VectorStore) StoreEmbedding(id string, embedding []float32, metadata map[string]interface{}) error {
    query := `
        INSERT INTO embeddings (id, vector, metadata, created_at)
        VALUES ($1, $2, $3, NOW())
        ON CONFLICT (id) DO UPDATE SET
            vector = EXCLUDED.vector,
            metadata = EXCLUDED.metadata,
            updated_at = NOW()
    `
    
    metadataJSON, _ := json.Marshal(metadata)
    _, err := vs.db.Exec(query, id, pq.Array(embedding), metadataJSON)
    return err
}

func (vs *VectorStore) SearchSimilar(queryVector []float32, limit int) ([]SimilarityResult, error) {
    query := `
        SELECT id, metadata, 1 - (vector <=> $1) as similarity
        FROM embeddings
        ORDER BY vector <=> $1
        LIMIT $2
    `
    
    rows, err := vs.db.Query(query, pq.Array(queryVector), limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var results []SimilarityResult
    for rows.Next() {
        var result SimilarityResult
        var metadataJSON []byte
        
        err := rows.Scan(&result.ID, &metadataJSON, &result.Similarity)
        if err != nil {
            continue
        }
        
        json.Unmarshal(metadataJSON, &result.Metadata)
        results = append(results, result)
    }
    
    return results, nil
}
```

## 5. 数据库设计优化

### 5.1 核心表结构
```sql
-- 工单表
CREATE TABLE tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    priority ticket_priority NOT NULL DEFAULT 'medium',
    status ticket_status NOT NULL DEFAULT 'open',
    category_id UUID REFERENCES ticket_categories(id),
    requester_id UUID NOT NULL REFERENCES users(id),
    assignee_id UUID REFERENCES users(id),
    due_date TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    tenant_id UUID NOT NULL REFERENCES tenants(id)
);

-- 创建索引
CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_assignee ON tickets(assignee_id);
CREATE INDEX idx_tickets_created_at ON tickets(created_at);
CREATE INDEX idx_tickets_tenant ON tickets(tenant_id);
CREATE INDEX idx_tickets_priority_status ON tickets(priority, status);

-- 向量嵌入表
CREATE TABLE embeddings (
    id VARCHAR(255) PRIMARY KEY,
    vector vector(1536),  -- OpenAI embedding维度
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 创建向量索引
CREATE INDEX ON embeddings USING ivfflat (vector vector_cosine_ops);
```

### 5.2 分区策略
```sql
-- 按时间分区的工单表
CREATE TABLE tickets_partitioned (
    LIKE tickets INCLUDING ALL
) PARTITION BY RANGE (created_at);

-- 创建月度分区
CREATE TABLE tickets_2024_01 PARTITION OF tickets_partitioned
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE tickets_2024_02 PARTITION OF tickets_partitioned
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
```

## 6. 性能优化方案

### 6.1 缓存策略
```go
// infrastructure/cache/redis_cache.go
package cache

type RedisCache struct {
    client *redis.Client
}

func (c *RedisCache) GetTickets(filters TicketFilters) ([]Ticket, error) {
    key := fmt.Sprintf("tickets:%s", filters.Hash())
    
    // 尝试从缓存获取
    cached, err := c.client.Get(context.Background(), key).Result()
    if err == nil {
        var tickets []Ticket
        json.Unmarshal([]byte(cached), &tickets)
        return tickets, nil
    }
    
    // 缓存未命中，从数据库查询
    tickets, err := c.ticketRepo.FindByFilters(filters)
    if err != nil {
        return nil, err
    }
    
    // 写入缓存
    data, _ := json.Marshal(tickets)
    c.client.Set(context.Background(), key, data, 5*time.Minute)
    
    return tickets, nil
}
```

### 6.2 数据库连接池优化
```go
// database/connection.go
func InitDatabase(cfg *DatabaseConfig) (*ent.Client, error) {
    db, err := sql.Open("postgres", cfg.DSN)
    if err != nil {
        return nil, err
    }
    
    // 连接池配置
    db.SetMaxOpenConns(25)                 // 最大连接数
    db.SetMaxIdleConns(5)                  // 最大空闲连接数
    db.SetConnMaxLifetime(5 * time.Minute) // 连接最大生命周期
    db.SetConnMaxIdleTime(1 * time.Minute) // 空闲连接超时
    
    // 创建Ent客户端
    client := ent.NewClient(ent.Driver(sql.OpenDB("postgres", db)))
    
    return client, nil
}
```

## 7. 安全框架设计

### 7.1 JWT认证中间件
```go
// middleware/auth_middleware.go
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := extractToken(c)
        if token == "" {
            c.JSON(401, gin.H{"error": "未提供认证令牌"})
            c.Abort()
            return
        }
        
        claims, err := validateJWT(token, jwtSecret)
        if err != nil {
            c.JSON(401, gin.H{"error": "无效的认证令牌"})
            c.Abort()
            return
        }
        
        // 设置用户上下文
        c.Set("user_id", claims.UserID)
        c.Set("tenant_id", claims.TenantID)
        c.Set("roles", claims.Roles)
        
        c.Next()
    }
}
```

### 7.2 权限控制
```go
// middleware/rbac_middleware.go
func RBACMiddleware(requiredPermission string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userRoles, exists := c.Get("roles")
        if !exists {
            c.JSON(403, gin.H{"error": "权限不足"})
            c.Abort()
            return
        }
        
        hasPermission := checkPermission(userRoles.([]string), requiredPermission)
        if !hasPermission {
            c.JSON(403, gin.H{"error": "权限不足"})
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

## 8. 监控和日志

### 8.1 结构化日志
```go
// middleware/logger_middleware.go
func LoggerMiddleware() gin.HandlerFunc {
    return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
        log := map[string]interface{}{
            "timestamp":   param.TimeStamp.Format(time.RFC3339),
            "method":      param.Method,
            "path":        param.Path,
            "status":      param.StatusCode,
            "latency":     param.Latency.String(),
            "client_ip":   param.ClientIP,
            "user_agent":  param.Request.UserAgent(),
            "request_id":  param.Request.Header.Get("X-Request-ID"),
        }
        
        logJSON, _ := json.Marshal(log)
        return string(logJSON) + "\n"
    })
}
```

### 8.2 性能监控
```go
// middleware/metrics_middleware.go
var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )
    
    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "http_request_duration_seconds",
            Help: "HTTP request duration in seconds",
        },
        []string{"method", "endpoint"},
    )
)

func MetricsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        c.Next()
        
        duration := time.Since(start).Seconds()
        status := strconv.Itoa(c.Writer.Status())
        
        httpRequestsTotal.WithLabelValues(c.Request.Method, c.FullPath(), status).Inc()
        httpRequestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(duration)
    }
}
```

这个架构设计文档提供了完整的ITSM系统代码架构方案，包括后端的领域驱动设计、前端的组件化架构、AI服务集成、数据库优化、性能优化和安全框架等核心内容。接下来我将继续完善具体的实施计划。