'use client';

/**
 * 知识库文章编辑页面
 * 修复：列表/详情页“编辑”按钮指向 /knowledge/articles/[id]/edit，但路由缺失导致 404
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
  message,
  Typography,
  Breadcrumb,
  Skeleton,
} from 'antd';
import { ArrowLeft, Save } from 'lucide-react';
import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';

const { Title } = Typography;
const { TextArea } = Input;

export default function EditKnowledgeArticlePage() {
  const router = useRouter();
  const { id } = useParams() as { id: string };
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [fetching, setFetching] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [categories, setCategories] = useState<{ id: number; name: string }[]>([]);

  useEffect(() => {
    KnowledgeBaseApi.getCategories()
      .then((data: any) => {
        const list = Array.isArray(data) ? data : data?.categories || [];
        setCategories(list.map((c: any) => ({ id: c.id, name: c.name })));
      })
      .catch(() => {
        // fallback 默认分类
        setCategories([
          { id: 1, name: '故障处理' },
          { id: 2, name: '操作指南' },
          { id: 3, name: '常见问题' },
        ]);
      });
  }, []);

  useEffect(() => {
    if (!id) return;
    setFetching(true);
    KnowledgeBaseApi.getArticle(id)
      .then(article => {
        const matched = categories.find(
          c => c.name === article.categoryName || String(c.id) === String(article.categoryId),
        );
        form.setFieldsValue({
          title: article.title,
          content: article.content,
          categoryId: matched?.id ?? 1,
          tags: article.tags || [],
        });
      })
      .catch(() => {
        setNotFound(true);
      })
      .finally(() => setFetching(false));
    // categories 加载完成后重新匹配一次默认分类
  }, [id, categories, form]);

  const onFinish = async (values: any) => {
    setLoading(true);
    try {
      await KnowledgeBaseApi.updateArticle(id, {
        title: values.title,
        content: values.content,
        category:
          categories.find(c => c.id === values.categoryId)?.name || String(values.categoryId),
        tags: values.tags || [],
      });
      message.success('文章更新成功');
      router.push(`/knowledge/articles/${id}`);
    } catch (e: any) {
      message.error('更新失败：' + (e?.message || '未知错误'));
    } finally {
      setLoading(false);
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
              name="categoryId"
              label="分类"
              rules={[{ required: true, message: '请选择分类' }]}
            >
              <Select
                placeholder="选择分类"
                options={categories.map(c => ({ label: c.name, value: c.id }))}
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
                <Button type="primary" htmlType="submit" icon={<Save />} loading={loading}>
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
