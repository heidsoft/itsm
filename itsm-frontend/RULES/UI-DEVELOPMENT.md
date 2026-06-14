# ITSM 前端 UI 开发规范

> **版本**: 1.1
> **最后更新**: 2026-06-14
> **目的**: 针对 ITSM 前端开发中的常见 UI 问题制定约束，确保界面的一致性、可读性和专业性。

---

## 速查表

| 问题类型 | 检查点 | 修复优先级 | 关联规则 |
|----------|--------|------------|----------|
| 配色对比度 | 无灰色透明背景 + 白色文字组合 | P0 | FRONTEND-NO-GO.md §1 |
| 开发状态文字 | 无"开发中"、"Under construction"等提示 | P0 | FRONTEND-NO-GO.md §2 |
| 看板卡片文字 | 数字下方次要文字不使用透明背景 | P0 | FRONTEND-NO-GO.md §0.5 |
| 通知按钮状态 | 入口按钮有明确激活/非激活区分 | P0 | FRONTEND-NO-GO.md §5.1 |
| 标签输入样式 | Select mode="tags" 使用自定义 tagRender | P0 | FRONTEND-NO-GO.md §5.2 |
| 导航错误处理 | 可能报错的功能有 try-catch 包装 | P0 | FRONTEND-NO-GO.md §6.1 |
| 卡片尺寸 | 同一行卡片结构一致，无堆叠 Statistic | P1 | FRONTEND-NO-GO.md §3 |
| Focus 样式 | 输入框无过于醒目的蓝色 focus ring | P1 | FRONTEND-NO-GO.md §5.3 |
| 按钮状态 | 激活/非激活状态有明显区分 | P2 | FRONTEND-NO-GO.md §5 |

---

### 0.5 看板卡片数字下方文字透明度

**问题**：事件管理页的看板卡片，数字下方文字太透明了，看不清楚。

```tsx
// ❌ 错误：数字下方文字使用透明背景 + 白色文字
<Card className="bg-gray-800">
  <Statistic value={42} />
  <Text className="text-white/40">个待处理</Text>  {/* 透明度太高，不可读 */}
</Card>

// ✅ 正确：使用 Ant Design 文字类型，自动适配主题
<Statistic value={42} />
<Typography.Text type="secondary">个待处理</Typography.Text>

// ✅ 正确：使用固定颜色（非透明）
<Card>
  <Statistic value={42} />
  <span className="text-gray-400">个待处理</span>  {/* 固定颜色，无透明度 */}
</Card>
```

---

## 1. 配色与对比度

### 1.1 禁止灰色透明背景配白色文字

**问题**：灰色透明背景配白色文字导致对比度不足，难以阅读。

**解决方案**：

```tsx
// ❌ 错误：灰色透明背景 + 白色文字 = 对比度不足
<div className="bg-gray-100/50 text-white">文字</div>

// ✅ 正确方案 1：深色背景 + 白色文字
<div className="bg-gray-900 text-white">文字</div>

// ✅ 正确方案 2：浅色背景 + 深色文字
<div className="bg-gray-100 text-gray-900">文字</div>

// ✅ 正确方案 3：使用 Ant Design 的 secondary 文字类型（自动适配主题）
<Typography.Text type="secondary">次要文字</Typography.Text>
```

### 1.2 标签（Tag）配色规则

```tsx
// ❌ 错误：灰色透明标签 + 白色文字 = 不可读
<span className="bg-gray-200/50 text-white">标签</span>

// ✅ 正确：使用明确的颜色组合
<span className="bg-blue-100 text-blue-800">标签</span>
// 或使用 Ant Design Tag 组件
<Tag color="blue">标签</Tag>
```

### 1.3 按钮状态颜色

```tsx
// ❌ 错误：白色背景按钮在某些主题下不可见
<Button className="bg-white">按钮</Button>

// ✅ 正确：使用明确的按钮类型
<Button type="primary">主要按钮</Button>
<Button type="default">默认按钮</Button>
<Button type="text">文字按钮</Button>
```

---

## 2. 开发状态提示清理

### 2.1 禁止保留开发中/Under Construction 文字

**问题**：页面出现"开发中"、"Under construction"等状态文字。

**解决方案**：

