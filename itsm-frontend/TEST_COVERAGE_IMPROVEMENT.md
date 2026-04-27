# 测试覆盖提升方案

## 现状分析

### 当前测试状况
- **测试文件数**: 32个测试文件
- **测试框架**: Jest + Playwright
- **测试类型**: 单元测试、集成测试、E2E测试、性能测试、UX测试

### Jest配置
```typescript
{
  collectCoverage: true,
  collectCoverageFrom: [
    'src/components/ui/**/*.{ts,tsx}',
    'src/components/common/**/*.{ts,tsx}',
    'src/app/lib/**/*.{ts,tsx}',
    'src/lib/**/*.{ts,tsx}',
  ],
  coverageThreshold: {}, // 暂时未设置阈值
}
```

## 测试策略

### 阶段1: 基础组件测试（目标覆盖率: 60%）

#### UI组件测试优先级
1. **StatCard** - 统计卡片组件
2. **LoadingSpinner** - 加载指示器
3. **LoadingEmptyError** - 状态管理组件
4. **OptimizedImage** - 图片优化组件
5. **GlobalSearch** - 全局搜索组件

#### 示例测试用例
```typescript
// StatCard.test.tsx
import { render, screen } from '@testing-library/react';
import { StatCard } from './StatCard';

describe('StatCard', () => {
  it('应该正确渲染统计数据', () => {
    render(<StatCard title="总工单" value={100} />);
    expect(screen.getByText('总工单')).toBeInTheDocument();
    expect(screen.getByText('100')).toBeInTheDocument();
  });

  it('应该支持自定义颜色', () => {
    render(<StatCard title="测试" value={50} color="#ff0000" />);
    const valueElement = screen.getByText('50');
    expect(valueElement).toHaveStyle({ color: '#ff0000' });
  });

  it('应该显示加载状态', () => {
    render(<StatCard title="测试" value={0} loading />);
    expect(screen.getByRole('status')).toBeInTheDocument();
  });
});
```

### 阶段2: 业务组件测试（目标覆盖率: 80%）

#### 核心业务组件
1. **TicketList** - 工单列表
2. **TicketKanban** - 工单看板
3. **ChangeList** - 变更列表
4. **KnowledgeArticle** - 知识库文章
5. **CMDBList** - 配置项列表

#### 测试要点
- 数据渲染正确性
- 用户交互（点击、筛选、排序）
- 状态变化（加载、错误、空状态）
- API调用mock

### 阶段3: Hooks测试（目标覆盖率: 90%）

#### 关键Hooks
1. **useResponsive** - 响应式检测
2. **useAuth** - 认证状态
3. **useTickets** - 工单数据
4. **useLayoutStore** - 布局状态

#### 示例Hook测试
```typescript
// useResponsive.test.ts
import { renderHook, act } from '@testing-library/react';
import { useResponsive } from './useResponsive';

describe('useResponsive', () => {
  it('应该正确检测移动端', () => {
    window.innerWidth = 500;
    const { result } = renderHook(() => useResponsive());
    expect(result.current.isMobile).toBe(true);
    expect(result.current.isDesktop).toBe(false);
  });

  it('应该正确检测桌面端', () => {
    window.innerWidth = 1200;
    const { result } = renderHook(() => useResponsive());
    expect(result.current.isMobile).toBe(false);
    expect(result.current.isDesktop).toBe(true);
  });
});
```

### 阶段4: 工具函数测试（目标覆盖率: 95%）

#### 工具函数分类
1. **格式化函数** - formatters.ts
2. **验证函数** - validators.ts
3. **工具函数** - component-utils.ts
4. **错误处理** - error-handler.tsx

## 测试模板

