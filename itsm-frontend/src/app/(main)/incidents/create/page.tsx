'use client';

import React, { useState } from 'react';
import { Button, Card, Form, Input, Select, DatePicker, Upload, Space, Row, Col, message, Tabs, Typography, Divider } from 'antd';
import { ArrowLeftOutlined, UploadOutlined, PlusOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import { IncidentAPI } from '@/lib/api/incident-api';

const { Title, Text } = Typography;
const { TextArea } = Input;
const { Option } = Select;

export default function CreateIncidentPage() {
  const router = useRouter();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('basic');

  const handleSubmit = async (values: any) => {
    setLoading(true);
    try {
      await IncidentAPI.createIncident({
        title: values.title,
        description: values.description,
        priority: values.priority,
        category: values.category,
        impact: values.impact,
        urgency: values.urgency,
        assigned_to: values.assigned_to,
      });
      message.success('事件创建成功');
      router.push('/incidents');
    } catch (error) {
      console.error('Failed to create incident:', error);
      message.error('创建失败，请重试');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="p-6 min-h-screen bg-gray-50">
      {/* 返回按钮 */}
      <div className="mb-6">
        <Button
          type="link"
          icon={<ArrowLeftOutlined />}
          onClick={() => router.back()}
          style={{ paddingLeft: 0 }}
        >
          返回列表
        </Button>
      </div>

      {/* 页面标题 */}
      <div className="mb-6">
        <Title level={2} style={{ marginBottom: 4 }}>创建事件</Title>
        <Text type="secondary">填写事件信息以创建新的事件记录</Text>
      </div>

      <Row gutter={24}>
        {/* 左侧表单 */}
        <Col xs={24} lg={16}>
          <Card>
            <Form
              form={form}
              layout="vertical"
              onFinish={handleSubmit}
              initialValues={{
                priority: 'medium',
                impact: 'medium',
                urgency: 'medium',
              }}
            >
              <Tabs
                activeKey={activeTab}
                onChange={setActiveTab}
                items={[
                  {
                    key: 'basic',
                    label: '基本信息',
                    children: (
                      <>
                        <Form.Item
                          name="title"
                          label="事件标题"
                          rules={[{ required: true, message: '请输入事件标题' }]}
                        >
                          <Input placeholder="简要描述事件" maxLength={200} showCount />
                        </Form.Item>

                        <Form.Item
                          name="description"
                          label="详细描述"
                          rules={[{ required: true, message: '请输入事件描述' }]}
                        >
                          <TextArea
                            rows={6}
                            placeholder="详细描述事件的发生情况、影响范围、错误信息等"
                          />
                        </Form.Item>

                        <Row gutter={16}>
                          <Col span={8}>
                            <Form.Item
                              name="priority"
                              label="优先级"
                              rules={[{ required: true }]}
                            >
                              <Select>
                                <Option value="critical">紧急</Option>
                                <Option value="high">高</Option>
                                <Option value="medium">中</Option>
                                <Option value="low">低</Option>
                              </Select>
                            </Form.Item>
                          </Col>
                          <Col span={8}>
                            <Form.Item
                              name="category"
                              label="事件分类"
                            >
                              <Select placeholder="选择分类">
                                <Option value="hardware">硬件故障</Option>
                                <Option value="software">软件故障</Option>
                                <Option value="network">网络问题</Option>
                                <Option value="security">安全问题</Option>
                                <Option value="other">其他</Option>
                              </Select>
                            </Form.Item>
                          </Col>
                          <Col span={8}>
                            <Form.Item
                              name="assigned_to"
                              label="指派给"
                            >
                              <Select placeholder="选择负责人" allowClear>
                                <Option value={1}>张三</Option>
                                <Option value={2}>李四</Option>
                                <Option value={3}>王五</Option>
                              </Select>
                            </Form.Item>
                          </Col>
                        </Row>
                      </>
                    ),
                  },
                  {
                    key: 'impact',
                    label: '影响分析',
                    children: (
                      <>
                        <Row gutter={16}>
                          <Col span={12}>
                            <Form.Item
                              name="impact"
                              label="影响范围"
                              rules={[{ required: true }]}
                            >
                              <Select>
                                <Option value="critical">全局</Option>
                                <Option value="high">部门级</Option>
                                <Option value="medium">团队级</Option>
                                <Option value="low">个人</Option>
                              </Select>
                            </Form.Item>
                          </Col>
                          <Col span={12}>
                            <Form.Item
                              name="urgency"
                              label="紧急程度"
                              rules={[{ required: true }]}
                            >
                              <Select>
                                <Option value="critical">紧急</Option>
                                <Option value="high">高</Option>
                                <Option value="medium">中</Option>
                                <Option value="low">低</Option>
                              </Select>
                            </Form.Item>
                          </Col>
                        </Row>

                        <Form.Item
                          name="affected_systems"
                          label="受影响系统"
                        >
                          <Select mode="multiple" placeholder="选择受影响的系统" allowClear>
                            <Option value="web">Web网站</Option>
                            <Option value="api">API服务</Option>
                            <Option value="database">数据库</Option>
                            <Option value="network">网络</Option>
                            <Option value="storage">存储</Option>
                          </Select>
                        </Form.Item>

                        <Form.Item
                          name="root_cause"
                          label="初步原因分析"
                        >
                          <TextArea
                            rows={4}
                            placeholder="初步分析可能的原因"
                          />
                        </Form.Item>
                      </>
                    ),
                  },
                  {
                    key: 'attachment',
                    label: '附件',
                    children: (
                      <>
                        <Form.Item
                          name="attachments"
                          label="上传附件"
                          valuePropName="fileList"
                          getValueFromEvent={(e) => {
                            if (Array.isArray(e)) return e;
                            return e?.fileList;
                          }}
                        >
                          <Upload name="logo" action="/upload.do" listType="text">
                            <Button icon={<UploadOutlined />}>上传附件</Button>
                          </Upload>
                        </Form.Item>
                        <Text type="secondary">
                          支持上传图片、文档等附件，单个文件不超过10MB
                        </Text>
                      </>
                    ),
                  },
                ]}
              />

              <Divider />

              <Form.Item className="!mb-0">
                <Space>
                  <Button type="primary" htmlType="submit" loading={loading}>
                    提交
                  </Button>
                  <Button onClick={() => router.back()}>取消</Button>
                </Space>
              </Form.Item>
            </Form>
          </Card>
        </Col>

        {/* 右侧信息 */}
        <Col xs={24} lg={8}>
          <Card title="创建提示" className="mb-4">
            <Space direction="vertical" className="w-full">
              <div>
                <Text strong>优先级说明</Text>
                <ul className="mt-2 text-sm text-gray-600">
                  <li>🔴 紧急：系统完全不可用</li>
                  <li>🟠 高：核心功能受影响</li>
                  <li>🔵 中：非核心功能受影响</li>
                  <li>🟢 低：轻微问题</li>
                </ul>
              </div>
              <Divider className="!my-2" />
              <div>
                <Text strong>紧急联系方式</Text>
                <ul className="mt-2 text-sm text-gray-600">
                  <li>电话：400-XXX-XXXX</li>
                  <li>邮箱：support@example.com</li>
                </ul>
              </div>
            </Space>
          </Card>
        </Col>
      </Row>
    </div>
  );
}
