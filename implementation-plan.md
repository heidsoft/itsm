# ITSM系统代码实施计划

## 1. 实施概览

### 1.1 实施策略
- **分阶段交付**: 采用敏捷开发模式，每2周一个迭代
- **MVP优先**: 先实现核心功能，再逐步完善
- **并行开发**: 前后端同步开发，API优先设计
- **持续集成**: 每日构建，自动化测试

### 1.2 团队配置
```
项目团队 (8人)
├── 技术负责人 (1人) - 架构设计、技术决策
├── 后端开发 (3人) - Go开发、API设计
├── 前端开发 (2人) - Next.js开发、UI实现  
├── AI工程师 (1人) - AI服务集成、算法优化
└── 测试工程师 (1人) - 测试用例、质量保证
```

## 2. 第一阶段：基础架构搭建 (第1-2周)

### 2.1 后端基础架构

#### 2.1.1 项目结构重构
```bash
# 创建新的项目结构
mkdir -p itsm-backend/{domain,application,infrastructure,interface,shared}
mkdir -p itsm-backend/domain/{ticket,incident,user,ai}
mkdir -p itsm-backend/infrastructure/{database,cache,ai,external}
```

#### 2.1.2 核心代码实现

**配置管理优化**
```go
// shared/config/config.go
package config

type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Database DatabaseConfig `yaml:"database"`
    Redis    RedisConfig    `yaml:"redis"`
    AI       AIConfig       `yaml:"ai"`
    Security SecurityConfig `yaml:"security"`
}

type AIConfig struct {
    Provider string `yaml:"provider"` // openai, azure, local
    APIKey   string `yaml:"api_key"`
    Endpoint string `yaml:"endpoint"`
    Model    string `yaml:"model"`
}

type SecurityConfig struct {
    JWTSecret     string        `yaml:"jwt_secret"`
    TokenExpiry   time.Duration `yaml:"token_expiry"`
    RefreshExpiry time.Duration `yaml:"refresh_expiry"`
    RateLimit     RateLimit     `yaml:"rate_limit"`
}

type RateLimit struct {
    RequestsPerMinute int `yaml:"requests_per_minute"`
    BurstSize         int `yaml:"burst_size"`
}
```

**领域实体重构**
```go
// domain/ticket/entity/ticket.go
package entity

import (
    "time"
    "errors"
    "itsm-backend/shared/valueobject"
)

var (
    ErrInvalidTicketStatus = errors.New("invalid ticket status")
    ErrCannotAssignClosedTicket = errors.New("cannot assign closed ticket")
)

type Ticket struct {
    id          TicketID
    title       string
    description string
    priority    Priority
    status      Status
    category    *Category
    requester   UserID
    assignee    *UserID
    createdAt   time.Time
    updatedAt   time.Time
    dueDate     *time.Time
    
    // 领域事件
    events []DomainEvent
}

// 工厂方法
func NewTicket(title, description string, priority Priority, requester UserID) *Ticket {
    return &Ticket{
        id:          NewTicketID(),
        title:       title,
        description: description,
        priority:    priority,
        status:      StatusOpen,
        requester:   requester,
        createdAt:   time.Now(),
        updatedAt:   time.Now(),
        events:      []DomainEvent{},
    }
}

// 领域行为
func (t *Ticket) Assign(assigneeID UserID) error {
    if t.status == StatusClosed {
        return ErrCannotAssignClosedTicket
    }
    
    oldAssignee := t.assignee
    t.assignee = &assigneeID
    t.updatedAt = time.Now()
    
    // 发布领域事件
    t.addEvent(NewTicketAssignedEvent(t.id, oldAssignee, &assigneeID))
    return nil
}

func (t *Ticket) UpdateStatus(newStatus Status) error {
    if !t.canTransitionTo(newStatus) {
        return ErrInvalidTicketStatus
    }
    
    oldStatus := t.status
    t.status = newStatus
    t.updatedAt = time.Now()
    
    t.addEvent(NewTicketStatusChangedEvent(t.id, oldStatus, newStatus))
    return nil
}

func (t *Ticket) canTransitionTo(newStatus Status) bool {
    validTransitions := map[Status][]Status{
        StatusOpen:       {StatusInProgress, StatusClosed},
        StatusInProgress: {StatusResolved, StatusOpen, StatusClosed},
        StatusResolved:   {StatusClosed, StatusOpen},
        StatusClosed:     {StatusOpen},
    }
    
    allowed, exists := validTransitions[t.status]
    if !exists {
        return false
    }
    
    for _, status := range allowed {
        if status == newStatus {
            return true
        }
    }
    return false
}
```

