# 📘 Trae 项目开发规则：ITSM 系统（Go + Gin + Ent）

---

## 1. 框架版本与主要依赖

### 后端技术栈

* 编程语言：Go 1.22+
* Web 框架：Gin v1.9.1
* ORM 工具：Ent v0.13.0
* 数据库：PostgreSQL v15+
* 配置管理：spf13/viper
* JSON 处理：Go 原生 encoding/json
* 日志组件：uber-go/zap v1.26.0
* 接口文档：swaggo/gin-swagger
* 鉴权机制：JWT + 基于角色的访问控制（RBAC）

### 前端技术栈

* 框架：Next.js 15.3.4 (App Router)
* 编程语言：TypeScript 5.x
* UI 框架：React 19.0.0
* 样式：Tailwind CSS 4.x
* 图标库：Lucide React v0.525.0
* 图表库：Recharts v3.0.2
* 状态管理：Zustand v5.0.6
* 代码规范：ESLint + Next.js 配置

---

## 2. 禁止使用的 APIs 与代码风格限制

### 后端限制

为了保持结构一致、便于协作与自动生成代码，**禁止使用以下做法：**

* ❌ 不使用 `database/sql` 或 `gorm`，统一采用 `Ent` ORM
* ❌ 不允许在 controller 中直接访问数据库
* ❌ 不使用 `fmt.Println()` 或 `log.Println()`，应统一使用 `zap.Sugar()` 结构化日志
* ❌ 不允许未封装的错误直接 panic，应通过 `errors.Wrap()` 或统一错误返回结构处理
* ❌ 禁止返回非统一格式的响应，应使用：

```go
// 成功响应
Success(c, data)

// 错误响应
Fail(c, code, "错误信息")
```

* ❌ 不允许写死用户 ID 或硬编码权限逻辑，应通过中间件注入用户身份并进行验证

### 前端限制

为了保持代码一致性和可维护性，**禁止使用以下做法：**

* ❌ 不使用 Pages Router，统一采用 App Router
* ❌ 不使用 CSS Modules 或 styled-components，统一使用 Tailwind CSS
* ❌ 不使用 `console.log()` 进行调试，应使用适当的调试工具
* ❌ 不允许在组件中直接调用 API，应通过统一的 API 层
* ❌ 禁止硬编码 API 地址，应使用环境变量配置
* ❌ 不允许在客户端组件中使用服务端专用的 API
* ❌ 禁止直接操作 localStorage，应通过封装的服务类

---

## 3. 测试框架与测试要求

### 后端测试

* 单元测试框架：`stretchr/testify`
* 所有 `service` 层必须配有对应的 `_test.go` 文件
* 所有接口建议使用 `httptest.NewRecorder` 进行 handler 级测试
* 测试方式采用 Table Driven Test 风格
* 所有错误路径需要覆盖
* 推荐使用 `enttest.NewClient()` 创建测试用数据库环境（内存或临时文件）

### 前端测试

* 单元测试框架：Jest + React Testing Library
* 组件测试：每个复杂组件应配有对应的 `.test.tsx` 文件
* API 层测试：所有 API 调用应进行 mock 测试
* 端到端测试：关键业务流程使用 Playwright 或 Cypress
* 类型检查：严格的 TypeScript 配置，不允许 `any` 类型

---

## ✅ 统一接口响应规范

所有 API 接口返回结构如下：

```json
{
  "code": 0,
  "message": "操作成功",
  "data": {}
}
```

* 成功时：`code = 0`
* 参数错误：`code = 1001+`
* 鉴权失败：`code = 2001`
* 服务内部错误：`code = 5001`

### 后端响应规范

#### 统一响应结构

