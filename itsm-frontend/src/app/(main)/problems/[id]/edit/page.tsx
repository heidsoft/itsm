'use client';

import React, { useState, useEffect } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { Button, Card, Form, Input, Select, App, Row, Col, Space, Divider } from 'antd';
import { ArrowLeft, Save } from 'lucide-react';
import { ProblemApi } from '@/lib/api/problem-api';
import {
  ProblemCategoryOptions,
  isKnownProblemCategory,
} from '@/constants/problem';
import { useI18n } from '@/lib/i18n';

const { TextArea } = Input;

interface ProblemData {
  id: number;
  title: string;
  description: string;
  priority: string;
  category: string;
  status: string;
  rootCause: string;
  impact: string;
}

export default function ProblemEditPage() {
  const router = useRouter();
  const params = useParams();
  const id = params?.id as string;
  const { message } = App.useApp();
  const { t } = useI18n();
  const [form] = Form.useForm();
  // 提交动作态（数据加载由 fetching 负责）
  const [actionLoading, setActionLoading] = useState(false);
  const [fetching, setFetching] = useState(false);
  const [problemData, setProblemData] = useState<ProblemData | null>(null);

  // Fetch problem data
  useEffect(() => {
    if (!id) return;

    const fetchProblem = async () => {
      setFetching(true);
      try {
        const data = await ProblemApi.getProblem(Number(id));
        setProblemData(data);
        // 后端可能带不在前枚举里的 category（旧数据 / 脏数据），
        // antd v6 Select 不识别时表现为“空白”，此处直接显示原始字符串 + 后缀提示。
        const rawCategory = typeof data.category === 'string' ? data.category : '';
        const safeCategory = isKnownProblemCategory(rawCategory) ? rawCategory : '';
        form.setFieldsValue({
          title: data.title,
          description: data.description,
          priority: data.priority,
          category: safeCategory,
          status: data.status,
          rootCause: data.rootCause,
          impact: data.impact,
        });
        if (rawCategory && !safeCategory) {
          message.warning(`Original category “${rawCategory}” is not in the current enumeration, please select again`);
        }
      } catch (error) {
        message.error(t('problems.getFailed'));
        router.push('/problems');
      } finally {
        setFetching(false);
      }
    };

    fetchProblem();
  }, [id, form, router]);

  const handleSubmit = async (values: {
    title: string;
    description: string;
    priority: string;
    category: string;
    status: string;
    rootCause: string;
    impact: string;
  }) => {
    if (!id) return;

    setActionLoading(true);
    try {
      await ProblemApi.updateProblem(Number(id), values);
      message.success(t('problems.updateSuccess'));
      router.push(`/problems/${id}`);
    } catch (error) {
      message.error(t('problems.updateFailed'));
    } finally {
      setActionLoading(false);
    }
  };

  const handleCancel = () => {
    router.back();
  };

  return (
    <div className="p-6 min-h-screen bg-gray-50">
      <div className="mb-6">
        <Button
          type="link"
          icon={<ArrowLeft />}
          onClick={() => router.back()}
          style={{ paddingLeft: 0, color: '#666' }}
        >

          Back
        </Button>
      </div>

      <Card
        title={
          <span className="text-lg font-medium">Edit question - #{problemData?.id}</span>
        }
        loading={fetching}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          initialValues={{
            priority: 'medium',
            status: 'open',
          }}
        >
          <Row gutter={24}>
            <Col span={24}>
              <Form.Item
                name="title"
                label="Question title"
                rules={[{ required: true, message: 'Please enter a question title' }]}
              >
                <Input placeholder="Please enter a question title" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={12}>
              <Form.Item
                name="status"
                label="Status"
                rules={[{ required: true, message: 'Please select status' }]}
              >
                <Select placeholder="Please select status" options={[{ value: "open", label: "Pending" }, { value: "investigating", label: "Investigating" }, { value: "resolved", label: "Resolved" }, { value: "closed", label: "Closed" }]} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="priority"
                label="Priority"
                rules={[{ required: true, message: 'Please select priority' }]}
              >
                <Select placeholder="Please select priority" options={[{ value: "low", label: "Low" }, { value: "medium", label: "Medium" }, { value: "high", label: "High" }, { value: "critical", label: "Urgent" }]} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={24}>
              <Form.Item name="category" label="Category">
                <Select
                  placeholder="Please select a category"
                  allowClear
                  // 旧数据 category 可能不在前枚举里，先把原始值附加为额外选项，
                  // 这样既能保留原有内容、又能让 Select 正常显示旧值。
                  options={[
                    ...ProblemCategoryOptions,
                    ...(problemData &&
                    problemData.category &&
                    !isKnownProblemCategory(problemData.category)
                      ? [
                          {
                            value: problemData.category,
                            label: `${problemData.category}(old value)`,
                          },
                        ]
                      : []),
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={24}>
              <Form.Item name="description" label="Problem description">
                <TextArea rows={4} placeholder="Please describe the problem in detail" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={24}>
              <Form.Item name="rootCause" label="Root Cause Analysis">
                <TextArea rows={4} placeholder="Describe the root cause in detail" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={24}>
              <Form.Item name="impact" label="Impact Range">
                <TextArea rows={3} placeholder="Please describe the scope of the problem" />
              </Form.Item>
            </Col>
          </Row>

          <Divider />

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" icon={<Save />} loading={actionLoading}>
                Save
              </Button>
              <Button onClick={handleCancel}>Cancel</Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