#### 2.1.3 数据库迁移脚本
```sql
-- migrations/001_create_base_tables.sql

-- 用户表
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255),
    avatar_url VARCHAR(500),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    tenant_id UUID NOT NULL
);

-- 工单表
CREATE TABLE tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    priority VARCHAR(20) NOT NULL DEFAULT 'medium',
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    category_id UUID,
    requester_id UUID NOT NULL REFERENCES users(id),
    assignee_id UUID REFERENCES users(id),
    due_date TIMESTAMP,
    resolution TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    tenant_id UUID NOT NULL,
    
    CONSTRAINT chk_priority CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    CONSTRAINT chk_status CHECK (status IN ('open', 'in_progress', 'resolved', 'closed'))
);

-- 工单评论表
CREATE TABLE ticket_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id),
    content TEXT NOT NULL,
    is_internal BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    tenant_id UUID NOT NULL
);

-- 工单附件表
CREATE TABLE ticket_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100),
    uploaded_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    tenant_id UUID NOT NULL
);

-- 创建索引
CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_assignee ON tickets(assignee_id);
CREATE INDEX idx_tickets_requester ON tickets(requester_id);
CREATE INDEX idx_tickets_created_at ON tickets(created_at);
CREATE INDEX idx_tickets_tenant ON tickets(tenant_id);
CREATE INDEX idx_tickets_priority_status ON tickets(priority, status);

CREATE INDEX idx_ticket_comments_ticket ON ticket_comments(ticket_id);
CREATE INDEX idx_ticket_comments_created_at ON ticket_comments(created_at);

CREATE INDEX idx_ticket_attachments_ticket ON ticket_attachments(ticket_id);
```

### 2.2 前端基础架构

#### 2.2.1 项目配置优化
```typescript
// next.config.ts
import type { NextConfig } from 'next'

const nextConfig: NextConfig = {
  experimental: {
    appDir: true,
  },
  env: {
    NEXT_PUBLIC_API_BASE_URL: process.env.NEXT_PUBLIC_API_BASE_URL,
    NEXT_PUBLIC_WS_URL: process.env.NEXT_PUBLIC_WS_URL,
  },
  images: {
    domains: ['localhost', 'api.example.com'],
  },
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: `${process.env.NEXT_PUBLIC_API_BASE_URL}/api/v1/:path*`,
      },
    ]
  },
}

export default nextConfig
```

#### 2.2.2 类型定义
```typescript
// types/ticket.ts
export interface Ticket {
  id: string
  title: string
  description: string
  priority: TicketPriority
  status: TicketStatus
  category?: TicketCategory
  requester: User
  assignee?: User
  dueDate?: string
  resolution?: string
  createdAt: string
  updatedAt: string
  comments?: TicketComment[]
  attachments?: TicketAttachment[]
}

export type TicketPriority = 'low' | 'medium' | 'high' | 'urgent'
export type TicketStatus = 'open' | 'in_progress' | 'resolved' | 'closed'

export interface TicketFilters {
  status?: TicketStatus[]
  priority?: TicketPriority[]
  assigneeId?: string
  requesterId?: string
  categoryId?: string
  dateRange?: {
    start: string
    end: string
  }
  search?: string
}

export interface CreateTicketData {
  title: string
  description: string
  priority: TicketPriority
  categoryId?: string
  dueDate?: string
}
```

