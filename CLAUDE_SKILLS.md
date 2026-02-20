# ITSM Development Skills

本文档以 Skill Card 形式组织，方便 Claude Code 直接使用。

---

## Skill 1: Next.js + TypeScript + Ant Design 项目初始化

**适用场景**: 新建 ITSM 或类似管理后台项目

**执行步骤**:
```bash
# 1. 创建项目
npx create-next-app@latest project-name \
  --typescript --tailwind --eslint --app --src-dir \
  --import-alias "@/*" --no-turbopack

# 2. 安装 UI 库
npm install antd @ant-design/pro-components lucide-react dayjs

# 3. 安装状态管理
npm install zustand @tanstack/react-query axios

# 4. 安装测试
npm install -D jest @types/jest @testing-library/react @testing-library/jest-dom @playwright/test

# 5. 配置 tsconfig.json
# 添加 "paths": {"@/*": ["./src/*"]}
```

**关键配置**:
- `tsconfig.json`: `skipLibCheck: true`, `paths` 别名
- `next.config.js`: 图片域名白名单
- `.eslintrc.json`: 规则定制

---

## Skill 2: API 层封装模式

**适用场景**: 需要统一 API 调用处理

**代码模板**:
```typescript
// HttpClient 封装
class HttpClient {
  private client = axios.create({ baseURL: '/api/v1', timeout: 30000 });

  constructor() {
    this.client.interceptors.request.use((config) => {
      const token = localStorage.getItem('token');
      if (token) config.headers.Authorization = `Bearer ${token}`;
      return config;
    });
    this.client.interceptors.response.use(
      (r) => (r.data.code === 0 ? r.data.data : Promise.reject(r.data.message)),
      (e) => Promise.reject(e)
    );
  }

  get<T>(url: string, params?: object) {
    return this.client.get<T>(url, { params });
  }
  post<T>(url: string, data?: object) {
    return this.client.post<T>(url, data);
  }
}

export const api = new HttpClient();
```

**Skill 调用**:
```
创建 API 文件: src/lib/api/[module]-api.ts
使用: import { ModuleApi } from '@/lib/api/module-api';
```

---

## Skill 3: Zustand 全局状态 + TanStack Query

**适用场景**: 需要客户端缓存 + 服务端状态管理

**代码模板**:
```typescript
// src/lib/store/[module]-store.ts
import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface State { user: User | null; token: string | null }
interface Actions { login: (u: User, t: string) => void; logout: () => void }

export const useStore = create<State & Actions>()(
  persist(
    (set) => ({
      user: null, token: null,
      login: (u, t) => set({ user: u, token: t }),
      logout: () => set({ user: null, token: null }),
    }),
    { name: 'storage' }
  )
);

// src/lib/hooks/use[Module].ts
import { useQuery, useMutation } from '@tanstack/react-query';

export const useData = (id: number) =>
  useQuery({ queryKey: ['key', id], queryFn: () => api.get(`/path/${id}`) });

export const useCreate = () =>
  useMutation({ mutationFn: (data: object) => api.post('/path', data) });
```

---

## Skill 4: 工单 CRUD 页面快速搭建

**适用场景**: 需要新增工单类型管理页面

**代码模板**:
```typescript
// page.tsx
'use client';
import { useState } from 'react';
import { Table, Button, Space, message, Modal, Form, Input, Select } from 'antd';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

export default function ListPage() {
  const [form] = Form.useForm();
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ['tickets'], queryFn: () => api.get('/tickets')
  });

  const create = useMutation({
    mutationFn: (d: object) => api.post('/tickets', d),
    onSuccess: () => { queryClient.invalidateQueries(['tickets']); }
  });

  const columns = [
    { title: '标题', dataIndex: 'title' },
    { title: '状态', dataIndex: 'status' },
    { title: '创建时间', dataIndex: 'created_at' },
  ];

  return (
    <Table loading={isLoading} dataSource={data?.tickets} columns={columns} rowKey="id" />
  );
}
```

---

## Skill 5: BPMN 工作流设计器集成

**适用场景**: 需要可视化工作流编辑功能

**代码模板**:
```typescript
// src/components/workflow/BPMNDesigner.tsx
'use client';
import { useEffect, useRef } from 'react';
import BpmnModeler from 'bpmn-js/lib/Modeler';
import 'bpmn-js/dist/assets/diagram-js.css';
import 'bpmn-js/dist/assets/bpmn-font/css/bpmn.css';

interface Props { xml?: string; onChange: (xml: string) => void }

export function BPMNDesigner({ xml, onChange }: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const modeler = useRef<BpmnModeler | null>(null);

  useEffect(() => {
    if (!ref.current) return;
    modeler.current = new BpmnModeler({ container: ref.current });
    xml && modeler.current.importXML(xml);
    modeler.current.on('commandStack.changed', () => {
      modeler.current?.saveXML({ format: true }).then(({ xml }) => onChange(xml));
    });
  }, []);

  return <div ref={ref} style={{ height: 500 }} />;
}
```

---

## Skill 6: 类型冲突解决

