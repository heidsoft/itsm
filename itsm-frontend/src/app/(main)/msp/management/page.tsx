'use client';

import { useState, useEffect } from 'react';
import {
  Card,
  Form,
  Select,
  Input,
  Button,
  Table,
  App,
  Modal,
  Space,
  Tag,
  Alert,
  Spin,
} from 'antd';
import { Plus } from 'lucide-react';
import MSPService from '@/lib/services/msp-service';
import { UserApi } from '@/lib/api/user-api';
import type { MSPAllocation, CreateAllocationRequest } from '@/types/msp';
import { BACKEND_MAX_PAGE_SIZE } from '@/lib/api/pagination';
import { formatDateTime } from '@/lib/formatters';
export default function MSPManagementPage() {
  const { message } = App.useApp();
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
        setAccessError('You do not have permission to access this page');
      }
    } catch (err: unknown) {
      setAccessError(err instanceof Error ? err.message : 'Failed to check permissions');
    }
  };

  if (accessError) {
    return (
      <div style={{ padding: 24 }}>
        <Alert title={accessError} type="error" showIcon />
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
        UserApi.getUsers({ page: 1, pageSize: BACKEND_MAX_PAGE_SIZE }).catch(() => ({ items: [] })),
      ]);
      setAllocations(allocRes.allocations);
      setCustomers(custRes.customers);

      const allUsers = usersRes.items;
      if (allUsers.length > 0) {
        setMSPUsers(allUsers.map((u) => ({ id: u.id, username: u.username || u.name || '' })));
      } else {
        // 降级：从已有分配中提取用户
        const uniqueUsers = Array.from(
          new Map(
            allocRes.allocations.map(a => [
              a.mspUserId,
              { id: a.mspUserId, username: a.mspUsername || '' },
            ])
          ).values()
        );
        setMSPUsers(uniqueUsers);
      }
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : 'Failed to load data');
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = async (values: CreateAllocationRequest) => {
    try {
      await MSPService.createAllocation(values);
      message.success('Allocation created successfully');
      setModalVisible(false);
      form.resetFields();
      loadData();
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : 'Failed to create allocation');
    }
  };

  const handleDeallocate = async (allocation: MSPAllocation) => {
    Modal.confirm({
      title: 'Confirm deallocation',
      content: `Confirm to cancel ${allocation.mspUsername} To the customer ${allocation.customerTenantName} Assignment?`,
      onOk: async () => {
        setDeallocateLoading(true);
        try {
          await MSPService.deallocate(allocation.mspUserId, allocation.customerTenantId);
          message.success('Allocation deallocated');
          loadData();
        } catch (err: unknown) {
          message.error(err instanceof Error ? err.message : 'Deallocation failed');
        } finally {
          setDeallocateLoading(false);
        }
      },
    });
  };

  const columns = [
    {
      title: 'MSP employee',
      dataIndex: 'mspUsername',
      key:'mspUsername',
    },
    {
      title: 'Customer',
      dataIndex: 'customerTenantName',
      key:'customerTenantName',
    },
    {
      title: 'Role',
      dataIndex: 'role',
      key: 'role',
      render: (role: string) => {
        const config = {
          primary: { color: 'green', text: 'Main support' },
          backup: { color: 'orange', text: 'Backup' },
          specialist: { color: 'blue', text: 'Expert' },
        };
        const cfg = config[role as keyof typeof config] || { color: 'default', text: role };
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    {
      title: 'Allocate time',
      dataIndex: 'assignedAt',
      key: 'assignedAt',
      render: (date: string) => formatDateTime(date),
    },
    {
      title: 'Actions',
      key: 'action',
      render: (_: unknown, record: MSPAllocation) => (
        <Space size="small">
          <Button size="small" onClick={() => handleDeallocate(record)} danger>

            Dismissed
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Card
        title="MSP Allocation Management"
        extra={
          <Button type="primary" icon={<Plus />} onClick={() => setModalVisible(true)}>

            New allocation
          </Button>
        }
      >
        <Alert
          title="MSP Manager can assign customer tenants to employees, and after assignment, access the corresponding work orders through the X-Customer-Tenant-ID header"
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
        title="Create MSP allocation"
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          form.resetFields();
        }}
        footer={null}
      >
        <Form form={form} layout="vertical" onFinish={handleCreate}>
          <Form.Item
            name="mspUserId"
            label="MSP employee"
            rules={[{ required: true, message: 'Please select an MSP employee' }]}
          >
            <Select placeholder="Select MSP Employee" showSearch optionFilterProp="children" options={mspUsers.map(user => ({ value: user.id, label: user.username }))} />
          </Form.Item>

          <Form.Item
            name="customerTenantId"
            label="Customer Tenant"
            rules={[{ required: true, message: 'Please select customer tenant' }]}
          >
            <Select placeholder="Select customer tenant" showSearch optionFilterProp="children" options={customers.map(cust => ({ value: cust.id, label: <>
                  {cust.code} - {cust.name}
                </> }))} />
          </Form.Item>

          <Form.Item
            name="role"
            label="Assign roles"
            initialValue="primary"
            rules={[{ required: true, message: 'Please select assignment role' }]}
          >
            <Select placeholder="Select role" options={[{ value: "primary", label: "Primary Support (Primary)" }, { value: "backup", label: "Backup" }, { value: "specialist", label: "Specialist" }]} />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">

                Create
              </Button>
              <Button onClick={() => setModalVisible(false)}>Cancel</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
