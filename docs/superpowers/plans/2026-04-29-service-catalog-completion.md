# 服务目录完善 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 完成服务目录模块，实现服务项浏览、服务管理、服务请求审批的完整闭环

**Architecture:**
- 前端：Next.js App Router，新增 `detail/[id]` 页面、`edit/[id]` 页面、审批处理页面
- 后端：在现有 `handlers/service_catalog/` 基础上增强，新增搜索API、统计API
- 数据流：前端 → API → Handler → Service → Repository → Ent

**Tech Stack:** Next.js, Ant Design, Go/Gin, Ent ORM

---

## 一、当前文件结构分析

### 后端 (handlers/service_catalog/)
| 文件 | 职责 | 状态 |
|------|------|------|
| `entity.go` | 内部实体定义 | ✅ |
| `repository_impl.go` | 数据库操作 | ✅ |
| `service.go` | 业务逻辑 | ✅ |
| `handler.go` | HTTP处理 | ✅ |

### 后端 (ent/schema/servicecatalog.go)
- 字段：name, description, category, icon, service_type, price, delivery_time, unit
- 审批：requires_approval, approval_level, approvers
- SLA：sla_response_time, sla_resolution_time
- CI关联：ci_type_id, cloud_service_id
- 表单配置：form_schema, available_regions, available_specs
- 状态：status, is_active, sort_order, tenant_id

### 前端 (service-catalog/)
| 文件 | 职责 |
|------|------|
| `page.tsx` | 服务列表页 |
| `components/ServiceItemCard.tsx` | 服务卡片 |
| `components/CreateServiceModal.tsx` | 创建服务 |
| `components/ServiceCatalogFilters.tsx` | 筛选器 |
| `components/ServiceCatalogStats.tsx` | 统计 |
| `request/[serviceId]/page.tsx` | 服务请求页 |
| `request/forms/ApplyRdsForm.tsx` | RDS申请表单 |
| `request/forms/ApplyVmForm.tsx` | VM申请表单 |
| `request/forms/ExpandOssForm.tsx` | OSS扩容表单 |

---

## 二、任务分解

### Task 1: 服务详情页 (Service Detail Page)

**Files:**
- Create: `itsm-frontend/src/app/(main)/service-catalog/detail/[id]/page.tsx`

- [ ] **Step 1: 创建服务详情页骨架**

```tsx
'use client';

import React, { useEffect, useState } from 'react';
import { Card, Descriptions, Tag, Button, Row, Col, App, Spin, Empty } from 'antd';
import { useParams, useRouter } from 'next/navigation';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import type { ServiceItem } from '@/types/biz/service-catalog';
import { useI18n } from '@/lib/i18n';

export default function ServiceDetailPage() {
  const params = useParams();
  const router = useRouter();
  const { message } = App.useApp();
  const { t } = useI18n();
  const [service, setService] = useState<ServiceItem | null>(null);
  const [loading, setLoading] = useState(true);

  const serviceId = params.id as string;

  useEffect(() => {
    const loadService = async () => {
      try {
        setLoading(true);
        const data = await ServiceCatalogApi.getService(serviceId);
        setService(data);
      } catch (error) {
        message.error(t('common.getFailed'));
        router.push('/service-catalog');
      } finally {
        setLoading(false);
      }
    };
    loadService();
  }, [serviceId, message, router, t]);

  if (loading) {
    return <Spin className="flex justify-center py-12" />;
  }

  if (!service) {
    return <Empty description={t('service.notFound')} />;
  }

  return (
    <div className="p-6">
      <Card>
        <Descriptions title={service.name} bordered column={2}>
          <Descriptions.Item label={t('service.category')}>
            {service.category}
          </Descriptions.Item>
          <Descriptions.Item label={t('service.status')}>
            <Tag color={service.status === 'published' ? 'green' : 'default'}>
              {service.status}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('service.description')} span={2}>
            {service.shortDescription || service.fullDescription}
          </Descriptions.Item>
          {service.availability?.responseTime && (
            <Descriptions.Item label={t('service.deliveryTime')}>
              {service.availability.responseTime}
            </Descriptions.Item>
          )}
        </Descriptions>

        <div className="mt-6 flex gap-4">
          <Button type="primary" onClick={() => router.push(`/service-catalog/request/${serviceId}`)}>
            {t('service.request')}
          </Button>
          <Button onClick={() => router.push('/service-catalog')}>
            {t('common.back')}
          </Button>
        </div>
      </Card>
    </div>
  );
}
```

