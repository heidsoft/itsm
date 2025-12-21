# ITSM系统编码规范与命名标准

## 📋 概述

本文档定义了ITSM系统的统一编码规范和命名标准，旨在提高代码可读性、维护性和团队协作效率。

## 🏗️ 项目结构规范

### 1. 目录结构命名

```
itsm/
├── itsm-frontend/          # 前端项目
│   ├── src/
│   │   ├── app/           # Next.js App Router
│   │   ├── components/    # 组件目录
│   │   │   ├── ui/       # UI基础组件
│   │   │   ├── business/ # 业务组件
│   │   │   └── forms/    # 表单组件
│   │   ├── lib/           # 工具库
│   │   │   ├── api/      # API客户端
│   │   │   ├── utils/    # 工具函数
│   │   │   └── config/   # 配置管理
│   │   ├── stores/        # 状态管理
│   │   ├── types/         # 类型定义
│   │   └── styles/       # 样式文件
│   └── public/           # 静态资源
├── itsm-backend/           # 后端项目
│   ├── controller/         # 控制器层
│   ├── service/          # 服务层
│   ├── dto/              # 数据传输对象
│   ├── ent/              # 实体层
│   ├── middleware/        # 中间件
│   ├── router/           # 路由
│   └── config/           # 配置
└── scripts/              # 脚本文件
```

## 📁 文件命名规范

### 1. 前端文件命名

#### 组件文件
```typescript
// ✅ 正确命名
components/ui/Button.tsx           // PascalCase
components/ui/Input.tsx            // PascalCase
components/business/TicketDetail.tsx // PascalCase
components/forms/EnhancedInput.tsx // PascalCase

// ❌ 错误命名
components/ui/button.tsx           // 小写开头
components/ui/buttonComponent.tsx  // 后缀不必要
components/ui/Button_Component.tsx // 下划线分隔
```

#### 工具文件
```typescript
// ✅ 正确命名
lib/utils/formatDate.ts            // camelCase
lib/utils/validation.ts            // camelCase
lib/api/ticket-api.ts             // kebab-case
lib/config/app-config.ts           // kebab-case

// ❌ 错误命名
lib/utils/FormatDate.ts           // PascalCase
lib/utils/format_date.ts          // snake_case
lib/api/ticketApi.ts              // 驼峰混合
```

#### 类型文件
```typescript
// ✅ 正确命名
types/api-types.ts                // kebab-case
types/ticket-types.ts             // kebab-case
types/user-types.ts               // kebab-case

// ❌ 错误命名
types/apiTypes.ts                // camelCase
types/api_types.ts               // snake_case
```

#### 页面文件
```typescript
// ✅ 正确命名 (Next.js App Router)
app/(auth)/login/page.tsx        // 固定文件名
app/(main)/dashboard/page.tsx     // 固定文件名
app/(main)/tickets/[id]/page.tsx // 动态路由

// ✅ 正确命名 (布局文件)
app/(auth)/layout.tsx            // 布局组件
app/(main)/layout.tsx            // 布局组件

// ✅ 正确命名 (组件文件)
components/ui/loading.tsx         // 全小写
components/ui/error.tsx           // 全小写
```

### 2. 后端文件命名

#### Go源文件
```go
// ✅ 正确命名
controller/ticket_controller.go     // snake_case
service/ticket_service.go          // snake_case
dto/ticket_dto.go                 // snake_case
middleware/auth_middleware.go      // snake_case
router/api_router.go              // snake_case

// ❌ 错误命名
controller/ticketController.go     // camelCase
service/ticketservice.go          // 无分隔符
controller/ticket-controller.go    // 短横线分隔
```

#### 测试文件
```go
// ✅ 正确命名
controller/ticket_controller_test.go   // _test.go后缀
service/ticket_service_test.go        // _test.go后缀
dto/ticket_dto_test.go               // _test.go后缀

// ❌ 错误命名
controller/ticket_test.go            // 缺少controller
test/ticket_controller_test.go       // 错误目录
controller/ticket_controller.tests.go // 错误后缀
```

#### 配置文件
```go
// ✅ 正确命名
config/app_config.go               // snake_case
config/db_config.go                 // snake_case
config/auth_config.go               // snake_case

// ❌ 错误命名
config/appConfig.go                // camelCase
config/app-config.go               // 短横线分隔
```

### 3. 脚本文件命名

