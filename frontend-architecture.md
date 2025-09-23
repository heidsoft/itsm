# ITSM 前端架构设计

## 1. 技术栈选型

### 核心框架
- **Next.js 14**: React全栈框架，支持SSR/SSG
- **TypeScript**: 类型安全的JavaScript超集
- **Tailwind CSS**: 原子化CSS框架
- **React Query/TanStack Query**: 数据获取和状态管理
- **Zustand**: 轻量级状态管理
- **React Hook Form**: 表单处理
- **Zod**: 数据验证

### UI组件库
- **Shadcn/ui**: 基于Radix UI的组件库
- **Lucide React**: 图标库
- **Recharts**: 图表库
- **React Table**: 表格组件

## 2. 项目结构设计

```
src/
├── app/                    # Next.js App Router
│   ├── (auth)/            # 认证相关页面组
│   │   ├── login/
│   │   └── register/
│   ├── (dashboard)/       # 主应用页面组
│   │   ├── dashboard/     # 仪表板
│   │   ├── incidents/     # 事件管理
│   │   ├── tickets/       # 工单管理
│   │   ├── problems/      # 问题管理
│   │   ├── changes/       # 变更管理
│   │   ├── cmdb/          # 配置管理
│   │   ├── workflow/      # 工作流管理
│   │   ├── reports/       # 报表分析
│   │   └── admin/         # 系统管理
│   ├── api/               # API路由
│   ├── globals.css        # 全局样式
│   ├── layout.tsx         # 根布局
│   └── page.tsx           # 首页
├── components/            # 可复用组件
│   ├── ui/               # 基础UI组件
│   ├── forms/            # 表单组件
│   ├── charts/           # 图表组件
│   ├── tables/           # 表格组件
│   ├── layout/           # 布局组件
│   └── business/         # 业务组件
├── lib/                  # 工具库
│   ├── api/              # API客户端
│   ├── auth/             # 认证相关
│   ├── utils/            # 工具函数
│   ├── validations/      # 数据验证
│   └── constants/        # 常量定义
├── hooks/                # 自定义Hooks
├── stores/               # 状态管理
├── types/                # TypeScript类型定义
└── styles/               # 样式文件
```

## 3. 组件架构设计

### 3.1 组件分层

```typescript
// 基础UI组件层 (components/ui/)
export interface ButtonProps {
  variant?: 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost' | 'link'
  size?: 'default' | 'sm' | 'lg' | 'icon'
  children: React.ReactNode
}

// 业务组件层 (components/business/)
export interface TicketCardProps {
  ticket: Ticket
  onStatusChange: (ticketId: string, status: TicketStatus) => void
  onAssign: (ticketId: string, assigneeId: string) => void
}

// 页面组件层 (app/)
export interface TicketListPageProps {
  searchParams: {
    status?: string
    priority?: string
    assignee?: string
    page?: string
  }
}
```

### 3.2 表单组件设计

```typescript
// components/forms/TicketForm.tsx
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { ticketSchema } from '@/lib/validations/ticket'

interface TicketFormProps {
  initialData?: Partial<Ticket>
  onSubmit: (data: TicketFormData) => Promise<void>
  mode: 'create' | 'edit'
}

export function TicketForm({ initialData, onSubmit, mode }: TicketFormProps) {
  const form = useForm<TicketFormData>({
    resolver: zodResolver(ticketSchema),
    defaultValues: initialData
  })

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
        <FormField
          control={form.control}
          name="title"
          render={({ field }) => (
            <FormItem>
              <FormLabel>工单标题</FormLabel>
              <FormControl>
                <Input placeholder="请输入工单标题" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        {/* 其他表单字段 */}
      </form>
    </Form>
  )
}
```

### 3.3 表格组件设计

