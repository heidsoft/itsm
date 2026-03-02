/**
 * 标准列表页面模板
 */

'use client';

import { useState } from 'react';
import { Table, Button, Space, message, Card } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';

interface DataItem {
  id: number;
  [key: string]: unknown;
}

interface PageState {
  page: number;
  pageSize: number;
  total: number;
}

export function createListPage<T extends DataItem>(config: {
  name: string;
  fetchApi: (params: PageState) => Promise<{ list: T[]; total: number }>;
  createApi?: (data: Partial<T>) => Promise<T>;
  deleteApi?: (id: number) => Promise<void>;
  columns: ColumnsType<T>;
  createPath?: string;
}) {
  return function ListPage() {
    const [loading, setLoading] = useState(false);
    const [data, setData] = useState<T[]>([]);
    const [pagination, setPagination] = useState<PageState>({ page: 1, pageSize: 10, total: 0 });

    const loadData = async () => {
      setLoading(true);
      try {
        const result = await config.fetchApi(pagination);
        setData(result.list);
        setPagination(p => ({ ...p, total: result.total }));
      } finally {
        setLoading(false);
      }
    };

    const handleDelete = async (id: number) => {
      if (!config.deleteApi) return;
      await config.deleteApi(id);
      message.success('删除成功');
      loadData();
    };

    const columns = [
      ...config.columns,
      {
        title: '操作',
        render: (_: unknown, record: T) => (
          <Space>
            <Button size='small' danger onClick={() => handleDelete(record.id)}>
              删除
            </Button>
          </Space>
        ),
      },
    ];

    return (
      <Card
        title={`${config.name}管理`}
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={loadData}>
              刷新
            </Button>
            <Button type='primary' icon={<PlusOutlined />}>
              新建{config.name}
            </Button>
          </Space>
        }
      >
        <Table<T>
          loading={loading}
          dataSource={data}
          rowKey='id'
          columns={columns}
          pagination={{
            current: pagination.page,
            pageSize: pagination.pageSize,
            total: pagination.total,
            onChange: (page, pageSize) => setPagination(p => ({ ...p, page, pageSize })),
          }}
        />
      </Card>
    );
  };
}
