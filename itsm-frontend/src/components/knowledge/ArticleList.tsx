'use client';

/**
 * 知识库文章列表组件
 */

import React, { useState, useEffect, useCallback } from 'react';
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
  Breadcrumb,
  Empty,
  App,
  Alert,
  Skeleton,
} from 'antd';
import { Search, Plus, Pencil, Trash2, Eye, RotateCcw } from 'lucide-react';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';
import { usePermissions } from '@/lib/hooks/use-permissions';
import {
  KnowledgeStatus,
  KnowledgeStatusLabels,
  KnowledgeStatusColors,
} from '@/constants/knowledge';
import type { KnowledgeArticle, ArticleQuery } from '@/types/knowledge-base';
interface ArticleListProps {
  showHeader?: boolean;
}

const ArticleList: React.FC<ArticleListProps> = ({ showHeader = true }) => {
  const router = useRouter();
  const { message } = App.useApp();
  const { hasPermission } = usePermissions();
  // 后端把新增/更新/发布统一挂 knowledge:write，删除挂 knowledge:delete；
  // 只读角色（end_user 只有 knowledge:read）此前仍能看到这些按钮，点击只会 403。
  const canWriteArticle = hasPermission('knowledge', 'write');
  const canDeleteArticle = hasPermission('knowledge', 'delete');
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<KnowledgeArticle[]>([]);
  const [total, setTotal] = useState(0);
  const [categories, setCategories] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [form] = Form.useForm();

  const [query, setQuery] = useState<ArticleQuery>({
    page: 1,
    pageSize: 10,
  });

  const loadCategories = async () => {
    try {
      const res = await KnowledgeBaseApi.getCategories();
      // 后端返回该租户已使用的分类名称集合；纯数字项是把分类 ID 当名称写入的脏数据。
      setCategories((res || []).filter(name => !/^\d+$/.test(String(name))));
    } catch (e) {
      // console.error(e);
    }
  };

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      let values = {};
      try {
        values = await form.validateFields();
      } catch (e) {
        // Form validation may fail on initial load, use empty values
      }
      const resp = await KnowledgeBaseApi.getArticles({
        ...query,
        ...values,
        categoryId: (values as unknown as { category?: number }).category ? String((values as unknown as { category?: number }).category) : undefined,
      });
      // HTTP client already extracts data, so resp is ListKnowledgeArticlesResponse
      const articles = resp?.items ?? [];
      const total = resp?.total || 0;
      setData(articles as unknown as KnowledgeArticle[]);
      setTotal(total);
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Unknown error');
      message.error('Failed to load article list');
    } finally {
      setLoading(false);
    }
  }, [form, query, message]);

  useEffect(() => {
    loadCategories();
    loadData();

  }, [loadData]);

  const handleSearch = () => {
    if (query.page === 1) loadData();
    else setQuery(prev => ({ ...prev, page: 1 }));
  };

  const handleReset = () => {
    form.resetFields();
    if (query.page === 1) loadData();
    else setQuery(prev => ({ ...prev, page: 1 }));
  };

  const handleDelete = (id: string | number) => {
    Modal.confirm({
      title: 'Are you sure you want to delete this article?',
      content: 'Cannot be restored after deletion.',
      onOk: async () => {
        try {
          await KnowledgeBaseApi.deleteArticle(id.toString());
          message.success('Deleted');
          loadData();
        } catch (e) {
          message.error('Delete failed');
        }
      },
    });
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 70,
    },
    {
      title: 'Title',
      dataIndex: 'title',
      ellipsis: true,
      render: (text: string, record: KnowledgeArticle) => (
        <a onClick={() => router.push(`/knowledge/articles/${record.id}`)}>{text}</a>
      ),
    },
    {
      title: 'Category',
      dataIndex: 'category',
      width: 120,
    },
    {
      title: 'Tags',
      dataIndex: 'tags',
      render: (tags: string[]) => (
        <>{Array.isArray(tags) && tags.map(tag => <Tag key={tag}>{tag}</Tag>)}</>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      width: 100,
      render: (status: string) => {
        const s = status === 'published' ? KnowledgeStatus.PUBLISHED : KnowledgeStatus.DRAFT;
        return <Tag color={KnowledgeStatusColors[s]}>{KnowledgeStatusLabels[s]}</Tag>;
      },
    },
    {
      title: 'Updated At',
      dataIndex: 'updatedAt',
      width: 160,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: 'Actions',
      key: 'action',
      width: 120,
      render: (_: unknown, record: KnowledgeArticle) => (
        <Space size="small">
          {canWriteArticle && (
            <Tooltip title="Edit">
              <Button
                size="small"
                aria-label={`Edit Article ${record.title}`}
                icon={<Pencil />}
                onClick={() => router.push(`/knowledge/articles/${record.id}/edit`)}
                style={{ backgroundColor: '#3b82f6', color: '#fff', border: 'none' }}
              />
            </Tooltip>
          )}
          {canDeleteArticle && (
            <Tooltip title="Delete">
              <Button
                size="small"
                aria-label={`Delete article ${record.title}`}
                icon={<Trash2 />}
                onClick={() => handleDelete(record.id)}
                style={{ backgroundColor: '#ef4444', color: '#fff', border: 'none' }}
              />
            </Tooltip>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div className="p-6">
      {showHeader && (
        <div className="flex justify-between items-center mb-6">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Knowledge Base</h1>
            <p className="text-gray-500 mt-1">Create, maintain and share solutions and best practices</p>
          </div>
          {canWriteArticle && (
            <Button
              type="primary"
              icon={<Plus />}
              onClick={() => router.push('/knowledge/articles/new')}
              size="large"
            >

              New Article
            </Button>
          )}
        </div>
      )}

      <Card className="rounded-lg shadow-sm border border-gray-200">
        <Form form={form} layout="inline" className="mb-6 flex-wrap gap-y-4" onFinish={handleSearch}>
          <Form.Item name="search" className="mb-0">
            <Input
              placeholder="Search title/content"
              allowClear
              prefix={<Search className="text-gray-400" />}
              className="w-64"
              aria-label="Search knowledge base articles"
            />
          </Form.Item>
          <Form.Item name="category" className="mb-0">
            <Select
              placeholder="Category"
              className="w-36"
              allowClear

              style={{ backgroundColor: '#fafafa', borderRadius: '6px' }}
             options={Array.isArray(categories) ? categories.map(c => ({ value: c, label: c })) : []} />
          </Form.Item>
          <Form.Item name="status" className="mb-0">
            <Select
              placeholder="Status"
              className="w-28"
              allowClear

              style={{ backgroundColor: '#fafafa', borderRadius: '6px' }}
             options={[{ value: "published", label: "Published" }, { value: "draft", label: "Draft" }]} />
          </Form.Item>
          <Form.Item className="mb-0">
            <Space>
              <Button type="primary" htmlType="submit" loading={loading}>

                Query
              </Button>
              <Button onClick={handleReset} disabled={loading}>Reset</Button>
            </Space>
          </Form.Item>
        </Form>

        {error ? (
          <Alert
            type="error"
            showIcon
            title="Knowledge base article failed to load"
            description={error}
            action={<Button size="small" onClick={loadData}>Retry</Button>}
          />
        ) : loading && data.length === 0 ? (
          <Skeleton active paragraph={{ rows: 8 }} />
        ) : data.length === 0 ? (
          <Empty description="No knowledge articles yet" image={Empty.PRESENTED_IMAGE_SIMPLE}>
            <Button type="primary" onClick={() => router.push('/knowledge/articles/new')}>

              Create first article
            </Button>
          </Empty>
        ) : (
          <Table
            rowKey="id"
            columns={columns as any}
            dataSource={data}
            loading={loading}
            pagination={{
              current: query.page,
              pageSize: query.pageSize,
              total: total,
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

export default ArticleList;