#### 2.2.3 API客户端实现
```typescript
// lib/api/ticket-api.ts
import { httpClient } from './http-client'
import type { Ticket, TicketFilters, CreateTicketData } from '@/types/ticket'

export class TicketAPI {
  async getTickets(filters?: TicketFilters): Promise<{
    data: Ticket[]
    total: number
    page: number
    pageSize: number
  }> {
    const params = new URLSearchParams()
    
    if (filters?.status?.length) {
      params.append('status', filters.status.join(','))
    }
    if (filters?.priority?.length) {
      params.append('priority', filters.priority.join(','))
    }
    if (filters?.assigneeId) {
      params.append('assignee_id', filters.assigneeId)
    }
    if (filters?.search) {
      params.append('search', filters.search)
    }
    
    return httpClient.get(`/tickets?${params.toString()}`)
  }

  async getTicket(id: string): Promise<Ticket> {
    return httpClient.get(`/tickets/${id}`)
  }

  async createTicket(data: CreateTicketData): Promise<Ticket> {
    return httpClient.post('/tickets', data)
  }

  async updateTicket(id: string, data: Partial<CreateTicketData>): Promise<Ticket> {
    return httpClient.put(`/tickets/${id}`, data)
  }

  async assignTicket(id: string, assigneeId: string): Promise<void> {
    return httpClient.post(`/tickets/${id}/assign`, { assignee_id: assigneeId })
  }

  async updateStatus(id: string, status: TicketStatus): Promise<void> {
    return httpClient.post(`/tickets/${id}/status`, { status })
  }

  async addComment(id: string, content: string, isInternal = false): Promise<void> {
    return httpClient.post(`/tickets/${id}/comments`, { content, is_internal: isInternal })
  }
}

export const ticketAPI = new TicketAPI()
```

## 3. 第二阶段：核心功能开发 (第3-6周)

### 3.1 工单管理系统

#### 3.1.1 后端服务实现
```go
// application/service/ticket_service.go
package service

import (
    "context"
    "itsm-backend/domain/ticket/entity"
    "itsm-backend/domain/ticket/repository"
    "itsm-backend/shared/event"
)

type TicketApplicationService struct {
    ticketRepo     repository.TicketRepository
    userRepo       repository.UserRepository
    eventPublisher event.Publisher
    aiService      AIService
    logger         Logger
}

func NewTicketApplicationService(
    ticketRepo repository.TicketRepository,
    userRepo repository.UserRepository,
    eventPublisher event.Publisher,
    aiService AIService,
    logger Logger,
) *TicketApplicationService {
    return &TicketApplicationService{
        ticketRepo:     ticketRepo,
        userRepo:       userRepo,
        eventPublisher: eventPublisher,
        aiService:      aiService,
        logger:         logger,
    }
}

func (s *TicketApplicationService) CreateTicket(ctx context.Context, cmd CreateTicketCommand) (*TicketDTO, error) {
    // 1. 验证输入
    if err := cmd.Validate(); err != nil {
        return nil, NewValidationError("invalid input", err)
    }
    
    // 2. 验证用户权限
    requester, err := s.userRepo.FindByID(ctx, cmd.RequesterID)
    if err != nil {
        return nil, err
    }
    
    // 3. 创建领域对象
    ticket := entity.NewTicket(
        cmd.Title,
        cmd.Description,
        cmd.Priority,
        cmd.RequesterID,
    )
    
    // 4. AI智能分类
    if category, err := s.aiService.ClassifyTicket(ctx, ticket); err == nil {
        ticket.SetCategory(category)
    }
    
    // 5. 智能分配
    if assignee, err := s.aiService.SuggestAssignee(ctx, ticket); err == nil {
        ticket.Assign(assignee)
    }
    
    // 6. 保存到仓储
    if err := s.ticketRepo.Save(ctx, ticket); err != nil {
        s.logger.Error("failed to save ticket", "error", err)
        return nil, err
    }
    
    // 7. 发布事件
    for _, event := range ticket.GetEvents() {
        s.eventPublisher.Publish(ctx, event)
    }
    
    s.logger.Info("ticket created", "ticket_id", ticket.ID(), "requester", cmd.RequesterID)
    
    return s.toDTO(ticket), nil
}

func (s *TicketApplicationService) AssignTicket(ctx context.Context, ticketID string, assigneeID string) error {
    // 1. 获取工单
    ticket, err := s.ticketRepo.FindByID(ctx, ticketID)
    if err != nil {
        return err
    }
    
    // 2. 验证分配人员
    assignee, err := s.userRepo.FindByID(ctx, assigneeID)
    if err != nil {
        return err
    }
    
    // 3. 执行分配
    if err := ticket.Assign(assigneeID); err != nil {
        return err
    }
    
    // 4. 保存更改
    if err := s.ticketRepo.Save(ctx, ticket); err != nil {
        return err
    }
    
    // 5. 发布事件
    for _, event := range ticket.GetEvents() {
        s.eventPublisher.Publish(ctx, event)
    }
    
    return nil
}
```