```typescript
// components/tables/TicketTable.tsx
import { useReactTable, getCoreRowModel, ColumnDef } from '@tanstack/react-table'

interface TicketTableProps {
  data: Ticket[]
  loading?: boolean
  onRowClick?: (ticket: Ticket) => void
}

export const ticketColumns: ColumnDef<Ticket>[] = [
  {
    accessorKey: 'id',
    header: '工单ID',
    cell: ({ row }) => (
      <Badge variant="outline">{row.getValue('id')}</Badge>
    )
  },
  {
    accessorKey: 'title',
    header: '标题',
    cell: ({ row }) => (
      <div className="max-w-[200px] truncate">
        {row.getValue('title')}
      </div>
    )
  },
  {
    accessorKey: 'status',
    header: '状态',
    cell: ({ row }) => (
      <StatusBadge status={row.getValue('status')} />
    )
  }
]

export function TicketTable({ data, loading, onRowClick }: TicketTableProps) {
  const table = useReactTable({
    data,
    columns: ticketColumns,
    getCoreRowModel: getCoreRowModel()
  })

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          {table.getHeaderGroups().map((headerGroup) => (
            <TableRow key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <TableHead key={header.id}>
                  {header.isPlaceholder
                    ? null
                    : flexRender(header.column.columnDef.header, header.getContext())
                  }
                </TableHead>
              ))}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {loading ? (
            <TableRow>
              <TableCell colSpan={ticketColumns.length} className="h-24 text-center">
                <Spinner />
              </TableCell>
            </TableRow>
          ) : table.getRowModel().rows?.length ? (
            table.getRowModel().rows.map((row) => (
              <TableRow
                key={row.id}
                className="cursor-pointer hover:bg-muted/50"
                onClick={() => onRowClick?.(row.original)}
              >
                {row.getVisibleCells().map((cell) => (
                  <TableCell key={cell.id}>
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </TableCell>
                ))}
              </TableRow>
            ))
          ) : (
            <TableRow>
              <TableCell colSpan={ticketColumns.length} className="h-24 text-center">
                暂无数据
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  )
}
```

## 4. 状态管理设计

### 4.1 全局状态 (Zustand)

```typescript
// stores/auth.ts
interface AuthState {
  user: User | null
  token: string | null
  isAuthenticated: boolean
  login: (credentials: LoginCredentials) => Promise<void>
  logout: () => void
  refreshToken: () => Promise<void>
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  token: null,
  isAuthenticated: false,
  
  login: async (credentials) => {
    try {
      const response = await authApi.login(credentials)
      set({
        user: response.user,
        token: response.token,
        isAuthenticated: true
      })
      localStorage.setItem('token', response.token)
    } catch (error) {
      throw error
    }
  },
  
  logout: () => {
    set({ user: null, token: null, isAuthenticated: false })
    localStorage.removeItem('token')
  },
  
  refreshToken: async () => {
    try {
      const response = await authApi.refreshToken()
      set({ token: response.token })
      localStorage.setItem('token', response.token)
    } catch (error) {
      get().logout()
      throw error
    }
  }
}))

// stores/ui.ts
interface UIState {
  sidebarCollapsed: boolean
  theme: 'light' | 'dark' | 'system'
  notifications: Notification[]
  toggleSidebar: () => void
  setTheme: (theme: 'light' | 'dark' | 'system') => void
  addNotification: (notification: Omit<Notification, 'id'>) => void
  removeNotification: (id: string) => void
}

export const useUIStore = create<UIState>((set) => ({
  sidebarCollapsed: false,
  theme: 'system',
  notifications: [],
  
  toggleSidebar: () => set((state) => ({ 
    sidebarCollapsed: !state.sidebarCollapsed 
  })),
  
  setTheme: (theme) => set({ theme }),
  
  addNotification: (notification) => set((state) => ({
    notifications: [...state.notifications, { 
      ...notification, 
      id: crypto.randomUUID() 
    }]
  })),
  
  removeNotification: (id) => set((state) => ({
    notifications: state.notifications.filter(n => n.id !== id)
  }))
}))
```

### 4.2 服务端状态 (React Query)

```typescript
// hooks/useTickets.ts
export function useTickets(params?: TicketQueryParams) {
  return useQuery({
    queryKey: ['tickets', params],
    queryFn: () => ticketApi.getTickets(params),
    staleTime: 5 * 60 * 1000, // 5分钟
    gcTime: 10 * 60 * 1000,   // 10分钟
  })
}

export function useTicket(id: string) {
  return useQuery({
    queryKey: ['ticket', id],
    queryFn: () => ticketApi.getTicket(id),
    enabled: !!id,
  })
}

export function useCreateTicket() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ticketApi.createTicket,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tickets'] })
      toast.success('工单创建成功')
    },
    onError: (error) => {
      toast.error('工单创建失败: ' + error.message)
    }
  })
}

export function useUpdateTicket() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateTicketData }) =>
      ticketApi.updateTicket(id, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: ['ticket', id] })
      queryClient.invalidateQueries({ queryKey: ['tickets'] })
      toast.success('工单更新成功')
    },
    onError: (error) => {
      toast.error('工单更新失败: ' + error.message)
    }
  })
}
```

