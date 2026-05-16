'use client';

import { useState, useEffect } from 'react';
import {
  Card,
  Form,
  Select,
  Input,
  Button,
  Table,
  message,
  Modal,
  Space,
  Tag,
  Alert,
  Spin,
} from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import MSPService from '@/services/msp-service';
import { UserApi } from '@/lib/api/user-api';
import type { MSPAllocation, CreateAllocationRequest } from '@/types/msp';

const { Option } = Select;

export default function MSPManagementPage() {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [hasAccess, setHasAccess] = useState(false);
  const [allocations, setAllocations] = useState<MSPAllocation[]>([]);
  const [customers, setCustomers] = useState<{ id: number; code: string; name: string }[]>([]);
  const [mspUsers, setMSPUsers] = useState<{ id: number; username: string }[]>([]);
  const [modalVisible, setModalVisible] = useState(false);
  const [selectedAllocation, setSelectedAllocation] = useState<MSPAllocation | null>(null);
  const [deallocateLoading, setDeallocateLoading] = useState(false);
  const [accessError, setAccessError] = useState<string | null>(null);

  useEffect(() => {
    checkAccess();
  }, []);

  const checkAccess = async () => {
    try {
      const { isMSP, isAdmin } = await MSPService.isMSPUser();
      if (isMSP || isAdmin) {
        setHasAccess(true);
        loadData();
      } else {
        setAccessError('您没有权限访问此页面');
      }
    } catch (err: any) {
      setAccessError(err.message || '检查权限失败');
    }
  };

  if (accessError) {
    return (
      <div style={{ padding: 24 }}>
        <Alert message={accessError} type="error" showIcon />
      </div>
    );
  }

  if (!hasAccess) {
    return (
      <div style={{ padding: 24, textAlign: 'center' }}>
        <Spin size="large" />
      </div>
    );
  }

  const loadData = async () => {
    setLoading(true);
    try {
      const [allocRes, custRes, usersRes] = await Promise.all([
        MSPService.getAllocations(),
        MSPService.getCustomers(),
        UserApi.getUsers({ page: 1, page_size: 200 }).catch(() => ({ users: [] })),
      ]);
      setAllocations(allocRes.allocations);
      setCustomers(custRes.customers);

      // 从用户 API 获取完整用户列表
      const allUsers = (usersRes as any)?.users || (usersRes as any)?.data?.users || [];
      if (allUsers.length > 0) {
        setMSPUsers(allUsers.map((u: any) => ({ id: u.id, username: u.username || u.name || '' })));
      } else {
        // 降级：从已有分配中提取用户
        const uniqueUsers = Array.from(
          new Map(
            allocRes.allocations.map(a => [
              a.msp_user_id,
              { id: a.msp_user_id, username: a.msp_username || '' },
            ])
          ).values()
        );
        setMSPUsers(uniqueUsers);
      }
    } catch (err: any) {
      message.error(err.message || '加载数据失败');
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = async (values: CreateAllocationRequest) => {
    try {
      await MSPService.createAllocation(values);
      message.success('分配创建成功');
      setModalVisible(false);
      form.resetFields();
      loadData();
    } catch (err: any) {
      message.error(err.message || '创建分配失败');
    }
  };

  const handleDeallocate = async (allocation: MSPAllocation) => {
    Modal.confirm({
      title: '确认解除分配',
      content: `确定要解除 ${allocation.msp_username} 对客户 ${allocation.customer_name} 的分配吗？`,
      onOk: async () => {
        setDeallocateLoading(true);
        try {
          await MSPService.deallocate(allocation.msp_user_id, allocation.customer_tenant_id);
          message.success('分配已解除');
          loadData();
        } catch (err: any) {
          message.error(err.message || '解除分配失败');
        } finally {
          setDeallocateLoading(false);
        }
      },
    });
  };

  const columns = [
    {
      title: 'MSP 员工',
      dataIndex: 'msp_username',
      key: 'msp_username',
    },
    {
      title: '客户',
      dataIndex: 'customer_name',
      key: 'customer_name',
    },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      render: (role: string) => {
        const config = {
          primary: { color: 'green', text: '主支持' },
          backup: { color: 'orange', text: '备份' },
          specialist: { color: 'blue', text: '专家' },
        };
        const cfg = config[role as keyof typeof config] || { color: 'default', text: role };
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    {
      title: '分配时间',
      dataIndex: 'assignedAt',
      key: 'assignedAt',
      render: (date: string) => new Date(date).toLocaleString('zh-CN'),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: MSPAllocation) => (
        <Space size="small">
          <Button size="small" onClick={() => handleDeallocate(record)} danger>
            解除
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Card
        title="MSP 分配管理"
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalVisible(true)}>
            新建分配
          </Button>
        }
      >
        <Alert
          message="使用说明"
          description="MSP Manager 可以为 MSP 员工分配客户租户。分配后，MSP 员工即可通过 X-Customer-Tenant-ID 头访问对应客户的工单。"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />

        <Table
          columns={columns}
          dataSource={allocations}
          rowKey="id"
          loading={loading}
          pagination={{ pageSize: 10 }}
        />
      </Card>

      <Modal
        title="创建 MSP 分配"
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          form.resetFields();
        }}
        footer={null}
      >
        <Form form={form} layout="vertical" onFinish={handleCreate}>
          <Form.Item
            name="msp_user_id"
            label="MSP 员工"
            rules={[{ required: true, message: '请选择 MSP 员工' }]}
          >
            <Select placeholder="选择 MSP 员工" showSearch optionFilterProp="children">
              {mspUsers.map(user => (
                <Option key={user.id} value={user.id}>
                  {user.username}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="customer_tenant_id"
            label="客户租户"
            rules={[{ required: true, message: '请选择客户租户' }]}
          >
            <Select placeholder="选择客户租户" showSearch optionFilterProp="children">
              {customers.map(cust => (
                <Option key={cust.id} value={cust.id}>
                  {cust.code} - {cust.name}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="role"
            label="分配角色"
            initialValue="primary"
            rules={[{ required: true, message: '请选择分配角色' }]}
          >
            <Select placeholder="选择角色">
              <Option value="primary">主支持 (Primary)</Option>
              <Option value="backup">备份 (Backup)</Option>
              <Option value="specialist">专家 (Specialist)</Option>
            </Select>
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                创建
              </Button>
              <Button onClick={() => setModalVisible(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
