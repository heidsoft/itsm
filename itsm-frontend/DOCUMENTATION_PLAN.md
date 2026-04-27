# 文档完善方案

## 文档体系规划

### 1. Storybook组件文档

#### Storybook配置
```bash
# 安装Storybook
npx storybook@latest init
```

#### 目录结构
```
itsm-frontend/
├── .storybook/
│   ├── main.ts
│   ├── preview.tsx
│   └── theme.ts
├── src/components/
│   ├── ui/
│   │   ├── StatCard/
│   │   │   ├── StatCard.tsx
│   │   │   ├── StatCard.stories.tsx
│   │   │   ├── StatCard.test.tsx
│   │   │   └── StatCard.md
│   │   └── LoadingSpinner/
│   └── business/
│       └── TicketList/
```

#### .storybook/main.ts
```typescript
import type { StorybookConfig } from '@storybook/nextjs';

const config: StorybookConfig = {
  stories: ['../src/**/*.mdx', '../src/**/*.stories.@(js|jsx|mjs|ts|tsx)'],
  addons: [
    '@storybook/addon-links',
    '@storybook/addon-essentials',
    '@storybook/addon-interactions',
    '@storybook/addon-a11y',
    '@storybook/addon-coverage',
  ],
  framework: {
    name: '@storybook/nextjs',
    options: {},
  },
  docs: {
    autodocs: true,
  },
};

export default config;
```

#### .storybook/preview.tsx
```typescript
import type { Preview } from '@storybook/react';
import { AntdRegistry } from '@ant-design/nextjs-registry';
import '../src/styles/globals.css';

const preview: Preview = {
  decorators: [
    (Story) => (
      <AntdRegistry>
        <Story />
      </AntdRegistry>
    ),
  ],
  parameters: {
    actions: { argTypesRegex: '^on[A-Z].*' },
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
  },
};

export default preview;
```

### 2. 组件Story示例

#### StatCard.stories.tsx
```typescript
import type { Meta, StoryObj } from '@storybook/react';
import { StatCard } from './StatCard';

const meta: Meta<typeof StatCard> = {
  title: 'UI/StatCard',
  component: StatCard,
  tags: ['autodocs'],
  argTypes: {
    title: { control: 'text' },
    value: { control: 'number' },
    color: { control: 'color' },
    loading: { control: 'boolean' },
  },
};

export default meta;
type Story = StoryObj<typeof StatCard>;

export const Default: Story = {
  args: {
    title: '总工单',
    value: 1234,
  },
};

export const WithIcon: Story = {
  args: {
    title: '活跃用户',
    value: 567,
    icon: <UserOutlined />,
    color: '#3b82f6',
  },
};

export const Loading: Story = {
  args: {
    title: '加载中',
    value: 0,
    loading: true,
  },
};

export const Error: Story = {
  args: {
    title: '错误',
    value: 0,
    error: '无法加载数据',
  },
};
```

#### TicketList.stories.tsx
```typescript
import type { Meta, StoryObj } from '@storybook/react';
import { TicketList } from './TicketList';

const meta: Meta<typeof TicketList> = {
  title: 'Business/TicketList',
  component: TicketList,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
};

export default meta;
type Story = StoryObj<typeof TicketList>;

export const Default: Story = {
  args: {
    tickets: [
      { id: 1, title: '工单1', status: 'open', priority: 'high' },
      { id: 2, title: '工单2', status: 'resolved', priority: 'medium' },
    ],
  },
};

export const Empty: Story = {
  args: {
    tickets: [],
  },
};

export const Loading: Story = {
  args: {
    loading: true,
  },
};
```

### 3. API文档

#### OpenAPI/Swagger配置
```yaml
# openapi.yaml
openapi: 3.0.0
info:
  title: ITSM API
  version: 1.0.0
  description: ITSM系统API文档

paths:
  /api/tickets:
    get:
      summary: 获取工单列表
      tags: [Tickets]
      parameters:
        - name: page
          in: query
          schema:
            type: integer
            default: 1
        - name: pageSize
          in: query
          schema:
            type: integer
            default: 20
      responses:
        '200':
          description: 成功
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TicketListResponse'
```