#### 3.1.2 前端组件实现
```typescript
// components/tickets/TicketList.tsx
'use client'

import React, { useState, useEffect } from 'react'
import { Table, Tag, Button, Space, Input, Select, DatePicker } from 'antd'
import { SearchOutlined, PlusOutlined, FilterOutlined } from '@ant-design/icons'
import { useTicketStore } from '@/lib/store/ticket-store'
import { TicketCard } from './TicketCard'
import { CreateTicketModal } from './CreateTicketModal'
import type { Ticket, TicketFilters } from '@/types/ticket'

const { Search } = Input
const { RangePicker } = DatePicker

export const TicketList: React.FC = () => {
  const {
    tickets,
    loading,
    filters,
    fetchTickets,
    setFilters,
  } = useTicketStore()
  
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [viewMode, setViewMode] = useState<'table' | 'card'>('table')

  useEffect(() => {
    fetchTickets(filters)
  }, [filters, fetchTickets])

  const handleFilterChange = (key: keyof TicketFilters, value: any) => {
    setFilters({ ...filters, [key]: value })
  }

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 100,
      render: (id: string) => (
        <span className="font-mono text-xs">#{id.slice(-8)}</span>
      ),
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      ellipsis: true,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      render: (priority: string) => {
        const colors = {
          low: 'green',
          medium: 'orange',
          high: 'red',
          urgent: 'purple',
        }
        return <Tag color={colors[priority]}>{priority}</Tag>
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: string) => {
        const colors = {
          open: 'blue',
          in_progress: 'orange',
          resolved: 'green',
          closed: 'gray',
        }
        return <Tag color={colors[status]}>{status}</Tag>
      },
    },
    {
      title: '分配人',
      dataIndex: 'assignee',
      key: 'assignee',
      width: 120,
      render: (assignee: any) => assignee?.name || '未分配',
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 150,
      render: (date: string) => new Date(date).toLocaleDateString(),
    },
    {
      title: '操作',
      key: 'actions',
      width: 150,
      render: (_, record: Ticket) => (
        <Space>
          <Button size="small" type="link">
            查看
          </Button>
          <Button size="small" type="link">
            编辑
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      {/* 工具栏 */}
      <div className="flex justify-between items-center">
        <div className="flex space-x-4">
          <Search
            placeholder="搜索工单..."
            allowClear
            style={{ width: 300 }}
            onSearch={(value) => handleFilterChange('search', value)}
          />
          
          <Select
            mode="multiple"
            placeholder="状态筛选"
            style={{ width: 200 }}
            onChange={(value) => handleFilterChange('status', value)}
            options={[
              { label: '开放', value: 'open' },
              { label: '处理中', value: 'in_progress' },
              { label: '已解决', value: 'resolved' },
              { label: '已关闭', value: 'closed' },
            ]}
          />
          
          <Select
            mode="multiple"
            placeholder="优先级筛选"
            style={{ width: 200 }}
            onChange={(value) => handleFilterChange('priority', value)}
            options={[
              { label: '低', value: 'low' },
              { label: '中', value: 'medium' },
              { label: '高', value: 'high' },
              { label: '紧急', value: 'urgent' },
            ]}
          />
        </div>
        
        <Space>
          <Button
            icon={<FilterOutlined />}
            onClick={() => setViewMode(viewMode === 'table' ? 'card' : 'table')}
          >
            {viewMode === 'table' ? '卡片视图' : '表格视图'}
          </Button>
          
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setShowCreateModal(true)}
          >
            创建工单
          </Button>
        </Space>
      </div>

      {/* 内容区域 */}
      {viewMode === 'table' ? (
        <Table
          columns={columns}
          dataSource={tickets}
          loading={loading}
          rowKey="id"
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条记录`,
          }}
        />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {tickets.map((ticket) => (
            <TicketCard key={ticket.id} ticket={ticket} />
          ))}
        </div>
      )}

      {/* 创建工单模态框 */}
      <CreateTicketModal
        open={showCreateModal}
        onCancel={() => setShowCreateModal(false)}
        onSuccess={() => {
          setShowCreateModal(false)
          fetchTickets(filters)
        }}
      />
    </div>
  )
}
```

### 3.2 AI服务集成

#### 3.2.1 AI服务实现
```go
// infrastructure/ai/openai_service.go
package ai

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    
    "github.com/sashabaranov/go-openai"
)

type OpenAIService struct {
    client      *openai.Client
    vectorStore VectorStore
    logger      Logger
}

