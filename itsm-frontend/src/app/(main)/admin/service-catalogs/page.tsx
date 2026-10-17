'use client';

import {
  RefreshCw,
  AlertCircle,
  Download,
  BookOpen,
  CheckCircle,
  Filter,
  Search,
  Plus,
  Clock,
  Eye,
  Edit,
  Trash2,
  MoreHorizontal,
} from 'lucide-react';

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Card,
  Table,
  Button,
  Input,
  InputNumber,
  Modal,
  Form,
  Select,
  Space,
  Row,
  Col,
  Statistic,
  Typography,
  Popconfirm,
  Dropdown,
  Tooltip,
  App,
  Tag,
  Descriptions,
  Spin,
  Switch,
  Alert,
} from 'antd';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import { CMDBApi } from '@/lib/api/cmdb-api';
import { BatchActionBar, type BatchAction } from '@/components/business/BatchActionBar';
import { buildCsv, csvDate, downloadCsv } from '@/lib/utils/csv-export';
import type { ServiceItem, CreateServiceItemRequest } from '@/types/service-catalog';
import { ServiceStatus } from '@/types/service-catalog';
import type { CIType, CloudService } from '@/types/biz/cmdb';

const { Title, Text } = Typography;
const { TextArea } = Input;

/** 弹窗表单值：与后端可写字段同名，交付周期用天。 */
interface CatalogFormValues {
  name: string;
  category: string;
  description: string;
  serviceType?: ServiceItem['serviceType'];
  deliveryDays?: number;
  price?: number;
  unit?: string;
  requiresApproval?: boolean;
  approvalLevel?: number;
  sortOrder?: number;
  slaResponseTime?: number;
  slaResolutionTime?: number;
  ciTypeId?: number;
  cloudServiceId?: number;
  status: ServiceStatus;
}

/** 只影响服务端查询的过滤条件；列表的搜索/分类/状态/关联筛选都必须在后端生效。 */
interface CatalogFilters {
  q: string;
  status: ServiceStatus | '';
  category: string;
  ciTypeId?: number;
  cloudServiceId?: number;
}

const EMPTY_FILTERS: CatalogFilters = { q: '', status: '', category: '' };

// 产品默认分类；后端 category 是自由文本，真实分类另从 stats 合并进来。
const DEFAULT_CATEGORIES = ['云服务', '基础设施', '应用服务', '数据服务'];

/** 导出的硬上限：超过它必须显式提示，禁止静默导出半截 CSV。 */
const MAX_EXPORT_RECORDS = 5000;

// 浏览器单 origin 连接池约 6，批量 fan-out 必须限流（AGENTS.md Pattern E）。
const BATCH_CONCURRENCY = 5;

async function runBatch<T, R>(
  items: T[],
  worker: (item: T) => Promise<R>,
): Promise<{ succeeded: R[]; failed: { item: T; message: string }[] }> {
  const succeeded: R[] = [];
  const failed: { item: T; message: string }[] = [];
  let cursor = 0;

  const lanes = Array.from({ length: Math.min(BATCH_CONCURRENCY, items.length) }, async () => {
    while (cursor < items.length) {
      const index = cursor;
      cursor += 1;
      const item = items[index];
      try {
        succeeded.push(await worker(item));
      } catch (error) {
        failed.push({
          item,
          message: error instanceof Error ? error.message : '未知错误',
        });
      }
    }
  });
  await Promise.all(lanes);
  return { succeeded, failed };
}