```tsx
// ❌ 错误：保留开发状态提示
<ManagementNotice
  message="复杂域页面开始统一收口"
  description="总览、云资源、核对和配置项详情将逐步复用同一套页面头部..."
/>

// ✅ 正确：移除或替换为功能性说明
<ManagementNotice
  message="配置项总数"
  description="管理配置项、云资源同步、关系拓扑和核对结果。"
/>
```

### 2.2 常见需要移除的提示文本

| 模式 | 示例 | 修复方式 |
|------|------|----------|
| 开发状态 | "开发中"、"Under construction"、"In development" | 移除或替换为功能说明 |
| 技术细节暴露 | "复杂域页面开始统一收口" | 替换为用户友好的功能描述 |
| 内部说明 | "设计态与运行态已分离处理" | 移除或简化为功能说明 |
| 渲染状态 | "报表页已切换为稳态渲染" | 移除，保留错误处理即可 |
| 实现进度 | "将逐步复用..."、"图表数据失败时..." | 移除，错误消息已足够 |

### 2.3 如何处理功能未完成的情况

```tsx
// ❌ 错误：显示开发状态文字
<div className="bg-yellow-50 p-4">
  <Text type="warning">此功能开发中...</Text>
</div>

// ✅ 正确：使用空状态组件或禁用功能
<Button disabled icon={<Plus />}>
  新增配置项（功能开发中）
</Button>

// 或使用 LoadingEmptyError 组件
<LoadingEmptyError
  state="empty"
  empty={{
    title: '暂无数据',
    description: '功能开发中，请稍后再试。',
  }}
/>
```

---

## 3. 卡片尺寸一致性

### 3.1 统计卡片统一尺寸

**问题**：SLA 合规率卡片与其他卡片大小不一致。

**解决方案**：

```tsx
// ✅ 统一使用相同的 Col 配置
<Row gutter={[16, 16]}>
  <Col xs={24} sm={12} md={6}>
    <Card>
      <Statistic title="总SLA数量" value={10} />
    </Card>
  </Col>
  <Col xs={24} sm={12} md={6}>
    <Card>
      <Statistic title="生效中" value={8} />
    </Card>
  </Col>
  <Col xs={24} sm={12} md={6}>
    <Card>
      <Statistic title="违规数量" value={2} />
    </Card>
  </Col>
  <Col xs={24} sm={12} md={6}>
    {/* 合规率卡片需要与其他卡片保持一致 */}
    <Card>
      <Statistic title="合规率" value={95} suffix="%" />
      {/* 不要在同一个 Card 中堆叠多个 Statistic */}
    </Card>
  </Col>
</Row>
```

### 3.2 避免在一个 Card 中堆叠多个 Statistic

```tsx
// ❌ 错误：一个 Card 中有两个 Statistic
<Card>
  <Statistic title="合规率" value={95} suffix="%" />
  <Statistic title="总违规" value={5} />  {/* 导致卡片高度不一致 */}
</Card>

// ✅ 正确：拆分到独立的 Card
<Card>
  <Statistic title="合规率" value={95} suffix="%" precision={1} />
</Card>
<Card>
  <Statistic title="总违规" value={5} />
</Card>
```

### 3.3 KPI 卡片尺寸一致

```tsx
// AdvancedAnalytics.tsx 中的 KPI 卡片
<Row gutter={[16, 16]}>
  {kpiData.map((kpi, index) => (
    <Col xs={24} sm={12} md={6} key={index}>
      <Card className="kpi-card">  {/* 确保 kpi-card 样式统一 */}
        {/* ... */}
      </Card>
    </Col>
  ))}
</Row>
```

---

## 4. 表单输入样式

### 4.1 Tag 输入框样式

**问题**：标签输入框确认后标签是灰色透明文字白色，看不清楚。

**解决方案**：

```tsx
// ❌ 错误：使用默认样式
<Select mode="tags" placeholder="输入标签后回车" />

// ✅ 正确：自定义 tag 样式或使用 token
<Select
  mode="tags"
  placeholder="输入标签后回车"
  tagRender={(props) => (
    <Tag
      {...props}
      className="bg-blue-100 text-blue-800 border-blue-300"
    >
      {props.label}
    </Tag>
  )}
/>
```

### 4.2 Focus 状态样式

