'use client';

/**
 * 配置项 (CI) 列表组件
 */

import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  Table,
  Tag,
  Button,
  Card,
  Space,
  Tooltip,
  Input,
  Select,
  App,
  Modal,
  Empty,
  Dropdown,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { Search, Plus, Pencil, Trash2, Download, Eye, RotateCcw } from 'lucide-react';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { CMDBApi } from '@/lib/api/';
import { CIStatus, CIStatusLabels } from '@/constants/cmdb';
import type { ConfigurationItem, CIType } from '@/types/biz/cmdb';

const { Option } = Select;

const statusColors: Record<string, string> = {
  [CIStatus.ACTIVE]: 'green',
  [CIStatus.INACTIVE]: 'default',
  [CIStatus.MAINTENANCE]: 'orange',
  [CIStatus.DECOMMISSIONED]: 'red',
};

const getErrorMessage = (error: unknown): string => {
  if (error instanceof Error) return error.message;
  if (typeof error === 'string') return error;
  return '';
};

const CIList: React.FC = () => {
  const router = useRouter();
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<ConfigurationItem[]>([]);
  const [total, setTotal] = useState(0);
  const [types, setTypes] = useState<CIType[]>([]);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const requestIdRef = useRef(0);
  const isMountedRef = useRef(true);
  const [filters, setFilters] = useState<{
    search: string;
    ciTypeId?: number;
    status?: string;
  }>({
    search: '',
  });
  const filtersRef = useRef(filters);

  const updateFilters = (next: typeof filters) => {
    filtersRef.current = next;
    setFilters(next);
  };

  const [query, setQuery] = useState({
    offset: 0,
    limit: 10,
  });

  const loadTypes = useCallback(async () => {
    try {
      const res = await CMDBApi.getCITypes();
      if (!isMountedRef.current) return;
      // 支持多种响应格式
      const list = (res as any)?.data ?? (res as any)?.items ?? res;
      setTypes(Array.isArray(list) ? list : []);
    } catch (e) {
      if (!isMountedRef.current) return;
      message.error('加载资产类型失败');
    }
  }, []);

  const loadData = useCallback(async () => {
    const requestId = ++requestIdRef.current;
    setLoading(true);
    try {
      const currentFilters = filtersRef.current;
      const resp = await CMDBApi.getCIs({
        offset: query.offset,
        limit: query.limit,
        ciTypeId: currentFilters.ciTypeId,
        search: currentFilters.search || undefined,
        status: currentFilters.status,
      });
      if (!isMountedRef.current || requestId !== requestIdRef.current) return;
	  setData(resp.items ?? []);
      setTotal(resp.total ?? 0);
    } catch (error) {
      if (!isMountedRef.current || requestId !== requestIdRef.current) return;
      const errorMessage = getErrorMessage(error);
      message.error(errorMessage ? `加载配置项列表失败：${errorMessage}` : '加载配置项列表失败');
    } finally {
      if (isMountedRef.current && requestId === requestIdRef.current) {
        setLoading(false);
      }
    }
  }, [query]);

  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  useEffect(() => {
    loadTypes();
  }, [loadTypes]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleSearch = () => {
    if (query.offset === 0) {
      loadData();
      return;
    }
    setQuery(prev => ({ ...prev, offset: 0 }));
  };

  const handleDelete = (id: number) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除此配置项吗？相关关系也将受到影响。',
      onOk: async () => {
        try {
          await CMDBApi.deleteCI(String(id));
          message.success('删除成功');
          loadData();
        } catch (e) {
          const errorMessage = getErrorMessage(e);
          message.error(errorMessage ? `删除失败：${errorMessage}` : '删除失败');
        }
      },
    });
  };

  // 批量删除
  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要删除的配置项');
      return;
    }
    Modal.confirm({
      title: '批量删除',
      content: `确定要删除选中的 ${selectedRowKeys.length} 个配置项吗？此操作不可恢复。`,
      onOk: async () => {
        try {
          const deletePromises = selectedRowKeys.map(id => CMDBApi.deleteCI(String(id)));
          await Promise.all(deletePromises);
          message.success(`成功删除 ${selectedRowKeys.length} 个配置项`);
          setSelectedRowKeys([]);
          loadData();
        } catch (e) {
          message.error('批量删除部分失败，请检查');
          loadData();
        }
      },
    });
  };

  // 导出选中项
  const handleExport = () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要导出的配置项');
      return;
    }
    const selectedData = data.filter(item => selectedRowKeys.includes(item.id));
    const csvContent = [
      ['ID', '名称', '类型', '云厂商', '状态', '型号', '厂商', '最后更新'].join(','),
      ...selectedData.map(item => [
        item.id,
        item.name,
        types.find(t => t.id === item.ciTypeId)?.name || (item as any).ciType || '',
        item.cloudProvider || (item as any).cloudProvider || '',
        item.status,
        item.model || '',
        item.vendor || '',
        item.updatedAt || '',
      ].map(v => `"${String(v).replace(/"/g, '""')}"`).join(',')),
    ].join('\n');

    const blob = new Blob(['\ufeff' + csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `配置项导出_${dayjs().format('YYYYMMDD_HHmmss')}.csv`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    message.success(`已导出 ${selectedData.length} 条记录`);
  };

  // 表格行选择配置
  const rowSelection = {
    selectedRowKeys,
    onChange: (keys: React.Key[]) => setSelectedRowKeys(keys),
  };

  const columns: ColumnsType<ConfigurationItem> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 70,
    },
    {
      title: '资产名称',
      dataIndex: 'name',
      ellipsis: true,
      render: (text: string, record: ConfigurationItem) => (
        <Button
          type="link"
          onClick={() => router.push(`/cmdb/cis/${record.id}`)}
          style={{ padding: 0, height: 'auto' }}
        >
          {text}
        </Button>
      ),
    },
    {
      title: '类型',
      width: 120,
      render: (_: unknown, record: ConfigurationItem) => {
        const typeId = record.ciTypeId ?? record.ciTypeId;
        return types.find(t => t.id === typeId)?.name || record.type || `类型 ${typeId}`;
      },
    },
    {
      title: '云厂商',
      width: 120,
      render: (_: unknown, record: ConfigurationItem) => record.cloudProvider ?? record.cloudProvider ?? '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (status: CIStatus) => (
        <Tag color={statusColors[status]}>{CIStatusLabels[status] || status}</Tag>
      ),
    },
    {
      title: '型号/厂商',
      key:'modelVendor',
      width: 180,
      render: (_: unknown, record: ConfigurationItem) => (
        <span>
          {record.model || '-'} / {record.vendor || '-'}
        </span>
      ),
    },
    {
      title: '最后更新',
      width: 160,
      render: (_: unknown, record: ConfigurationItem) => {
        const date = record.updatedAt ?? record.updatedAt;
        return date ? dayjs(date).format('YYYY-MM-DD HH:mm') : '-';
      },
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: unknown, record: ConfigurationItem) => (
        <Space size="small">
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<Pencil />}
              aria-label="编辑"
              onClick={() => router.push(`/cmdb/cis/${record.id}/edit`)}
            />
          </Tooltip>
          <Tooltip title="删除">
            <Button
              type="text"
              danger
              icon={<Trash2 />}
              aria-label="删除"
              onClick={() => handleDelete(record.id)}
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <div className="p-6">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">配置管理</h1>
          <p className="text-gray-500 mt-1">管理和维护系统中的所有配置项(CI)及其关系</p>
        </div>
        <Space wrap>
          <Button
            type="primary"
            icon={<Plus />}
            onClick={() => router.push('/cmdb/cis/create')}
          >
            录入资产
          </Button>
        </Space>
      </div>

      <Card className="rounded-lg shadow-sm border border-gray-200">
        {/* 搜索工具栏 */}
        <div className="mb-4 flex flex-wrap items-center gap-3">
          <Input
            placeholder="搜索名称/序列号"
            allowClear
            value={filters.search}
            onChange={event =>
              updateFilters({
                ...filtersRef.current,
                search: event.target.value,
              })
            }
            onClear={() =>
              updateFilters({
                ...filtersRef.current,
                search: '',
              })
            }
            prefix={<Search className="text-gray-400" />}
            style={{ width: 200 }}
          />
          <Select
            aria-label="资产类型"
            placeholder="资产类型"
            style={{ width: 140 }}
            allowClear
            value={filters.ciTypeId}
            onChange={value => updateFilters({ ...filtersRef.current, ciTypeId: value })}
          >
            {types.map(t => (
              <Option key={t.id} value={t.id}>
                {t.name}
              </Option>
            ))}
          </Select>
          <Select
            aria-label="状态"
            placeholder="状态"
            style={{ width: 110 }}
            allowClear
            value={filters.status}
            onChange={value => updateFilters({ ...filtersRef.current, status: value })}
          >
            {Object.entries(CIStatusLabels).map(([value, label]) => (
              <Option key={value} value={value}>
                {label}
              </Option>
            ))}
          </Select>
          <Space>
            <Button type="primary" onClick={handleSearch}>
              查询
            </Button>
            <Button icon={<RotateCcw />} onClick={loadData} loading={loading}>
              刷新
            </Button>
          </Space>
        </div>

        {/* 批量操作工具栏 */}
        {selectedRowKeys.length > 0 && (
          <div className="mb-4 p-3 bg-blue-50 rounded-lg flex items-center gap-3">
            <span className="text-sm text-blue-700">
              已选择 <strong>{selectedRowKeys.length}</strong> 项
            </span>
            <Space>
              <Button size="small" icon={<Download />} onClick={handleExport}>
                导出
              </Button>
              <Button size="small" danger icon={<Trash2 />} onClick={handleBatchDelete}>
                批量删除
              </Button>
            </Space>
            <Button size="small" type="link" onClick={() => setSelectedRowKeys([])}>
              取消选择
            </Button>
          </div>
        )}

        <Table
          rowKey="id"
          rowSelection={rowSelection}
          columns={columns}
          dataSource={data}
          loading={loading}
          locale={{
            emptyText: (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无配置项数据">
                <Button type="primary" onClick={() => router.push('/cmdb/cis/create')}>
                  创建第一个配置项
                </Button>
              </Empty>
            ),
          }}
          pagination={{
            current: Math.floor(query.offset / query.limit) + 1,
            pageSize: query.limit,
            total: total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: total => `共 ${total} 条记录`,
            pageSizeOptions: ['10', '20', '50', '100'],
            onChange: (page, pageSize) => setQuery({ offset: (page - 1) * pageSize, limit: pageSize }),
          }}
          scroll={{ x: 1200 }}
          getPopupContainer={node => node.parentElement || document.body}
        />
      </Card>
    </div>
  );
};

export default CIList;