const ServiceCatalogManagement = () => {
  const { message } = App.useApp();
  const [catalogs, setCatalogs] = useState<ServiceItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [showModal, setShowModal] = useState(false);
  const [editingCatalog, setEditingCatalog] = useState<ServiceItem | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [searchInput, setSearchInput] = useState('');
  const [filters, setFilters] = useState<CatalogFilters>(EMPTY_FILTERS);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [ciTypes, setCiTypes] = useState<CIType[]>([]);
  const [cloudServices, setCloudServices] = useState<CloudService[]>([]);
  const [optionsLoading, setOptionsLoading] = useState(false);
  // 查看详情弹窗（2026-10-14 接线：原「查看详情」按钮无 onClick）
  const [detailVisible, setDetailVisible] = useState(false);
  const [detailCatalog, setDetailCatalog] = useState<ServiceItem | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [form] = Form.useForm();
  // 过滤条件变化时，慢的那次请求不能覆盖新的结果（AGENTS.md Pattern A）。
  const requestSeqRef = useRef(0);

  const handleViewDetail = async (catalog: ServiceItem) => {
    setDetailVisible(true);
    setDetailLoading(true);
    try {
      // 拉详情保证拿到最新全量字段，而非列表行的投影
      const full = await ServiceCatalogApi.getService(catalog.id.toString());
      setDetailCatalog(full);
    } catch {
      // 详情拉取失败时退化为列表行数据，至少不空白
      setDetailCatalog(catalog);
      message.warning('详情加载失败，展示列表缓存数据');
    } finally {
      setDetailLoading(false);
    }
  };

  // 统计数据来自 /service-catalogs/stats：整租户口径，不受列表筛选影响。
  // categories 同时作为分类筛选的选项来源：分类是后端自由文本，选项必须来自真实数据，
  // 不能写死四个候选值，否则筛不到已有目录。
  const [stats, setStats] = useState({
    total: 0,
    enabled: 0,
    disabled: 0,
    categories: [] as { name: string; count: number }[],
  });

  const fetchStats = useCallback(async () => {
    try {
      const data = await ServiceCatalogApi.getCatalogStats();
      setStats({
        total: data.totalServices,
        enabled: data.publishedServices,
        disabled: data.totalServices - data.publishedServices,
        categories: Object.entries(data.categories ?? {})
          .map(([name, count]) => ({ name, count }))
          .sort((a, b) => b.count - a.count || a.name.localeCompare(b.name, 'zh-CN')),
      });
    } catch (error) {
      setLoadError(error instanceof Error ? error.message : '服务目录统计加载失败');
    }
  }, []);

  // 默认四项只是起点：库里已有别的分类时必须能选回原值，否则编辑会把分类悄悄改掉。
  const categoryOptions = useMemo(() => {
    const names = [...DEFAULT_CATEGORIES];
    for (const item of stats.categories) {
      if (!names.includes(item.name)) names.push(item.name);
    }
    return names.map(name => ({ value: name, label: name }));
  }, [stats.categories]);

  // 获取服务目录列表
  const fetchCatalogs = useCallback(async () => {
    const seq = ++requestSeqRef.current;
    try {
      setLoading(true);
      setLoadError(null);
      const response = await ServiceCatalogApi.getServices({
        page,
        pageSize,
        ...(filters.q ? { search: filters.q } : {}),
        ...(filters.status ? { status: filters.status } : {}),
        ...(filters.category ? { category: filters.category } : {}),
        ...(filters.ciTypeId ? { ciTypeId: filters.ciTypeId } : {}),
        ...(filters.cloudServiceId ? { cloudServiceId: filters.cloudServiceId } : {}),
      });
      if (seq !== requestSeqRef.current) return;
      setCatalogs(response.services);
      setTotal(response.total);
    } catch (error) {
      if (seq !== requestSeqRef.current) return;
      setCatalogs([]);
      setTotal(0);
      setLoadError(error instanceof Error ? error.message : '获取服务目录失败');
    } finally {
      if (seq === requestSeqRef.current) setLoading(false);
    }
  }, [page, pageSize, filters]);

  const applyFilters = useCallback((next: Partial<CatalogFilters>) => {
    setFilters(previous => ({ ...previous, ...next }));
    setPage(1);
  }, []);

  const hasActiveFilters =
    filters.q !== '' ||
    filters.status !== '' ||
    filters.category !== '' ||
    filters.ciTypeId !== undefined ||
    filters.cloudServiceId !== undefined;

  const refresh = useCallback(() => {
    void Promise.all([fetchCatalogs(), fetchStats()]);
  }, [fetchCatalogs, fetchStats]);

  useEffect(() => {
    fetchCatalogs();
  }, [fetchCatalogs]);

  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  useEffect(() => {
    const loadOptions = async () => {
      try {
        setOptionsLoading(true);
        const [types, services] = await Promise.all([
          CMDBApi.getCITypes(),
          CMDBApi.getCloudServices(),
        ]);
        setCiTypes(types || []);
        setCloudServices(services.items);
      } catch {
        message.error('加载 CMDB 选项失败，关联筛选暂不可用');
      } finally {
        setOptionsLoading(false);
      }
    };
    loadOptions();
  }, [message]);

  // 表单字段与后端可写 DTO 一一对应；交付周期用天，不再伪装成 availability.responseTime。
  const handleSubmit = async (values: CatalogFormValues) => {
    if (submitting) return;
    // 「关联云服务必须选 CI 类型」由后端校验并返回中文提示，前端不再重复这套规则。
    const payload: CreateServiceItemRequest = {
      name: values.name,
      category: values.category,
      shortDescription: values.description,
      fullDescription: values.description,
      serviceType: values.serviceType,
      deliveryDays: values.deliveryDays,
      // 表单是这两个字段的权威视图：清空即「未配置/免费」，必须显式提交 0 和空串，
      // 省略会让后端 PATCH 保持原值，页面显示与库里数据就此分叉。
      price: values.price ?? 0,
      unit: values.unit ?? '',
      requiresApproval: values.requiresApproval ?? false,
      approvalLevel: values.approvalLevel ?? 0,
      sortOrder: values.sortOrder ?? 0,
      slaResponseTime: values.slaResponseTime,
      slaResolutionTime: values.slaResolutionTime,
      // Select 清空后是 undefined，而后端 PATCH 只认显式 0；省略会让关联原样留在库里，
      // 页面显示「未关联」但新申请仍按旧 CI 类型履约。
      ciTypeId: values.ciTypeId ?? 0,
      cloudServiceId: values.cloudServiceId ?? 0,
      status: values.status,
    };
    setSubmitting(true);
    try {
      if (editingCatalog) {
        await ServiceCatalogApi.updateService(editingCatalog.id, payload);
        message.success('服务目录已更新');
      } else {
        await ServiceCatalogApi.createService(payload);
        message.success('服务目录已创建');
      }
      setShowModal(false);
      setEditingCatalog(null);
      form.resetFields();
      refresh();
    } catch (error) {
      // 名称重复、状态非法、关联资源不存在都是后端业务校验，中文提示必须原样透出，
      // 否则管理员只看到「创建失败」而无从定位。
      message.error(error instanceof Error ? error.message : '保存服务目录失败');
    } finally {
      setSubmitting(false);
    }
  };

  // 编辑服务目录：表单可见的字段都要回填，否则保存时会被清空。
  const handleEdit = (catalog: ServiceItem) => {
    setEditingCatalog(catalog);
    form.setFieldsValue({
      name: catalog.name,
      category: catalog.category,
      description: catalog.shortDescription || catalog.fullDescription,
      serviceType: catalog.serviceType,
      deliveryDays: catalog.deliveryDays,
      price: catalog.price,
      unit: catalog.unit,
      requiresApproval: catalog.requiresApproval ?? false,
      approvalLevel: catalog.approvalLevel ?? 0,
      sortOrder: catalog.sortOrder ?? 0,
      slaResponseTime: catalog.slaResponseTime,
      slaResolutionTime: catalog.slaResolutionTime,
      ciTypeId: catalog.ciTypeId,
      cloudServiceId: catalog.cloudServiceId,
      status: catalog.status,
    });
    setShowModal(true);
  };

  const handleDelete = async (id: string) => {
    try {
      await ServiceCatalogApi.deleteService(id);
      message.success('已删除，目录会转为「已停用」');
      setSelectedRowKeys(previous => previous.filter(key => String(key) !== id));
      refresh();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除失败');
    }
  };

  // 批量删除：限流 + 保留失败选中，让失败项可以原地重试。
  const handleBatchDelete = async () => {
    const ids = selectedRowKeys.map(String);
    setSubmitting(true);
    try {
      const { succeeded, failed } = await runBatch(ids, id =>
        ServiceCatalogApi.deleteService(id),
      );
      setSelectedRowKeys(failed.map(entry => entry.item));
      if (failed.length === 0) {
        message.success(`已删除 ${succeeded.length} 个服务目录`);
      } else {
        message.warning(
          `${succeeded.length} 个成功、${failed.length} 个失败（${failed[0].message}），失败项已保留选中`,
        );
      }
      refresh();
    } finally {
      setSubmitting(false);
    }
  };

  // 批量发布/停用：只提交 status，其余字段由后端 PATCH 语义保持原值。
  const handleBatchStatusChange = async (status: ServiceStatus) => {
    const ids = selectedRowKeys.map(String);
    const action = status === ServiceStatus.PUBLISHED ? '发布' : '停用';
    setSubmitting(true);
    try {
      const { succeeded, failed } = await runBatch(ids, id =>
        ServiceCatalogApi.updateService(id, { status }),
      );
      setSelectedRowKeys(failed.map(entry => entry.item));
      if (failed.length === 0) {
        message.success(`已${action} ${succeeded.length} 个服务目录`);
      } else {
        message.warning(
          `${succeeded.length} 个${action}成功、${failed.length} 个失败（${failed[0].message}），失败项已保留选中`,
        );
      }
      refresh();
    } finally {
      setSubmitting(false);
    }
  };

  const handleStatusChange = async (catalog: ServiceItem, status: ServiceStatus) => {
    try {
      await ServiceCatalogApi.updateService(catalog.id, { status });
      message.success(status === ServiceStatus.PUBLISHED ? '服务目录已发布' : '服务目录已停用');
      refresh();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '状态更新失败');
    }
  };

  // 复制走后端真实字段，副本固定为已停用，确认后再发布；重复副本名由后端返回冲突提示。
  const handleDuplicate = async (catalog: ServiceItem) => {
    try {
      await ServiceCatalogApi.createService({
        name: `${catalog.name} 副本`,
        category: catalog.category,
        shortDescription: catalog.shortDescription,
        fullDescription: catalog.fullDescription,
        serviceType: catalog.serviceType,
        deliveryDays: catalog.deliveryDays,
        unit: catalog.unit,
        price: catalog.price,
        requiresApproval: catalog.requiresApproval ?? false,
        approvalLevel: catalog.approvalLevel ?? 0,
        sortOrder: catalog.sortOrder ?? 0,
        slaResponseTime: catalog.slaResponseTime,
        slaResolutionTime: catalog.slaResolutionTime,
        ciTypeId: catalog.ciTypeId,
        cloudServiceId: catalog.cloudServiceId,
        status: ServiceStatus.RETIRED,
      });
      message.success('已复制为「已停用」目录，确认内容后再发布');
      refresh();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '复制服务目录失败');
    }
  };

  // 导出按当前筛选条件翻页取全，并使用项目统一的 CSV 工具（含 BOM 与公式注入防护）。
  const handleExport = async () => {
    setSubmitting(true);
    try {
      const { services, total, complete } = await ServiceCatalogApi.getAllServices(
        {
          ...(filters.q ? { search: filters.q } : {}),
          ...(filters.status ? { status: filters.status } : {}),
          ...(filters.category ? { category: filters.category } : {}),
          ...(filters.ciTypeId ? { ciTypeId: filters.ciTypeId } : {}),
          ...(filters.cloudServiceId ? { cloudServiceId: filters.cloudServiceId } : {}),
        },
        MAX_EXPORT_RECORDS,
      );
      if (services.length === 0) {
        message.warning('当前筛选条件下没有可导出的服务目录');
        return;
      }
      const csv = buildCsv<ServiceItem>(
        [
          { title: 'ID', render: row => row.id },
          { title: '服务名称', render: row => row.name },
          { title: '分类', render: row => row.category },
          { title: '状态', render: row => row.status },
          { title: '描述', render: row => row.shortDescription },
          { title: '服务类型', render: row => row.serviceType ?? '' },
          { title: '交付天数', render: row => row.deliveryDays ?? '' },
          { title: '计价单位', render: row => row.unit ?? '' },
          { title: '价格', render: row => row.price ?? '' },
          { title: '需审批', render: row => (row.requiresApproval ? '是' : '否') },
          { title: '审批级别', render: row => row.approvalLevel ?? '' },
          {
            title: '关联CI类型',
            render: row => ciTypes.find(item => item.id === row.ciTypeId)?.name ?? '',
          },
          {
            title: '关联云服务',
            render: row => {
              const service = cloudServices.find(item => item.id === row.cloudServiceId);
              return service ? `${service.serviceName} (${service.resourceTypeName})` : '';
            },
          },
          { title: '创建时间', render: row => csvDate(String(row.createdAt)) },
          { title: '更新时间', render: row => csvDate(String(row.updatedAt)) },
        ],
        services,
      );
      downloadCsv(`service-catalogs-${new Date().toISOString().slice(0, 10)}`, csv);
      if (complete) {
        message.success(`已导出 ${services.length} 条服务目录`);
      } else {
        message.warning(
          `当前筛选共 ${total} 条，超过单次导出上限 ${MAX_EXPORT_RECORDS} 条，本次仅导出前 ${services.length} 条，请收窄条件后重试`,
        );
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '导出失败');
    } finally {
      setSubmitting(false);
    }
  };

  // 表格列定义
  const columns = [
    {
      title: '服务名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: ServiceItem) => (
        <div>
          <div className="font-medium text-gray-900">{name}</div>
          <div className="text-sm text-gray-500 mt-1">{record.shortDescription}</div>
        </div>
      ),
    },
    {
      title: '分类',
      dataIndex: 'category',
      key: 'category',
      width: 120,
      render: (category: string) => <Tag color="blue">{category}</Tag>,
    },
    {
      title: '关联CI类型',
      dataIndex: 'ciTypeId',
      key: 'ciTypeId',
      width: 140,
      render: (_: number | undefined, record: ServiceItem) => {
        const type = ciTypes.find(item => item.id === record.ciTypeId);
        return type ? type.name : '-';
      },
    },
    {
      title: '关联云服务',
      dataIndex: 'cloudServiceId',
      key: 'cloudServiceId',
      width: 180,
      render: (_: number | undefined, record: ServiceItem) => {
        const service = cloudServices.find(item => item.id === record.cloudServiceId);
        return service ? `${service.serviceName} (${service.resourceTypeName})` : '-';
      },
    },
    {
      title: '交付周期',
      key: 'deliveryDays',
      width: 120,
      render: (_: unknown, record: ServiceItem) => (
        <span className="text-sm flex items-center">
          <Clock className="w-4 h-4 mr-1" />
          {record.deliveryDays != null ? `${record.deliveryDays} 天` : '未配置'}
        </span>
      ),
    },
    {
      title: '价格',
      key: 'price',
      width: 110,
      render: (_: unknown, record: ServiceItem) =>
        record.price != null ? `${record.price}${record.unit ? ` / ${record.unit}` : ''}` : '免费',
    },
    {
      title: '审批',
      key: 'approval',
      width: 110,
      render: (_: unknown, record: ServiceItem) =>
        record.requiresApproval ? (
          <Tag color="orange">{`${record.approvalLevel ?? 0} 级`}</Tag>
        ) : (
          <Text type="secondary">无需审批</Text>
        ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: ServiceStatus) => (
        <Tag
          color={status === ServiceStatus.PUBLISHED ? 'green' : 'red'}
          icon={
            status === ServiceStatus.PUBLISHED ? (
              <CheckCircle className="w-3 h-3" />
            ) : (
              <AlertCircle className="w-3 h-3" />
            )
          }
        >
          {status === ServiceStatus.PUBLISHED ? '已发布' : '已停用'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 150,
      render: (date: string) => (
        <span className="text-sm text-gray-600">{new Date(date).toLocaleDateString('zh-CN')}</span>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      render: (_: unknown, record: ServiceItem) => (
        <Space>
          <Tooltip title="查看详情">
            <Button
              aria-label="查看详情"
              type="text"
              icon={<Eye className="w-4 h-4" />}
              size="small"
              onClick={() => handleViewDetail(record)}
            />
          </Tooltip>
          <Tooltip title="编辑">
            <Button
              aria-label="编辑"
              type="text"
              icon={<Edit className="w-4 h-4" />}
              onClick={() => handleEdit(record)}
              size="small"
            />
          </Tooltip>
          <Popconfirm
            title="确定要删除这个服务目录吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Tooltip title="删除">
              <Button aria-label="删除" type="text" icon={<Trash2 className="w-4 h-4" />} danger size="small" />
            </Tooltip>
          </Popconfirm>
          <Dropdown
            menu={{
              items: [
                {
                  key: 'enable',
                  label: '发布',
                  icon: <CheckCircle className="w-4 h-4" />,
                  disabled: record.status === ServiceStatus.PUBLISHED,
                  onClick: () => handleStatusChange(record, ServiceStatus.PUBLISHED),
                },
                {
                  key: 'disable',
                  label: '停用',
                  icon: <AlertCircle className="w-4 h-4" />,
                  disabled: record.status === ServiceStatus.RETIRED,
                  onClick: () => handleStatusChange(record, ServiceStatus.RETIRED),
                },
                {
                  type: 'divider' as const,
                },
                {
                  key: 'duplicate',
                  label: '复制',
                  icon: <Plus className="w-4 h-4" />,
                  onClick: () => handleDuplicate(record),
                },
              ],
            }}
          >
            <Button aria-label="更多操作" type="text" icon={<MoreHorizontal className="w-4 h-4" />} size="small" />
          </Dropdown>
        </Space>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* 页面标题 */}
      <div>
        <Title level={2} className="!mb-2">
          <BookOpen className="inline-block w-6 h-6 mr-2" />
          服务目录管理
        </Title>
        <Text type="secondary">管理IT服务目录和服务分类</Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="总服务数"
              value={stats.total}
              prefix={<BookOpen className="w-5 h-5" />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="启用服务"
              value={stats.enabled}
              prefix={<CheckCircle className="w-5 h-5" />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="禁用服务"
              value={stats.disabled}
              prefix={<AlertCircle className="w-5 h-5" />}
              styles={{ content: { color: '#ff4d4f' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="服务分类"
              value={stats.categories.length}
              prefix={<Filter className="w-5 h-5" />}
              styles={{ content: { color: '#722ed1' } }}
            />
          </Card>
        </Col>
      </Row>

      {/* 工具栏：所有筛选都提交给服务端，页面上不再对当前页二次过滤 */}
      <Card className="enterprise-card">
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} sm={12} md={6}>
            <Input
              placeholder="搜索名称或描述..."
              prefix={<Search className="w-4 h-4" />}
              value={searchInput}
              onChange={event => {
                setSearchInput(event.target.value);
                if (event.target.value === '') applyFilters({ q: '' });
              }}
              onPressEnter={() => applyFilters({ q: searchInput.trim() })}
              allowClear
            />
          </Col>
          <Col xs={24} sm={12} md={4}>
            <Select
              placeholder="状态筛选"
              value={filters.status || undefined}
              onChange={value => applyFilters({ status: (value ?? '') as CatalogFilters['status'] })}
              allowClear
              style={{ width: '100%' }}
              options={[
                { value: ServiceStatus.PUBLISHED, label: '已发布' },
                { value: ServiceStatus.RETIRED, label: '已停用' },
              ]}
            />
          </Col>
          <Col xs={24} sm={12} md={4}>
            <Select
              placeholder="分类筛选"
              value={filters.category || undefined}
              onChange={value => applyFilters({ category: value ?? '' })}
              allowClear
              style={{ width: '100%' }}
              options={stats.categories.map(item => ({
                value: item.name,
                label: `${item.name}（${item.count}）`,
              }))}
            />
          </Col>
          <Col xs={24} sm={12} md={5}>
            <Select
              placeholder="CI类型筛选"
              value={filters.ciTypeId}
              onChange={value => applyFilters({ ciTypeId: value })}
              allowClear
              loading={optionsLoading}
              style={{ width: '100%' }}
              options={ciTypes.map(type => ({ value: type.id, label: type.name }))}
            />
          </Col>
          <Col xs={24} sm={12} md={5}>
            <Select
              placeholder="云服务筛选"
              value={filters.cloudServiceId}
              onChange={value => applyFilters({ cloudServiceId: value })}
              allowClear
              loading={optionsLoading}
              showSearch
              optionFilterProp="label"
              style={{ width: '100%' }}
              options={cloudServices.map(service => ({
                value: service.id,
                label: `${service.serviceName} (${service.resourceTypeName})`,
              }))}
            />
          </Col>
          <Col xs={24} sm={24} md={24}>
            <Space>
              <Button
                type="primary"
                icon={<Plus className="w-4 h-4" />}
                onClick={() => {
                  setEditingCatalog(null);
                  form.resetFields();
                  setShowModal(true);
                }}
              >
                新建服务目录
              </Button>
              <Button
                icon={<Download className="w-4 h-4" />}
                loading={submitting}
                onClick={handleExport}
              >
                按当前筛选导出
              </Button>
              <Button icon={<RefreshCw className="w-4 h-4" />} onClick={refresh}>
                刷新
              </Button>
              {hasActiveFilters && (
                <Button
                  type="link"
                  onClick={() => {
                    setSearchInput('');
                    setFilters(EMPTY_FILTERS);
                    setPage(1);
                  }}
                >
                  清空筛选
                </Button>
              )}
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 批量操作工具栏（统一 BatchActionBar） */}
      <BatchActionBar
        selectedCount={selectedRowKeys.length}
        itemLabel="服务目录"
        onClear={() => setSelectedRowKeys([])}
        actions={[
          {
            key: 'publish',
            label: '批量发布',
            type: 'primary',
            onClick: () => handleBatchStatusChange(ServiceStatus.PUBLISHED),
          },
          {
            key: 'retire',
            label: '批量停用',
            onClick: () => handleBatchStatusChange(ServiceStatus.RETIRED),
          },
          {
            key: 'delete',
            label: '批量删除',
            danger: true,
            confirmTitle: `确定删除选中的 ${selectedRowKeys.length} 个服务目录？此操作不可撤销`,
            onClick: handleBatchDelete,
          } as BatchAction,
        ]}
      />

      {/* 加载失败必须显式可见，不能静默显示空表格假装「没有数据」 */}
      {loadError && (
        <Alert
          type="error"
          showIcon
          title="服务目录加载失败"
          description={loadError}
          action={
            <Button size="small" icon={<RefreshCw className="w-4 h-4" />} onClick={refresh}>
              重试
            </Button>
          }
        />
      )}

      {/* 服务目录表格：分页、总数和数据都由服务端返回 */}
      <Card className="enterprise-card">
        <Table
          columns={columns}
          dataSource={catalogs}
          rowKey="id"
          loading={loading}
          rowSelection={{
            selectedRowKeys,
            onChange: keys => setSelectedRowKeys(keys),
            selections: [Table.SELECTION_ALL, Table.SELECTION_INVERT, Table.SELECTION_NONE],
          }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: recordCount => `共 ${recordCount} 条记录`,
          }}
          onChange={pagination => {
            setPage(pagination.current ?? 1);
            setPageSize(pagination.pageSize ?? 20);
          }}
          scroll={{ x: 1000 }}
        />
      </Card>

      {/* 创建/编辑模态框 */}
      <Modal
        title={
          <span>
            <BookOpen className="w-4 h-4 mr-2" />
            {editingCatalog ? '编辑服务目录' : '新建服务目录'}
          </span>
        }
        open={showModal}
        onCancel={() => {
          if (submitting) return;
          setShowModal(false);
          setEditingCatalog(null);
          form.resetFields();
        }}
        footer={null}
        width={600}
        // 十几项配置填到一半，点中遮罩就整份清空重来没有任何提示。
        maskClosable={false}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          initialValues={{ status: ServiceStatus.PUBLISHED, requiresApproval: true, approvalLevel: 1 }}
        >
          <Form.Item
            name="name"
            label="服务名称"
            rules={[{ required: true, message: '请输入服务名称' }]}
          >
            <Input placeholder="请输入服务名称" maxLength={100} showCount />
          </Form.Item>

          <Form.Item
            name="category"
            label="服务分类"
            rules={[{ required: true, message: '请选择服务分类' }]}
          >
            {/* 后端 category 是自由文本，真实取值来自 stats；只给固定四项会让已有分类选不中。 */}
            <Select
              placeholder="请选择服务分类"
              showSearch
              options={categoryOptions}
            />
          </Form.Item>

          <Form.Item
            name="description"
            label="服务描述"
            rules={[{ required: true, message: '请输入服务描述' }]}
          >
            <TextArea rows={3} placeholder="请输入服务描述" showCount maxLength={500} />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="serviceType" label="服务类型" tooltip="用于动态加载对应的申请表单">
                <Select placeholder="选择服务类型" allowClear options={[
                  { value: 'vm', label: '云服务器 (VM)' },
                  { value: 'rds', label: '数据库 (RDS)' },
                  { value: 'oss', label: '对象存储 (OSS)' },
                  { value: 'network', label: '网络服务' },
                  { value: 'storage', label: '存储服务' },
                  { value: 'security', label: '安全服务' },
                  { value: 'custom', label: '自定义服务' },
                ]} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="deliveryDays"
                label="交付周期"
                extra="按自然日保存，范围 1–3650 天"
                rules={[
                  {
                    type: 'integer',
                    min: 1,
                    max: 3650,
                    message: '交付天数必须是 1 到 3650 之间的整数',
                  },
                ]}
              >
                <InputNumber min={1} max={3650} precision={0} suffix="天" style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>

          {/* 计价配置：留空即「未配置」，导出与门户展示都读这两个字段 */}
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="price" label="价格" tooltip="留空表示免费或未定价">
                <InputNumber min={0} precision={2} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="unit" label="计价单位">
                <Select
                  placeholder="未配置"
                  allowClear
                  options={[
                    { value: '月', label: '月' },
                    { value: '次', label: '次' },
                    { value: '用户', label: '用户' },
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>

          {/* 审批配置 */}
          <div className="bg-gray-50 p-4 rounded-lg mb-4">
            <Text strong className="block mb-3">
              审批配置
            </Text>
            <Row gutter={16}>
              <Col span={8}>
                <Form.Item name="requiresApproval" label="需要审批" valuePropName="checked">
                  <Switch checkedChildren="是" unCheckedChildren="否" />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item
                  name="approvalLevel"
                  label="审批级别"
                  tooltip="0 表示未配置审批级别；需要审批时建议至少 1 级"
                  dependencies={['requiresApproval']}
                  rules={[
                    {
                      validator: (_, value) =>
                        form.getFieldValue('requiresApproval') && !value
                          ? Promise.reject(new Error('开启审批后必须选择 1–3 级'))
                          : Promise.resolve(),
                    },
                  ]}
                >
                  <InputNumber min={0} max={3} placeholder="0-3" style={{ width: '100%' }} />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item name="sortOrder" label="排序" tooltip="门户列表升序，0 为默认">
                  <InputNumber min={0} placeholder="0" style={{ width: '100%' }} />
                </Form.Item>
              </Col>
            </Row>
          </div>

          {/* SLA配置 */}
          <div className="bg-blue-50 p-4 rounded-lg mb-4">
            <Text strong className="block mb-3">
              SLA配置
            </Text>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="slaResponseTime" label="响应时间(分钟)" tooltip="SLA响应时间要求">
                  <InputNumber min={0} placeholder="例如：30" style={{ width: '100%' }} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="slaResolutionTime"
                  label="解决时间(分钟)"
                  tooltip="SLA解决时间要求"
                >
                  <InputNumber min={0} placeholder="例如：240" style={{ width: '100%' }} />
                </Form.Item>
              </Col>
            </Row>
          </div>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="ciTypeId" label="关联CI类型">
                <Select placeholder="选择CI类型" allowClear loading={optionsLoading} options={ciTypes.map(type => ({ value: type.id, label: type.name }))} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="cloudServiceId" label="关联云服务">
                <Select
                  placeholder="选择云服务"
                  allowClear
                  loading={optionsLoading}
                  showSearch
                  optionFilterProp="label"
                  options={cloudServices.map(service => ({
                    value: service.id,
                    label: `${service.serviceName} (${service.resourceTypeName})`,
                  }))}
                />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="status"
                label="状态"
                extra="后端只有「已发布 / 已停用」两态，草稿不是服务目录的状态"
                rules={[{ required: true, message: '请选择状态' }]}
              >
                <Select options={[
                  { value: ServiceStatus.PUBLISHED, label: '已发布' },
                  { value: ServiceStatus.RETIRED, label: '已停用' },
                ]} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item className="mb-0 mt-6">
            <Space className="w-full justify-end">
              <Button
                disabled={submitting}
                onClick={() => {
                  setShowModal(false);
                  setEditingCatalog(null);
                  form.resetFields();
                }}
              >
                取消
              </Button>
              <Button type="primary" htmlType="submit" loading={submitting}>
                {editingCatalog ? '更新' : '创建'}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 服务目录详情弹窗 */}
      <Modal
        title="服务目录详情"
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={<Button onClick={() => setDetailVisible(false)}>关闭</Button>}
        width={640}
      >
        <Spin spinning={detailLoading}>
          {detailCatalog && (
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label="ID">{detailCatalog.id}</Descriptions.Item>
              <Descriptions.Item label="名称">{detailCatalog.name}</Descriptions.Item>
              <Descriptions.Item label="分类">{detailCatalog.category}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={detailCatalog.status === ServiceStatus.PUBLISHED ? 'green' : 'red'}>
                  {detailCatalog.status === ServiceStatus.PUBLISHED ? '已发布' : '已停用'}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="需要审批">
                {detailCatalog.requiresApproval ? `是（${detailCatalog.approvalLevel ?? 0} 级）` : '否'}
              </Descriptions.Item>
              <Descriptions.Item label="CI 类型">
                {ciTypes.find(item => item.id === detailCatalog.ciTypeId)?.name || '未关联'}
              </Descriptions.Item>
              <Descriptions.Item label="云服务">
                {cloudServices.find(item => item.id === detailCatalog.cloudServiceId)?.serviceName || '未关联'}
              </Descriptions.Item>
              <Descriptions.Item label="交付周期">
                {detailCatalog.deliveryDays != null ? `${detailCatalog.deliveryDays} 天` : '未配置'}
              </Descriptions.Item>
              <Descriptions.Item label="价格">
                {detailCatalog.price != null && detailCatalog.price > 0
                  ? `${detailCatalog.price}${detailCatalog.unit ? ` / ${detailCatalog.unit}` : ''}`
                  : '免费'}
              </Descriptions.Item>
              <Descriptions.Item label="服务类型">
                {detailCatalog.serviceType ?? '未配置'}
              </Descriptions.Item>
              <Descriptions.Item label="排序">
                {detailCatalog.sortOrder ?? 0}
              </Descriptions.Item>
              <Descriptions.Item label="简介" span={2}>
                {detailCatalog.shortDescription || '—'}
              </Descriptions.Item>
              <Descriptions.Item label="详细说明" span={2}>
                {detailCatalog.fullDescription || '—'}
              </Descriptions.Item>
            </Descriptions>
          )}
        </Spin>
      </Modal>
    </div>
  );
};

export default ServiceCatalogManagement;