**问题**：新建知识库文章标签输入框光标出现蓝色框（focus ring 过强）。

**解决方案**：

```tsx
// ❌ 错误：focus ring 过强，覆盖过多区域
<Select className="focus:ring-4 focus:ring-blue-500" />

// ✅ 正确：使用适度的 focus 样式
<Select className="focus:ring-2 focus:ring-primary focus:ring-offset-0" />

// ✅ 正确：禁用 focus ring，改用 border 颜色变化
<Select
  className="focus:border-primary border-gray-300 transition-colors"
  styles={{
    focus: { borderColor: 'var(--ant-primary-color)' },
  }}
/>
```

### 4.3 看板卡片数字下方文字

**问题**：事件管理页的看板卡片，数字下方文字太透明了。

**解决方案**：

```tsx
// ✅ 使用 Ant Design 文字类型
<Statistic value={42} />
<Typography.Text type="secondary">个待处理</Typography.Text>

// ✅ 或使用固定颜色（非透明）
<Card>
  <Statistic value={42} />
  <span className="text-gray-400">个待处理</span>
</Card>
```

---

## 5. 导航和按钮状态

### 5.1 通知中心入口按钮

**问题**：通知中心入口按钮未显示明确状态变化。

**解决方案**：

```tsx
// ✅ 使用 Badge 显示未读数量
<Badge count={unreadCount} size="small">
  <Button
    type={notificationsOpen ? 'primary' : 'text'}
    icon={<Bell className={notificationsOpen ? 'text-primary' : ''} />}
    onClick={() => setNotificationsOpen(!notificationsOpen)}
  />
</Badge>

// ✅ 或使用 Dropdown 显示激活状态
<Dropdown
  trigger={['click']}
  open={notificationsOpen}
  onOpenChange={setNotificationsOpen}
>
  <Button type="text" icon={<Bell />}>通知</Button>
</Dropdown>
```

---

## 6. 功能异常处理

### 6.1 资产服务拓扑报错

**问题**：点击资产服务拓扑弹出报错。

**解决方案**：

```tsx
// ✅ 添加错误处理和加载状态
const ServiceTopology = ({ assetId }) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetchTopology(assetId)
      .catch((err) => {
        setError(err.message);
        message.error('加载拓扑图失败：' + err.message);
      })
      .finally(() => setLoading(false));
  }, [assetId]);

  if (loading) return <Skeleton active />;
  if (error) return <Alert type="error" message={error} />;

  return <TopologyChart data={topologyData} />;
};
```

### 6.2 链接和按钮的错误处理

```tsx
// ✅ 使用 try-catch 包装可能导致错误的操作
<Button
  onClick={async () => {
    try {
      await router.push('/cmdb/topology');
    } catch (error) {
      message.error('页面加载失败，请稍后重试');
    }
  }}
>
  资产服务拓扑
</Button>

// ✅ 或使用 disabled 状态防止错误操作
<Button
  disabled={!isTopologyAvailable}
  onClick={() => router.push('/cmdb/topology')}
>
  资产服务拓扑
</Button>
```

---

## 7. 检查清单

在提交代码前，检查以下项目：

- [ ] 没有灰色透明背景配白色文字的组合
- [ ] 所有"开发中"、"Under construction"等状态文字已移除
- [ ] 看板卡片数字下方次要文字不使用透明背景
- [ ] 统计卡片/KPI 卡片尺寸一致，无堆叠多个 Statistic
- [ ] Tag 输入框（Select mode="tags"）使用自定义 tagRender，标签清晰可读
- [ ] 通知中心入口按钮有明确的激活/非激活状态区分（Badge + type）
- [ ] 所有可能导致报错的操作有 try-catch 错误处理
- [ ] 表单输入框的 focus 状态不过于醒目（focus:ring-2 而非 focus:ring-4）
- [ ] 所有导航按钮（router.push）有错误处理

---

## 8. 快速修复命令

