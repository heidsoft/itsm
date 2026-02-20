# ITSM项目开发经验总结

本文档记录ITSM项目开发过程中积累的可复用经验。

---

## 一、Next.js + TypeScript 项目初始化

### 1.1 项目脚手架命令

```bash
# 创建 Next.js 项目
npx create-next-app@latest itsm-frontend \
  --typescript \
  --tailwind \
  --eslint \
  --app \
  --src-dir \
  --import-alias "@/*" \
  --no-turbopack

# 安装核心依赖
npm install antd @ant-design/pro-components \
  zustand @tanstack/react-query \
  axios dayjs \
  lucide-react

# 安装开发依赖
npm install -D @types/node @types/react @types/react-dom \
  jest @types/jest @testing-library/react @testing-library/jest-dom \
  @playwright/test \
  eslint eslint-config-prettier
```

### 1.2 tsconfig.json 配置

```json
{
  "compilerOptions": {
    "target": "ES2017",
    "lib": ["dom", "dom.iterable", "esnext"],
    "allowJs": true,
    "skipLibCheck": true,
    "strict": true,
    "noEmit": true,
    "esModuleInterop": true,
    "module": "esnext",
    "moduleResolution": "bundler",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "jsx": "preserve",
    "incremental": true,
    "plugins": [{ "name": "next" }],
    "paths": {
      "@/*": ["./src/*"]
    }
  }
}
```

---

## 二、API层设计模式

### 2.1 HttpClient 统一封装

```typescript
// src/lib/api/http-client.ts
import axios, { AxiosInstance, AxiosRequestConfig } from 'axios';

class HttpClient {
  private client: AxiosInstance;
  private token: string | null = null;
  private tenantId: number | null = null;

  constructor() {
    this.client = axios.create({
      baseURL: process.env.NEXT_PUBLIC_API_URL || '/api/v1',
      timeout: 30000,
      headers: { 'Content-Type': 'application/json' },
    });

    this.client.interceptors.request.use((config) => {
      if (this.token) config.headers.Authorization = `Bearer ${this.token}`;
      if (this.tenantId) config.headers['X-Tenant-ID'] = String(this.tenantId);
      return config;
    });

    this.client.interceptors.response.use(
      (response) => {
        const { code, message, data } = response.data;
        if (code === 0) return data;
        throw new Error(message || 'Request failed');
      },
      (error) => {
        console.error('[ERROR] "Request failed:', error);
        throw error;
      }
    );
  }

  setToken(token: string) { this.token = token; }
  setTenantId(id: number) { this.tenantId = id; }

  async get<T>(url: string, params?: object) {
    return this.client.get<T>(url, { params }).then((r) => r.data);
  }
  async post<T>(url: string, data?: object) {
    return this.client.post<T>(url, data).then((r) => r.data);
  }
  async put<T>(url: string, data?: object) {
    return this.client.put<T>(url, data).then((r) => r.data);
  }
  async delete<T>(url: string) {
    return this.client.delete<T>(url).then((r) => r.data);
  }
}

export const httpClient = new HttpClient();
```

### 2.2 API 服务层封装

```typescript
// src/lib/api/ticket-api.ts
import { httpClient } from './http-client';

interface Ticket {
  id: number;
  title: string;
  status: string;
}

export const TicketApi = {
  list: (params?: { page?: number; pageSize?: number; status?: string }) =>
    httpClient.get<{ tickets: Ticket[]; total: number }>('/tickets', params),

  get: (id: number) =>
    httpClient.get<Ticket>(`/tickets/${id}`),

  create: (data: { title: string; description: string }) =>
    httpClient.post<Ticket>('/tickets', data),

  update: (id: number, data: Partial<Ticket>) =>
    httpClient.put<Ticket>(`/tickets/${id}`, data),

  delete: (id: number) =>
    httpClient.delete(`/tickets/${id}`),
};
```

---

## 三、状态管理方案

### 3.1 Zustand 全局状态

```typescript
// src/lib/store/auth-store.ts
import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface User {
  id: number;
  name: string;
  email: string;
}

interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  login: (user: User, token: string) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      token: null,
      isAuthenticated: false,
      login: (user, token) =>
        set({ user, token, isAuthenticated: true }),
      logout: () =>
        set({ user: null, token: null, isAuthenticated: false }),
    }),
    { name: 'auth-storage' }
  )
);
```

### 3.2 TanStack Query 服务端状态