- [ ] **Step 2: 添加服务项卡片跳转链接**

Modify: `itsm-frontend/src/app/(main)/service-catalog/components/ServiceItemCard.tsx:30-40`

找到 `ServiceItemCard` 组件中点击事件处理，添加跳转：

```tsx
const handleClick = () => {
  router.push(`/service-catalog/detail/${catalog.id}`);
};
```

- [ ] **Step 3: 运行验证**

Run: `cd itsm-frontend && npm run dev`
Expected: 访问 `/service-catalog/detail/1` 显示服务详情

- [ ] **Step 4: 提交**

```bash
git add itsm-frontend/src/app/\(main\)/service-catalog/detail/
git commit -m "feat(service-catalog): add service detail page"
```

---

### Task 2: 服务编辑功能 (Service Edit)

**Files:**
- Create: `itsm-frontend/src/app/(main)/service-catalog/edit/[id]/page.tsx`
- Modify: `itsm-frontend/src/app/(main)/service-catalog/components/ServiceItemCard.tsx:35-45` (添加编辑入口)

- [ ] **Step 1: 创建服务编辑页面**

```tsx
'use client';

import React, { useEffect, useState } from 'react';
import { Card, Form, Input, Select, InputNumber, Switch, Button, App, Spin, message } from 'antd';
import { useParams, useRouter } from 'next/navigation';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import { useI18n } from '@/lib/i18n';

const { TextArea } = Input;

export default function EditServicePage() {
  const params = useParams();
  const router = useRouter();
  const { message: appMessage } = App.useApp();
  const { t } = useI18n();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [initialLoading, setInitialLoading] = useState(true);

  const serviceId = params.id as string;

  useEffect(() => {
    const loadService = async () => {
      try {
        const data = await ServiceCatalogApi.getService(serviceId);
        form.setFieldsValue({
          name: data.name,
          category: data.category,
          description: data.shortDescription || data.fullDescription,
          deliveryTime: data.availability?.responseTime,
          status: data.status === 'published' ? 'enabled' : 'disabled',
        });
      } catch (error) {
        appMessage.error(t('common.getFailed'));
        router.push('/service-catalog');
      } finally {
        setInitialLoading(false);
      }
    };
    loadService();
  }, [serviceId, form, appMessage, router, t]);

  const handleSubmit = async (values: any) => {
    try {
      setLoading(true);
      await ServiceCatalogApi.updateService(serviceId, {
        name: values.name,
        category: values.category,
        shortDescription: values.description,
        availability: { responseTime: values.deliveryTime },
      });
      appMessage.success(t('common.saveSuccess'));
      router.push('/service-catalog');
    } catch (error) {
      appMessage.error(t('common.saveFailed'));
    } finally {
      setLoading(false);
    }
  };

  if (initialLoading) {
    return <Spin className="flex justify-center py-12" />;
  }

  return (
    <div className="p-6 max-w-2xl">
      <Card title={t('service.editService')}>
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item label={t('service.name')} name="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label={t('service.category')} name="category">
            <Input />
          </Form.Item>
          <Form.Item label={t('service.description')} name="description">
            <TextArea rows={4} />
          </Form.Item>
          <Form.Item label={t('service.deliveryTime')} name="deliveryTime">
            <InputNumber min={1} className="w-full" />
          </Form.Item>
          <Form.Item label={t('service.status')} name="status" valuePropName="checked">
            <Switch checkedChildren="enabled" unCheckedChildren="disabled" />
          </Form.Item>
          <Form.Item>
            <div className="flex gap-4">
              <Button type="primary" htmlType="submit" loading={loading}>
                {t('common.save')}
              </Button>
              <Button onClick={() => router.push('/service-catalog')}>
                {t('common.cancel')}
              </Button>
            </div>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
```

- [ ] **Step 2: 在服务项卡片添更多操作入口**