```bash
# === P0 必检 ===

# 检查是否包含开发状态文字
grep -rEn "开发中|Under construction|开发状态|逐步复用|稳态渲染|设计态与运行态|复杂域页面|统一收口" src --include="*.tsx"

# 检查是否有灰色透明背景 + 白色文字
grep -rEn "bg-gray-[0-9]+/[0-4][0-9] text-white|text-white/[0-9] text-white" src --include="*.tsx"

# 检查 Select mode="tags" 是否缺少 tagRender
grep -rEn 'mode="tags"' src --include="*.tsx" | while read f; do
  file=$(echo "$f" | cut -d: -f1)
  if ! grep -q "tagRender" "$file"; then
    echo "MISSING tagRender: $file"
  fi
done

# 检查通知中心按钮是否缺少状态区分
grep -rEn "<Button[^>]*icon=\{<Bell" src --include="*.tsx" | grep -v "Badge\|notificationsOpen\|type=.*primary"

# 检查看板卡片数字下方文字是否使用透明背景
grep -rEn "text-white/[0-9]" src --include="*.tsx"

# === P1 推荐检查 ===

# 检查 focus ring 样式是否过于醒目
grep -rEn "focus:ring-[4-9]" src --include="*.tsx"

# 检查无错误处理的导航按钮
grep -rEn "onClick=\{.*router\.push" src --include="*.tsx" | grep -v "try\|catch\|error\|message\." | grep -v "disabled="

# === P2 可选检查 ===

# 检查大文件（超过 800 行）
find src -name "*.tsx" -exec wc -l {} \; | awk '$1 > 800 {print $2, $1" lines"}'
```

---

## 9. 参考文件与常见问题映射

| 页面 | 文件路径 | 常见问题 | 关联规则 |
|------|----------|----------|----------|
| CMDB | `app/(main)/cmdb/page.tsx` | 页面头部有开发状态提示 | NO-GO §2 |
| 报表中心 | `app/(main)/reports/page.tsx` | 有"稳态渲染"提示 | NO-GO §2 |
| 工作流管理 | `app/(main)/workflow/page.tsx` | 有"设计态与运行态"提示 | NO-GO §2 |
| SLA服务级别 | `app/(main)/sla/page.tsx` | 合规率卡片有堆叠 Statistic，卡片高度不一致 | NO-GO §3 |
| 高级分析 | `components/reports/AdvancedAnalytics.tsx` | KPI 卡片尺寸不一致 | NO-GO §3 |
| 知识库新建 | `app/(main)/knowledge/articles/new/page.tsx` | 标签输入框缺少 tagRender，focus ring 过强 | NO-GO §5.2, §5.3 |
| 用户组管理 | `app/(main)/admin/groups/page.tsx` | 权限标签样式不清晰 | NO-GO §1.2 |
| 事件管理 | `app/(main)/tickets/page.tsx` | 看板卡片数字下方文字透明度太高 | NO-GO §0.5 |
| 全局 | 通知中心组件 | 入口按钮无状态变化 | NO-GO §5.1 |
| 全局 | 资产服务拓扑入口 | 导航无错误处理 | NO-GO §6.1 |

---

## 10. 已修复问题记录

| 修复日期 | 页面 | 问题 | 状态 | 关联规则 |
|----------|------|------|------|----------|
| 2026-06-14 | `workflow/page.tsx` | 移除"设计态与运行态已分离处理"提示 | ✅ | NO-GO §2 |
| 2026-06-14 | `cmdb/page.tsx` | 移除"复杂域页面开始统一收口"提示 | ✅ | NO-GO §2 |
| 2026-06-14 | `reports/page.tsx` | 移除"稳态渲染"提示 | ✅ | NO-GO §2 |
| 2026-06-14 | `sla/page.tsx` | 合规率卡片拆分为独立 Card | ✅ | NO-GO §3 |
| 2026-06-14 | `sla/page.tsx` | SLA 卡片改用 `align="stretch"` 和 `h-full w-full` 确保高度一致 | ✅ | NO-GO §3 |
| 2026-06-14 | `knowledge/articles/new/page.tsx` | 标签使用自定义样式 `bg-blue-100 text-blue-800` | ✅ | NO-GO §5.2 |
| 2026-06-14 | `admin/groups/page.tsx` | 权限标签使用自定义样式 | ✅ | NO-GO §1.2 |

---

## 11. 规则文件索引

| 文件 | 用途 |
|------|------|
| `UI-DEVELOPMENT.md` | 开发规范和解决方案示例 |
| `FRONTEND-NO-GO.md` | 负面清单（禁止的反模式），含自检命令 |