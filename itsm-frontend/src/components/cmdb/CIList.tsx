'use client';

/**
 * 配置项 (CI) 列表组件
 * - useCIsQuery 负责竞态/卸载/loading/error，无需手写 AbortController 和 cancelledRef
 * - useCITypesQuery 缓存 10 分钟（CI 类型列表稳定）
 * - useDeleteCIMutation 成功后自动 invalidate 列表
 */

import React, { useState, useCallback, useMemo, useRef, useEffect } from 'react';
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
import { useOperationPermissions } from '@/lib/hooks/use-permissions';
import { buildCsv, downloadCsv } from '@/lib/utils/csv-export';
const statusColors: Record<string, string> = {
  [CIStatus.ACTIVE]: 'green',
  [CIStatus.INACTIVE]: 'default',
  [CIStatus.MAINTENANCE]: 'orange',
  [CIStatus.DECOMMISSIONED]: 'red',
};

// ci_types.name 是租户内唯一代码（server/database/...），中文名在 description。
// 展示层统一优先中文描述，缺失再回退代码/原始类型，避免列表与导出出现英文代码。
const getCITypeLabel = (types: CIType[], ciTypeId: number | undefined, fallback?: string): string => {
  const type = types.find(item => item.id === ciTypeId);
  return type?.description?.trim() || type?.name || fallback || '';
};