Modify: `itsm-frontend/src/app/(main)/service-catalog/components/ServiceItemCard.tsx`

在卡片底部添加操作按钮区域（编辑、发布/停用）：

```tsx
<div className="service-card-actions mt-4 flex gap-2">
  <Button size="small" onClick={(e) => { e.stopPropagation(); router.push(`/service-catalog/edit/${catalog.id}`); }}>
    {t('common.edit')}
  </Button>
  {catalog.status === 'published' ? (
    <Button size="small" onClick={(e) => { e.stopPropagation(); handleRetire(catalog.id); }}>
      {t('service.retire')}
    </Button>
  ) : (
    <Button size="small" type="primary" onClick={(e) => { e.stopPropagation(); handlePublish(catalog.id); }}>
      {t('service.publish')}
    </Button>
  )}
</div>
```

添加 `handlePublish` 和 `handleRetire` 函数：

```tsx
const handlePublish = async (id: string) => {
  try {
    await ServiceCatalogApi.publishService(id);
    message.success(t('service.publishSuccess'));
    loadServiceCatalogs();
  } catch (error) {
    message.error(t('service.publishFailed'));
  }
};

const handleRetire = async (id: string) => {
  try {
    await ServiceCatalogApi.retireService(id);
    message.success(t('service.retireSuccess'));
    loadServiceCatalogs();
  } catch (error) {
    message.error(t('service.retireFailed'));
  }
};
```

- [ ] **Step 3: 运行验证**

Run: `cd itsm-frontend && npm run dev`
Expected:
- 访问 `/service-catalog` 点击卡片上的"编辑"按钮跳转到编辑页
- 编辑页可以修改服务信息并保存

- [ ] **Step 4: 提交**

```bash
git add itsm-frontend/src/app/\(main\)/service-catalog/edit/
git add itsm-frontend/src/app/\(main\)/service-catalog/components/ServiceItemCard.tsx
git commit -m "feat(service-catalog): add service edit functionality"
```

---

### Task 3: 服务搜索后端API

**Files:**
- Modify: `itsm-backend/handlers/service_catalog/service.go` (新增Search方法)
- Modify: `itsm-backend/handlers/service_catalog/handler.go` (新增Search handler)
- Modify: `itsm-backend/handlers/service_catalog/repository_impl.go` (新增Search repo method)

- [ ] **Step 1: 添加Repository搜索方法**

Modify: `itsm-backend/handlers/service_catalog/repository_impl.go`

添加 `Search` 方法到 Repository 接口和实现：

```go
// Repository interface 新增
Search(ctx context.Context, tenantID int, keyword string, filters ListFilters) ([]*ServiceCatalog, int, error)

// 实现
func (r *repo) Search(ctx context.Context, tenantID int, keyword string, filters ListFilters) ([]*ServiceCatalog, int, error) {
    query := r.client.ServiceCatalog.Query().
        Where(
            servicecatalog.TenantID(tenantID),
            servicecatalog.StatusEQ("enabled"),
        )

    // 关键词搜索：name OR description
    if keyword != "" {
        query = query.Where(
            or(
                servicecatalog.NameContains(keyword),
                servicecatalog.DescriptionContains(keyword),
            ),
        )
    }

    // 分类过滤
    if filters.Category != "" {
        query = query.Where(servicecatalog.Category(filters.Category))
    }

    // 获取总数
    total, err := query.Count(ctx)
    if err != nil {
        return nil, 0, err
    }

    // 分页
    offset := (filters.Page - 1) * filters.Size
    catalogs, err := query.
        Order(webhook.FieldNameGeneralWebhooks, OrderDesc).
        Offset(offset).
        Limit(filters.Size).
        All(ctx)
    if err != nil {
        return nil, 0, err
    }

    return catalogs, total, nil
}
```

注意：需要导入 `entsql` 或使用 `servicecatalog.NameContains` (ent内置支持)。

- [ ] **Step 2: 添加Service搜索方法**

Modify: `itsm-backend/handlers/service_catalog/service.go`

