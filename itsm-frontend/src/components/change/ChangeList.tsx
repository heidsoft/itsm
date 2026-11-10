'use client';

/**
 * 变更列表组件
 */

import React, { useState, useEffect } from 'react';
import {
  Table,
  Tag,
  Button,
  Card,
  Space,
  Tooltip,
  Input,
  Select,
  Form,
  Modal,
  Empty,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { Search, Plus, Pencil, Trash2, Eye, RefreshCw } from 'lucide-react';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { ChangeApi } from '@/lib/api/';
import { notify } from '@/lib/notify';
import type {
  ChangeType,
  ChangePriority} from '@/constants/change';
import {
  ChangeStatus,
  ChangeImpact,
  ChangeRisk,
  ChangeStatusLabels,
  ChangeTypeLabels,
  ChangePriorityLabels,
} from '@/constants/change';
import type { Change, ChangeQuery } from '@/types/biz/change';

// 颜色映射
const statusColors: Record<string, string> = {
  [ChangeStatus.DRAFT]: 'default',
  [ChangeStatus.PENDING]: 'orange',
  [ChangeStatus.APPROVED]: 'cyan',
  [ChangeStatus.IN_PROGRESS]: 'blue',
  [ChangeStatus.COMPLETED]: 'green',
  [ChangeStatus.REJECTED]: 'red',
  [ChangeStatus.ROLLED_BACK]: 'magenta',
  [ChangeStatus.CANCELLED]: 'default',
};

interface ChangeListProps {
  showHeader?: boolean;
  /** 外部传入的搜索关键词 */
  search?: string;
  /** 外部传入的状态筛选 */
  status?: string;
  /** 外部传入的风险等级筛选 */
  risk?: string;
  /** 受控分页：嵌入模式（showHeader=false）下由父级 Pagination 驱动 */
  page?: number;
  pageSize?: number;
  /**
   * 受控数据：父级传入后组件不再自己发请求，只负责渲染。
   *
   * 嵌入模式下父级（/changes）原本自己拉一份数据用于分页与空态，组件内部再拉
   * 一份用于表格——同一份数据两个源、两次请求，筛选或翻页时两个响应竞态，
   * 会出现「分页显示 37 条但表体是上一页」这类自相矛盾的界面。父级传 items
   * 即声明「数据我负责」，组件退化为纯展示。
   */
  items?: Change[];
  /** 受控总数（配合 items 使用） */
  total?: number;
  /** 受控加载态（配合 items 使用） */
  loading?: boolean;
  /**
   * 数据发生变化需要父级重新拉取（受控模式下删除成功后触发）。
   * 受控时组件内 loadData 是空操作，删除后若不通知父级，界面会停留在旧数据。
   */
  onDataChange?: () => void;
}

const ChangeList: React.FC<ChangeListProps> = ({
  showHeader = true,
  search,
  status,
  risk,
  page,
  pageSize,
  items,
  total,
  loading: loadingProp,
  onDataChange,
}) => {
  const router = useRouter();
  const isControlled = items !== undefined;
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<Change[]>([]);
  // 非受控模式下的总数；受控模式读父级传入的 total  prop。
  const [internalTotal, setInternalTotal] = useState(0);
  const [form] = Form.useForm();

  const [query, setQuery] = useState<ChangeQuery>({
    page: 1,
    pageSize: 10,
  });

  // 当外部筛选条件变化时，同步到表单并重新加载
  useEffect(() => {
    if (search !== undefined || status !== undefined || risk !== undefined) {
      form.setFieldsValue({
        search: search || undefined,
        status: status || undefined,
      });
      setQuery(prev => ({ ...prev, page: 1 }));
    }
  }, [search, status, risk, form]);

  const loadData = async () => {
    // 受控模式下数据由父级提供，组件内不得再发请求（否则就是第二份数据源）。
    if (isControlled) return;
    setLoading(true);
    try {
      const values = await form.validateFields().catch(() => ({}));
      const resp = await ChangeApi.getChanges({
        ...query,
        ...values,
        riskLevel: risk || undefined,
      });
      setData(resp.items as unknown as Change[]);
      setInternalTotal(resp.total);
    } catch (error) {
      // 只在有实际错误时显示失败消息，不是因为表单验证导致的
      if (
        error &&
        typeof error === 'object' &&
        !('name' in error && (error as { name?: string }).name === 'ValidationError')
      ) {
        console.error('Failed to load changes:', error);
        notify.error(error, { context: 'Loading changelist' });
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();

  }, [query]);

  // 受控分页同步：嵌入模式下表格自身 pagination={false}，分页控件在父级。
  // 2026-10-14 修复：此前不接收父级页码，父级 Pagination 翻页只改父 state，
  // 表格停在第 1 页；此处把父级 page/pageSize 同步进内部 query 触发重取。
  useEffect(() => {
    if (showHeader) return; // 独立模式自管分页，不受外部控制
    setQuery(prev => ({
      ...prev,
      page: page ?? prev.page,
      pageSize: pageSize ?? prev.pageSize,
    }));
  }, [showHeader, page, pageSize]);
  const handleSearch = () => {
    setQuery(prev => ({ ...prev, page: 1 }));
  };

  const handleReset = () => {
    form.resetFields();
    setQuery({ page: 1, pageSize: 10 });
  };

  const handleDelete = (id: number, title: string) => {
    Modal.confirm({
      title: 'Confirm Delete',
      content: (
        <div>
          <p>Are you sure you want to delete the change request"{title}"?"</p>
          <p style={{ color: '#ff4d4f' }}>This operation cannot be undone.</p>
        </div>
      ),
      okText: 'Confirm Delete',
      okType: 'danger',
      cancelText: 'Cancel',
      onOk: async () => {
        try {
          await ChangeApi.deleteChange(id);
          notify.success('Change deleted');
          loadData();
          onDataChange?.();
        } catch (error) {
          notify.error(error, { context: 'Delete changes' });
        }
      },
    });
  };

  const columns: ColumnsType<Change> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 70,
    },
    {
      title: 'Title',
      dataIndex: 'title',
      ellipsis: true,
    },
    {
      title: 'Model',
      dataIndex: 'type',
      width: 100,
      render: (type: ChangeType) => ChangeTypeLabels[type] || type,
    },
    {
      title: 'Status',
      dataIndex: 'status',
      width: 100,
      render: (status: ChangeStatus) => (
        <Tag color={statusColors[status]}>{ChangeStatusLabels[status]}</Tag>
      ),
    },
    {
      title: 'Priority',
      dataIndex: 'priority',
      width: 80,
      render: (priority: ChangePriority) => <Tag>{ChangePriorityLabels[priority]}</Tag>,
    },
    {
      title: 'Creator',
      dataIndex:'createdByName',
      width: 100,
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      width: 160,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: 'Actions',
      key: 'action',
      width: 150,
      render: (_: unknown, record: Change) => (
        <Space size="small">
          <Tooltip title="View Details">
            <Button
              type="text"
              icon={<Eye />}
              onClick={() => router.push(`/changes/${record.id}`)}
              aria-label={`View changes ${record.title}`}
            />
          </Tooltip>
          <Tooltip title="Edit">
            <Button
              type="text"
              icon={<Pencil />}
              onClick={() => router.push(`/changes/${record.id}/edit`)}
              aria-label={`Edit Change ${record.title}`}
            />
          </Tooltip>
          <Tooltip title="Delete">
            <Button
              aria-label="Delete"
              type="text"
              danger
              icon={<Trash2 />}
              onClick={() => handleDelete(record.id, record.title)}
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  // 受控模式用父级给的数据与加载态，非受控模式用组件自己拉的那份。
  const rows = isControlled ? (items as Change[]) : data;
  const rowTotal = isControlled ? (total ?? rows.length) : internalTotal;
  const spinning = isControlled ? Boolean(loadingProp) : loading;

  // When embedded in BusinessPageTemplate (showHeader=false), render a bare table
  // without Card wrapper, search form, or internal pagination.
  if (!showHeader) {
    return (
      <Table
        rowKey="id"
        columns={columns}
        dataSource={rows}
        loading={spinning}
        pagination={false}
        scroll={{ x: 1000 }}
        getPopupContainer={node => node.parentElement || document.body}
      />
    );
  }

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Changes</h1>
          <p className="text-gray-500 mt-1">Manage change requests for IT infrastructure and services to minimize change risk</p>
        </div>
        <Button
          type="primary"
          icon={<Plus />}
          onClick={() => router.push('/changes/new')}
          size="large"
        >

          New Change
        </Button>
      </div>

      <Card className="rounded-lg shadow-sm border border-[var(--color-border-primary)]">
        <Form form={form} layout="inline" className="mb-6 flex-wrap gap-y-4">
          <Form.Item name="search" className="mb-0">
            <Input
              placeholder="Search title"
              allowClear
              prefix={<Search className="text-gray-400" />}
              className="w-64"
            />
          </Form.Item>
          <Form.Item name="status" className="mb-0">
            <Select placeholder="Status" className="w-32" allowClear options={Object.entries(ChangeStatusLabels).map(([value, label]) => ({
                value,
                label,
              }))} />
          </Form.Item>
          <Form.Item name="type" className="mb-0">
            <Select placeholder="Type" className="w-32" allowClear options={Object.entries(ChangeTypeLabels).map(([value, label]) => ({
                value,
                label,
              }))} />
          </Form.Item>
          <Form.Item className="mb-0">
            <Space>
              <Button type="primary" onClick={handleSearch}>

                Query
              </Button>
              <Button aria-label="Refresh" icon={<RefreshCw />} onClick={loadData} />
            </Space>
          </Form.Item>
        </Form>

        {rows.length === 0 && !spinning ? (
          <Empty description="No change records" image={Empty.PRESENTED_IMAGE_SIMPLE}>
            <Button type="primary" onClick={() => router.push('/changes/new')}>

              Create Change
            </Button>
          </Empty>
        ) : (
          <Table
            rowKey="id"
            columns={columns}
            dataSource={rows}
            loading={spinning}
            pagination={{
              current: query.page,
              pageSize: query.pageSize,
              total: rowTotal,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: total => `Total ${total} records`,
              pageSizeOptions: ['10', '20', '50', '100'],
              onChange: (page, pageSize) => setQuery(prev => ({ ...prev, page, pageSize })),
            }}
            scroll={{ x: 1000 }}
            getPopupContainer={node => node.parentElement || document.body}
          />
        )}
      </Card>
    </div>
  );
};

export default ChangeList;
