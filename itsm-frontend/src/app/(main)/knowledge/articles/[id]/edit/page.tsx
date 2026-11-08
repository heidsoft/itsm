'use client';

/**
 * 知识库文章编辑页面
 */

import React, { useEffect, useState } from 'react';
import { useRouter, useParams } from 'next/navigation';
import {
  Card,
  Form,
  Input,
  Select,
  Tag,
  Button,
  Space,
  App,
  Typography,
  Breadcrumb,
  Skeleton,
  AutoComplete,
} from 'antd';
import { ArrowLeft, Save } from 'lucide-react';
import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';

const { Title } = Typography;
const { TextArea } = Input;

export default function EditKnowledgeArticlePage() {
  const router = useRouter();
  const { id } = useParams() as { id: string };
  const [form] = Form.useForm();
  const { message } = App.useApp();
  // 提交动作态（数据加载由 fetching 负责）。
  const [actionLoading, setActionLoading] = useState(false);
  const [fetching, setFetching] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [categories, setCategories] = useState<string[]>([]);

  useEffect(() => {
    KnowledgeBaseApi.getCategories()
      .then(list => {
        // 后端返回该租户已使用的分类名称集合；纯数字项是把分类 ID 当名称写入的脏数据。
        setCategories((list || []).filter(name => !/^\d+$/.test(String(name))));
      })
      .catch(() => {
        message.error('加载知识分类失败，可手动输入分类名称');
      });
  }, [message]);

  useEffect(() => {
    if (!id) return;
    setFetching(true);
    KnowledgeBaseApi.getArticle(id)
      .then(article => {
        form.setFieldsValue({
          title: article.title,
          content: article.content,
          // 后端契约字段是 category（分类名称）；历史脏数据把分类 ID
          // 存成了名称，回填时按「未分类」留给用户重新输入。
          category: /^\d+$/.test(article.category ?? '')
            ? undefined
            : article.category,
          tags: article.tags || [],
        });
      })
      .catch(() => {
        setNotFound(true);
      })
      .finally(() => setFetching(false));
  }, [id, form]);

  const onFinish = async (values: { title: string; content: string; category: string; tags?: string[] }) => {
    setActionLoading(true);
    try {
      await KnowledgeBaseApi.updateArticle(id, {
        title: values.title,
        content: values.content,
        category: values.category.trim(),
        tags: values.tags || [],
      });
      message.success('文章更新成功');
      router.push(`/knowledge/articles/${id}`);
    } catch (e: unknown) {
      message.error('更新失败：' + (e instanceof Error ? e.message : '未知错误'));
    } finally {
      setActionLoading(false);
    }
  };

  if (notFound) {
    return (
      <div className="max-w-4xl mx-auto p-6">
        <Card>
          <Title level={4}>文章不存在或已被删除</Title>
          <Button onClick={() => router.push('/knowledge')}>返回知识库</Button>
        </Card>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto p-6">
      <Breadcrumb
        items={[
          { title: '知识库', href: '/knowledge' },
          { title: '编辑文章' },
        ]}
        className="mb-4"
      />
      <Card>
        <Space className="mb-4">
          <Button icon={<ArrowLeft />} onClick={() => router.push(`/knowledge/articles/${id}`)}>
            返回
          </Button>
          <Title level={3} style={{ margin: 0 }}>
            编辑知识库文章
          </Title>
        </Space>
        {fetching ? (
          <Skeleton active paragraph={{ rows: 10 }} />
        ) : (
          <Form form={form} layout="vertical" onFinish={onFinish} initialValues={{ tags: [] }}>
            <Form.Item
              name="title"
              label="标题"
              rules={[{ required: true, message: '请输入标题' }]}
            >
              <Input placeholder="例如：VPN 拨号失败排查指南" maxLength={200} />
            </Form.Item>

            <Form.Item
              name="category"
              label="分类"
              rules={[{ required: true, message: '请选择或输入分类' }]}
            >
              <AutoComplete
                placeholder="选择或输入分类，如「故障排查」"
                options={categories.map(name => ({ value: name }))}
                filterOption={(input, option) =>
                  String(option?.value ?? '').toLowerCase().includes(input.toLowerCase())
                }
              />
            </Form.Item>

            <Form.Item name="tags" label="标签">
              <Select
                mode="tags"
                placeholder="输入标签后回车"
                tagRender={({ label, closable, onClose }) => (
                  <Tag
                    closable={closable}
                    onClose={onClose}
                    className="bg-blue-100 text-blue-800 border-blue-300 mr-1 mb-1"
                  >
                    {label}
                  </Tag>
                )}
              />
            </Form.Item>

            <Form.Item
              name="content"
              label="内容（支持 Markdown）"
              rules={[{ required: true, message: '请输入内容' }]}
            >
              <TextArea rows={15} placeholder="# 问题描述&#10;&#10;请输入内容..." />
            </Form.Item>

            <Form.Item>
              <Space>
                <Button type="primary" htmlType="submit" icon={<Save />} loading={actionLoading}>
                  保存修改
                </Button>
                <Button onClick={() => router.push(`/knowledge/articles/${id}`)}>取消</Button>
              </Space>
            </Form.Item>
          </Form>
        )}
      </Card>
    </div>
  );
}