func NewOpenAIService(apiKey string, vectorStore VectorStore, logger Logger) *OpenAIService {
    client := openai.NewClient(apiKey)
    return &OpenAIService{
        client:      client,
        vectorStore: vectorStore,
        logger:      logger,
    }
}

func (s *OpenAIService) ClassifyTicket(ctx context.Context, ticket *entity.Ticket) (*Category, error) {
    prompt := fmt.Sprintf(`
请分析以下工单并进行分类：

标题：%s
描述：%s

请从以下类别中选择最合适的一个：
1. 硬件问题 - 服务器、网络设备、打印机等硬件故障
2. 软件问题 - 应用程序错误、系统软件问题
3. 网络问题 - 网络连接、带宽、DNS等网络相关问题
4. 权限问题 - 账户权限、访问控制相关问题
5. 数据问题 - 数据丢失、数据同步、备份恢复等
6. 其他 - 不属于以上类别的问题

请只返回类别名称，不要包含其他内容。
`, ticket.Title(), ticket.Description())

    resp, err := s.client.CreateChatCompletion(
        ctx,
        openai.ChatCompletionRequest{
            Model: openai.GPT3Dot5Turbo,
            Messages: []openai.ChatCompletionMessage{
                {
                    Role:    openai.ChatMessageRoleUser,
                    Content: prompt,
                },
            },
            Temperature: 0.1,
            MaxTokens:   50,
        },
    )

    if err != nil {
        s.logger.Error("failed to classify ticket", "error", err)
        return nil, err
    }

    if len(resp.Choices) == 0 {
        return nil, fmt.Errorf("no classification result")
    }

    categoryName := strings.TrimSpace(resp.Choices[0].Message.Content)
    category := NewCategory(categoryName)
    
    s.logger.Info("ticket classified", 
        "ticket_id", ticket.ID(), 
        "category", categoryName)
    
    return category, nil
}

func (s *OpenAIService) SuggestAssignee(ctx context.Context, ticket *entity.Ticket) (*UserID, error) {
    // 1. 获取相似的历史工单
    embedding, err := s.createEmbedding(ctx, ticket.Title()+" "+ticket.Description())
    if err != nil {
        return nil, err
    }
    
    similarTickets, err := s.vectorStore.SearchSimilar(ctx, embedding, 5)
    if err != nil {
        return nil, err
    }
    
    // 2. 分析历史分配模式
    assigneeStats := make(map[string]int)
    for _, similar := range similarTickets {
        if assigneeID, ok := similar.Metadata["assignee_id"].(string); ok {
            assigneeStats[assigneeID]++
        }
    }
    
    // 3. 找到最佳匹配的分配人
    var bestAssignee string
    maxCount := 0
    for assigneeID, count := range assigneeStats {
        if count > maxCount {
            maxCount = count
            bestAssignee = assigneeID
        }
    }
    
    if bestAssignee == "" {
        return nil, fmt.Errorf("no suitable assignee found")
    }
    
    userID := NewUserID(bestAssignee)
    return &userID, nil
}

func (s *OpenAIService) createEmbedding(ctx context.Context, text string) ([]float32, error) {
    resp, err := s.client.CreateEmbeddings(
        ctx,
        openai.EmbeddingRequest{
            Input: []string{text},
            Model: openai.AdaEmbeddingV2,
        },
    )
    
    if err != nil {
        return nil, err
    }
    
    if len(resp.Data) == 0 {
        return nil, fmt.Errorf("no embedding generated")
    }
    
    return resp.Data[0].Embedding, nil
}
```

## 4. 第三阶段：高级功能开发 (第7-10周)

### 4.1 工作流引擎

#### 4.1.1 BPMN工作流实现
```go
// domain/workflow/engine/bpmn_engine.go
package engine

type BPMNEngine struct {
    processRepo ProcessRepository
    taskRepo    TaskRepository
    eventBus    EventBus
}

func (e *BPMNEngine) StartProcess(ctx context.Context, processKey string, variables map[string]interface{}) (*ProcessInstance, error) {
    // 1. 获取流程定义
    processDef, err := e.processRepo.FindByKey(ctx, processKey)
    if err != nil {
        return nil, err
    }
    
    // 2. 创建流程实例
    instance := NewProcessInstance(processDef.ID, variables)
    
    // 3. 启动流程
    startEvent := processDef.GetStartEvent()
    if err := e.executeElement(ctx, instance, startEvent); err != nil {
        return nil, err
    }
    
    return instance, nil
}