```typescript
// src/lib/hooks/useTickets.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { TicketApi } from '@/lib/api/ticket-api';

export const useTickets = (params?: { page?: number; status?: string }) =>
  useQuery({
    queryKey: ['tickets', params],
    queryFn: () => TicketApi.list(params),
  });

export const useTicket = (id: number) =>
  useQuery({
    queryKey: ['ticket', id],
    queryFn: () => TicketApi.get(id),
    enabled: !!id,
  });

export const useCreateTicket = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: TicketApi.create,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ['tickets'] }),
  });
};
```

---

## 四、类型定义规范

### 4.1 避免重复导出冲突

```typescript
// src/types/index.ts

// 问题：多个文件导出同名类型导致冲突
// 解决：明确指定导出类型，使用类型别名区分

// API 层类型 (蛇形命名)
export type { UserBasicInfo } from '../lib/api/types';

// 业务层类型 (驼峰命名)
export type {
  UserRole,
  UserStatus,
  User as BusinessUser,
} from './user';

// 组件 Props 类型
export interface TicketListProps {
  tickets: Ticket[];
  onSelect: (id: number) => void;
}
```

### 4.2 统一 API 类型文件

```typescript
// src/lib/api/types.ts

export type TicketPriority = 'low' | 'medium' | 'high' | 'critical';
export type TicketStatus = 'new' | 'open' | 'in_progress' | 'pending' | 'resolved' | 'closed';
export type TicketType = 'incident' | 'problem' | 'change' | 'service_request';

export interface UserBasicInfo {
  id: number;
  name: string;
  username?: string;
  email?: string;
  avatar?: string;
}

export interface Ticket {
  id: number;
  ticket_number: string;
  title: string;
  priority: TicketPriority;
  status: TicketStatus;
  assignee?: UserBasicInfo;
  created_at: string;
}

export interface TicketListResponse {
  tickets: Ticket[];
  total: number;
  page: number;
  size: number;
}
```

---

## 五、测试配置

### 5.1 Jest 配置

```javascript
// jest.config.js
const nextJest = require('next/jest.js');

const createJestConfig = nextJest({
  dir: './',
});

const customJestConfig = {
  setupFilesAfterEnv: ['<rootDir>/jest.setup.js'],
  testEnvironment: 'jest-environment-jsdom',
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
    '^lodash-es$': 'lodash',
  },
  collectCoverageFrom: [
    'src/components/**/*.{ts,tsx}',
    'src/lib/**/*.{ts,tsx}',
  ],
};

module.exports = createJestConfig(customJestConfig);
```

### 5.2 测试工具函数

```typescript
// src/lib/test-utils.tsx

// 模拟API响应
export const createMockApiResponse = <T>(data: T) => ({
  code: 0,
  message: 'success',
  data,
});

// 模拟分页响应
export const createMockPaginatedResponse = <T>(items: T[], page = 1, pageSize = 10) => ({
  items,
  total: items.length,
  page,
  pageSize,
  totalPages: Math.ceil(items.length / pageSize),
});

// 延迟函数
export const delay = (ms: number) => new Promise((r) => setTimeout(r, ms));
```

### 5.3 React Testing Library 示例

```typescript
// src/components/__tests__/Button.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { Button } from '../Button';

describe('Button', () => {
  it('renders correctly', () => {
    render(<Button>Click me</Button>);
    expect(screen.getByRole('button')).toHaveTextContent('Click me');
  });

  it('calls onClick', () => {
    const onClick = jest.fn();
    render(<Button onClick={onClick}>Click me</Button>);
    fireEvent.click(screen.getByRole('button'));
    expect(onClick).toHaveBeenCalledTimes(1);
  });
});
```

---

## 六、CI/CD 配置

### 6.1 GitHub Actions 工作流

```yaml
# .github/workflows/frontend-ci.yml
name: Frontend CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
      - run: npm ci
      - run: npm run lint:check

  type-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
      - run: npm ci
      - run: npm run type-check

  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
      - run: npm ci
      - run: npm test -- --passWithNoTests
```

---

## 七、问题排查经验

### 7.1 TypeScript 类型冲突

**问题**：多个文件导出同名类型导致冲突

**解决方案**：
```typescript
// 避免使用 export * 批量导出
// 改用显式类型导出
export type { UserBasicInfo } from '../lib/api/types';
```

### 7.2 API 404 问题排查流程