## 5. 路由设计

### 5.1 App Router 结构

```typescript
// app/layout.tsx - 根布局
export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="zh-CN">
      <body className={cn("min-h-screen bg-background font-sans antialiased", fontSans.variable)}>
        <Providers>
          <div className="relative flex min-h-screen flex-col">
            <Header />
            <div className="flex-1">{children}</div>
            <Footer />
          </div>
          <Toaster />
        </Providers>
      </body>
    </html>
  )
}

// app/(dashboard)/layout.tsx - 仪表板布局
export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="flex h-screen">
      <Sidebar />
      <main className="flex-1 overflow-y-auto">
        <div className="container mx-auto p-6">
          {children}
        </div>
      </main>
    </div>
  )
}
```

### 5.2 页面组件设计

```typescript
// app/(dashboard)/tickets/page.tsx
interface TicketListPageProps {
  searchParams: {
    status?: string
    priority?: string
    assignee?: string
    page?: string
  }
}

export default function TicketListPage({ searchParams }: TicketListPageProps) {
  const [filters, setFilters] = useState<TicketFilters>({
    status: searchParams.status,
    priority: searchParams.priority,
    assignee: searchParams.assignee,
  })
  
  const { data: tickets, isLoading, error } = useTickets({
    ...filters,
    page: Number(searchParams.page) || 1,
    limit: 20
  })

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">工单管理</h1>
        <Button asChild>
          <Link href="/tickets/new">
            <Plus className="mr-2 h-4 w-4" />
            新建工单
          </Link>
        </Button>
      </div>
      
      <TicketFilters filters={filters} onFiltersChange={setFilters} />
      
      {error ? (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>错误</AlertTitle>
          <AlertDescription>
            加载工单列表失败: {error.message}
          </AlertDescription>
        </Alert>
      ) : (
        <TicketTable 
          data={tickets?.data || []} 
          loading={isLoading}
          onRowClick={(ticket) => router.push(`/tickets/${ticket.id}`)}
        />
      )}
      
      {tickets && (
        <Pagination
          currentPage={tickets.pagination.page}
          totalPages={tickets.pagination.totalPages}
          onPageChange={(page) => {
            const params = new URLSearchParams(searchParams)
            params.set('page', page.toString())
            router.push(`/tickets?${params.toString()}`)
          }}
        />
      )}
    </div>
  )
}

// app/(dashboard)/tickets/[id]/page.tsx
interface TicketDetailPageProps {
  params: { id: string }
}

export default function TicketDetailPage({ params }: TicketDetailPageProps) {
  const { data: ticket, isLoading, error } = useTicket(params.id)
  const updateTicket = useUpdateTicket()

  if (isLoading) return <TicketDetailSkeleton />
  if (error) return <ErrorPage error={error} />
  if (!ticket) return <NotFoundPage />

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Button variant="ghost" size="sm" asChild>
            <Link href="/tickets">
              <ArrowLeft className="mr-2 h-4 w-4" />
              返回列表
            </Link>
          </Button>
          <h1 className="text-3xl font-bold">{ticket.title}</h1>
        </div>
        <TicketActions ticket={ticket} />
      </div>
      
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-6">
          <TicketContent ticket={ticket} />
          <TicketComments ticketId={ticket.id} />
        </div>
        <div className="space-y-6">
          <TicketSidebar 
            ticket={ticket} 
            onUpdate={(data) => updateTicket.mutate({ id: ticket.id, data })}
          />
        </div>
      </div>
    </div>
  )
}
```

## 6. API客户端设计

### 6.1 HTTP客户端封装