**适用场景**: 多处导出同名类型导致 TS 错误

**解决方案**:
```typescript
// src/types/index.ts

// ❌ 错误：export * 冲突
// export * from './user';
// export * from './api/types';

// ✅ 正确：显式导出
export type { UserRole, UserStatus } from './user';
export type { UserBasicInfo } from '../lib/api/types';

// 使用处区分命名
import type { User as BusinessUser } from '@/types/user';
import type { UserBasicInfo } from '@/lib/api/types';
```

---

## Skill 7: 组件测试 (React Testing Library)

**适用场景**: 确保组件功能正确

**代码模板**:
```typescript
// __tests__/Button.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';

describe('Button', () => {
  it('renders and handles click', () => {
    const onClick = jest.fn();
    render(<Button onClick={onClick}>Click</Button>);
    expect(screen.getByRole('button')).toHaveTextContent('Click');
    fireEvent.click(screen.getByRole('button'));
    expect(onClick).toHaveBeenCalled();
  });
});
```

**运行**: `npm test -- --testPathPattern="Button"`

---

## Skill 8: E2E 测试 (Playwright)

**适用场景**: 端到端用户流程测试

**代码模板**:
```typescript
// e2e/tickets.spec.ts
import { test, expect } from '@playwright/test';

test('create ticket flow', async ({ page }) => {
  await page.goto('/tickets');
  await page.click('text=新建工单');
  await page.fill('input[title*="标题"]', '测试工单');
  await page.click('button:has-text("提交")');
  await expect(page).toHaveURL(/\/\d+/);
});
```

---

## Skill 9: API 404 问题排查

**执行步骤**:
```bash
# 1. 检查前端端点
grep -r "api/v1" src/lib/api/ | grep -v test

# 2. 检查后端路由
grep -r "knowledge" itsm-backend/controller/

# 3. 验证连通性
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/knowledge/articles

# 4. 常见问题:
# - 前端: /api/v1/knowledge/categories
# - 后端: /api/v1/knowledge-articles/categories
# - 修复: 统一路径前缀
```

---

## Skill 10: ESLint 规则禁用

**适用场景**: 需要临时跳过某些检查

**用法**:
```typescript
/* eslint-disable @typescript-eslint/no-explicit-any */
// 代码...
/* eslint-enable @typescript-eslint/no-explicit-any */

// 或针对单行
const data: any = {}; // eslint-disable-line @typescript-eslint/no-explicit-any
```

---

## Skill 11: ITSM 数据模型

**适用场景**: 理解业务实体关系

```
Ticket (工单)
├── Incident (事件)     -- 临时问题
├── Problem (问题)      -- 根本原因
├── Change (变更)       -- 计划变更
└── ServiceRequest     -- 服务请求

关联:
Ticket ←→ KnowledgeArticle (知识关联)
Ticket ←→ ConfigurationItem (CI关联)
Ticket ←→ Ticket (父子/依赖/相关)
```

---

## Skill 12: 权限控制模式

**适用场景**: 基于角色的访问控制

**代码模板**:
```typescript
// src/lib/hooks/usePermission.ts
export function usePermission() {
  const { user } = useAuthStore();

  const hasPermission = (code: string) =>
    user?.permissions?.includes(code) ?? false;

  const hasRole = (role: string) =>
    user?.role === role;

  return { hasPermission, hasRole, user };
}

// 使用
const { hasPermission } = usePermission();
if (!hasPermission('ticket:create')) return null;
```

---

## Skill 13: Loading + Error 状态处理

**适用场景**: API 数据加载状态统一处理

**代码模板**:
```typescript
// src/components/PageWrapper.tsx
export function PageWrapper<T>({
  isLoading,
  error,
  data,
  render,
  loadingText = '加载中...'
}: Props<T>) {
  if (isLoading) return <Spin tip={loadingText} />;
  if (error) return <Alert type="error" message={String(error)} />;
  return render(data);
}

// 使用
<PageWrapper
  isLoading={isLoading}
  error={error}
  data={data}
  render={(d) => <Table dataSource={d?.list} />}
/>
```

---

## Skill 14: 列表页分页模式

**适用场景**: 带分页的表格页面

**代码模板**:
```typescript
const [pagination, setPagination] = useState({ current: 1, pageSize: 10 });
const { data, isLoading } = useQuery({
  queryKey: ['list', pagination],
  queryFn: () => api.get('/list', pagination)
});

const handleChange = (p: typeof pagination) => setPagination(p);

<Table
  loading={isLoading}
  dataSource={data?.list}
  rowKey="id"
  pagination={{ ...pagination, total: data?.total, onChange: handleChange }}
/>
```

---

## Skill 15: GitHub Actions CI 模板

**适用场景**: 自动化检查和测试

**代码模板**:
```yaml
# .github/workflows/ci.yml
name: CI
on: [push, pull_request]
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: '20' }
      - run: npm ci && npm run lint:check
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: '20' }
      - run: npm ci && npm test -- --passWithNoTests
```

---

> 最后更新: 2024-02
