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
export default function ProblemEditPage() {
  const router = useRouter();
  const params = useParams();
  const id = params?.id as string;
  const { message } = App.useApp();
  const { t } = useI18n();
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
          message.warning(`原分类 “${rawCategory}” 不在当前枚举内，请重新选择`);
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

  const handleSubmit = async (values: any) => {
    if (!id) return;

    setLoading(true);
    try {
      await ProblemApi.updateProblem(Number(id), values);
      message.success(t('problems.updateSuccess'));
      router.push(`/problems/${id}`);
    } catch (error) {
      message.error(t('problems.updateFailed'));
    } finally {
      setLoading(false);
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
          返回
        </Button>
      </div>

      <Card
        title={
          <span className="text-lg font-medium">编辑问题 - #{problemData?.id}</span>
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
                label="问题标题"
                rules={[{ required: true, message: '请输入问题标题' }]}
              >
                <Input placeholder="请输入问题标题" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={12}>
              <Form.Item
                name="status"
                label="状态"
                rules={[{ required: true, message: '请选择状态' }]}
              >
                <Select placeholder="请选择状态" options={[{ value: "open", label: "待处理" }, { value: "investigating", label: "调查中" }, { value: "resolved", label: "已解决" }, { value: "closed", label: "已关闭" }]} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="priority"
                label="优先级"
                rules={[{ required: true, message: '请选择优先级' }]}
              >
                <Select placeholder="请选择优先级" options={[{ value: "low", label: "低" }, { value: "medium", label: "中" }, { value: "high", label: "高" }, { value: "critical", label: "紧急" }]} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={24}>
              <Form.Item name="category" label="分类">
                <Select
                  placeholder="请选择分类"
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
                            label: `${problemData.category}（旧值）`,
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
              <Form.Item name="description" label="问题描述">
                <TextArea rows={4} placeholder="请详细描述问题情况" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={24}>
              <Form.Item name="rootCause" label="根本原因分析">
                <TextArea rows={4} placeholder="请详细描述问题的根本原因" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={24}>
              <Form.Item name="impact" label="影响范围">
                <TextArea rows={3} placeholder="请描述问题的影响范围" />
              </Form.Item>
            </Col>
          </Row>

          <Divider />

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" icon={<Save />} loading={loading}>
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
