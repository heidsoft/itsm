'use client';

import React, { useState, useEffect } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { Button, Card, Form, Input, Select, Row, Col, Space, Divider, App } from 'antd';
import { ArrowLeftOutlined, SaveOutlined } from '@ant-design/icons';
import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';

const { TextArea } = Input;
const { Option } = Select;

export default function EditArticlePage() {
  const router = useRouter();
  const { message } = App.useApp();
  const params = useParams();
  const id = params?.id as string;
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [fetching, setFetching] = useState(false);
  const [categories, setCategories] = useState<string[]>([]);
  const [articleData, setArticleData] = useState<any>(null);

  // Load categories
  useEffect(() => {
    const loadCategories = async () => {
      try {
        const res = await KnowledgeBaseApi.getCategories();
        // Map KnowledgeCategory objects to strings for backward compatibility
        const categoryNames = (res || []).map((cat: any) => cat.name || cat.id || String(cat));
        setCategories(categoryNames);
      } catch (error) {
        setCategories(['故障排查', '解决方案', '操作流程', '最佳实践', '技术文档']);
      }
    };
    loadCategories();
  }, []);

  // Fetch article data
  useEffect(() => {
    if (!id) return;

    const fetchArticle = async () => {
      setFetching(true);
      try {
        const data = await KnowledgeBaseApi.getArticle(id);
        setArticleData(data);
        const anyData = data as any;
        form.setFieldsValue({
          title: data.title,
          content: data.content,
          category: data.categoryId || anyData.category,
          tags: data.tags || [],
        });
        // Set page title
        if (typeof document !== 'undefined') {
          document.title = `编辑: ${data.title} - 知识库`;
        }
      } catch (error) {
        // Use alert as fallback since message might not be ready
        if (typeof window !== 'undefined') {
          alert('获取文章失败');
        }
        router.push('/knowledge');
      } finally {
        setFetching(false);
      }
    };

    fetchArticle();
  }, [id, form, router]);

  const handleSubmit = async (values: any) => {
    if (!id) return;

    setLoading(true);
    try {
      await KnowledgeBaseApi.updateArticle(id, {
        title: values.title,
        content: values.content,
        categoryId: values.category,
        tags: values.tags || [],
      });
      message.success('更新成功');
      router.push(`/knowledge/articles/${id}`);
    } catch (error) {
      message.error('更新失败');
    } finally {
      setLoading(false);
    }
  };

  const handleCancel = () => {
    router.back();
  };

  return (
    <div className='p-6 min-h-screen bg-gray-50'>
      <div className='mb-6'>
        <Button
          type='link'
          icon={<ArrowLeftOutlined />}
          onClick={() => router.back()}
          style={{ paddingLeft: 0, color: '#666' }}
        >
          返回
        </Button>
      </div>

      <Card
        title={<span className='text-lg font-medium'>编辑知识库文章</span>}
        loading={fetching}
      >
        <Form
          form={form}
          layout='vertical'
          onFinish={handleSubmit}
        >
          <Row gutter={24}>
            <Col span={24}>
              <Form.Item
                name='title'
                label='文章标题'
                rules={[{ required: true, message: '请输入文章标题' }]}
              >
                <Input placeholder='请输入文章标题' />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={12}>
              <Form.Item
                name='category'
                label='分类'
                rules={[{ required: true, message: '请选择分类' }]}
              >
                <Select placeholder='请选择分类'>
                  {categories.map((cat, idx) => (
                    <Option key={idx} value={cat}>{cat}</Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name='tags'
                label='标签'
              >
                <Select
                  mode='tags'
                  placeholder='输入标签后按回车添加'
                  style={{ width: '100%' }}
                />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={24}>
              <Form.Item
                name='content'
                label='文章内容'
                rules={[{ required: true, message: '请输入文章内容' }]}
              >
                <TextArea
                  rows={16}
                  placeholder='请使用 Markdown 格式编写文章内容...
# 标题1
## 标题2

正文内容...

### 步骤
1. 步骤一
2. 步骤二

### 注意事项
- 注意一
- 注意二'
                />
              </Form.Item>
            </Col>
          </Row>

          <Divider />

          <Form.Item>
            <Space>
              <Button type='primary' htmlType='submit' icon={<SaveOutlined />} loading={loading}>
                保存
              </Button>
              <Button onClick={handleCancel}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