```go
// common/response.go
type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

// 响应码定义
const (
    SuccessCode        = 0
    ParamErrorCode     = 1001
    ValidationError    = 1002
    AuthFailedCode     = 2001
    ForbiddenCode      = 2003
    NotFoundCode       = 4004
    InternalErrorCode  = 5001
)

// 成功响应
func Success(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, Response{
        Code:    SuccessCode,
        Message: "success",
        Data:    data,
    })
}

// 失败响应
func Fail(c *gin.Context, code int, message string) {
    c.JSON(http.StatusOK, Response{
        Code:    code,
        Message: message,
    })
}
```

#### 控制器使用规范

```go
// 正确的控制器写法
func (c *IncidentController) ListIncidents(ctx *gin.Context) {
    var req dto.ListIncidentsRequest
    if err := ctx.ShouldBindQuery(&req); err != nil {
        common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
        return
    }

    response, err := c.incidentService.ListIncidents(ctx, &req, tenantID.(int))
    if err != nil {
        c.logger.Errorw("Failed to list incidents", "error", err)
        common.Fail(ctx, common.InternalErrorCode, err.Error())
        return
    }

    common.Success(ctx, response)
}
```

### 前端响应处理规范

#### HTTP客户端规范

```typescript
// lib/http-client.ts
interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

class HttpClient {
  private async request<T>(endpoint: string, config: RequestConfig): Promise<T> {
    // ... 请求逻辑
    
    const responseData = await response.json() as ApiResponse<T>;
    
    // 检查响应码
    if (responseData.code !== 0) {
      throw new Error(responseData.message || '请求失败');
    }
    
    return responseData.data;
  }
}
```

#### API层规范

```typescript
// lib/incident-api.ts
export class IncidentAPI {
  static async listIncidents(params: ListIncidentsRequest = {}): Promise<ListIncidentsResponse> {
    try {
      const response = await httpClient.get<ListIncidentsResponse>('/api/incidents', params);
      return response;
    } catch (error) {
      console.error('IncidentAPI.listIncidents error:', error);
      throw error;
    }
  }
}
```

#### 页面组件规范

```typescript
// 页面组件中的错误处理
const fetchIncidents = async () => {
  try {
    setLoading(true);
    const response = await IncidentAPI.listIncidents(params);
    
    if (!response || !response.incidents) {
      throw new Error('API响应数据格式错误');
    }
    
    setIncidents(response.incidents);
    setTotal(response.total);
  } catch (error) {
    console.error('Failed to fetch incidents:', error);
    setError(error instanceof Error ? error.message : '获取事件列表失败');
  } finally {
    setLoading(false);
  }
};
```

### 错误处理规范

#### 后端错误处理

* **参数验证错误**：使用 `common.Fail(ctx, common.ParamErrorCode, message)`
* **认证失败**：使用 `common.Fail(ctx, common.AuthFailedCode, message)`
* **权限不足**：使用 `common.Fail(ctx, common.ForbiddenCode, message)`
* **资源不存在**：使用 `common.Fail(ctx, common.NotFoundCode, message)`
* **内部错误**：使用 `common.Fail(ctx, common.InternalErrorCode, message)`

#### 前端错误处理

* **网络错误**：显示友好的网络错误提示
* **业务错误**：显示后端返回的具体错误信息
* **数据格式错误**：记录详细日志，显示通用错误提示

### 类型定义规范

#### 后端DTO规范

```go
// dto/incident_dto.go
type ListIncidentsResponse struct {
    Incidents []*Incident `json:"incidents"`
    Total     int         `json:"total"`
    Page      int         `json:"page"`
    PageSize  int         `json:"page_size"`
}

type CreateIncidentRequest struct {
    Title       string                 `json:"title" binding:"required,min=2,max=200"`
    Description string                 `json:"description" binding:"required,min=10,max=5000"`
    Priority    string                 `json:"priority" binding:"required,oneof=low medium high critical"`
    FormFields  map[string]interface{} `json:"form_fields"`
}
```

#### 前端类型定义规范

