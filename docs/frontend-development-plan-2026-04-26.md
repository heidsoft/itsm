# ITSM 前端功能开发计划

## 开发概览

按优先级开发3个前端界面：
1. PIR管理界面
2. 问题趋势分析界面
3. 知识库审核界面

---

## 1. PIR管理界面开发

### 前端页面路径
- `itsm-frontend/src/app/(main)/changes/[id]/pir/page.tsx` - PIR创建/查看页面

### 功能需求
**PIR创建表单：**
- 变更信息展示
- 总体结果选择（成功/部分成功/失败）
- 目标达成选择
- 成功总结（文本框）
- 遇到的问题（文本框）
- 经验教训（文本框）
- 改进建议（文本框）
- 实际开始/结束时间
- 是否回滚（条件显示回滚原因）

**PIR查看页面：**
- PIR详情展示
- 修改功能

### API调用
- `GET /api/changes/{id}` - 获取变更信息
- `POST /api/changes/{id}/pir` - 创建PIR
- `GET /api/changes/{id}/pir` - 获取PIR详情

---

## 2. 问题趋势分析界面开发

### 前端页面路径
- `itsm-frontend/src/app/(main)/problems/trends/page.tsx` - 问题趋势分析页面

### 功能需求
**趋势图表：**
- 问题数量趋势（折线图）
- 问题分类分布（饼图）
- 问题优先级分布（柱状图）
- 月度趋势对比

**数据展示：**
- 问题总数、已解决、未解决
- 解决率、平均解决时间
- Top 5问题分类

**筛选功能：**
- 时间范围选择
- 分类筛选
- 优先级筛选

### API调用
- `GET /api/problems/trends?start_date=&end_date=` - 获取趋势数据
- `GET /api/problems/hotspots` - 获取热点区域

---

## 3. 知识库审核界面开发

### 前端页面路径
- `itsm-frontend/src/app/(main)/knowledge/reviews/page.tsx` - 审核队列页面
- `itsm-frontend/src/app/(main)/knowledge/reviews/[id]/page.tsx` - 审核详情页面

### 功能需求
**审核队列：**
- 待审核文章列表
- 文章标题、作者、提交时间
- 快速操作（通过/拒绝）

**审核详情：**
- 文章内容预览
- 审核意见输入
- 通过/拒绝按钮
- 历史记录

### API调用
- `GET /api/knowledge/reviews/pending` - 获取待审核列表
- `POST /api/knowledge/reviews/{id}/approve` - 审核通过
- `POST /api/knowledge/reviews/{id}/reject` - 审核拒绝

---

## 4. API服务层开发

### 创建API服务文件
```
itsm-frontend/src/lib/api/
  ├── pir-api.ts          - PIR API
  ├── problem-trend-api.ts - 问题趋势 API
  └── knowledge-review-api.ts - 知识库审核 API
```

### API服务示例
```typescript
// pir-api.ts
export const PIRApi = {
  createPIR: async (changeId: number, data: PIRFormData) => {
    const response = await fetch(`/api/changes/${changeId}/pir`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return response.json();
  },
  
  getPIR: async (changeId: number) => {
    const response = await fetch(`/api/changes/${changeId}/pir`);
    return response.json();
  },
};
```

---
## 5. 开发时间估算

| 功能 | 工作量 | 优先级 |
|------|--------|--------|
| PIR界面 | 2-3天 | 高 |
| 问题趋势界面 | 1-2天 | 高 |
| 知识库审核界面 | 1-2天 | 高 |
| API服务层 | 0.5天 | 高 |
| 测试验证 | 1天 | 高 |

**总计：5-8天**

---

## 6. 测试验证计划

### 单元测试
- 组件渲染测试
- API调用测试
- 表单验证测试

### 集成测试
- 前后端联调
- API集成测试

### E2E测试
- PIR创建流程
- 趋势数据展示
- 审核流程

---

## 7. 部署检查清单

- [ ] 前端页面已创建
- [ ] API服务层已实现
- [ ] 单元测试已编写
- [ ] 路由已配置
- [ ] 权限已设置
- [ ] 文档已更新

---

## 总结

已规划3个前端界面的开发计划，预计5-8天可完成。建议按优先级依次实现，每完成一个功能立即测试验证。
