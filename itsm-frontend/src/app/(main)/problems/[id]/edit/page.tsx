'use client';

import React, { useState, useEffect } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { Button, Card, Form, Input, Select, message, Row, Col, Space, Divider } from 'antd';
import { ArrowLeftOutlined, SaveOutlined } from '@ant-design/icons';
import { ProblemApi } from '@/lib/api/problem-api';
import { problemService } from '@/lib/services/problem-service';

const { TextArea } = Input;
const { Option } = Select;

export default function ProblemEditPage() {
  const router = useRouter();
  const params = useParams();
  const id = params?.id as string;
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [fetching, setFetching] = useState(false);
  const [problemData, setProblemData] = useState<any>(null);

  // Fetch problem data
  useEffect(() => {
    if (!id) return;

    const fetchProblem = async () => {
      setFetching(true);
      try {
        const resp = await ProblemApi.getProblem(Number(id));
        const data = resp as any;
        setProblemData(data);
        form.setFieldsValue({
          title: data.title,
          description: data.description,
          priority: data.priority,
          category: data.category,
          status: data.status,
          root_cause: data.root_cause,
          impact: data.impact,
        });
      } catch (error) {
        message.error('获取问题失败');
        router.push('/problems');
      } finally {
        setFetching(false);
      }
    };

    fetchProblem();
  }, [id, form, router]);

  const handleSubmit = async (values: any) => {
    if (!id) return;

    setLoading(true);
    try {
      await ProblemApi.updateProblem(Number(id), values);
      message.success('更新成功');
      router.push(`/problems/${id}`);
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
        title={
          <span className='text-lg font-medium'>
            编辑问题 - {problemData?.problem_number}
          </span>
        }
        loading={fetching}
      >
        <Form
          form={form}
          layout='vertical'
          onFinish={handleSubmit}
          initialValues={{
            priority: 'medium',
            status: 'open',
          }}
        >
          <Row gutter={24}>
            <Col span={24}>
              <Form.Item
                name='title'
                label='问题标题'
                rules={[{ required: true, message: '请输入问题标题' }]}
              >
                <Input placeholder='请输入问题标题' />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={12}>
              <Form.Item
                name='status'
                label='状态'
                rules={[{ required: true, message: '请选择状态' }]}
              >
                <Select placeholder='请选择状态'>
                  <Option value='open'>打开</Option>
                  <Option value='in_progress'>进行中</Option>
                  <Option value='resolved'>已解决</Option>
                  <Option value='closed'>已关闭</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name='priority'
                label='优先级'
                rules={[{ required: true, message: '请选择优先级' }]}
              >
                <Select placeholder='请选择优先级'>
                  <Option value='low'>低</Option>
                  <Option value='medium'>中</Option>
                  <Option value='high'>高</Option>
                  <Option value='critical'>紧急</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={24}>
              <Form.Item name='category' label='分类'>
                <Select placeholder='请选择分类' allowClear>
                  <Option value='系统问题'>系统问题</Option>
                  <Option value='网络问题'>网络问题</Option>
                  <Option value='数据库问题'>数据库问题</Option>
                  <Option value='应用问题'>应用问题</Option>
                  <Option value='安全问题'>安全问题</Option>
                  <Option value='硬件问题'>硬件问题</Option>
                  <Option value='其他'>其他</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={24}>
              <Form.Item name='description' label='问题描述'>
                <TextArea rows={4} placeholder='请详细描述问题情况' />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={24}>
              <Form.Item name='root_cause' label='根本原因分析'>
                <TextArea rows={4} placeholder='请详细描述问题的根本原因' />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={24}>
              <Form.Item name='impact' label='影响范围'>
                <TextArea rows={3} placeholder='请描述问题的影响范围' />
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