```typescript
// lib/api/client.ts
class ApiClient {
  private baseURL: string
  private defaultHeaders: Record<string, string>

  constructor(baseURL: string) {
    this.baseURL = baseURL
    this.defaultHeaders = {
      'Content-Type': 'application/json',
    }
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<ApiResponse<T>> {
    const url = `${this.baseURL}${endpoint}`
    const token = useAuthStore.getState().token

    const config: RequestInit = {
      ...options,
      headers: {
        ...this.defaultHeaders,
        ...(token && { Authorization: `Bearer ${token}` }),
        ...options.headers,
      },
    }

    try {
      const response = await fetch(url, config)
      
      if (!response.ok) {
        const error = await response.json().catch(() => ({}))
        throw new ApiError(response.status, error.message || response.statusText)
      }

      const data = await response.json()
      return data
    } catch (error) {
      if (error instanceof ApiError) {
        throw error
      }
      throw new ApiError(0, '网络请求失败')
    }
  }

  async get<T>(endpoint: string, params?: Record<string, any>): Promise<ApiResponse<T>> {
    const url = params ? `${endpoint}?${new URLSearchParams(params)}` : endpoint
    return this.request<T>(url, { method: 'GET' })
  }

  async post<T>(endpoint: string, data?: any): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    })
  }

  async put<T>(endpoint: string, data?: any): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, {
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    })
  }

  async delete<T>(endpoint: string): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, { method: 'DELETE' })
  }
}

export const apiClient = new ApiClient(process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api')
```

### 6.2 业务API封装

```typescript
// lib/api/ticket.ts
export const ticketApi = {
  getTickets: (params?: TicketQueryParams): Promise<PaginatedResponse<Ticket>> =>
    apiClient.get('/tickets', params),

  getTicket: (id: string): Promise<Ticket> =>
    apiClient.get(`/tickets/${id}`),

  createTicket: (data: CreateTicketData): Promise<Ticket> =>
    apiClient.post('/tickets', data),

  updateTicket: (id: string, data: UpdateTicketData): Promise<Ticket> =>
    apiClient.put(`/tickets/${id}`, data),

  deleteTicket: (id: string): Promise<void> =>
    apiClient.delete(`/tickets/${id}`),

  assignTicket: (id: string, assigneeId: string): Promise<Ticket> =>
    apiClient.put(`/tickets/${id}/assign`, { assigneeId }),

  updateStatus: (id: string, status: TicketStatus): Promise<Ticket> =>
    apiClient.put(`/tickets/${id}/status`, { status }),

  addComment: (id: string, content: string): Promise<Comment> =>
    apiClient.post(`/tickets/${id}/comments`, { content }),

  getComments: (id: string): Promise<Comment[]> =>
    apiClient.get(`/tickets/${id}/comments`),
}
```

## 7. 性能优化策略

### 7.1 代码分割和懒加载

```typescript
// 页面级别的代码分割
const TicketDetailPage = lazy(() => import('./TicketDetailPage'))
const ReportsPage = lazy(() => import('./ReportsPage'))

// 组件级别的懒加载
const ChartComponent = lazy(() => import('@/components/charts/ChartComponent'))

// 使用Suspense包装
<Suspense fallback={<PageSkeleton />}>
  <TicketDetailPage />
</Suspense>
```

### 7.2 数据缓存和预取

```typescript
// 预取相关数据
export function useTicketWithPrefetch(id: string) {
  const queryClient = useQueryClient()
  
  const ticketQuery = useQuery({
    queryKey: ['ticket', id],
    queryFn: () => ticketApi.getTicket(id),
  })

  // 预取评论数据
  useEffect(() => {
    if (ticketQuery.data) {
      queryClient.prefetchQuery({
        queryKey: ['ticket-comments', id],
        queryFn: () => ticketApi.getComments(id),
      })
    }
  }, [ticketQuery.data, id, queryClient])

  return ticketQuery
}
```

### 7.3 虚拟化长列表

```typescript
// components/VirtualizedTable.tsx
import { FixedSizeList as List } from 'react-window'

interface VirtualizedTableProps {
  items: any[]
  height: number
  itemHeight: number
  renderItem: ({ index, style }: { index: number; style: React.CSSProperties }) => React.ReactNode
}

export function VirtualizedTable({ items, height, itemHeight, renderItem }: VirtualizedTableProps) {
  return (
    <List
      height={height}
      itemCount={items.length}
      itemSize={itemHeight}
      itemData={items}
    >
      {renderItem}
    </List>
  )
}
```