func (e *BPMNEngine) CompleteTask(ctx context.Context, taskID string, variables map[string]interface{}) error {
    // 1. 获取任务
    task, err := e.taskRepo.FindByID(ctx, taskID)
    if err != nil {
        return err
    }
    
    // 2. 完成任务
    task.Complete(variables)
    
    // 3. 继续流程执行
    nextElements := task.GetOutgoingElements()
    for _, element := range nextElements {
        if err := e.executeElement(ctx, task.ProcessInstance, element); err != nil {
            return err
        }
    }
    
    return nil
}
```

### 4.2 智能分析和报表

#### 4.2.1 预测性分析
```go
// application/service/analytics_service.go
package service

type AnalyticsService struct {
    ticketRepo repository.TicketRepository
    aiService  AIService
    cache      Cache
}

func (s *AnalyticsService) PredictTicketVolume(ctx context.Context, days int) (*VolumePredict, error) {
    // 1. 获取历史数据
    historicalData, err := s.ticketRepo.GetDailyTicketCounts(ctx, 90) // 90天历史数据
    if err != nil {
        return nil, err
    }
    
    // 2. 使用时间序列分析预测
    prediction, err := s.aiService.PredictTimeSeries(ctx, historicalData, days)
    if err != nil {
        return nil, err
    }
    
    // 3. 缓存结果
    cacheKey := fmt.Sprintf("volume_predict_%d", days)
    s.cache.Set(ctx, cacheKey, prediction, time.Hour)
    
    return prediction, nil
}

func (s *AnalyticsService) AnalyzeResolutionTrends(ctx context.Context) (*ResolutionTrends, error) {
    // 1. 获取解决时间数据
    resolutionData, err := s.ticketRepo.GetResolutionTimeStats(ctx, 30)
    if err != nil {
        return nil, err
    }
    
    // 2. 计算趋势指标
    trends := &ResolutionTrends{
        AverageResolutionTime: calculateAverage(resolutionData),
        TrendDirection:        calculateTrend(resolutionData),
        CategoryBreakdown:     calculateCategoryBreakdown(resolutionData),
    }
    
    return trends, nil
}
```

#### 4.2.2 前端报表组件
```typescript
// components/analytics/PredictiveAnalytics.tsx
'use client'

import React, { useState, useEffect } from 'react'
import { Card, Row, Col, Select, DatePicker, Spin } from 'antd'
import { Line, Bar, Pie } from '@ant-design/plots'
import { analyticsAPI } from '@/lib/api/analytics-api'

export const PredictiveAnalytics: React.FC = () => {
  const [loading, setLoading] = useState(false)
  const [volumeData, setVolumeData] = useState([])
  const [resolutionTrends, setResolutionTrends] = useState(null)
  const [timeRange, setTimeRange] = useState(30)

  useEffect(() => {
    loadAnalyticsData()
  }, [timeRange])

  const loadAnalyticsData = async () => {
    setLoading(true)
    try {
      const [volumePredict, trends] = await Promise.all([
        analyticsAPI.predictTicketVolume(timeRange),
        analyticsAPI.getResolutionTrends()
      ])
      
      setVolumeData(volumePredict.data)
      setResolutionTrends(trends)
    } catch (error) {
      console.error('Failed to load analytics data:', error)
    } finally {
      setLoading(false)
    }
  }

  const volumeConfig = {
    data: volumeData,
    xField: 'date',
    yField: 'count',
    seriesField: 'type',
    smooth: true,
    animation: {
      appear: {
        animation: 'path-in',
        duration: 1000,
      },
    },
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h2 className="text-2xl font-bold">预测性分析</h2>
        <Select
          value={timeRange}
          onChange={setTimeRange}
          options={[
            { label: '7天', value: 7 },
            { label: '30天', value: 30 },
            { label: '90天', value: 90 },
          ]}
        />
      </div>

      <Spin spinning={loading}>
        <Row gutter={[16, 16]}>
          <Col span={24}>
            <Card title="工单量预测">
              <Line {...volumeConfig} />
            </Card>
          </Col>
          
          <Col span={12}>
            <Card title="解决时间趋势">
              {resolutionTrends && (
                <Bar
                  data={resolutionTrends.categoryBreakdown}
                  xField="category"
                  yField="averageTime"
                  colorField="category"
                />
              )}
            </Card>
          </Col>
          
          <Col span={12}>
            <Card title="优先级分布">
              {resolutionTrends && (
                <Pie
                  data={resolutionTrends.priorityDistribution}
                  angleField="count"
                  colorField="priority"
                  radius={0.8}
                />
              )}
            </Card>
          </Col>
        </Row>
      </Spin>
    </div>
  )
}
```

## 5. 第四阶段：性能优化和测试 (第11-12周)

### 5.1 性能优化

#### 5.1.1 数据库优化
```sql
-- 创建复合索引优化查询
CREATE INDEX CONCURRENTLY idx_tickets_composite 
ON tickets(tenant_id, status, priority, created_at DESC);