1. 检查前端 API 端点路径
2. 对照后端路由定义
3. 确认 HTTP 方法 (GET/POST/PUT/DELETE)
4. 检查认证 Token 是否传递

```bash
# 快速排查命令
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/knowledge/categories
```

### 7.3 ESLint 规则禁用策略

```typescript
// 文件顶部添加 eslint 禁用注释
/* eslint-disable @typescript-eslint/no-explicit-any */
// 或针对特定行
// eslint-disable-next-line @typescript-eslint/no-explicit-any
```

---

## 八、ITSM 业务模型

### 8.1 核心实体关系

```
Ticket (工单)
  ├── Incident (事件) - 临时性问题
  ├── Problem (问题) - 根本原因分析
  ├── Change (变更) - 计划性变更
  └── ServiceRequest (服务请求) - 用户请求

关联关系:
Ticket ←→ Ticket (父子/依赖/相关)
Ticket ←→ KnowledgeArticle (知识关联)
Ticket ←→ ConfigurationItem (CI关联)
```

### 8.2 工单状态流转

```
Incident: new → investigating → identified → monitoring → resolved → closed
Problem: new → in_progress → root_cause_analysis → solution_implemented → closed
Change: draft → pending_approval → approved → in_progress → completed/rolled_back
```

---

## 九、可复用代码模板

### 9.1 标准 CRUD 组件

```typescript
// src/app/components/StandardList.tsx
'use client';

import { useState } from 'react';
import { Table, Button, Space, message } from 'antd';
import type { TableColumn } from '@/components/ui';

interface StandardListProps<T> {
  fetchApi: (params: any) => Promise<{ data: T[]; total: number }>;
  columns: TableColumn<T>[];
  createPath?: string;
  deleteApi?: (id: number) => Promise<void>;
}

export function StandardList<T extends { id: number }>({
  fetchApi,
  columns,
  createPath,
  deleteApi,
}: StandardListProps<T>) {
  const [data, setData] = useState<T[]>([]);
  const [loading, setLoading] = useState(false);
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10, total: 0 });

  const loadData = async () => {
    setLoading(true);
    try {
      const { data: result, total } = await fetchApi(pagination);
      setData(result);
      setPagination((p) => ({ ...p, total }));
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: number) => {
    if (!deleteApi) return;
    await deleteApi(id);
    message.success('删除成功');
    loadData();
  };

  return (
    <Table<T>
      loading={loading}
      dataSource={data}
      rowKey="id"
      pagination={pagination}
      onPaginationChange={(p) => setPagination({ ...pagination, ...p })}
      columns={[
        ...columns,
        {
          title: '操作',
          render: (_, record) => (
            <Space>
              <Button size="small" onClick={() => handleDelete(record.id)}>删除</Button>
            </Space>
          ),
        },
      ]}
    />
  );
}
```

---

## 十、开发检查清单

### 10.1 新增 API

- [ ] 统一在 `src/lib/api/` 目录下创建
- [ ] 使用 HttpClient 封装
- [ ] 添加 TypeScript 类型定义
- [ ] 对应 Hook (useXxx) 在 `src/lib/hooks/`
- [ ] 更新 API 文档

### 10.2 新增组件

- [ ] 组件文件: `src/components/[category]/[Name].tsx`
- [ ] 测试文件: `src/components/[category]/__tests__/[Name].test.tsx`
- [ ] 导出到 `src/components/[category]/index.ts`
- [ ] 添加 displayName
- [ ] Props 类型完整

### 10.3 新增页面

- [ ] 页面文件: `src/app/(main)/[module]/page.tsx`
- [ ] 添加路由守卫 (如需要认证)
- [ ] 权限检查
- [ ] Loading 状态处理
- [ ] Error Boundary 包装

---

## 十一、常见问题速查

| 问题 | 解决方案 |
|------|---------|
| `export *` 冲突 | 改用显式导出 `export type { X }` |
| API 404 | 检查 `/api/v1/` 前缀是否正确 |
| 组件不渲染 | 检查 `use client` 指令 |
| Zustand 状态丢失 | 确认使用 `persist` middleware |
| Jest 测试失败 | 检查 `jest-environment-jsdom` |
| ESLint 警告 | 添加注释或修复代码 |
| 类型错误 | 使用 `// @ts-ignore` 临时解决 |

---

> 文档最后更新: 2024-02
> 维护者: ITSM Development Team