```typescript
// lib/incident-api.ts
export interface Incident {
  id: number;
  title: string;
  description: string;
  status: string;
  priority: string;
  source: string;
  type: string;
  incident_number: string;
  is_major_incident: boolean;
  created_at: string;
  updated_at: string;
}

export interface ListIncidentsResponse {
  incidents: Incident[];
  total: number;
  page: number;
  page_size: number;
}

export interface ListIncidentsRequest {
  page?: number;
  page_size?: number;
  status?: string;
  priority?: string;
  source?: string;
  type?: string;
  assignee_id?: number;
  is_major_incident?: boolean;
  keyword?: string;
}
```

### 调试与日志规范

#### 后端日志规范

```go
// 使用结构化日志
c.logger.Errorw("Failed to list incidents", 
    "error", err,
    "tenant_id", tenantID,
    "params", req)
```

#### 前端调试规范

```typescript
// 在API层添加调试日志
console.log('IncidentAPI.listIncidents called with params:', params);
console.log('IncidentAPI.listIncidents response:', response);

// 在HTTP客户端添加请求响应日志
console.log('HTTP Client Request:', { url, method, headers });
console.log('HTTP Client Response:', { status, statusText });
console.log('HTTP Client Raw Response Data:', responseData);
```

### 性能优化规范

#### 分页查询规范

```typescript
// 前端分页参数
const params = {
  page: 1,
  page_size: 20,
  status: filter === "全部" ? undefined : filter,
  is_major_incident: showMajorIncidents ? true : undefined,
};
```

```go
// 后端分页处理
if req.Page <= 0 {
    req.Page = 1
}
if req.PageSize <= 0 {
    req.PageSize = 20
}
```

#### 缓存策略规范

* 使用 React Query 或 SWR 进行数据缓存
* 设置合理的缓存过期时间
* 实现请求去重和防抖

```typescript
// 使用 SWR 进行数据缓存
export const useIncidents = (params: ListIncidentsRequest) => {
  const { data, error, mutate } = useSWR(
    ['/api/incidents', params],
    () => IncidentAPI.listIncidents(params),
    {
      revalidateOnFocus: false,
      dedupingInterval: 5000,
    }
  );
  
  return {
    incidents: data?.incidents || [],
    total: data?.total || 0,
    isLoading: !error && !data,
    isError: error,
    refresh: mutate,
  };
};
```

---

## 📁 项目结构建议

### 后端结构

| 目录            | 说明                   |
| ------------- | -------------------- |
| `controller/` | 控制器，仅接收参数并调用 service |
| `service/`    | 核心业务逻辑处理             |
| `ent/schema/` | 数据建模（自动生成 CRUD）      |
| `middleware/` | JWT、权限、日志、异常处理       |
| `router/`     | 路由注册                 |
| `config/`     | 配置文件、初始化数据库等         |
| `tests/`      | 单元测试与 mock 测试        |

### 前端结构

| 目录                    | 说明                     |
| --------------------- | ---------------------- |
| `src/app/`            | Next.js App Router 根目录  |
| `src/app/components/` | 可复用组件                  |
| `src/app/lib/`        | 工具函数、API 客户端、状态管理     |
| `src/app/hooks/`      | 自定义 React Hooks       |
| `src/app/(pages)/`    | 页面组件（按功能模块分组）          |
| `src/app/globals.css` | 全局样式                   |
| `public/`             | 静态资源                   |

---

## 🎨 前端开发规范

### 组件开发规范

* 所有组件必须使用 TypeScript
* 组件文件名使用 PascalCase（如 `UserProfile.tsx`）
* 页面文件使用 `page.tsx`，布局文件使用 `layout.tsx`
* 组件必须导出类型定义的 Props 接口

```typescript
interface UserProfileProps {
  userId: string;
  onUpdate?: (user: User) => void;
}

export const UserProfile: React.FC<UserProfileProps> = ({ userId, onUpdate }) => {
  // 组件实现
};
```

### 状态管理规范

* 使用 Zustand 进行全局状态管理
* 本地状态优先使用 `useState` 和 `useReducer`
* 复杂状态逻辑封装为自定义 Hooks