```bash
# ✅ 正确命名
scripts/build.sh                   # snake_case + .sh
scripts/deploy-production.sh        # snake_case + .sh
scripts/code-quality-check.sh       # kebab-case + .sh
scripts/setup-dev-env.sh           # snake_case + .sh

# ❌ 错误命名
scripts/buildScript.sh             # camelCase
scripts/build.sh.bak               // 错误后缀
scripts/BUILD.SH                  // 大写
```

## 🏷️ 命名约定

### 1. 前端命名规范

#### 组件命名
```typescript
// ✅ 正确 - 组件名使用PascalCase
export const TicketDetail: React.FC<TicketDetailProps> = ({ ticket }) => {
  return <div>{ticket.title}</div>;
};

// ✅ 正确 - Props接口命名
interface TicketDetailProps {
  ticket: Ticket;
  onUpdate: (ticket: Ticket) => void;
  className?: string;
}

// ✅ 正确 - Hook命名
export const useTicketData = () => {
  const [tickets, setTickets] = useState<Ticket[]>([]);
  return { tickets, setTickets };
};

// ✅ 正确 - 常量命名
const API_ENDPOINTS = {
  TICKETS: '/api/tickets',
  USERS: '/api/users',
} as const;

// ✅ 正确 - 变量和函数命名
const ticketData = [];           // camelCase
function fetchTicketData() {}     // camelCase
const handleSubmit = () => {}     // camelCase

// ❌ 错误命名
export const ticketDetail = () => {};  // 小写开头
const ticket_data = [];                // snake_case
function FetchTicketData() {}           // 大写开头
```

#### 类型定义
```typescript
// ✅ 正确 - 接口命名
interface Ticket {
  id: number;
  title: string;
  status: TicketStatus;
}

// ✅ 正确 - 类型别名
type TicketStatus = 'open' | 'closed' | 'pending';

// ✅ 正确 - 枚举命名
enum Priority {
  Low = 'low',
  Medium = 'medium',
  High = 'high',
  Critical = 'critical',
}

// ✅ 正确 - 泛型命名
interface ApiResponse<T> {
  data: T;
  message: string;
  code: number;
}

// ❌ 错误命名
interface ticket {}                // 小写
type ticketStatus = string;        // camelCase
enum priority { ... }              // 小写
```

### 2. 后端命名规范

#### 结构体命名
```go
// ✅ 正确 - 结构体使用PascalCase
type TicketController struct {
    ticketService *TicketService
    logger        *zap.SugaredLogger
}

type TicketService struct {
    client *ent.Client
    logger *zap.SugaredLogger
}

// ✅ 正确 - 接口命名
type TicketRepository interface {
    Create(ctx context.Context, ticket *Ticket) error
    GetByID(ctx context.Context, id int) (*Ticket, error)
}

// ✅ 正确 - DTO命名
type CreateTicketRequest struct {
    Title       string `json:"title"`
    Description string `json:"description"`
    Priority    string `json:"priority"`
}

type TicketResponse struct {
    ID          int    `json:"id"`
    Title       string `json:"title"`
    Status      string `json:"status"`
}

// ❌ 错误命名
type ticketController struct {}   // 小写开头
type ticket_service struct {}    // 下划线
type CreateTicketRequest struct { // 大写字母开头
    title string `json:"title"`
}
```

#### 函数和方法命名
```go
// ✅ 正确 - 公开函数/方法使用PascalCase
func NewTicketController(service *TicketService) *TicketController {
    return &TicketController{ticketService: service}
}

func (tc *TicketController) CreateTicket(c *gin.Context) {
    // 实现
}

// ✅ 正确 - 私有函数使用小写字母开头
func validateTicketData(ticket *CreateTicketRequest) error {
    // 验证逻辑
}

func (ts *TicketService) generateTicketNumber(ctx context.Context) (string, error) {
    // 生成工单编号
}

// ✅ 正确 - 构造函数命名
func NewTicketService(client *ent.Client, logger *zap.SugaredLogger) *TicketService {
    return &TicketService{client: client, logger: logger}
}

// ❌ 错误命名
func newTicketController() {}       // 小写开头
func (tc *TicketController) createTicket() {} // 小写开头
func generateTicketNumber() {}       // 混合使用
```