```go
func (s *Service) Search(ctx context.Context, tenantID int, keyword string, filters ListFilters) ([]*ServiceCatalog, int, error) {
    if filters.Page < 1 {
        filters.Page = 1
    }
    if filters.Size < 1 {
        filters.Size = 20
    }
    return s.repo.Search(ctx, tenantID, keyword, filters)
}
```

- [ ] **Step 3: 添加Handler搜索端点**

Modify: `itsm-backend/handlers/service_catalog/handler.go`

```go
// Search handles GET /api/v1/service-catalogs/search?q=xxx
func (h *Handler) Search(c *gin.Context) {
    keyword := c.Query("q")
    tenantID, exists := c.Get("tenant_id")
    if !exists {
        common.Fail(c, 2001, "租户信息缺失")
        return
    }

    filters := ListFilters{
        Category: c.Query("category"),
        Status:   "enabled",
        Page:     1,
        Size:     20,
    }

    catalogs, total, err := h.service.Search(c.Request.Context(), tenantID.(int), keyword, filters)
    if err != nil {
        common.Fail(c, 5001, err.Error())
        return
    }

    var responses []dto.ServiceCatalogResponse
    for _, cat := range catalogs {
        responses = append(responses, h.toDTO(cat))
    }

    common.Success(c, dto.ServiceCatalogListResponse{
        Catalogs: responses,
        Total:    total,
        Page:     filters.Page,
        Size:     filters.Size,
    })
}
```

- [ ] **Step 4: 注册路由**

Modify: `itsm-backend/router/router.go` 或 `itsm-backend/internal/bootstrap/app.go`

添加路由：
```go
serviceCatalogGroup.GET("/search", serviceCatalogHandler.Search)
```

- [ ] **Step 5: 测试验证**

Run: `curl "http://localhost:8080/api/v1/service-catalogs/search?q=vm"`
Expected: 返回包含"vm"的服务列表

- [ ] **Step 6: 提交**

```bash
git add itsm-backend/handlers/service_catalog/
git commit -m "feat(service-catalog): add search API"
```

---

### Task 4: 前端搜索功能集成

**Files:**
- Modify: `itsm-frontend/src/app/(main)/service-catalog/components/ServiceCatalogFilters.tsx` (添加搜索输入框)
- Modify: `itsm-frontend/src/app/(main)/service-catalog/hooks/useServiceCatalogData.ts` (支持搜索参数)

- [ ] **Step 1: 确认现有的 Filter 组件结构**

Read: `itsm-frontend/src/app/(main)/service-catalog/components/ServiceCatalogFilters.tsx`

查看现有筛选组件，然后添加搜索输入框。

- [ ] **Step 2: 修改 useServiceCatalogData 支持搜索**

Modify: `itsm-frontend/src/app/(main)/service-catalog/hooks/useServiceCatalogData.ts`

添加 `searchText` 状态并传给API：

```typescript
const [searchText, setSearchText] = useState('');

const loadServiceCatalogs = useCallback(async () => {
  setLoading(true);
  try {
    const { services, total } = await ServiceCatalogApi.getServices({
      page: currentPage,
      pageSize: pageSize,
      category: categoryFilter,
      status: 'published',
      search: searchText,  // 新增
    });
    setCatalogs(services);
    setTotal(total);
  } catch (error) {
    // error handling
  } finally {
    setLoading(false);
  }
}, [currentPage, pageSize, categoryFilter, searchText]);

// 当 searchText 变化时重新加载
useEffect(() => {
  loadServiceCatalogs();
}, [loadServiceCatalogs]);
```

- [ ] **Step 3: 提交**

```bash
git add itsm-frontend/src/app/\(main\)/service-catalog/
git commit -m "feat(service-catalog): integrate search functionality"
```

---

### Task 5: 服务审批管理页面

**Files:**
- Create: `itsm-frontend/src/app/(main)/service-catalog/approvals/page.tsx`

- [ ] **Step 1: 创建审批列表页**