## 8. 错误处理和用户体验

### 8.1 错误边界

```typescript
// components/ErrorBoundary.tsx
interface ErrorBoundaryState {
  hasError: boolean
  error?: Error
}

export class ErrorBoundary extends Component<
  { children: ReactNode; fallback?: ComponentType<{ error: Error }> },
  ErrorBoundaryState
> {
  constructor(props: any) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('Error caught by boundary:', error, errorInfo)
    // 发送错误报告到监控服务
  }

  render() {
    if (this.state.hasError) {
      const FallbackComponent = this.props.fallback || DefaultErrorFallback
      return <FallbackComponent error={this.state.error!} />
    }

    return this.props.children
  }
}
```

### 8.2 加载状态和骨架屏

```typescript
// components/skeletons/TicketListSkeleton.tsx
export function TicketListSkeleton() {
  return (
    <div className="space-y-4">
      {Array.from({ length: 10 }).map((_, i) => (
        <div key={i} className="flex items-center space-x-4 p-4 border rounded-lg">
          <Skeleton className="h-4 w-16" />
          <Skeleton className="h-4 w-48" />
          <Skeleton className="h-4 w-20" />
          <Skeleton className="h-4 w-24" />
        </div>
      ))}
    </div>
  )
}
```

## 9. 国际化支持

```typescript
// lib/i18n.ts
import { createInstance } from 'i18next'
import { initReactI18next } from 'react-i18next'

const i18n = createInstance()

i18n
  .use(initReactI18next)
  .init({
    lng: 'zh-CN',
    fallbackLng: 'en',
    resources: {
      'zh-CN': {
        translation: {
          'ticket.title': '工单标题',
          'ticket.status': '状态',
          'ticket.priority': '优先级',
        }
      },
      'en': {
        translation: {
          'ticket.title': 'Ticket Title',
          'ticket.status': 'Status',
          'ticket.priority': 'Priority',
        }
      }
    }
  })

export default i18n
```

## 10. 测试策略

### 10.1 单元测试

```typescript
// __tests__/components/TicketCard.test.tsx
import { render, screen, fireEvent } from '@testing-library/react'
import { TicketCard } from '@/components/business/TicketCard'

describe('TicketCard', () => {
  const mockTicket = {
    id: '1',
    title: 'Test Ticket',
    status: 'open',
    priority: 'high'
  }

  it('renders ticket information correctly', () => {
    render(<TicketCard ticket={mockTicket} />)
    
    expect(screen.getByText('Test Ticket')).toBeInTheDocument()
    expect(screen.getByText('open')).toBeInTheDocument()
    expect(screen.getByText('high')).toBeInTheDocument()
  })

  it('calls onStatusChange when status is updated', () => {
    const mockOnStatusChange = jest.fn()
    render(
      <TicketCard 
        ticket={mockTicket} 
        onStatusChange={mockOnStatusChange}
      />
    )
    
    fireEvent.click(screen.getByRole('button', { name: /change status/i }))
    expect(mockOnStatusChange).toHaveBeenCalledWith('1', 'in-progress')
  })
})
```

### 10.2 集成测试

```typescript
// __tests__/pages/tickets.test.tsx
import { render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import TicketListPage from '@/app/(dashboard)/tickets/page'

const createTestQueryClient = () => new QueryClient({
  defaultOptions: {
    queries: { retry: false },
    mutations: { retry: false },
  },
})

describe('TicketListPage', () => {
  it('displays tickets when loaded successfully', async () => {
    const queryClient = createTestQueryClient()
    
    render(
      <QueryClientProvider client={queryClient}>
        <TicketListPage searchParams={{}} />
      </QueryClientProvider>
    )

    await waitFor(() => {
      expect(screen.getByText('工单管理')).toBeInTheDocument()
    })
  })
})
```

这个前端架构设计提供了完整的技术栈选型、项目结构、组件设计、状态管理、路由设计、API客户端、性能优化、错误处理、国际化和测试策略，为ITSM系统的前端开发提供了详细的指导。