-- 创建部分索引
CREATE INDEX CONCURRENTLY idx_tickets_open 
ON tickets(assignee_id, created_at) 
WHERE status IN ('open', 'in_progress');

-- 创建表达式索引
CREATE INDEX CONCURRENTLY idx_tickets_search 
ON tickets USING gin(to_tsvector('english', title || ' ' || description));
```

#### 5.1.2 缓存策略实现
```go
// infrastructure/cache/layered_cache.go
package cache

type LayeredCache struct {
    l1Cache *sync.Map     // 内存缓存
    l2Cache *redis.Client // Redis缓存
    l3Cache Database      // 数据库
}

func (c *LayeredCache) Get(ctx context.Context, key string) (interface{}, error) {
    // L1: 内存缓存
    if value, ok := c.l1Cache.Load(key); ok {
        return value, nil
    }
    
    // L2: Redis缓存
    if value, err := c.l2Cache.Get(ctx, key).Result(); err == nil {
        var result interface{}
        json.Unmarshal([]byte(value), &result)
        
        // 回写到L1
        c.l1Cache.Store(key, result)
        return result, nil
    }
    
    // L3: 数据库
    value, err := c.l3Cache.Get(ctx, key)
    if err != nil {
        return nil, err
    }
    
    // 回写到L2和L1
    data, _ := json.Marshal(value)
    c.l2Cache.Set(ctx, key, data, time.Hour)
    c.l1Cache.Store(key, value)
    
    return value, nil
}
```

### 5.2 测试策略

#### 5.2.1 单元测试
```go
// application/service/ticket_service_test.go
package service_test

func TestTicketApplicationService_CreateTicket(t *testing.T) {
    // 准备测试数据
    mockRepo := &MockTicketRepository{}
    mockAI := &MockAIService{}
    mockPublisher := &MockEventPublisher{}
    
    service := NewTicketApplicationService(mockRepo, nil, mockPublisher, mockAI, nil)
    
    // 测试用例
    tests := []struct {
        name    string
        command CreateTicketCommand
        want    *TicketDTO
        wantErr bool
    }{
        {
            name: "successful creation",
            command: CreateTicketCommand{
                Title:       "Test Ticket",
                Description: "Test Description",
                Priority:    "high",
                RequesterID: "user-123",
            },
            want: &TicketDTO{
                Title:    "Test Ticket",
                Priority: "high",
            },
            wantErr: false,
        },
        {
            name: "invalid input",
            command: CreateTicketCommand{
                Title: "", // 空标题应该失败
            },
            want:    nil,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := service.CreateTicket(context.Background(), tt.command)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("CreateTicket() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if !tt.wantErr && got.Title != tt.want.Title {
                t.Errorf("CreateTicket() got = %v, want %v", got, tt.want)
            }
        })
    }
}
```

#### 5.2.2 集成测试
```go
// test/integration/ticket_integration_test.go
package integration_test

func TestTicketWorkflow(t *testing.T) {
    // 启动测试数据库
    testDB := setupTestDatabase(t)
    defer testDB.Close()
    
    // 创建测试客户端
    client := setupTestClient(t, testDB)
    
    // 测试完整的工单流程
    t.Run("complete ticket workflow", func(t *testing.T) {
        // 1. 创建工单
        createReq := CreateTicketRequest{
            Title:       "Integration Test Ticket",
            Description: "Test Description",
            Priority:    "high",
        }
        
        resp, err := client.CreateTicket(createReq)
        require.NoError(t, err)
        require.NotEmpty(t, resp.ID)
        
        ticketID := resp.ID
        
        // 2. 分配工单
        err = client.AssignTicket(ticketID, "assignee-123")
        require.NoError(t, err)
        
        // 3. 更新状态
        err = client.UpdateTicketStatus(ticketID, "in_progress")
        require.NoError(t, err)
        
        // 4. 添加评论
        err = client.AddComment(ticketID, "Working on this issue")
        require.NoError(t, err)
        
        // 5. 解决工单
        err = client.ResolveTicket(ticketID, "Issue resolved")
        require.NoError(t, err)
        
        // 6. 验证最终状态
        ticket, err := client.GetTicket(ticketID)
        require.NoError(t, err)
        assert.Equal(t, "resolved", ticket.Status)
        assert.NotEmpty(t, ticket.Resolution)
    })
}
```

#### 5.2.3 前端测试
```typescript
// components/tickets/__tests__/TicketList.test.tsx
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { TicketList } from '../TicketList'
import { useTicketStore } from '@/lib/store/ticket-store'