### 组件测试模板
```typescript
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ComponentName } from './ComponentName';

describe('ComponentName', () => {
  // 基础渲染测试
  it('应该正确渲染', () => {
    render(<ComponentName />);
    expect(screen.getByRole('region')).toBeInTheDocument();
  });

  // Props测试
  describe('Props', () => {
    it('应该接受title prop', () => {
      render(<ComponentName title="测试标题" />);
      expect(screen.getByText('测试标题')).toBeInTheDocument();
    });
  });

  // 交互测试
  describe('交互', () => {
    it('应该响应点击事件', async () => {
      const handleClick = jest.fn();
      render(<ComponentName onClick={handleClick} />);
      
      await userEvent.click(screen.getByRole('button'));
      expect(handleClick).toHaveBeenCalledTimes(1);
    });
  });

  // 状态测试
  describe('状态', () => {
    it('应该显示加载状态', () => {
      render(<ComponentName loading />);
      expect(screen.getByRole('status')).toBeInTheDocument();
    });

    it('应该显示错误状态', () => {
      render(<ComponentName error="出错了" />);
      expect(screen.getByText('出错了')).toBeInTheDocument();
    });
  });

  // 可访问性测试
  describe('可访问性', () => {
    it('应该有正确的ARIA标签', () => {
      render(<ComponentName />);
      expect(screen.getByRole('button')).toHaveAttribute('aria-label');
    });
  });
});
```

### API Mock模板
```typescript
// __mocks__/api.ts
export const mockTicketApi = {
  getTickets: jest.fn(),
  createTicket: jest.fn(),
  updateTicket: jest.fn(),
  deleteTicket: jest.fn(),
};

// 使用
import { mockTicketApi } from '__mocks__/api';
jest.mock('@/lib/api/ticket-api', () => mockTicketApi);

describe('TicketList', () => {
  beforeEach(() => {
    mockTicketApi.getTickets.mockResolvedValue({
      tickets: [],
      total: 0,
    });
  });
});
```

## 覆盖率目标

### 分阶段目标

| 阶段 | 组件类型 | 当前覆盖率 | 目标覆盖率 | 时间 |
|------|---------|-----------|-----------|------|
| 1 | UI基础组件 | ~20% | 60% | 1周 |
| 2 | 业务组件 | ~10% | 80% | 2周 |
| 3 | Hooks | ~5% | 90% | 1周 |
| 4 | 工具函数 | ~30% | 95% | 1周 |
| **总体** | **全部** | **~15%** | **80%** | **5周** |

### 具体目标

#### 组件覆盖率
```json
{
  "coverageThreshold": {
    "global": {
      "branches": 80,
      "functions": 80,
      "lines": 80,
      "statements": 80
    },
    "./src/components/ui/": {
      "branches": 90,
      "functions": 90,
      "lines": 90,
      "statements": 90
    },
    "./src/lib/": {
      "branches": 85,
      "functions": 85,
      "lines": 85,
      "statements": 85
    }
  }
}
```

## 测试执行

### 运行命令
```bash
# 运行所有测试
npm run test

# 运行特定测试
npm run test ComponentName

# 生成覆盖率报告
npm run test:coverage

# 监听模式
npm run test:watch

# CI模式
npm run test:ci
```

### CI集成
```yaml
# .github/workflows/test.yml
name: Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-node@v2
      - run: npm ci
      - run: npm run test:ci
      - uses: codecov/codecov-action@v2
```

## 测试清单

### 已完成测试
- [x] LoadingSpinner组件
- [ ] StatCard组件
- [ ] LoadingEmptyError组件
- [ ] OptimizedImage组件
- [ ] useResponsive Hook

### 待完成测试
- [ ] TicketList组件
- [ ] TicketKanban组件
- [ ] ChangeList组件
- [ ] 所有表单组件
- [ ] 所有Hooks

## 最佳实践

### 1. 测试命名
```typescript
// ✅ 好的命名
it('应该在点击提交按钮后显示成功消息', () => {});

// ❌ 差的命名
it('test1', () => {});
```

### 2. 测试隔离
```typescript
beforeEach(() => {
  jest.clearAllMocks();
  cleanup();
});
```

### 3. 查询优先级
```typescript
// 优先使用 getByRole
screen.getByRole('button', { name: /提交/i });

// 其次使用 getByLabelText
screen.getByLabelText('用户名');

// 避免使用 getByTestId
screen.getByTestId('submit-button'); // 仅在必要时使用
```

### 4. 异步测试
```typescript
// ✅ 正确
await waitFor(() => {
  expect(screen.getByText('成功')).toBeInTheDocument();
});

// ❌ 错误
setTimeout(() => {
  expect(screen.getByText('成功')).toBeInTheDocument();
}, 1000);
```

---

**更新日期**: 2026-04-26  
**负责人**: 测试团队