const CIList: React.FC = () => {
  const router = useRouter();
  const { message } = App.useApp();
  // 写操作按 cmdb:write / cmdb:delete 门控：只读角色不该看到点了就 403 的按钮
  // （审计 3.2-⑥ 按钮级门控）。
  const { cmdb: cmdbPerms } = useOperationPermissions();
  const canCreateCI = cmdbPerms.canCreate();
  const canUpdateCI = cmdbPerms.canUpdate();
  const canDeleteCI = cmdbPerms.canDelete();
  const [filters, setFilters] = useState<{
    search: string;
    ciTypeId?: number;
    status?: string;
  }>({
    search: '',
  });

  const [query, setQuery] = useState({
    page: 1,
    pageSize: 10,
  });

  // React Query：CI 列表（自动竞态/缓存/重试；queryKey 含 filter+page 让缓存自然按维度隔离）
  const listQuery = useCIsQuery({
    page: query.page,
    pageSize: query.pageSize,
    ciTypeId: filters.ciTypeId,
    search: filters.search || undefined,
    status: filters.status,
  });
  const data = useMemo(() => listQuery.data?.items ?? [], [listQuery.data]);
  const total = useMemo(() => listQuery.data?.total ?? 0, [listQuery.data]);

  const typesQuery = useCITypesQuery();
  const types: CIType[] = useMemo(() => typesQuery.data ?? [], [typesQuery.data]);

  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 搜索框变化：300ms 防抖后自动查询
  // - searchInput 作为受控值提供即时输入反馈
  // - filters.search 在防抖延迟后才更新，驱动 React Query 重新请求
  // - 用 useRef 持有 timeout ID，新输入前清理旧定时器，避免重复请求
  const [searchInput, setSearchInput] = useState('');
  const searchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    // 组件卸载时清理未触发的防抖定时器，避免对已卸载组件 setState
    return () => {
      if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
    };
  }, []);

  const handleSearchInputChange = (value: string) => {
    setSearchInput(value);
    if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
    searchTimerRef.current = setTimeout(() => {
      setFilters(prev => ({ ...prev, search: value }));
      if (query.page !== 1) setQuery(prev => ({ ...prev, page: 1 }));
      searchTimerRef.current = null;
    }, 300);
  };

  // 下拉筛选变化：即时自动查询
  const handleFilterChange = (patch: Partial<typeof filters>) => {
    setFilters(prev => ({ ...prev, ...patch }));
    setQuery(prev => ({ ...prev, page: 1 }));
  };

  // 删除走 useDeleteCIMutation，自动 invalidate + onSuccess 提示
  const deleteMutation = useDeleteCIMutation();
  const handleDelete = (id: number) => {
    Modal.confirm({
      title: 'Confirm Delete',
      content: 'Are you sure you want to delete this configuration item? Related relationships will also be affected.',
      onOk: () => deleteMutation.mutateAsync(String(id)).catch(() => {}),
    });
  };

  // handleSearch：React Query 模式下由 queryKey 驱动，查询按钮触发当前过滤条件生效并回到第 1 页
  const handleSearch = () => {
    if (query.page !== 1) {
      setQuery(prev => ({ ...prev, page: 1 }));
    } else {
      listQuery.refetch();
    }
  };

  // 批量删除：走 useDeleteCIMutation，按条 mutateAsync + Promise.allSettled 统计成功/失败
  const handleBatchDelete = () => {
    if (selectedRowKeys.length === 0) {
      message.warning('Please select the configuration item to delete first');
      return;
    }
    Modal.confirm({
      title: 'Batch Delete',
      content: `Are you sure you want to delete the selected ${selectedRowKeys.length} Configuration item? This operation is irreversible.`,
      onOk: async () => {
        const results = await Promise.allSettled(
          selectedRowKeys.map(id => deleteMutation.mutateAsync(String(id))),
        );
        const succeeded = results.filter(r => r.status === 'fulfilled').length;
        const failed = results.length - succeeded;
        if (failed === 0) {
          message.success(`Deleted successfully ${succeeded} configuration items`);
        } else if (succeeded === 0) {
          message.error(`Batch deletion failed:${failed} None of the configuration items have been deleted`);
        } else {
          message.warning(`Deleted successfully ${succeeded} , failed ${failed} please check and try again`);
        }
        setSelectedRowKeys([]);
        listQuery.refetch();
      },
    });
  };

  // 导出选中项（CSV）：统一走 buildCsv，避免手写拼接绕过公式注入防护。
  // 旧实现只把引号翻倍，形如 =HYPERLINK(...) 的 CI 名称在 Excel/WPS 打开即执行。
  const handleExport = () => {
    if (selectedRowKeys.length === 0) {
      message.warning('Please select the configuration items to be exported first');
      return;
    }
    const selectedData = data.filter(item => selectedRowKeys.includes(item.id));
    const csv = buildCsv<ConfigurationItem>(
      [
        { title: 'ID', render: item => item.id },
        { title: "Name", render: item => item.name },
        { title: "Type", render: item => getCITypeLabel(types, item.ciTypeId, item.type) },
        { title: "Cloud Provider", render: item => item.cloudProvider || '' },
        { title: "Status", render: item => item.status },
        { title: "Model", render: item => item.model || '' },
        { title: "Vendor", render: item => item.vendor || '' },
        { title: "Last Update", render: item => item.updatedAt || '' },
      ],
      selectedData
    );
    downloadCsv(`Configuration item export_${dayjs().format('YYYYMMDD_HHmmss')}`, csv);
    message.success(`Exported ${selectedData.length} records`);
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
      title: 'Name',
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
      title: 'Type',
      width: 120,
      render: (_: unknown, record: ConfigurationItem) => {
        return getCITypeLabel(types, record.ciTypeId, record.type) || `Type ${record.ciTypeId}`;
      },
    },
    {
      title: 'Cloud Provider',
      width: 120,
      render: (_: unknown, record: ConfigurationItem) => record.cloudProvider || '-',
    },
    {
      title: 'Status',
      dataIndex: 'status',
      width: 100,
      render: (status: CIStatus) => (
        <Tag color={statusColors[status]}>{CIStatusLabels[status] || status}</Tag>
      ),
    },
    {
      title: 'Model/Manufacturer',
      key:'modelVendor',
      width: 180,
      render: (_: unknown, record: ConfigurationItem) => (
        <span>
          {record.model || '-'} / {record.vendor || '-'}
        </span>
      ),
    },
    {
      title: 'Last Update',
      width: 160,
      render: (_: unknown, record: ConfigurationItem) => {
        return record.updatedAt ? dayjs(record.updatedAt).format('YYYY-MM-DD HH:mm') : '-';
      },
    },
    {
      title: 'Actions',
      key: 'action',
      width: 120,
      render: (_: unknown, record: ConfigurationItem) => (
        <Space size="small">
          {canUpdateCI && (
            <Tooltip title="Edit">
              <Button
                type="text"
                icon={<Pencil />}
                aria-label="Edit"
                onClick={() => router.push(`/cmdb/cis/${record.id}/edit`)}
              />
            </Tooltip>
          )}
          {canDeleteCI && (
            <Tooltip title="Delete">
              <Button
                type="text"
                danger
                icon={<Trash2 />}
                aria-label="Delete"
                onClick={() => handleDelete(record.id)}
              />
            </Tooltip>
          )}
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
            placeholder="Search name/number"
            allowClear
            value={searchInput}
            onChange={event => handleSearchInputChange(event.target.value)}
            onClear={() => handleSearchInputChange('')}
            prefix={<Search className="text-gray-400" />}
            style={{ width: 200 }}
          />
          <Select
            aria-label="CI Types"
            placeholder="CI Types"
            style={{ width: 140 }}
            allowClear
            value={filters.ciTypeId}
            onChange={value => handleFilterChange({ ciTypeId: value })}
           options={types.map(t => ({ value: t.id, label: t.description?.trim() || t.name }))} />
          <Select
            aria-label="Status"
            placeholder="Status"
            style={{ width: 110 }}
            allowClear
            value={filters.status}
            onChange={value => handleFilterChange({ status: value })}
           options={Object.entries(CIStatusLabels).map(([value, label]) => ({ value: value, label: label }))} />
          <Space>
            <Button type="primary" onClick={handleSearch}>

              Query
            </Button>
            <Button icon={<RotateCcw />} onClick={() => listQuery.refetch()} loading={listQuery.isFetching}>

              Refresh
            </Button>
          </Space>
          {canCreateCI && (
            <Button
              type="primary"
              icon={<Plus />}
              className="ml-auto"
              onClick={() => router.push('/cmdb/cis/create')}
            >

              New Configuration Item
            </Button>
          )}
        </div>

        {/* 批量操作工具栏 */}
        {selectedRowKeys.length > 0 && (
          <div className="mb-4 p-3 bg-blue-50 rounded-lg flex items-center gap-3">
            <span className="text-sm text-blue-700">

              Selected <strong>{selectedRowKeys.length}</strong>  items
            </span>
            <Space>
              <Button size="small" icon={<Download />} onClick={handleExport}>

                Export
              </Button>
              {canDeleteCI && (
                <Button size="small" danger icon={<Trash2 />} onClick={handleBatchDelete}>

                  Batch Delete
                </Button>
              )}
            </Space>
            <Button size="small" type="link" onClick={() => setSelectedRowKeys([])}>

              Cancel Selection
            </Button>
          </div>
        )}

        <Table
          rowKey="id"
          rowSelection={rowSelection}
          columns={columns}
          dataSource={data}
          loading={listQuery.isPending}
          locale={{
            emptyText:
              listQuery.isPending && data.length === 0 ? (
                <LoadingEmptyError state="loading" minHeight={200} />
              ) : listQuery.isError ? (
                <LoadingEmptyError
                  state="error"
                  minHeight={200}
                  error={{
                    title: 'LoadingFailed',
                    description: 'Failed to load configuration item list',
                    actionText: 'Retry',
                    onAction: () => listQuery.refetch(),
                  }}
                />
              ) : (
                <LoadingEmptyError
                  state="empty"
                  minHeight={200}
                  empty={{
                    title: 'No configuration items yet',
                    description: 'There is no registered configuration item yet. You can create a new one first or import it from the cloud resource list.',
                    icon: <Database size={48} />,
                    actionText: 'Create configuration item',
                    onAction: () => router.push('/cmdb/cis/create'),
                  }}
                />
              ),
          }}
          pagination={{
            current: query.page,
            pageSize: query.pageSize,
            total: total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: total => `Total ${total} records`,
            pageSizeOptions: ['10', '20', '50', '100'],
            onChange: (page, pageSize) => setQuery({ ...query, page, pageSize }),
          }}
          scroll={{ x: 1200 }}
          getPopupContainer={node => node.parentElement || document.body}
        />
      </Card>
    </div>
  );
};

export default CIList;