```typescript
// store.ts
interface AuthState {
  user: User | null;
  token: string | null;
  login: (user: User, token: string) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  token: null,
  login: (user, token) => set({ user, token }),
  logout: () => set({ user: null, token: null }),
}));
```

### API 调用规范

* 统一使用 `httpClient` 类进行 API 调用
* 所有 API 接口必须定义 TypeScript 类型
* 错误处理统一在 API 层进行

```typescript
// api/user-api.ts
export class UserApi {
  static async getUser(id: string): Promise<ApiResponse<User>> {
    return httpClient.get<User>(`/api/users/${id}`);
  }
  
  static async updateUser(id: string, data: UpdateUserRequest): Promise<ApiResponse<User>> {
    return httpClient.put<User>(`/api/users/${id}`, data);
  }
}
```

### 样式规范

* 统一使用 Tailwind CSS 进行样式开发
* 组件样式使用 className 属性
* 复杂样式可提取为 CSS 变量或 Tailwind 配置

```typescript
// 推荐的样式写法
<div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
  <h2 className="text-xl font-semibold text-gray-900 mb-4">标题</h2>
  <p className="text-gray-600">内容</p>
</div>
```

---

## 🔐 认证与权限规范

### 前端认证流程

* 使用 JWT Token 进行身份验证
* Token 存储在 localStorage 中，通过 `AuthService` 管理
* 页面级权限通过路由守卫实现
* 组件级权限通过 HOC 或自定义 Hook 实现

```typescript
// 路由守卫示例
export const AuthGuard: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useAuthStore();
  
  if (!isAuthenticated) {
    return <LoginPage />;
  }
  
  return <>{children}</>;
};
```

---

## 🔄 前后端API联调对接规范

### 环境配置

#### 后端配置

* 后端服务默认运行在 `http://localhost:8080`
* CORS 配置允许前端域名访问
* 支持 `GET, POST, PUT, DELETE, OPTIONS` 方法
* 允许 `Content-Type, Authorization` 请求头

#### 前端配置

* API 基础地址通过环境变量 `NEXT_PUBLIC_API_URL` 配置
* 开发环境默认：`http://localhost:8080`
* 生产环境需要配置实际的后端服务地址

```typescript
// .env.local
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### API 版本管理

#### 版本控制策略

* API 版本通过 URL 路径进行管理：`/api/v1/tickets`
* 当前版本：v1
* 向后兼容原则：新版本不能破坏现有客户端
* 废弃版本提前通知，保留至少一个版本周期

```typescript
// 前端 API 版本配置
export const API_CONFIG = {
  BASE_URL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080',
  VERSION: 'v1',
  TIMEOUT: 30000, // 30秒超时
};

export const getApiUrl = (endpoint: string) => 
  `${API_CONFIG.BASE_URL}/api/${API_CONFIG.VERSION}${endpoint}`;
```

```go
// 后端版本路由组织
func SetupRoutes(r *gin.Engine) {
    // API v1 路由组
    v1 := r.Group("/api/v1")
    {
        // 公共路由（无需认证）
        public := v1.Group("")
        {
            public.POST("/login", authController.Login)
            public.POST("/refresh-token", authController.RefreshToken)
        }
        
        // 认证路由（需要 JWT）
        auth := v1.Group("").Use(middleware.AuthMiddleware())
        {
            auth.GET("/tickets", ticketController.GetTickets)
            auth.POST("/tickets", ticketController.CreateTicket)
        }
    }
}
```

### 性能优化规范

#### 前端性能优化

* **请求优化**：
  * 使用 React Query 或 SWR 进行数据缓存
  * 实现请求去重和防抖
  * 分页加载大数据集
  * 图片懒加载

```typescript
// 使用 SWR 进行数据缓存
import useSWR from 'swr';

