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
  const [loading, setLoading] = useState(false);
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
        message.error('Failed to load knowledge classification. You can enter the classification name manually.');
      });
  }, [message]);

  const onFinish = async (values: { title: string; content: string; category: string; tags?: string[] }) => {
    setLoading(true);
    try {
      const created = await KnowledgeBaseApi.createArticle({
        title: values.title,
        content: values.content,
        category: values.category.trim(),
        tags: values.tags || [],
      });
      message.success('Article created successfully');
      router.push(`/knowledge/articles/${created.id}`);
    } catch (e: unknown) {
      message.error('Creation failed:' + (e instanceof Error ? e.message : 'Unknown error'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-4xl mx-auto p-6">
      <Breadcrumb
        items={[{ title: 'Knowledge Base', href: '/knowledge' }, { title: 'New Article' }]}
        className="mb-4"
      />
      <Card>
        <Space className="mb-4">
          <Button icon={<ArrowLeft />} onClick={() => router.push('/knowledge')}>

            Back
          </Button>
          <Title level={3} style={{ margin: 0 }}>

            Create a new knowledge base article
          </Title>
        </Space>
        <Form
          form={form}
          layout="vertical"
          onFinish={onFinish}
          initialValues={{ tags: [] }}
        >
          <Form.Item name="title" label="Title" rules={[{ required: true, message: 'Please enter a title' }]}>
            <Input placeholder="Example: VPN dial-up failure troubleshooting guide" maxLength={200} />
          </Form.Item>

          {/* 分类在后端是自由字符串，因此用 AutoComplete：既可从已有分类中选择，
              也可直接输入新分类，避免后端分类为空时表单无法提交。 */}
          <Form.Item
            name="category"
            label="Category"
            rules={[{ required: true, message: 'Please select or enter a category' }]}
          >
            <AutoComplete
              placeholder="Select or enter a category, such as &quot;Troubleshooting&quot;"
              options={categories.map(name => ({ value: name }))}
              filterOption={(input, option) =>
                String(option?.value ?? '').toLowerCase().includes(input.toLowerCase())
              }
            />
          </Form.Item>

          <Form.Item name="tags" label="Tags">
            <Select
              mode="tags"
              placeholder="Enter the label and press Enter"
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
            label="Content (supports Markdown)"
            rules={[{ required: true, message: 'Please enter content' }]}
          >
            <TextArea
              rows={15}
              placeholder="#Problem description

Please enter content..."
            />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" icon={<Save />} loading={loading}>

                Save draft
              </Button>
              <Button onClick={() => router.push('/knowledge')}>Cancel</Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
