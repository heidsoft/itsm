'use client';

import React, { useState, useEffect, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { ArrowLeft } from 'lucide-react';
import { Form, Input, Select, Button, Card, message, Alert, Spin } from 'antd';
import { problemService, ProblemPriority } from '@/lib/services/problem-service';

const { TextArea } = Input;
const { Option } = Select;

const CreateProblemPageContent = () => {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const incidentId = searchParams.get('fromIncidentId');
    const incidentTitle = searchParams.get('incidentTitle');
    const incidentDescription = searchParams.get('incidentDescription');

    if (incidentId) {
      form.setFieldsValue({
        title: `由事件 ${incidentId} 引起的问题: ${incidentTitle || ''}`,
        description: `此问题由以下事件引发：\n事件ID: ${incidentId}\n事件标题: ${
          incidentTitle || ''
        }\n事件描述: ${incidentDescription || ''}\n\n请在此处填写问题的详细描述和根本原因分析...`,
      });
    }
  }, [searchParams, form]);

  const handleSubmit = async (values: any) => {
    setLoading(true);
    try {
      await problemService.createProblem({
        title: values.title,
        description: values.description,
        priority: values.priority,
        category: values.category,
        root_cause: values.root_cause,
        impact: values.impact,
        assignee_id: values.assignee_id,
      });

      message.success('问题创建成功！');
      router.push('/problems');
    } catch (error) {
      console.error('创建问题失败:', error);
      message.error('创建问题失败，请重试');
    } finally {
      setLoading(false);
    }
  };

  const priorityOptions = [
    { value: ProblemPriority.LOW, label: '低' },
    { value: ProblemPriority.MEDIUM, label: '中' },
    { value: ProblemPriority.HIGH, label: '高' },
    { value: ProblemPriority.CRITICAL, label: '紧急' },
  ];

  const categoryOptions = [
    { value: '系统问题', label: '系统问题' },
    { value: '网络问题', label: '网络问题' },
    { value: '数据库问题', label: '数据库问题' },
    { value: '应用问题', label: '应用问题' },
    { value: '安全问题', label: '安全问题' },
    { value: '硬件问题', label: '硬件问题' },
    { value: '其他', label: '其他' },
  ];

  return (
    <div className='p-10 bg-gray-50 min-h-full'>
      <header className='mb-8'>
        <button
          onClick={() => router.back()}
          className='flex items-center text-blue-600 hover:underline mb-4'
        >
          <ArrowLeft className='w-5 h-5 mr-2' />
          返回问题列表
        </button>
        <h2 className='text-4xl font-bold text-gray-800'>新建问题</h2>
        <p className='text-gray-500 mt-1'>识别、分析和解决IT服务的根本原因</p>
      </header>

      <Card className='shadow-md'>
        <Form
          form={form}
          layout='vertical'
          onFinish={handleSubmit}
          initialValues={{
            priority: ProblemPriority.MEDIUM,
            category: '系统问题',
          }}
        >
          {searchParams.get('fromIncidentId') && (
            <Alert
              message={`此问题由事件 ${searchParams.get('fromIncidentId')} 触发`}
              type='info'
              showIcon
              className='mb-6'
            />
          )}

          <Form.Item
            label='问题标题'
            name='title'
            rules={[
              { required: true, message: '请输入问题标题' },
              { min: 2, max: 200, message: '标题长度应在2-200字符之间' },
            ]}
          >
            <Input placeholder='简要描述问题内容' />
          </Form.Item>

          <Form.Item
            label='详细描述'
            name='description'
            rules={[
              { required: true, message: '请输入问题详细描述' },
              { min: 10, max: 5000, message: '描述长度应在10-5000字符之间' },
            ]}
          >
            <TextArea
              rows={6}
              placeholder='请提供问题的详细信息，包括影响范围、发生时间、已观察到的现象等...'
            />
          </Form.Item>

          <Form.Item
            label='优先级'
            name='priority'
            rules={[{ required: true, message: '请选择优先级' }]}
          >
            <Select placeholder='选择优先级'>
              {priorityOptions.map(option => (
                <Option key={option.value} value={option.value}>
                  {option.label}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            label='分类'
            name='category'
            rules={[{ required: true, message: '请选择分类' }]}
          >
            <Select placeholder='选择分类'>
              {categoryOptions.map(option => (
                <Option key={option.value} value={option.value}>
                  {option.label}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            label='根本原因分析 (RCA)'
            name='root_cause'
            rules={[
              { required: true, message: '请输入根本原因分析' },
              { min: 10, max: 5000, message: '内容长度应在10-5000字符之间' },
            ]}
          >
            <TextArea rows={4} placeholder='请详细说明问题的根本原因...' />
          </Form.Item>

          <Form.Item
            label='影响范围'
            name='impact'
            rules={[
              { required: true, message: '请输入影响范围' },
              { min: 10, max: 5000, message: '内容长度应在10-5000字符之间' },
            ]}
          >
            <TextArea rows={3} placeholder='请描述问题的影响范围，如影响哪些用户、服务或业务...' />
          </Form.Item>

          <Form.Item className='mb-0'>
            <div className='flex justify-end space-x-4'>
              <Button onClick={() => router.back()}>取消</Button>
              <Button type='primary' htmlType='submit' loading={loading}>
                创建问题
              </Button>
            </div>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
};

const CreateProblemPage = () => {
  return (
    <Suspense
      fallback={
        <div className='flex items-center justify-center min-h-screen'>
          <Spin size='large' />
        </div>
      }
    >
      <CreateProblemPageContent />
    </Suspense>
  );
};

export default CreateProblemPage;