#### 变量和常量命名
```go
// ✅ 正确 - 常量使用大写字母和下划线
const (
    API_VERSION = "v1"
    MAX_PAGE_SIZE = 100
    DEFAULT_TIMEOUT = 30 * time.Second
)

// ✅ 正确 - 包级别变量使用camelCase
var (
    defaultLogger *zap.SugaredLogger
    appConfig    *Config
)

// ✅ 正确 - 局部变量使用camelCase
func (tc *TicketController) CreateTicket(c *gin.Context) {
    var req CreateTicketRequest
    userID := c.GetInt("user_id")
    tenantID := c.GetInt("tenant_id")
    
    ticket, err := tc.ticketService.CreateTicket(c.Request.Context(), &req, tenantID)
}

// ❌ 错误命名
const apiVersion = "v1"           // 小写
var DefaultLogger *zap.SugaredLogger // 大写开头但非常量
func (tc *TicketController) CreateTicket(c *gin.Context) {
    var req CreateTicketRequest
    userId := c.GetInt("user_id")     // 不一致
    TenantId := c.GetInt("tenant_id") // 不一致
}
```

## 🔄 API命名规范

### 1. RESTful API端点

```go
// ✅ 正确的RESTful命名
GET    /api/v1/tickets              // 获取工单列表
POST   /api/v1/tickets              // 创建工单
GET    /api/v1/tickets/{id}         // 获取特定工单
PUT    /api/v1/tickets/{id}         // 更新工单
DELETE /api/v1/tickets/{id}         // 删除工单
PATCH  /api/v1/tickets/{id}/status   // 更新工单状态
POST   /api/v1/tickets/{id}/assign   // 分配工单

// ✅ 正确的嵌套资源命名
GET    /api/v1/tickets/{id}/comments    // 获取工单评论
POST   /api/v1/tickets/{id}/comments    // 添加工单评论
GET    /api/v1/users/{id}/tickets       // 获取用户的工单

// ❌ 错误命名
GET    /api/v1/getTickets              // 动词开头
POST   /api/v1/ticket                 // 单数形式
GET    /api/v1/ticket/{id}            // 不一致
PUT    /api/v1/tickets/{id}/update    // 资源更新
```

### 2. 查询参数命名

```typescript
// ✅ 正确的查询参数命名
GET /api/v1/tickets?page=1&page_size=20&sort_by=created_at&sort_order=desc&search=keyword

// ✅ 标准化的参数结构
interface TicketListParams {
  page?: number;          // 页码
  page_size?: number;     // 每页大小
  sort_by?: string;      // 排序字段
  sort_order?: 'asc' | 'desc'; // 排序方向
  search?: string;       // 搜索关键词
  status?: string;       // 状态筛选
  priority?: string;     // 优先级筛选
  date_from?: string;    // 开始日期
  date_to?: string;      // 结束日期
}

// ❌ 错误的参数命名
GET /api/v1/tickets?currentPage=1&limit=20&sortBy=createdAt&order=desc&keyword=search

// 问题：命名不一致，应该统一使用snake_case
```

## 📝 代码格式规范

### 1. TypeScript/JavaScript格式

```typescript
// ✅ 正确的导入格式
import React from 'react';
import { Button, Input } from '@/components/ui';
import type { Ticket, TicketStatus } from '@/types';
import { useTicketStore } from '@/stores';

// ✅ 正确的函数定义
export const TicketList: React.FC<TicketListProps> = ({ 
  tickets, 
  loading, 
  onUpdate 
}) => {
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  
  const handleUpdate = useCallback((ticket: Ticket) => {
    onUpdate?.(ticket);
  }, [onUpdate]);

  return (
    <div className="ticket-list">
      {tickets.map((ticket) => (
        <TicketCard 
          key={ticket.id} 
          ticket={ticket} 
          onUpdate={handleUpdate}
        />
      ))}
    </div>
  );
};

// ✅ 正确的接口定义
interface TicketListProps {
  tickets: Ticket[];
  loading?: boolean;
  onUpdate?: (ticket: Ticket) => void;
}
```

### 2. Go格式