export const useTickets = (page: number = 1, limit: number = 20) => {
  const { data, error, mutate } = useSWR(
    `/api/tickets?page=${page}&limit=${limit}`,
    TicketApi.getTickets,
    {
      revalidateOnFocus: false,
      dedupingInterval: 5000, // 5秒内去重
    }
  );
  
  return {
    tickets: data?.data,
    isLoading: !error && !data,
    isError: error,
    refresh: mutate,
  };
};
```

* **组件优化**：
  * 使用 React.memo 避免不必要的重渲染
  * 合理使用 useMemo 和 useCallback
  * 代码分割和懒加载

```typescript
// 组件优化示例
const TicketList = React.memo<TicketListProps>(({ tickets, onUpdate }) => {
  const handleTicketClick = useCallback((ticketId: number) => {
    // 处理点击事件
  }, []);
  
  const sortedTickets = useMemo(() => 
    tickets.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()),
    [tickets]
  );
  
  return (
    <div className="space-y-4">
      {sortedTickets.map(ticket => (
        <TicketCard 
          key={ticket.id} 
          ticket={ticket} 
          onClick={handleTicketClick}
        />
      ))}
    </div>
  );
});
```

#### 后端性能优化

* **数据库优化**：
  * 合理使用索引
  * 避免 N+1 查询问题
  * 使用分页查询
  * 数据库连接池配置

```go
// 避免 N+1 查询，使用 Ent 的 With 方法
func (s *TicketService) GetTicketsWithDetails(ctx context.Context, page, limit int) ([]*ent.Ticket, error) {
    return s.client.Ticket.Query().
        WithRequester().  // 预加载请求者信息
        WithAssignee().   // 预加载处理人信息
        Offset((page - 1) * limit).
        Limit(limit).
        Order(ent.Desc(ticket.FieldCreatedAt)).
        All(ctx)
}
```

* **缓存策略**：
  * Redis 缓存热点数据
  * 设置合理的过期时间
  * 缓存穿透和雪崩防护

```go
// Redis 缓存示例
func (s *TicketService) GetTicketFromCache(ctx context.Context, ticketID int) (*ent.Ticket, error) {
    cacheKey := fmt.Sprintf("ticket:%d", ticketID)
    
    // 尝试从缓存获取
    cached, err := s.redis.Get(ctx, cacheKey).Result()
    if err == nil {
        var ticket ent.Ticket
        if err := json.Unmarshal([]byte(cached), &ticket); err == nil {
            return &ticket, nil
        }
    }
    
    // 缓存未命中，从数据库获取
    ticket, err := s.client.Ticket.Get(ctx, ticketID)
    if err != nil {
        return nil, err
    }
    
    // 写入缓存
    ticketJSON, _ := json.Marshal(ticket)
    s.redis.Set(ctx, cacheKey, ticketJSON, 5*time.Minute)
    
    return ticket, nil
}
```

### 监控与告警

#### 前端监控

* **错误监控**：
  * 集成 Sentry 进行错误追踪
  * 记录用户操作路径
  * 性能指标监控

```typescript
// 错误边界组件
class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false };
  }
  
  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true };
  }
  
  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    // 发送错误到监控系统
    console.error('React Error Boundary caught an error:', error, errorInfo);
    // Sentry.captureException(error, { extra: errorInfo });
  }
  
  render() {
    if (this.state.hasError) {
      return (
        <div className="min-h-screen flex items-center justify-center">
          <div className="text-center">
            <h2 className="text-xl font-semibold text-gray-900 mb-2">出现了一些问题</h2>
            <p className="text-gray-600 mb-4">页面加载失败，请刷新重试</p>
            <button 
              onClick={() => window.location.reload()}
              className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
            >
              刷新页面
            </button>
          </div>
        </div>
      );
    }
    
    return this.props.children;
  }
}
```

#### 后端监控

* **日志规范**：
  * 结构化日志记录
  * 请求链路追踪
  * 关键业务操作审计

```go
// 结构化日志中间件
func LoggerMiddleware() gin.HandlerFunc {
    return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
        logData := map[string]interface{}{
            "timestamp":   param.TimeStamp.Format(time.RFC3339),
            "method":      param.Method,
            "path":        param.Path,
            "status_code": param.StatusCode,
            "latency":     param.Latency.String(),
            "client_ip":   param.ClientIP,
            "user_agent":  param.Request.UserAgent(),
        }
        
        // 添加用户信息（如果已认证）
        if userID, exists := param.Keys["user_id"]; exists {
            logData["user_id"] = userID
        }
        
        logJSON, _ := json.Marshal(logData)
        return string(logJSON) + "\n"
    })
}
```

* **健康检查**：
  * 数据库连接检查
  * Redis 连接检查
  * 外部服务依赖检查

```go
// 健康检查端点
func (h *HealthController) HealthCheck(c *gin.Context) {
    health := map[string]interface{}{
        "status":    "healthy",
        "timestamp": time.Now().Unix(),
        "version":   "1.0.0",
        "checks": map[string]interface{}{
            "database": h.checkDatabase(),
            "redis":    h.checkRedis(),
        },
    }
    
    // 如果任何检查失败，返回 503
    allHealthy := true
    for _, check := range health["checks"].(map[string]interface{}) {
        if check.(map[string]interface{})["status"] != "healthy" {
            allHealthy = false
            break
        }
    }
    
    if !allHealthy {
        health["status"] = "unhealthy"
        c.JSON(503, health)
        return
    }
    
    c.JSON(200, health)
}
```

### 安全规范

#### 数据安全

* **敏感数据处理**：
  * 密码必须加密存储
  * 个人信息脱敏显示
  * API 响应不包含敏感字段

```go
// 用户信息脱敏
type UserResponse struct {
    ID       int    `json:"id"`
    Username string `json:"username"`
    Email    string `json:"email"`
    Name     string `json:"name"`
    // 不包含密码等敏感信息
}

