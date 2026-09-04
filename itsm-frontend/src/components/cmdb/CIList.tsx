'use client';

/**
 * 配置项 (CI) 列表组件
 * P1-2：手写 useState + useEffect + isMountedRef + requestIdRef 替换为 React Query。
 *  - useCIsQuery 自动处理竞态/卸载/loading/error（无需 AbortController/cancelledRef）
 *  - useCITypesQuery 缓存 10 分钟（CI 类型列表稳定）
 *  - useDeleteCIMutation 成功后自动 invalidate 列表
 */

import React, { useState, useCallback, useMemo } from 'react';
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
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { Search, Plus, Pencil, Trash2, Download, Eye, RotateCcw, Database } from 'lucide-react';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';
import LoadingEmptyError from '@/components/ui/LoadingEmptyError';

import { CIStatus, CIStatusLabels } from '@/constants/cmdb';
import type { ConfigurationItem, CIType } from '@/types/biz/cmdb';
import {
  useCIsQuery,
  useCITypesQuery,
  useDeleteCIMutation,
} from '@/lib/hooks/useCMDB';
const statusColors: Record<string, string> = {
  [CIStatus.ACTIVE]: 'green',
  [CIStatus.INACTIVE]: 'default',
  [CIStatus.MAINTENANCE]: 'orange',
  [CIStatus.DECOMMISSIONED]: 'red',
};

