'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Typography,
  Modal,
  Form,
  Input,
  Select,
  Switch,
  message,
  Popconfirm,
  Tag,
  Row,
  Col,
  Statistic,
} from 'antd';
import {
  Plus,
  Edit,
  Trash2,
  GitMerge,
  RefreshCw,
} from 'lucide-react';
import type { ColumnsType } from 'antd/es/table';
import { httpClient } from '@/lib/api/http-client';

const { Title, Text } = Typography;
const { TextArea } = Input;

interface ApprovalWorkflow {
  id: number;
  name: string;
  description?: string;
  ticket_type?: string;
  priority?: string;
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
}

export default function ApprovalManagement() {
  const [workflows, setWorkflows] = useState<ApprovalWorkflow[]>([]);
  const [loading, setLoading] = useState(false);
  const [fetching, setFetching] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [selectedWorkflow, setSelectedWorkflow] = useState<ApprovalWorkflow | null>(null);
  const [form] = Form.useForm();
  const [stats, setStats] = useState({
    total: 0,
    active: 0,
    inactive: 0,
  });

  // 加载审批工作流数据
  const loadWorkflows = useCallback(async () => {
    setFetching(true);
    try {
      const response = await httpClient.get<{ items: ApprovalWorkflow[]; total: number }>(
        '/api/v1/approval-workflows',
        { page: 1, page_size: 100 }
      );
      setWorkflows(response.items || []);
      setStats({
        total: response.total || 0,
        active: (response.items || []).filter(w => w.is_active).length,
        inactive: (response.items || []).filter(w => !w.is_active).length,
      });
    } catch (error) {
      console.error('Failed to load workflows:', error);
      message.error('加载审批工作流失败');
    } finally {
      setFetching(false);
    }
  }, []);

  // 初始化加载
  useEffect(() => {
    loadWorkflows();
  }, [loadWorkflows]);

  // 处理保存
  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      const data = {
        ...values,
        is_active: values.is_active ?? true,
      };

      if (selectedWorkflow) {
        // 更新
        await httpClient.put(`/api/v1/approval-workflows/${selectedWorkflow.id}`, data);
        message.success('审批工作流更新成功');
      } else {
        // 创建
        await httpClient.post('/api/v1/approval-workflows', data);
        message.success('审批工作流创建成功');
      }

      setShowModal(false);
      form.resetFields();
      setSelectedWorkflow(null);
      loadWorkflows();
    } catch (error) {
      console.error('Failed to save workflow:', error);
      message.error('保存审批工作流失败');
    } finally {
      setLoading(false);
    }
  };

  // 处理删除
  const handleDelete = async (id: number) => {
    try {
      await httpClient.delete(`/api/v1/approval-workflows/${id}`);
      message.success('审批工作流删除成功');
      loadWorkflows();
    } catch (error) {
      console.error('Failed to delete workflow:', error);
      message.error('删除审批工作流失败');
    }
  };

  // 处理编辑
  const handleEdit = (record: ApprovalWorkflow) => {
    setSelectedWorkflow(record);
    form.setFieldsValue({
      name: record.name,
      description: record.description,
      ticket_type: record.ticket_type,
      priority: record.priority,
      is_active: record.is_active,
    });
    setShowModal(true);
  };

  // 处理切换状态
  const handleToggleStatus = async (record: ApprovalWorkflow) => {
    try {
      await httpClient.patch(`/api/v1/approval-workflows/${record.id}`, {
        is_active: !record.is_active,
      });
      message.success(`${record.is_active ? '停用' : '启用'}成功`);
      loadWorkflows();
    } catch (error) {
      console.error('Failed to toggle status:', error);
      message.error('状态切换失败');
    }
  };

  // 表格列定义
  const columns: ColumnsType<ApprovalWorkflow> = [
    {
      title: '工作流名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => (
        <Space>
          <GitMerge size={16} />
          <span className="font-medium">{text}</span>
        </Space>
      ),
    },
    {
      title: '工单类型',
      dataIndex: 'ticket_type',
      key: 'ticket_type',
      render: (text: string) => <Tag color="blue">{text || '-'}</Tag>,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      render: (text: string) => <Tag color="purple">{text || '-'}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'status',
      render: (isActive: boolean) => (
        <Tag color={isActive ? 'success' : 'default'}>
          {isActive ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '操作',
      key: 'actions',
      width: 200,
      render: (_: unknown, record: ApprovalWorkflow) => (
        <Space size="small">
          <Button
            type="text"
            icon={<Edit size={16} />}
            onClick={() => handleEdit(record)}
          />
          <Button
            type="text"
            onClick={() => handleToggleStatus(record)}
          >
            {record.is_active ? '停用' : '启用'}
          </Button>
          <Popconfirm
            title="确认删除"
            description={`确定要删除工作流"${record.name}"吗？`}
            onConfirm={() => handleDelete(record.id)}
            okText="确认"
            cancelText="取消"
          >
            <Button type="text" danger icon={<Trash2 size={16} />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <div className="mb-6">
        <Title level={2} className="!mb-2">
          <GitMerge className="mr-2" />
          审批管理
        </Title>
        <Text type="secondary">管理审批工作流和审批规则</Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={8}>
          <Card className="enterprise-card">
            <Statistic
              title="工作流总数"
              value={stats.total}
              prefix={<GitMerge />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <Card className="enterprise-card">
            <Statistic
              title="启用"
              value={stats.active}
              prefix={<GitMerge />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <Card className="enterprise-card">
            <Statistic
              title="禁用"
              value={stats.inactive}
              prefix={<GitMerge />}
            />
          </Card>
        </Col>
      </Row>

      {/* 操作栏 */}
      <Card className="mb-6">
        <Space>
          <Button
            type="primary"
            icon={<Plus size={16} />}
            onClick={() => {
              setSelectedWorkflow(null);
              form.resetFields();
              setShowModal(true);
            }}
          >
            新建工作流
          </Button>
          <Button
            icon={<RefreshCw size={16} />}
            onClick={() => loadWorkflows()}
            loading={fetching}
          >
            刷新
          </Button>
        </Space>
      </Card>

      {/* 工作流列表 */}
      <Card className="enterprise-card">
        <Table
          columns={columns}
          dataSource={workflows}
          rowKey="id"
          loading={fetching}
          pagination={{
            total: workflows.length,
            pageSize: 10,
            showSizeChanger: true,
            showTotal: total => `共 ${total} 条记录`,
          }}
          className="enterprise-table"
        />
      </Card>

      {/* 编辑模态框 */}
      <Modal
        title={
          <span>
            <Edit className="w-4 h-4 mr-2" />
            {selectedWorkflow ? '编辑工作流' : '新建工作流'}
          </span>
        }
        open={showModal}
        onOk={handleSave}
        onCancel={() => {
          setShowModal(false);
          setSelectedWorkflow(null);
          form.resetFields();
        }}
        width={600}
        confirmLoading={loading}
        okText="保存"
        cancelText="取消"
      >
        <Form form={form} layout="vertical" className="mt-4">
          <Form.Item
            label="工作流名称"
            name="name"
            rules={[{ required: true, message: '请输入工作流名称' }]}
          >
            <Input placeholder="请输入工作流名称" />
          </Form.Item>
          <Form.Item
            label="工单类型"
            name="ticket_type"
          >
            <Select
              placeholder="选择工单类型"
              allowClear
              options={[
                { label: '服务请求', value: 'service_request' },
                { label: '事件', value: 'incident' },
                { label: '问题', value: 'problem' },
                { label: '变更', value: 'change' },
              ]}
            />
          </Form.Item>
          <Form.Item
            label="优先级"
            name="priority"
          >
            <Select
              placeholder="选择优先级"
              allowClear
              options={[
                { label: '紧急', value: 'urgent' },
                { label: '高', value: 'high' },
                { label: '中', value: 'medium' },
                { label: '低', value: 'low' },
              ]}
            />
          </Form.Item>
          <Form.Item
            label="描述"
            name="description"
          >
            <TextArea rows={3} placeholder="请输入工作流描述" />
          </Form.Item>
          <Form.Item
            label="状态"
            name="is_active"
            valuePropName="checked"
          >
            <Switch checkedChildren="启用" unCheckedChildren="禁用" defaultChecked />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