func (u *User) ToResponse() *UserResponse {
    return &UserResponse{
        ID:       u.ID,
        Username: u.Username,
        Email:    maskEmail(u.Email), // 邮箱脱敏
        Name:     u.Name,
    }
}
```

#### 输入验证

* **前端验证**：
  * 表单数据客户端验证
  * XSS 防护
  * 文件上传安全检查

```typescript
// 输入验证示例
export const validateTicketForm = (data: CreateTicketRequest): ValidationResult => {
  const errors: Record<string, string> = {};
  
  if (!data.title || data.title.trim().length < 2) {
    errors.title = '标题至少需要2个字符';
  }
  
  if (data.title && data.title.length > 200) {
    errors.title = '标题不能超过200个字符';
  }
  
  if (!data.description || data.description.trim().length < 10) {
    errors.description = '描述至少需要10个字符';
  }
  
  // XSS 防护 - 清理 HTML 标签
  if (data.description && /<script|javascript:/i.test(data.description)) {
    errors.description = '描述包含不安全的内容';
  }
  
  return {
    isValid: Object.keys(errors).length === 0,
    errors,
  };
};
```

* **后端验证**：
  * 严格的参数验证
  * SQL 注入防护
  * 权限验证

```go
// 后端参数验证
type CreateTicketRequest struct {
    Title       string                 `json:"title" binding:"required,min=2,max=200"`
    Description string                 `json:"description" binding:"required,min=10,max=5000"`
    Priority    string                 `json:"priority" binding:"required,oneof=low medium high critical"`
    FormFields  map[string]interface{} `json:"form_fields"`
}

func (tc *TicketController) CreateTicket(c *gin.Context) {
    var req CreateTicketRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        common.Fail(c, 1001, "参数验证失败: "+err.Error())
        return
    }
    
    // 额外的业务验证
    if err := tc.validateTicketData(&req); err != nil {
        common.Fail(c, 1002, err.Error())
        return
    }
    
    // 权限验证
    userID := c.GetInt("user_id")
    if !tc.authService.CanCreateTicket(userID) {
        common.Fail(c, 2003, "无权限创建工单")
        return
    }
    
    // 创建工单逻辑...
}
```

---

* ... existing code ...