```tsx
'use client';

import React, { useEffect, useState } from 'react';
import { Table, Tag, Button, Card, App, Space, Modal, Input } from 'antd';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import { useI18n } from '@/lib/i18n';

export default function ServiceApprovalsPage() {
  const { message } = App.useApp();
  const { t } = useI18n();
  const [requests, setRequests] = useState([]);
  const [loading, setLoading] = useState(false);
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [rejectModalVisible, setRejectModalVisible] = useState(false);
  const [selectedRequest, setSelectedRequest] = useState<any>(null);
  const [rejectReason, setRejectReason] = useState('');

  useEffect(() => {
    loadApprovals();
  }, []);

  const loadApprovals = async () => {
    setLoading(true);
    try {
      const data = await ServiceCatalogApi.getServiceRequests({ status: 'pending' });
      setRequests(data.requests || []);
    } catch (error) {
      message.error(t('common.getFailed'));
    } finally {
      setLoading(false);
    }
  };

  const handleApprove = async (id: number) => {
    try {
      await ServiceCatalogApi.approveServiceRequest(id);
      message.success(t('service.approveSuccess'));
      loadApprovals();
    } catch (error) {
      message.error(t('service.approveFailed'));
    }
  };

  const handleReject = async (id: number) => {
    if (!rejectReason.trim()) {
      message.error(t('service.rejectReasonRequired'));
      return;
    }
    try {
      await ServiceCatalogApi.rejectServiceRequest(id, rejectReason);
      message.success(t('service.rejectSuccess'));
      setRejectModalVisible(false);
      loadApprovals();
    } catch (error) {
      message.error(t('service.rejectFailed'));
    }
  };

  const columns = [
    { title: t('service.requestId'), dataIndex: 'id' },
    { title: t('service.serviceName'), dataIndex: 'serviceName' },
    { title: t('service.requester'), dataIndex: 'requesterName' },
    { title: t('service.createdAt'), dataIndex: 'createdAt' },
    {
      title: t('service.status'),
      render: (_, record) => (
        <Tag color="orange">{t('service.pending')}</Tag>
      ),
    },
    {
      title: t('common.actions'),
      render: (_, record) => (
        <Space>
          <Button size="small" type="primary" onClick={() => handleApprove(record.id)}>
            {t('service.approve')}
          </Button>
          <Button size="small" onClick={() => { setSelectedRequest(record); setDetailModalVisible(true); }}>
            {t('common.view')}
          </Button>
          <Button size="small" danger onClick={() => { setSelectedRequest(record); setRejectModalVisible(true); }}>
            {t('service.reject')}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div className="p-6">
      <Card title={t('service.pendingApprovals')}>
        <Table columns={columns} dataSource={requests} loading={loading} rowKey="id" />
      </Card>

      {/* 详情弹窗 */}
      <Modal title={t('service.requestDetail')} open={detailModalVisible} onCancel={() => setDetailModalVisible(false)} footer={null}>
        {selectedRequest && (
          <div>
            <p><strong>{t('service.serviceName')}:</strong> {selectedRequest.serviceName}</p>
            <p><strong>{t('service.requester')}:</strong> {selectedRequest.requesterName}</p>
            <p><strong>{t('service.reason')}:</strong> {selectedRequest.reason}</p>
          </div>
        )}
      </Modal>

      {/* 拒绝弹窗 */}
      <Modal
        title={t('service.rejectRequest')}
        open={rejectModalVisible}
        onCancel={() => setRejectModalVisible(false)}
        onOk={() => handleReject(selectedRequest?.id)}
      >
        <p>{t('service.rejectReason')}</p>
        <Input.TextArea value={rejectReason} onChange={(e) => setRejectReason(e.target.value)} rows={4} />
      </Modal>
    </div>
  );
}
```

- [ ] **Step 2: 注册路由（菜单）**

Modify: `itsm-frontend/src/components/layout/sidebar/menu-config.ts`

添加审批菜单项：
```typescript
{
  key: '/service-catalog/approvals',
  label: t('service.pendingApprovals'),
  icon: <CheckCircleOutlined />,
}
```

- [ ] **Step 3: 提交**

```bash
git add itsm-frontend/src/app/\(main\)/service-catalog/approvals/
git add itsm-frontend/src/components/layout/sidebar/menu-config.ts
git commit -m "feat(service-catalog): add approval management page"
```

---

### Task 6: 服务目录统计API

**Files:**
- Modify: `itsm-backend/handlers/service_catalog/service.go` (新增Stats方法)
- Modify: `itsm-backend/handlers/service_catalog/handler.go` (新增Stats handler)