```go
// ✅ 正确的Go格式
package controller

import (
    "context"
    "fmt"
    "net/http"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

type TicketController struct {
    ticketService *TicketService
    logger        *zap.SugaredLogger
}

func NewTicketController(service *TicketService, logger *zap.SugaredLogger) *TicketController {
    return &TicketController{
        ticketService: service,
        logger:        logger,
    }
}

func (tc *TicketController) CreateTicket(c *gin.Context) {
    var req CreateTicketRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        tc.logger.Errorw("Failed to bind request", "error", err)
        c.JSON(http.StatusBadRequest, gin.H{
            "code":    1001,
            "message": "请求参数错误",
        })
        return
    }

    tenantID, userID := tc.getContextParams(c)
    ticket, err := tc.ticketService.CreateTicket(c.Request.Context(), &req, tenantID, userID)
    if err != nil {
        tc.handleError(c, err)
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "code":    200,
        "message": "success",
        "data":    ticket,
    })
}
```

## 🏷️ 注释规范

### 1. 前端注释

```typescript
// ✅ 单行注释 - 使用双斜杠
const API_TIMEOUT = 30000; // API请求超时时间（毫秒）

// ✅ 多行注释 - 使用双斜杠
// TicketDetail组件用于显示工单的详细信息
// 支持编辑、删除、评论等操作
// 支持移动端自适应布局

/**
 * TicketDetail组件属性接口
 * 
 * @interface TicketDetailProps
 * @property {Ticket} ticket - 工单数据
 * @property {(ticket: Ticket) => void} onUpdate - 更新回调
 * @property {string} [className] - 自定义CSS类名
 * @property {boolean} [editable=true] - 是否可编辑
 */
interface TicketDetailProps {
  ticket: Ticket;
  onUpdate: (ticket: Ticket) => void;
  className?: string;
  editable?: boolean;
}

/**
 * 获取工单列表
 * 
 * @param {TicketListParams} params - 查询参数
 * @param {number} params.page - 页码
 * @param {number} params.page_size - 每页大小
 * @param {string} params.sort_by - 排序字段
 * @param {'asc'|'desc'} params.sort_order - 排序方向
 * @returns {Promise<TicketListResponse>} 工单列表响应
 * @throws {ApiError} API请求错误
 */
export const fetchTickets = async (params: TicketListParams): Promise<TicketListResponse> => {
  // 实现逻辑
};
```

### 2. 后端注释

```go
// Package controller 提供HTTP请求处理功能
package controller

import (
    "context"
    "fmt"
)

// TicketController 处理工单相关的HTTP请求
// 提供工单的CRUD操作、状态管理、分配等功能
type TicketController struct {
    ticketService *TicketService // 工单服务
    logger        *zap.SugaredLogger // 日志记录器
}

// NewTicketController 创建工单控制器实例
// 
// 参数:
//   - service: 工单服务实例
//   - logger: 日志记录器实例
//
// 返回:
//   - *TicketController: 工单控制器实例
func NewTicketController(service *TicketService, logger *zap.SugaredLogger) *TicketController {
    return &TicketController{
        ticketService: service,
        logger:        logger,
    }
}

// CreateTicket 创建新的工单
// 处理POST /api/v1/tickets请求
//
// 请求体示例:
//   {
//     "title": "工单标题",
//     "description": "工单描述",
//     "priority": "high"
//   }
//
// 响应示例:
//   {
//     "code": 200,
//     "message": "success",
//     "data": {
//       "id": 1,
//       "title": "工单标题",
//       "status": "open"
//     }
//   }
//
// 错误码:
//   - 1001: 参数验证失败
//   - 1002: 权限不足
//   - 5001: 服务器内部错误
func (tc *TicketController) CreateTicket(c *gin.Context) {
    // 验证请求参数
    var req CreateTicketRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        tc.logger.Errorw("Failed to bind request", "error", err)
        c.JSON(400, gin.H{
            "code":    1001,
            "message": "请求参数错误",
        })
        return
    }

    // 获取上下文参数
    tenantID, userID := tc.getContextParams(c)

    // 调用服务层创建工单
    ticket, err := tc.ticketService.CreateTicket(c.Request.Context(), &req, tenantID, userID)
    if err != nil {
        tc.handleError(c, err)
        return
    }

    // 返回成功响应
    c.JSON(200, gin.H{
        "code":    200,
        "message": "success",
        "data":    ticket,
    })
}
```

## 🔧 工具配置

### 1. ESLint配置

