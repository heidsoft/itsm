# 图标使用规范指南

## 概述
本项目使用两种图标库：
1. **lucide-react** - 主要图标库，提供现代化的线性图标
2. **@ant-design/icons** - Ant Design 组件库的图标，用于与 Ant Design 组件配合使用

## 图标库选择原则

### 使用 lucide-react 的情况：
- 页面内容中的装饰性图标
- 按钮、链接等交互元素的图标
- 数据展示中的状态图标
- 导航菜单中的图标
- 表单控件中的图标

### 使用 @ant-design/icons 的情况：
- Ant Design 组件的 `icon` 属性
- 与 Ant Design 组件紧密集成的图标
- 需要保持与 Ant Design 设计语言一致的场景
- 上传、下载等特定功能的图标

## 导入规范

### lucide-react 图标导入
```typescript
import { 
  Search, 
  Plus, 
  Edit, 
  Trash2, 
  CheckCircle, 
  AlertTriangle,
  RefreshCw,
  FileText,
  User,
  Settings
} from "lucide-react";
```

### Ant Design 图标导入
```typescript
import { 
  SearchOutlined, 
  PlusOutlined, 
  EditOutlined, 
  DeleteOutlined,
  UploadOutlined,
  DownloadOutlined
} from "@ant-design/icons";
```

## 使用规范

### 1. 图标大小
- 小图标：16px (size={16})
- 标准图标：20px (size={20}) 
- 大图标：24px (size={24})
- 超大图标：32px (size={32})

### 2. 图标颜色
```typescript
// 使用主题色
<Search size={20} style={{ color: '#1890ff' }} />

// 使用语义色
<CheckCircle size={20} style={{ color: '#52c41a' }} />
<AlertTriangle size={20} style={{ color: '#faad14' }} />
<XCircle size={20} style={{ color: '#ff4d4f' }} />
```

### 3. 图标命名规范
- 使用 PascalCase 命名
- 图标名称应该清晰表达其功能
- 避免使用缩写，除非是通用缩写

## 常用图标映射表

| 功能 | lucide-react | @ant-design/icons |
|------|---------------|-------------------|
| 搜索 | Search | SearchOutlined |
| 添加 | Plus | PlusOutlined |
| 编辑 | Edit | EditOutlined |
| 删除 | Trash2 | DeleteOutlined |
| 查看 | Eye | EyeOutlined |
| 下载 | Download | DownloadOutlined |
| 上传 | Upload | UploadOutlined |
| 刷新 | RefreshCw | ReloadOutlined |
| 设置 | Settings | SettingOutlined |
| 用户 | User | UserOutlined |
| 文件 | FileText | FileTextOutlined |
| 检查 | CheckCircle | CheckCircleOutlined |
| 警告 | AlertTriangle | ExclamationCircleOutlined |
| 错误 | XCircle | CloseCircleOutlined |
| 信息 | Info | InfoCircleOutlined |
| 时钟 | Clock | ClockCircleOutlined |
| 日历 | Calendar | CalendarOutlined |
| 邮件 | Mail | MailOutlined |
| 电话 | Phone | PhoneOutlined |
| 位置 | MapPin | EnvironmentOutlined |
| 链接 | Link | LinkOutlined |
| 星标 | Star | StarOutlined |
| 标签 | Tag | TagsOutlined |
| 工作流 | Workflow | BranchesOutlined |
| 历史 | History | HistoryOutlined |
| 趋势 | TrendingUp | RiseOutlined |
| 数据库 | Database | DatabaseOutlined |
| 电子表格 | FileSpreadsheet | TableOutlined |
| 消息 | MessageSquare | MessageOutlined |
| 附件 | Paperclip | PaperClipOutlined |
| 发送 | Send | SendOutlined |
| 更多 | MoreHorizontal | MoreOutlined |
| 标志 | Flag | FlagOutlined |
| 书签 | Bookmark | BookOutlined |
| 图表 | BarChart3 | BarChartOutlined |
| 仪表盘 | Gauge | DashboardOutlined |
| 通知 | Bell | BellOutlined |
| 锁 | Lock | LockOutlined |
| 盾牌 | Shield | SafetyOutlined |
| 闪电 | Zap | ThunderboltOutlined |
| 地球 | Globe | GlobalOutlined |
| 闪光 | Sparkles | FireOutlined |

## 最佳实践

### 1. 一致性
- 在同一页面或组件中，保持图标风格一致
- 优先使用 lucide-react 图标，保持整体设计统一

### 2. 可访问性
- 为图标添加适当的 `aria-label` 属性
- 图标应该与文本配合使用，而不是完全替代文本

### 3. 性能优化
- 只导入需要的图标，避免导入整个图标库
- 使用图标时考虑懒加载

### 4. 响应式设计
- 图标大小应该适应不同的屏幕尺寸
- 在小屏幕上使用较小的图标

## 常见问题解决

### 1. 图标未定义错误
```typescript
// 错误：图标未导入
<Sparkles size={20} />

// 正确：先导入图标
import { Sparkles } from "lucide-react";
<Sparkles size={20} />
```

### 2. 图标类型错误
```typescript
// 错误：混用不同图标库
import { Search } from "lucide-react";
import { SearchOutlined } from "@ant-design/icons";

// 正确：选择一种图标库，保持一致性
import { Search } from "lucide-react";
```

### 3. 图标大小不一致
```typescript
// 错误：混用不同的大小单位
<Search size={20} />
<SearchOutlined style={{ fontSize: '16px' }} />

// 正确：统一使用 size 属性
<Search size={20} />
<SearchOutlined style={{ fontSize: '20px' }} />
```

## 迁移指南

### 从 Ant Design 图标迁移到 lucide-react
1. 查找对应的 lucide-react 图标
2. 更新导入语句
3. 调整图标属性（size 替代 style.fontSize）
4. 测试显示效果

### 从 lucide-react 迁移到 Ant Design 图标
1. 查找对应的 Ant Design 图标
2. 更新导入语句
3. 调整图标属性（style.fontSize 替代 size）
4. 测试显示效果

## 代码审查检查点

在代码审查时，请检查：
- [ ] 所有使用的图标都已正确导入
- [ ] 图标库使用保持一致
- [ ] 图标大小和颜色设置合理
- [ ] 图标具有适当的可访问性属性
- [ ] 没有未使用的图标导入