// Mock store
jest.mock('@/lib/store/ticket-store')

const mockUseTicketStore = useTicketStore as jest.MockedFunction<typeof useTicketStore>

describe('TicketList', () => {
  const mockStore = {
    tickets: [
      {
        id: '1',
        title: 'Test Ticket',
        priority: 'high',
        status: 'open',
        createdAt: '2024-01-01T00:00:00Z',
      },
    ],
    loading: false,
    filters: {},
    fetchTickets: jest.fn(),
    setFilters: jest.fn(),
  }

  beforeEach(() => {
    mockUseTicketStore.mockReturnValue(mockStore)
  })

  it('renders ticket list correctly', () => {
    render(<TicketList />)
    
    expect(screen.getByText('Test Ticket')).toBeInTheDocument()
    expect(screen.getByText('high')).toBeInTheDocument()
    expect(screen.getByText('open')).toBeInTheDocument()
  })

  it('handles search input', async () => {
    render(<TicketList />)
    
    const searchInput = screen.getByPlaceholderText('搜索工单...')
    fireEvent.change(searchInput, { target: { value: 'test search' } })
    fireEvent.keyPress(searchInput, { key: 'Enter', code: 'Enter' })
    
    await waitFor(() => {
      expect(mockStore.setFilters).toHaveBeenCalledWith({
        search: 'test search'
      })
    })
  })

  it('opens create ticket modal', () => {
    render(<TicketList />)
    
    const createButton = screen.getByText('创建工单')
    fireEvent.click(createButton)
    
    expect(screen.getByText('创建新工单')).toBeInTheDocument()
  })
})
```

## 6. 部署和运维

### 6.1 Docker化部署
```dockerfile
# Dockerfile.backend
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/config ./config

CMD ["./main"]
```

```dockerfile
# Dockerfile.frontend
FROM node:18-alpine AS builder

WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production

COPY . .
RUN npm run build

FROM node:18-alpine
WORKDIR /app

COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/package*.json ./
COPY --from=builder /app/node_modules ./node_modules

EXPOSE 3000
CMD ["npm", "start"]
```

### 6.2 Kubernetes部署配置
```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: itsm-backend
spec:
  replicas: 3
  selector:
    matchLabels:
      app: itsm-backend
  template:
    metadata:
      labels:
        app: itsm-backend
    spec:
      containers:
      - name: itsm-backend
        image: itsm-backend:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: itsm-secrets
              key: database-url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: itsm-secrets
              key: redis-url
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /api/v1/healthz
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /api/v1/healthz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

## 7. 项目交付清单

### 7.1 代码交付物
- [ ] 后端Go代码 (完整的DDD架构)
- [ ] 前端Next.js代码 (组件化架构)
- [ ] 数据库迁移脚本
- [ ] API文档 (Swagger)
- [ ] 单元测试 (覆盖率>80%)
- [ ] 集成测试
- [ ] 性能测试报告

### 7.2 部署交付物
- [ ] Docker镜像
- [ ] Kubernetes配置文件
- [ ] CI/CD流水线配置
- [ ] 监控配置 (Prometheus/Grafana)
- [ ] 日志配置 (ELK Stack)

### 7.3 文档交付物
- [ ] 系统架构文档
- [ ] API接口文档
- [ ] 用户操作手册
- [ ] 运维部署手册
- [ ] 故障排查手册

### 7.4 培训交付物
- [ ] 开发团队技术培训
- [ ] 运维团队部署培训
- [ ] 用户使用培训
- [ ] 管理员配置培训

这个实施计划提供了完整的12周开发路线图，确保ITSM系统能够按时高质量交付。每个阶段都有明确的目标、具体的代码实现和验收标准。