```json
{
  "rules": {
    // 命名规范
    "camelcase": ["error", { "properties": "always" }],
    "typescript/naming-convention": [
      "error",
      {
        "selector": "interface",
        "format": ["PascalCase"]
      },
      {
        "selector": "typeAlias",
        "format": ["PascalCase"]
      },
      {
        "selector": "variable",
        "format": ["camelCase", "UPPER_CASE"],
        "filter": {
          "regex": "^(?:[A-Z]|const .+$)",
          "match": false
        }
      },
      {
        "selector": "function",
        "format": ["camelCase"]
      }
    ]
  }
}
```

### 2. Go Vet和Lint配置

```bash
# .golangci.yml
linters:
  enable:
    - gofmt
    - goimports
    - govet
    - misspell
    - goconst
    - gocritic
    - gocyclo
    - gosec
    - ineffassign
    - misspell
    - unconvert
    - unparam
    - unused
    - varcheck
    - structcheck

linters-settings:
  goconst:
    min-len: 3
    min-occurrences: 3
  
  gocyclo:
    min-complexity: 15
  
  goimports:
    local-prefixes: itsm-backend
```

## 📋 检查清单

### 提交代码前的检查

#### 文件命名
- [ ] 文件名符合规范（前端的kebab-case/PascalCase，后端的snake_case）
- [ ] 文件名长度合理（不超过50个字符）
- [ ] 目录结构清晰（不超过4层深度）

#### 代码命名
- [ ] 变量名有意义且符合规范
- [ ] 函数/方法名清晰表达功能
- [ ] 类名/接口名使用正确的命名风格
- [ ] 常量名使用大写字母和下划线

#### 代码结构
- [ ] 函数职责单一，长度合理（不超过50行）
- [ ] 类/接口职责明确
- [ ] 代码注释充分且规范
- [ ] 导入语句格式正确

#### API设计
- [ ] RESTful API端点命名规范
- [ ] 查询参数命名一致
- [ ] 响应格式统一
- [ ] 错误处理完善

### Code Review要点

#### 可读性检查
- [ ] 代码逻辑清晰易懂
- [ ] 变量命名准确反映用途
- [ ] 函数名明确表达功能
- [ ] 没有冗余或无用代码

#### 一致性检查
- [ ] 命名风格与项目一致
- [ ] 代码结构与其他部分一致
- [ ] 错误处理方式一致
- [ ] 注释风格一致

#### 性能检查
- [ ] 没有明显的性能问题
- [ ] 资源使用合理
- [ ] 缓存策略适当
- [ ] 数据库查询优化

## 🎯 实施建议

### 1. 渐进式实施

**第一阶段（1周）**：
- 配置代码检查工具（ESLint, golangci-lint）
- 建立基础的命名规范文档
- 进行团队培训和宣贯

**第二阶段（2-4周）**：
- 重构现有不符合规范的代码
- 实施自动化代码检查
- 建立Code Review checklist

**第三阶段（持续）**：
- 定期检查和更新规范
- 根据项目发展调整标准
- 持续改进和优化

### 2. 工具集成

#### Pre-commit Hooks
```bash
#!/bin/sh
# .git/hooks/pre-commit

# 前端代码检查
if git diff --cached --name-only | grep -E '\.(ts|tsx)$'; then
    npm run lint:fix
    npm run type-check
fi

# 后端代码检查
if git diff --cached --name-only | grep -E '\.go$'; then
    go fmt ./...
    go vet ./...
    golangci-lint run
fi
```

#### CI/CD集成
```yaml
# .github/workflows/ci.yml
name: CI
on: [push, pull_request]

jobs:
  lint-and-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'
          
      - name: Setup Go
        uses: actions/setup-go@v3
        with:
          go-version: '1.19'
          
      - name: Frontend lint
        run: |
          cd itsm-frontend
          npm ci
          npm run lint
          npm run type-check
          
      - name: Backend lint
        run: |
          cd itsm-backend
          go fmt ./...
          go vet ./...
          golangci-lint run
```

### 3. 团队培训

#### 新员工入职培训
- 编码规范讲解
- 工具使用培训
- 代码Review流程培训

#### 定期技术分享
- 编码规范更新分享
- 最佳实践案例分享
- 问题代码案例分析

## 📚 参考资料

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/handbook/intro.html)
- [Airbnb JavaScript Style Guide](https://github.com/airbnb/javascript)
- [RESTful API Design Guide](https://restfulapi.net/)

---

**文档版本**: 1.0.0  
**最后更新**: 2025-12-21  
**维护者**: ITSM架构团队  
**审核**: 技术委员会