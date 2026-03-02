'use client';

/**
 * 知识库文章列表组件
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
  Breadcrumb,
  Empty,
  App,
} from 'antd';
import {
  SearchOutlined,
  EyeOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';
import {
  KnowledgeStatus,
  KnowledgeStatusLabels,
  KnowledgeStatusColors,
} from '@/constants/knowledge';
import type { KnowledgeArticle, ArticleQuery } from '@/types/knowledge-base';

const { Option } = Select;

const ArticleList: React.FC = () => {
  const router = useRouter();
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<KnowledgeArticle[]>([]);
  const [total, setTotal] = useState(0);
  const [categories, setCategories] = useState<string[]>([]);
  const [form] = Form.useForm();

  const [query, setQuery] = useState<ArticleQuery>({
    page: 1,
    pageSize: 10,
  });

  const loadCategories = async () => {
    try {
      const res = await KnowledgeBaseApi.getCategories();
      // Map KnowledgeCategory objects to strings for backward compatibility
      const categoryNames = (res || []).map((cat: any) => cat.name || cat.id || String(cat));
      setCategories(categoryNames);
    } catch (e) {
      // console.error(e);
    }
  };

  const loadData = async () => {
    setLoading(true);
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
        categoryId: (values as any).category || undefined,
      });
      // HTTP client already extracts data, so resp is ListKnowledgeArticlesResponse
      const articles = resp?.articles || [];
      const total = resp?.total || 0;
      setData(articles as unknown as KnowledgeArticle[]);
      setTotal(total);
    } catch (error) {
      message.error('加载文章列表失败: ' + (error instanceof Error ? error.message : '未知错误'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadCategories();
    loadData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query]);

  const handleSearch = () => {
    setQuery(prev => ({ ...prev, page: 1 }));
  };

  const handleDelete = (id: string | number) => {
    Modal.confirm({
      title: '确定要删除此文章吗？',
      content: '删除后无法恢复。',
      onOk: async () => {
        try {
          await KnowledgeBaseApi.deleteArticle(id.toString());
          message.success('删除成功');
          loadData();
        } catch (e) {
          message.error('删除失败');
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
      title: '标题',
      dataIndex: 'title',
      ellipsis: true,
      render: (text: string, record: KnowledgeArticle) => (
        <a onClick={() => router.push(`/knowledge/articles/${record.id}`)}>{text}</a>
      ),
    },
    {
      title: '分类',
      dataIndex: 'category',
      width: 120,
    },
    {
      title: '标签',
      dataIndex: 'tags',
      render: (tags: string[]) => (
        <>{Array.isArray(tags) && tags.map(tag => <Tag key={tag}>{tag}</Tag>)}</>
      ),
    },
    {
      title: '状态',
      dataIndex: 'is_published',
      width: 100,
      render: (is_published: boolean) => {
        const status = is_published ? KnowledgeStatus.PUBLISHED : KnowledgeStatus.DRAFT;
        return <Tag color={KnowledgeStatusColors[status]}>{KnowledgeStatusLabels[status]}</Tag>;
      },
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      width: 160,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: any, record: KnowledgeArticle) => (
        <Space size='small'>
          <Tooltip title='编辑'>
            <Button
              size='small'
              icon={<EditOutlined />}
              onClick={() => router.push(`/knowledge/articles/${record.id}/edit`)}
              style={{ backgroundColor: '#3b82f6', color: '#fff', border: 'none' }}
            />
          </Tooltip>
          <Tooltip title='删除'>
            <Button
              size='small'
              icon={<DeleteOutlined />}
              onClick={() => handleDelete(record.id)}
              style={{ backgroundColor: '#ef4444', color: '#fff', border: 'none' }}
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <div className='p-6'>
      <div className='flex justify-between items-center mb-6'>
        <div>
          <h1 className='text-2xl font-bold text-gray-900'>知识库</h1>
          <p className='text-gray-500 mt-1'>创建、维护和分享解决方案与最佳实践</p>
        </div>
        <Button
          type='primary'
          icon={<PlusOutlined />}
          onClick={() => router.push('/knowledge/articles/new')}
          size='large'
        >
          新建文章
        </Button>
      </div>

      <Card className='rounded-lg shadow-sm border border-gray-200' variant='borderless'>
        <Form form={form} layout='inline' className='mb-6 flex-wrap gap-y-4'>
          <Form.Item name='search' className='mb-0'>
            <Input
              placeholder='搜索标题/内容'
              allowClear
              prefix={<SearchOutlined className='text-gray-400' />}
              className='w-64'
            />
          </Form.Item>
          <Form.Item name='category' className='mb-0'>
            <Select
              placeholder='分类'
              className='w-36'
              allowClear
              variant='borderless'
              style={{ backgroundColor: '#fafafa', borderRadius: '6px' }}
            >
              {Array.isArray(categories) &&
                categories.map((c, idx) => (
                  <Option key={idx} value={c}>
                    {c}
                  </Option>
                ))}
            </Select>
          </Form.Item>
          <Form.Item name='status' className='mb-0'>
            <Select
              placeholder='状态'
              className='w-28'
              allowClear
              variant='borderless'
              style={{ backgroundColor: '#fafafa', borderRadius: '6px' }}
            >
              <Option value='published'>已发布</Option>
              <Option value='draft'>草稿</Option>
            </Select>
          </Form.Item>
          <Form.Item className='mb-0'>
            <Space>
              <Button type='primary' onClick={handleSearch}>
                查询
              </Button>
              <Button onClick={loadData}>重置</Button>
            </Space>
          </Form.Item>
        </Form>

        {data.length === 0 && !loading ? (
          <Empty description='暂无知识库文章' image={Empty.PRESENTED_IMAGE_SIMPLE}>
            <Button type='primary' onClick={() => router.push('/knowledge/articles/new')}>
              创建第一篇文章
            </Button>
          </Empty>
        ) : (
          <Table
            rowKey='id'
            columns={columns as any}
            dataSource={data}
            loading={loading}
            pagination={{
              current: query.page,
              pageSize: query.pageSize,
              total: total,
              showSizeChanger: true,
              showTotal: total => `共 ${total} 条记录`,
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
