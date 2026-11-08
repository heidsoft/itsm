'use client';

/**
 * 知识库新建文章页面
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
  App,
  Typography,
  Breadcrumb,
  AutoComplete,
} from 'antd';
import { ArrowLeft, Save } from 'lucide-react';
import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';

const { Title } = Typography;
const { TextArea } = Input;

export default function NewKnowledgeArticlePage() {
  const router = useRouter();
  const [form] = Form.useForm();
  const { message } = App.useApp();
  // 提交动作态（表单页首帧是空表单，无列表假空态问题）。
  const [actionLoading, setActionLoading] = useState(false);
  const [categories, setCategories] = useState<string[]>([]);

  React.useEffect(() => {
    KnowledgeBaseApi.getCategories()
      .then(list => {
        // 后端返回的是该租户已使用的分类名称集合；纯数字项来自旧版把分类 ID
        // 当名称写入的脏数据，直接过滤，避免下拉里出现「1」。
        setCategories((list || []).filter(name => !/^\d+$/.test(String(name))));
      })
      .catch(() => {
        // 分类加载失败不阻塞写作，但必须让用户知道下拉不完整。
        message.error('加载知识分类失败，可手动输入分类名称');
      });
  }, [message]);

  const onFinish = async (values: { title: string; content: string; category: string; tags?: string[] }) => {
    setActionLoading(true);
    try {
      const created = await KnowledgeBaseApi.createArticle({
        title: values.title,
        content: values.content,
        category: values.category.trim(),
        tags: values.tags || [],
      });
      message.success('文章创建成功');
      router.push(`/knowledge/articles/${created.id}`);
    } catch (e: unknown) {
      message.error('创建失败：' + (e instanceof Error ? e.message : '未知错误'));
    } finally {
      setActionLoading(false);
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
          initialValues={{ tags: [] }}
        >
          <Form.Item name="title" label="标题" rules={[{ required: true, message: '请输入标题' }]}>
            <Input placeholder="例如：VPN 拨号失败排查指南" maxLength={200} />
          </Form.Item>

          {/* 分类在后端是自由字符串，因此用 AutoComplete：既可从已有分类中选择，
              也可直接输入新分类，避免后端分类为空时表单无法提交。 */}
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
            <TextArea
              rows={15}
              placeholder="# 问题描述&#10;&#10;请输入内容..."
            />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" icon={<Save />} loading={actionLoading}>
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
