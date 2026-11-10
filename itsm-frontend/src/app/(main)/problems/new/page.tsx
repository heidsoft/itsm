'use client';

import React, { useState, useEffect, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { ArrowLeft } from 'lucide-react';
import { Form, Input, Select, Button, Card, App, Alert, Spin } from 'antd';
import { ProblemApi } from '@/lib/api/problem-api';
import { ProblemPriority, ProblemCategoryOptions } from '@/constants/problem';
import { useI18n } from '@/lib/i18n';

const { TextArea } = Input;

const CreateProblemPageContent = () => {
  const router = useRouter();
  const { t } = useI18n();
  const searchParams = useSearchParams();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const { message } = App.useApp();

  useEffect(() => {
    const incidentId = searchParams.get('fromIncidentId');
    const incidentTitle = searchParams.get('incidentTitle');
    const incidentDescription = searchParams.get('incidentDescription');

    if (incidentId) {
      form.setFieldsValue({
        title: `by event ${incidentId} Problem caused by: ${incidentTitle || ''}`,
        description: `This problem is caused by the following event:
Event ID: ${incidentId}
Event title: ${
          incidentTitle || ''
        }
Event description: ${incidentDescription || ''}

Please fill in a detailed description and root cause analysis of the problem here...`,
      });
    }
  }, [searchParams, form]);

  const handleSubmit = async (values: {
    title: string;
    description: string;
    priority: string;
    category: string;
    rootCause: string;
    impact: string;
  }) => {
    setLoading(true);
    try {
      await ProblemApi.createProblem({
        title: values.title,
        description: values.description,
        priority: values.priority,
        category: values.category,
        rootCause: values.rootCause,
        impact: values.impact,
      });

      message.success(t('problems.createSuccess'));
      router.push('/problems');
    } catch (error) {
      console.error('Failed to create issue:', error);
      message.error(t('problems.createFailed'));
    } finally {
      setLoading(false);
    }
  };

  const priorityOptions = [
    { value: ProblemPriority.LOW, label: 'Low' },
    { value: ProblemPriority.MEDIUM, label: 'Medium' },
    { value: ProblemPriority.HIGH, label: 'High' },
    { value: ProblemPriority.CRITICAL, label: 'Urgent' },
  ];

  return (
    <div className="p-10 bg-gray-50 min-h-full">
      <header className="mb-8">
        <button
          onClick={() => router.back()}
          className="flex items-center text-blue-600 hover:underline mb-4"
          aria-label="Back to problem list"
        >
          <ArrowLeft className="w-5 h-5 mr-2" />

          Back to problem list
        </button>
        <h2 className="text-4xl font-bold text-gray-800">New Problem</h2>
        <p className="text-gray-500 mt-1">Identify, analyze and resolve root causes of IT services</p>
      </header>

      <Card className="shadow-md">
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          initialValues={{
            priority: ProblemPriority.MEDIUM,
            category: 'System problem',
          }}
        >
          {searchParams.get('fromIncidentId') && (
            <Alert
              title={`This issue is caused by the event ${searchParams.get('fromIncidentId')} trigger`}
              type="info"
              showIcon
              className="mb-6"
            />
          )}

          <Form.Item
            label="Question title"
            name="title"
            rules={[
              { required: true, message: 'Please enter a question title' },
              { min: 2, max: 200, message: 'Title length should be between 2-200 characters' },
            ]}
          >
            <Input placeholder="Briefly describe the problem content" />
          </Form.Item>

          <Form.Item
            label="Detailed description"
            name="description"
            rules={[
              { required: true, message: 'Please enter a detailed description of the problem' },
              { min: 10, max: 5000, message: 'Description length should be between 10-5000 characters' },
            ]}
          >
            <TextArea
              rows={6}
              placeholder="Please provide detailed information about the problem, including scope of impact, time of occurrence, observed phenomena, etc..."
            />
          </Form.Item>

          <Form.Item
            label="Priority"
            name="priority"
            rules={[{ required: true, message: 'Please select priority' }]}
          >
            <Select placeholder="Select priority" options={priorityOptions} />
          </Form.Item>

          <Form.Item
            label="Category"
            name="category"
            rules={[{ required: true, message: 'Please select a category' }]}
          >
            <Select placeholder="Select category" options={ProblemCategoryOptions} />
          </Form.Item>

          <Form.Item
            label="Root Cause Analysis (RCA)"
            name="rootCause"
            rules={[
              { required: true, message: 'Please enter root cause analysis' },
              { min: 10, max: 5000, message: 'Content length should be between 10-5000 characters' },
            ]}
          >
            <TextArea rows={4} placeholder="Please elaborate on the root cause of the problem..." />
          </Form.Item>

          <Form.Item
            label="Impact Range"
            name="impact"
            rules={[
              { required: true, message: 'Please enter the scope of influence' },
              { min: 10, max: 5000, message: 'Content length should be between 10-5000 characters' },
            ]}
          >
            <TextArea rows={3} placeholder="Please describe the scope of the problem, such as which users, services or businesses are affected..." />
          </Form.Item>

          <Form.Item className="mb-0">
            <div className="flex justify-end space-x-4">
              <Button onClick={() => router.back()}>Cancel</Button>
              <Button type="primary" htmlType="submit" loading={loading}>

                Create issue
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
        <div className="flex items-center justify-center min-h-screen">
          <Spin size="large" />
        </div>
      }
    >
      <CreateProblemPageContent />
    </Suspense>
  );
};

export default CreateProblemPage;
