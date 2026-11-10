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
import { serviceCategoryLabel, pricingUnitLabel } from '@/constants/service-catalog';

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { formatDateTime } from '@/lib/formatters';
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
const DEFAULT_CATEGORIES = ['Cloud services', 'Infrastructure', 'Application Service', 'Data Services'];

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
          message: error instanceof Error ? error.message : 'Unknown error',
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
      message.warning('Details loading failed, display list cache data');
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
          .sort((a, b) => b.count - a.count || a.name.localeCompare(b.name, 'en-US')),
      });
    } catch (error) {
      setLoadError(error instanceof Error ? error.message : 'Service catalog statistics loading failed');
    }
  }, []);

  // 默认四项只是起点：库里已有别的分类时必须能选回原值，否则编辑会把分类悄悄改掉。
  const categoryOptions = useMemo(() => {
    const names = [...DEFAULT_CATEGORIES];
    for (const item of stats.categories) {
      if (!names.includes(item.name)) names.push(item.name);
    }
    return names.map(name => ({ value: name, label: serviceCategoryLabel(name) }));
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
      setLoadError(error instanceof Error ? error.message : 'Failed to obtain service directory');
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
        message.error('Failed to load CMDB options, associated filtering is not available yet');
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
        message.success('Service catalog updated');
      } else {
        await ServiceCatalogApi.createService(payload);
        message.success('Service catalog created');
      }
      setShowModal(false);
      setEditingCatalog(null);
      form.resetFields();
      refresh();
    } catch (error) {
      // 名称重复、状态非法、关联资源不存在都是后端业务校验，中文提示必须原样透出，
      // 否则管理员只看到「创建失败」而无从定位。
      message.error(error instanceof Error ? error.message : 'Failed to save service catalog');
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
      message.success('Deleted, the directory will be converted to "Deactivated"');
      setSelectedRowKeys(previous => previous.filter(key => String(key) !== id));
      refresh();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Delete failed');
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
        message.success(`Deleted ${succeeded.length} service directory`);
      } else {
        message.warning(
          `${succeeded.length} success,${failed.length} failed (${failed[0].message}), the failed item remains selected`,
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
    const action = status === ServiceStatus.PUBLISHED ? 'Release' : 'Disable';
    setSubmitting(true);
    try {
      const { succeeded, failed } = await runBatch(ids, id =>
        ServiceCatalogApi.updateService(id, { status }),
      );
      setSelectedRowKeys(failed.map(entry => entry.item));
      if (failed.length === 0) {
        message.success(`Already ${action} ${succeeded.length} service directory`);
      } else {
        message.warning(
          `${succeeded.length} items ${action} Success,${failed.length} failed (${failed[0].message}), the failed item remains selected`,
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
      message.success(status === ServiceStatus.PUBLISHED ? 'Service catalog has been published' : 'Service Catalog is deactivated');
      refresh();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Failed to update status');
    }
  };

  // 复制走后端真实字段，副本固定为已停用，确认后再发布；重复副本名由后端返回冲突提示。
  const handleDuplicate = async (catalog: ServiceItem) => {
    try {
      await ServiceCatalogApi.createService({
        name: `${catalog.name} Copy`,
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
      message.success('Copied as "Deactivated" directory, confirm the content before publishing');
      refresh();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Failed to copy service directory');
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
        message.warning('There are no exportable service catalogs under the current filter conditions');
        return;
      }
      const csv = buildCsv<ServiceItem>(
        [
          { title: 'ID', render: row => row.id },
          { title: 'Service Name', render: row => row.name },
          { title: 'Category', render: row => serviceCategoryLabel(row.category) },
          { title: 'Status', render: row => row.status },
          { title: 'Description', render: row => row.shortDescription },
          { title: 'Service Type', render: row => row.serviceType ?? '' },
          { title: 'Delivery days', render: row => row.deliveryDays ?? '' },
          { title: 'Pricing unit', render: row => pricingUnitLabel(row.unit) },
          { title: 'Price', render: row => row.price ?? '' },
          { title: 'Approval required', render: row => (row.requiresApproval ? 'Yes' : 'No') },
          { title: 'Approval level', render: row => row.approvalLevel ?? '' },
          {
            title: 'Associated CI type',
            render: row => ciTypes.find(item => item.id === row.ciTypeId)?.name ?? '',
          },
          {
            title: 'Associated cloud service',
            render: row => {
              const service = cloudServices.find(item => item.id === row.cloudServiceId);
              return service ? `${service.serviceName} (${service.resourceTypeName})` : '';
            },
          },
          { title: 'Created', render: row => csvDate(String(row.createdAt)) },
          { title: 'Updated At', render: row => csvDate(String(row.updatedAt)) },
        ],
        services,
      );
      downloadCsv(`service-catalogs-${new Date().toISOString().slice(0, 10)}`, csv);
      if (complete) {
        message.success(`Exported ${services.length} service directory`);
      } else {
        message.warning(
          `Current filtering total ${total} items, exceeding the single export limit ${MAX_EXPORT_RECORDS} items, this time only the front ${services.length} Articles, please narrow the conditions and try again.`,
        );
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Export failed');
    } finally {
      setSubmitting(false);
    }
  };

  // 表格列定义
  const columns = [
    {
      title: 'Service Name',
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
      title: 'Category',
      dataIndex: 'category',
      key: 'category',
      width: 120,
      render: (category: string) => <Tag color="blue">{serviceCategoryLabel(category)}</Tag>,
    },
    {
      title: 'Associated CI type',
      dataIndex: 'ciTypeId',
      key: 'ciTypeId',
      width: 140,
      render: (_: number | undefined, record: ServiceItem) => {
        const type = ciTypes.find(item => item.id === record.ciTypeId);
        return type ? type.name : '-';
      },
    },
    {
      title: 'Associated cloud service',
      dataIndex: 'cloudServiceId',
      key: 'cloudServiceId',
      width: 180,
      render: (_: number | undefined, record: ServiceItem) => {
        const service = cloudServices.find(item => item.id === record.cloudServiceId);
        return service ? `${service.serviceName} (${service.resourceTypeName})` : '-';
      },
    },
    {
      title: 'Lead time',
      key: 'deliveryDays',
      width: 120,
      render: (_: unknown, record: ServiceItem) => (
        <span className="text-sm flex items-center">
          <Clock className="w-4 h-4 mr-1" />
          {record.deliveryDays != null ? `${record.deliveryDays} days` : 'Not configured'}
        </span>
      ),
    },
    {
      title: 'Price',
      key: 'price',
      width: 110,
      render: (_: unknown, record: ServiceItem) =>
        record.price != null ? `${record.price}${record.unit ? ` / ${pricingUnitLabel(record.unit)}` : ''}` : 'Free',
    },
    {
      title: 'Approval',
      key: 'approval',
      width: 110,
      render: (_: unknown, record: ServiceItem) =>
        record.requiresApproval ? (
          <Tag color="orange">{`${record.approvalLevel ?? 0} level`}</Tag>
        ) : (
          <Text type="secondary">No approval required</Text>
        ),
    },
    {
      title: 'Status',
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
          {status === ServiceStatus.PUBLISHED ? 'Published' : 'Disabled'}
        </Tag>
      ),
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 150,
      render: (date: string) => (
        <span className="text-sm text-gray-600">{formatDateTime(date)}</span>
      ),
    },
    {
      title: 'Actions',
      key: 'action',
      width: 180,
      render: (_: unknown, record: ServiceItem) => (
        <Space>
          <Tooltip title="View Details">
            <Button
              aria-label="View Details"
              type="text"
              icon={<Eye className="w-4 h-4" />}
              size="small"
              onClick={() => handleViewDetail(record)}
            />
          </Tooltip>
          <Tooltip title="Edit">
            <Button
              aria-label="Edit"
              type="text"
              icon={<Edit className="w-4 h-4" />}
              onClick={() => handleEdit(record)}
              size="small"
            />
          </Tooltip>
          <Popconfirm
            title="Are you sure you want to delete this service directory?"
            onConfirm={() => handleDelete(record.id)}
            okText="OK"
            cancelText="Cancel"
          >
            <Tooltip title="Delete">
              <Button aria-label="Delete" type="text" icon={<Trash2 className="w-4 h-4" />} danger size="small" />
            </Tooltip>
          </Popconfirm>
          <Dropdown
            menu={{
              items: [
                {
                  key: 'enable',
                  label: 'Release',
                  icon: <CheckCircle className="w-4 h-4" />,
                  disabled: record.status === ServiceStatus.PUBLISHED,
                  onClick: () => handleStatusChange(record, ServiceStatus.PUBLISHED),
                },
                {
                  key: 'disable',
                  label: 'Disable',
                  icon: <AlertCircle className="w-4 h-4" />,
                  disabled: record.status === ServiceStatus.RETIRED,
                  onClick: () => handleStatusChange(record, ServiceStatus.RETIRED),
                },
                {
                  type: 'divider' as const,
                },
                {
                  key: 'duplicate',
                  label: 'Copy',
                  icon: <Plus className="w-4 h-4" />,
                  onClick: () => handleDuplicate(record),
                },
              ],
            }}
          >
            <Button aria-label="More actions" type="text" icon={<MoreHorizontal className="w-4 h-4" />} size="small" />
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

          Service CatalogsManage
        </Title>
        <Text type="secondary">Manage IT service catalog and service classification</Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="Total number of services"
              value={stats.total}
              prefix={<BookOpen className="w-5 h-5" />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="Enable service"
              value={stats.enabled}
              prefix={<CheckCircle className="w-5 h-5" />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="Disable service"
              value={stats.disabled}
              prefix={<AlertCircle className="w-5 h-5" />}
              styles={{ content: { color: '#ff4d4f' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="enterprise-card">
            <Statistic
              title="Service Category"
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
              placeholder="Search name or description..."
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
              placeholder="Status"
              value={filters.status || undefined}
              onChange={value => applyFilters({ status: (value ?? '') as CatalogFilters['status'] })}
              allowClear
              style={{ width: '100%' }}
              options={[
                { value: ServiceStatus.PUBLISHED, label: 'Published' },
                { value: ServiceStatus.RETIRED, label: 'Disabled' },
              ]}
            />
          </Col>
          <Col xs={24} sm={12} md={4}>
            <Select
              placeholder="Category Filter"
              value={filters.category || undefined}
              onChange={value => applyFilters({ category: value ?? '' })}
              allowClear
              style={{ width: '100%' }}
              options={stats.categories.map(item => ({
                value: item.name,
                label: `${serviceCategoryLabel(item.name)} (${item.count})`,
              }))}
            />
          </Col>
          <Col xs={24} sm={12} md={5}>
            <Select
              placeholder="CI type filtering"
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
              placeholder="Cloud service filtering"
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

                Create a new service directory
              </Button>
              <Button
                icon={<Download className="w-4 h-4" />}
                loading={submitting}
                onClick={handleExport}
              >

                Export by current filter
              </Button>
              <Button icon={<RefreshCw className="w-4 h-4" />} onClick={refresh}>

                Refresh
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

                  Clear filter
                </Button>
              )}
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 批量操作工具栏（统一 BatchActionBar） */}
      <BatchActionBar
        selectedCount={selectedRowKeys.length}
        itemLabel="Service Catalog"
        onClear={() => setSelectedRowKeys([])}
        actions={[
          {
            key: 'publish',
            label: 'Batch publishing',
            type: 'primary',
            onClick: () => handleBatchStatusChange(ServiceStatus.PUBLISHED),
          },
          {
            key: 'retire',
            label: 'Batch Deactivate',
            onClick: () => handleBatchStatusChange(ServiceStatus.RETIRED),
          },
          {
            key: 'delete',
            label: 'Batch Delete',
            danger: true,
            confirmTitle: `Confirm to delete the selected ${selectedRowKeys.length} Service directory? This action is irreversible`,
            onClick: handleBatchDelete,
          } as BatchAction,
        ]}
      />

      {/* 加载失败必须显式可见，不能静默显示空表格假装「没有数据」 */}
      {loadError && (
        <Alert
          type="error"
          showIcon
          title="Failed to load service catalog"
          description={loadError}
          action={
            <Button size="small" icon={<RefreshCw className="w-4 h-4" />} onClick={refresh}>

              Retry
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
            showTotal: recordCount => `Total ${recordCount} records`,
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
            {editingCatalog ? 'Edit service directory' : 'Create a new service directory'}
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
            label="Service Name"
            rules={[{ required: true, message: 'Please enter service name' }]}
          >
            <Input placeholder="Please enter service name" maxLength={100} showCount />
          </Form.Item>

          <Form.Item
            name="category"
            label="Service Category"
            rules={[{ required: true, message: 'Please select service category' }]}
          >
            {/* 后端 category 是自由文本，真实取值来自 stats；只给固定四项会让已有分类选不中。 */}
            <Select
              placeholder="Please select service category"
              showSearch
              options={categoryOptions}
            />
          </Form.Item>

          <Form.Item
            name="description"
            label="Service Description"
            rules={[{ required: true, message: 'Please enter service description' }]}
          >
            <TextArea rows={3} placeholder="Please enter service description" showCount maxLength={500} />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="serviceType" label="Service Type" tooltip="Used to dynamically load the corresponding application form">
                <Select placeholder="Select service type" allowClear options={[
                  { value: 'vm', label: 'Cloud Server (VM)' },
                  { value: 'rds', label: 'Database (RDS)' },
                  { value: 'oss', label: 'Object Storage (OSS)' },
                  { value: 'network', label: 'Network Services' },
                  { value: 'storage', label: 'Storage Service' },
                  { value: 'security', label: 'Security Services' },
                  { value: 'custom', label: 'Custom service' },
                ]} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="deliveryDays"
                label="Lead time"
                extra="Save by natural day, range 1–3650 days"
                rules={[
                  {
                    type: 'integer',
                    min: 1,
                    max: 3650,
                    message: 'Delivery days must be an integer between 1 and 3650',
                  },
                ]}
              >
                <InputNumber min={1} max={3650} precision={0} suffix="days" style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>

          {/* 计价配置：留空即「未配置」，导出与门户展示都读这两个字段 */}
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="price" label="Price" tooltip="Leave blank to indicate free or unpriced">
                <InputNumber min={0} precision={2} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="unit" label="Pricing unit">
                <Select
                  placeholder="Not configured"
                  allowClear
                  options={[
                    { value: '月', label: 'month' },
                    { value: '次', label: 'times' },
                    { value: '用户', label: 'Users' },
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>

          {/* 审批配置 */}
          <div className="bg-gray-50 p-4 rounded-lg mb-4">
            <Text strong className="block mb-3">

              Approval
            </Text>
            <Row gutter={16}>
              <Col span={8}>
                <Form.Item name="requiresApproval" label="Approval required" valuePropName="checked">
                  <Switch checkedChildren="Yes" unCheckedChildren="No" />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item
                  name="approvalLevel"
                  label="Approval level"
                  tooltip="0 indicates that the approval level is not configured; when approval is required, it is recommended to use at least level 1"
                  dependencies={['requiresApproval']}
                  rules={[
                    {
                      validator: (_, value) =>
                        form.getFieldValue('requiresApproval') && !value
                          ? Promise.reject(new Error('Level 1–3 must be selected after opening approval'))
                          : Promise.resolve(),
                    },
                  ]}
                >
                  <InputNumber min={0} max={3} placeholder="0-3" style={{ width: '100%' }} />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item name="sortOrder" label="Sort Order" tooltip="Portal list in ascending order, 0 is the default">
                  <InputNumber min={0} placeholder="0" style={{ width: '100%' }} />
                </Form.Item>
              </Col>
            </Row>
          </div>

          {/* SLA配置 */}
          <div className="bg-blue-50 p-4 rounded-lg mb-4">
            <Text strong className="block mb-3">

              SLA
            </Text>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="slaResponseTime" label="Response Time (min)" tooltip="SLA response time requirements">
                  <InputNumber min={0} placeholder="For example: 30" style={{ width: '100%' }} />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="slaResolutionTime"
                  label="Resolution Time (min)"
                  tooltip="SLA resolution time requirements"
                >
                  <InputNumber min={0} placeholder="Example: 240" style={{ width: '100%' }} />
                </Form.Item>
              </Col>
            </Row>
          </div>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="ciTypeId" label="Associated CI type">
                <Select placeholder="Select CI type" allowClear loading={optionsLoading} options={ciTypes.map(type => ({ value: type.id, label: type.name }))} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="cloudServiceId" label="Associated cloud service">
                <Select
                  placeholder="Select cloud service"
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
                label="Status"
                extra="The backend only has two states: &quot;Published/Deactivated&quot;, and the draft is not the status of the service catalog."
                rules={[{ required: true, message: 'Please select status' }]}
              >
                <Select options={[
                  { value: ServiceStatus.PUBLISHED, label: 'Published' },
                  { value: ServiceStatus.RETIRED, label: 'Disabled' },
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

                Cancel
              </Button>
              <Button type="primary" htmlType="submit" loading={submitting}>
                {editingCatalog ? 'Update' : 'Create'}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 服务目录详情弹窗 */}
      <Modal
        title="Service catalog details"
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={<Button onClick={() => setDetailVisible(false)}>Close</Button>}
        width={640}
      >
        <Spin spinning={detailLoading}>
          {detailCatalog && (
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label="ID">{detailCatalog.id}</Descriptions.Item>
              <Descriptions.Item label="Name">{detailCatalog.name}</Descriptions.Item>
              <Descriptions.Item label="Category">{serviceCategoryLabel(detailCatalog.category)}</Descriptions.Item>
              <Descriptions.Item label="Status">
                <Tag color={detailCatalog.status === ServiceStatus.PUBLISHED ? 'green' : 'red'}>
                  {detailCatalog.status === ServiceStatus.PUBLISHED ? 'Published' : 'Disabled'}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="Approval required">
                {detailCatalog.requiresApproval ? `Yes(${detailCatalog.approvalLevel ?? 0} level)` : 'No'}
              </Descriptions.Item>
              <Descriptions.Item label="CI Types">
                {ciTypes.find(item => item.id === detailCatalog.ciTypeId)?.name || 'Not associated'}
              </Descriptions.Item>
              <Descriptions.Item label="Cloud services">
                {cloudServices.find(item => item.id === detailCatalog.cloudServiceId)?.serviceName || 'Not associated'}
              </Descriptions.Item>
              <Descriptions.Item label="Lead time">
                {detailCatalog.deliveryDays != null ? `${detailCatalog.deliveryDays} days` : 'Not configured'}
              </Descriptions.Item>
              <Descriptions.Item label="Price">
                {detailCatalog.price != null && detailCatalog.price > 0
                  ? `${detailCatalog.price}${detailCatalog.unit ? ` / ${pricingUnitLabel(detailCatalog.unit)}` : ''}`
                  : 'Free'}
              </Descriptions.Item>
              <Descriptions.Item label="Service Type">
                {detailCatalog.serviceType ?? 'Not configured'}
              </Descriptions.Item>
              <Descriptions.Item label="Sort Order">
                {detailCatalog.sortOrder ?? 0}
              </Descriptions.Item>
              <Descriptions.Item label="Introduction" span={2}>
                {detailCatalog.shortDescription || '—'}
              </Descriptions.Item>
              <Descriptions.Item label="Detailed description" span={2}>
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