#### API文档生成脚本
```bash
#!/bin/bash
# scripts/generate-api-docs.sh

# 从Go后端生成OpenAPI文档
swag init -g main.go -o ./docs

# 转换为前端TypeScript类型
npx openapi-typescript docs/swagger.json -o src/types/api.ts
```

### 4. 组件文档模板

#### README.md模板
```markdown
# ComponentName

简短描述组件的功能和用途。

## 安装

\`\`\`bash
npm install @itsm/components
\`\`\`

## 使用

\`\`\`tsx
import { ComponentName } from '@itsm/components';

function App() {
  return <ComponentName title="示例" />;
}
\`\`\`

## Props

| 属性 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| title | string | - | 标题 |
| loading | boolean | false | 加载状态 |

## 示例

### 基础用法
<Canvas of={Default} />

### 加载状态
<Canvas of={Loading} />

## API

### Props
<ArgsTable of={ComponentName} />

## 可访问性

- 支持键盘导航
- 包含ARIA标签
- 符合WCAG 2.1 AA标准

## 相关组件

- OtherComponent
- AnotherComponent
```

### 5. 文档生成工具

#### 组件文档生成脚本
```typescript
// scripts/generate-component-docs.ts
import fs from 'fs';
import path from 'path';

interface ComponentInfo {
  name: string;
  path: string;
  props: PropInfo[];
}

interface PropInfo {
  name: string;
  type: string;
  required: boolean;
  default: string | undefined;
  description: string;
}

function generateComponentDoc(component: ComponentInfo): string {
  return `# ${component.name}

## Props

| 属性 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
${component.props.map(prop => 
  `| ${prop.name} | \`${prop.type}\` | ${prop.required ? '是' : '否'} | ${prop.default || '-'} | ${prop.description} |`
).join('\n')}

## 示例

\`\`\`tsx
import { ${component.name} } from '@itsm/components';

<${component.name} ${component.props.slice(0, 2).map(p => `${p.name}="${p.name === 'title' ? '示例' : ''}"`).join(' ')} />
\`\`\`
`;
}

// 扫描组件目录
const componentsDir = './src/components';
const componentDirs = fs.readdirSync(componentsDir);

componentDirs.forEach(dir => {
  const componentPath = path.join(componentsDir, dir);
  if (fs.statSync(componentPath).isDirectory()) {
    // 解析TypeScript接口生成文档
    // ...
  }
});
```

### 6. 文档部署

#### GitHub Pages部署
```yaml
# .github/workflows/docs.yml
name: Deploy Docs
on:
  push:
    branches: [main]

jobs:
  deploy-storybook:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-node@v2
      - run: npm ci
      - run: npm run build-storybook
      - uses: peaceiris/actions-gh-pages@v3
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: ./storybook-static
```

### 7. 文档检查清单

#### 组件文档
- [ ] 组件描述
- [ ] Props列表
- [ ] 使用示例
- [ ] Storybook Stories
- [ ] 可访问性说明
- [ ] 单元测试

#### API文档
- [ ] OpenAPI规范
- [ ] TypeScript类型定义
- [ ] 请求/响应示例
- [ ] 错误码说明

#### 开发指南
- [ ] 快速开始
- [ ] 开发环境搭建
- [ ] 编码规范
- [ ] 测试指南
- [ ] 部署指南

## 实施计划

### Week 1: Storybook配置
- [ ] 安装Storybook
- [ ] 配置主题和插件
- [ ] 创建基础Stories

### Week 2: 组件文档
- [ ] UI组件文档（10个）
- [ ] 业务组件文档（20个）
- [ ] 工具函数文档

### Week 3: API文档
- [ ] OpenAPI规范生成
- [ ] TypeScript类型生成
- [ ] API文档部署

### Week 4: 文档部署
- [ ] GitHub Pages配置
- [ ] 自动化部署
- [ ] 文档站点优化

---

**更新日期**: 2026-04-26  
**负责人**: 文档团队