const CIList: React.FC = () => {
  const router = useRouter();
  const { message } = App.useApp();
  const [filters, setFilters] = useState<{
    search: string;
    ciTypeId?: number;
    status?: string;
  }>({
    search: '',
  });

  const [query, setQuery] = useState({
    offset: 0,
    limit: 10,
  });

  // React Query：CI 列表（自动竞态/缓存/重试；queryKey 含 filter+page 让缓存自然按维度隔离）
  const listQuery = useCIsQuery({
    offset: query.offset,
    limit: query.limit,
    ciTypeId: filters.ciTypeId,
    search: filters.search || undefined,
    status: filters.status,
  });
  const data = useMemo(() => listQuery.data?.items ?? [], [listQuery.data]);
  const total = useMemo(() => listQuery.data?.total ?? 0, [listQuery.data]);

  // React Query：CI 类型列表（缓存 10 分钟，避免重复请求）
  const typesQuery = useCITypesQuery();
  const types: CIType[] = useMemo(() => {
    const res = typesQuery.data as unknown;
    if (!res) return [];
    const list = (res as any)?.data ?? (res as any)?.items ?? res;
    return Array.isArray(list) ? list : [];
  }, [typesQuery.data]);

  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 搜索框变化：300ms 防抖后自动查询（节流）
  const [searchInput, setSearchInput] = useState('');
  const handleSearchInputChange = (value: string) => {
    setSearchInput(value);
    // React Query 由 queryKey 驱动，filters.search 通过 useMemo 派生；
    // 防抖通过延迟设置 filters 实现。
    const t = setTimeout(() => {
      setFilters(prev => ({ ...prev, search: value }));
      if (query.offset !== 0) setQuery(prev => ({ ...prev, offset: 0 }));
    }, 300);
    return () => clearTimeout(t);
  };

  // 下拉筛选变化：即时自动查询
  const handleFilterChange = (patch: Partial<typeof filters>) => {
    setFilters(prev => ({ ...prev, ...patch }));
    setQuery(prev => ({ ...prev, offset: 0 }));
  };

  // 删除走 useDeleteCIMutation，自动 invalidate + onSuccess 提示
  const deleteMutation = useDeleteCIMutation();
  const handleDelete = (id: number) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除此配置项吗？相关关系也将受到影响。',
      onOk: () => deleteMutation.mutateAsync(String(id)).catch(() => {}),
    });
  };

  // handleSearch：React Query 模式下由 queryKey 驱动，"查询"按钮触发当前过滤条件生效（offset 归零）
  const handleSearch = () => {
    if (query.offset !== 0) {
      setQuery(prev => ({ ...prev, offset: 0 }));
    } else {
      listQuery.refetch();
    }
  };

  // 批量删除：走 useDeleteCIMutation，按条 mutateAsync + Promise.allSettled 统计成功/失败
  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要删除的配置项');
      return;
    }
    Modal.confirm({
      title: '批量删除',
      content: `确定要删除选中的 ${selectedRowKeys.length} 个配置项吗？此操作不可恢复。`,
      onOk: async () => {
        const results = await Promise.allSettled(
          selectedRowKeys.map(id => deleteMutation.mutateAsync(String(id))),
        );
        const succeeded = results.filter(r => r.status === 'fulfilled').length;
        const failed = results.length - succeeded;
        if (failed === 0) {
          message.success(`成功删除 ${succeeded} 个配置项`);
        } else if (succeeded === 0) {
          message.error(`批量删除失败：${failed} 个配置项均未删除`);
        } else {
          message.warning(`成功删除 ${succeeded} 个，失败 ${failed} 个，请检查后重试`);
        }
        setSelectedRowKeys([]);
        listQuery.refetch();
      },
    });
  };

  // 导出选中项（CSV）：M1 占位实现，按列硬编码；M2 可切换为 xlsx 流式导出。
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
        return types.find(t => t.id === record.ciTypeId)?.name || record.type || `类型 ${record.ciTypeId}`;
      },
    },
    {
      title: '云厂商',
      width: 120,
      render: (_: unknown, record: ConfigurationItem) => record.cloudProvider || '-',
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
        return record.updatedAt ? dayjs(record.updatedAt).format('YYYY-MM-DD HH:mm') : '-';
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
    <div>
      <Card className="rounded-lg shadow-sm border border-gray-200">
        {/* 搜索工具栏 */}
        <div className="mb-4 flex flex-wrap items-center gap-3">
          <Input
            placeholder="搜索名称/序列号"
            allowClear
            value={filters.search}
            onChange={event => handleSearchInputChange(event.target.value)}
            onClear={() => handleSearchInputChange('')}
            prefix={<Search className="text-gray-400" />}
            style={{ width: 200 }}
          />
          <Select
            aria-label="资产类型"
            placeholder="资产类型"
            style={{ width: 140 }}
            allowClear
            value={filters.ciTypeId}
            onChange={value => handleFilterChange({ ciTypeId: value })}
           options={types.map(t => ({ value: t.id, label: t.name }))} />
          <Select
            aria-label="状态"
            placeholder="状态"
            style={{ width: 110 }}
            allowClear
            value={filters.status}
            onChange={value => handleFilterChange({ status: value })}
           options={Object.entries(CIStatusLabels).map(([value, label]) => ({ value: value, label: label }))} />
          <Space>
            <Button type="primary" onClick={handleSearch}>
              查询
            </Button>
            <Button icon={<RotateCcw />} onClick={() => listQuery.refetch()} loading={listQuery.isFetching}>
              刷新
            </Button>
          </Space>
          <Button
            type="primary"
            icon={<Plus />}
            className="ml-auto"
            onClick={() => router.push('/cmdb/cis/create')}
          >
            录入资产
          </Button>
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
          loading={listQuery.isLoading}
          locale={{
            emptyText:
              listQuery.isLoading && data.length === 0 ? (
                <LoadingEmptyError state="loading" minHeight={200} />
              ) : listQuery.isError ? (
                <LoadingEmptyError
                  state="error"
                  minHeight={200}
                  error={{
                    title: '加载失败',
                    description: '加载配置项列表失败',
                    actionText: '重试',
                    onAction: () => listQuery.refetch(),
                  }}
                />
              ) : (
                <LoadingEmptyError
                  state="empty"
                  minHeight={200}
                  empty={{
                    title: '暂无配置项数据',
                    description: '当前没有配置项数据',
                    icon: <Database size={48} />,
                    actionText: '创建第一个配置项',
                    onAction: () => router.push('/cmdb/cis/create'),
                  }}
                />
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
