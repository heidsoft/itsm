'use client';

/**
 * 知识库新建文章页面
 * B7 修复：原本 /knowledge/articles/new 路由 404
 */

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
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
} from 'antd';
import { ArrowLeft, Save } from 'lucide-react';
import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';

const { Title } = Typography;
const { TextArea } = Input;

export default function NewKnowledgeArticlePage() {
  const router = useRouter();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [categories, setCategories] = useState<{ id: number; name: string }[]>([]);

  React.useEffect(() => {
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

  const onFinish = async (values: any) => {
    setLoading(true);
    try {
      const created = await KnowledgeBaseApi.createArticle({
        title: values.title,
        content: values.content,
        category:
          categories.find(c => c.id === values.category_id)?.name || String(values.category_id),
        tags: values.tags || [],
      });
      message.success('文章创建成功');
      router.push(`/knowledge/articles/${(created as any).id}`);
    } catch (e: any) {
      message.error('创建失败：' + (e?.message || '未知错误'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-4xl mx-auto p-6">
      <Breadcrumb
        items={[{ title: '知识库', href: '/knowledge' }, { title: '新建文章' }]}
        className="mb-4"
      />
      <Card>
        <Space className="mb-4">
          <Button icon={<ArrowLeft />} onClick={() => router.push('/knowledge')}>
            返回
          </Button>
          <Title level={3} style={{ margin: 0 }}>
            新建知识库文章
          </Title>
        </Space>
        <Form
          form={form}
          layout="vertical"
          onFinish={onFinish}
          initialValues={{ category_id: 1, tags: [] }}
        >
          <Form.Item name="title" label="标题" rules={[{ required: true, message: '请输入标题' }]}>
            <Input placeholder="例如：VPN 拨号失败排查指南" maxLength={200} />
          </Form.Item>

          <Form.Item
            name="category_id"
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
            <TextArea
              rows={15}
              placeholder="# 问题描述&#10;&#10;请输入内容..."
            />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" icon={<Save />} loading={loading}>
                保存草稿
              </Button>
              <Button onClick={() => router.push('/knowledge')}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