- [ ] **Step 1: 添加Service统计方法**

```go
func (s *Service) GetStats(ctx context.Context, tenantID int) (*ServiceStats, error) {
    total, err := s.repo.Count(ctx, tenantID, ListFilters{})
    if err != nil {
        return nil, err
    }

    enabled, err := s.repo.Count(ctx, tenantID, ListFilters{Status: "enabled"})
    if err != nil {
        return nil, err
    }

    // 统计各分类数量
    byCategory, err := s.repo.CountByCategory(ctx, tenantID)
    if err != nil {
        return nil, err
    }

    return &ServiceStats{
        TotalServices:    total,
        PublishedServices: enabled,
        Categories:       byCategory,
    }, nil
}
```

- [ ] **Step 2: 添加Handler**

```go
// Stats handles GET /api/v1/service-catalogs/stats
func (h *Handler) Stats(c *gin.Context) {
    tenantID, exists := c.Get("tenant_id")
    if !exists {
        common.Fail(c, 2001, "租户信息缺失")
        return
    }

    stats, err := h.service.GetStats(c.Request.Context(), tenantID.(int))
    if err != nil {
        common.Fail(c, 5001, err.Error())
        return
    }

    common.Success(c, stats)
}
```

- [ ] **Step 3: Repository添加统计方法**

```go
func (r *repo) Count(ctx context.Context, tenantID int, filters ListFilters) (int, error) {
    query := r.client.ServiceCatalog.Query().Where(servicecatalog.TenantID(tenantID))
    if filters.Status != "" {
        query = query.Where(servicecatalog.Status(filters.Status))
    }
    return query.Count(ctx)
}

func (r *repo) CountByCategory(ctx context.Context, tenantID int) (map[string]int, error) {
    catalogs, err := r.client.ServiceCatalog.Query().
        Where(servicecatalog.TenantID(tenantID)).
        All(ctx)
    if err != nil {
        return nil, err
    }

    result := make(map[string]int)
    for _, cat := range catalogs {
        result[cat.Category]++
    }
    return result, nil
}
```

- [ ] **Step 4: 注册路由**

```go
serviceCatalogGroup.GET("/stats", serviceCatalogHandler.Stats)
```

- [ ] **Step 5: 提交**

```bash
git add itsm-backend/handlers/service_catalog/
git commit -m "feat(service-catalog): add stats API"
```

---

## 三、验收标准

### 服务目录完善验收检查清单

- [ ] 服务详情页可以查看服务完整信息
- [ ] 服务编辑页可以修改服务基本信息
- [ ] 服务卡片上可以快速发布/停用服务
- [ ] 后端搜索API可以按关键词搜索服务
- [ ] 前端搜索框可以搜索服务
- [ ] 审批页面可以查看待审批请求
- [ ] 审批页面可以批准/拒绝请求
- [ ] 统计API返回正确的服务数量

### 技术验收

- [ ] 所有API响应时间 < 200ms
- [ ] 前端无console.error
- [ ] 所有表单有validation
- [ ] 错误状态有友好提示

---

## 四、文件修改清单

| 操作 | 文件路径 |
|------|---------|
| Create | `itsm-frontend/src/app/(main)/service-catalog/detail/[id]/page.tsx` |
| Create | `itsm-frontend/src/app/(main)/service-catalog/edit/[id]/page.tsx` |
| Create | `itsm-frontend/src/app/(main)/service-catalog/approvals/page.tsx` |
| Modify | `itsm-frontend/src/app/(main)/service-catalog/components/ServiceItemCard.tsx` |
| Modify | `itsm-frontend/src/app/(main)/service-catalog/hooks/useServiceCatalogData.ts` |
| Modify | `itsm-frontend/src/components/layout/sidebar/menu-config.ts` |
| Modify | `itsm-backend/handlers/service_catalog/service.go` |
| Modify | `itsm-backend/handlers/service_catalog/handler.go` |
| Modify | `itsm-backend/handlers/service_catalog/repository_impl.go` |

---

**Plan complete and saved to `docs/superpowers/plans/2026-04-29-service-catalog-completion.md`. Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**