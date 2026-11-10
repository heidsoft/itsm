'use client';

import React, { useEffect, useState } from 'react';
import {
  App, Button, Card, Form, Input, Modal, Popconfirm, Space, Table, Tag, Typography,
} from 'antd';
import { PlusOutlined, ReloadOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import {
  emailIntakeService, type ServiceCustomer, type CustomerBranch,
} from '@/lib/services/emailIntakeService';

const { Title, Text } = Typography;

export default function CustomersPage() {
  const { message } = App.useApp();
  const [customers, setCustomers] = useState<ServiceCustomer[]>([]);
  const [branches, setBranches] = useState<CustomerBranch[]>([]);
  const [loading, setLoading] = useState(false);
  const [customerModal, setCustomerModal] = useState(false);
  const [branchModal, setBranchModal] = useState(false);
  const [editingCustomer, setEditingCustomer] = useState<ServiceCustomer | null>(null);
  const [selectedCustomer, setSelectedCustomer] = useState<ServiceCustomer | null>(null);
  const [customerForm] = Form.useForm();
  const [branchForm] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const [c, b] = await Promise.all([
        emailIntakeService.customers(),
        emailIntakeService.branches(),
      ]);
      setCustomers(c.items);
      setBranches(b.items);
    } catch {
      message.error('Failed to load data');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const saveCustomer = async () => {
    const values = await customerForm.validateFields();
    try {
      if (editingCustomer) {
        await emailIntakeService.updateCustomer(editingCustomer.id, { ...editingCustomer, ...values });
        message.success('Customer updated');
      } else {
        await emailIntakeService.createCustomer({ ...values, status: 'active' });
        message.success('Customer has been created');
      }
      setCustomerModal(false);
      customerForm.resetFields();
      setEditingCustomer(null);
      load();
    } catch { message.error('SaveFailed'); }
  };

  const disableCustomer = async (id: number) => {
    await emailIntakeService.disableCustomer(id);
    message.success('Customer deactivated');
    load();
  };

  const saveBranch = async () => {
    const values = await branchForm.validateFields();
    try {
      await emailIntakeService.createBranch({ ...values, customerId: selectedCustomer!.id, status: 'active' });
      message.success('Branch added');
      setBranchModal(false);
      branchForm.resetFields();
      load();
    } catch { message.error('SaveFailed'); }
  };

  const branchColumns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: 'Name', dataIndex: 'name' },
    { title: 'Status', dataIndex: 'status', render: (s: string) => <Tag color={s === 'active' ? 'green' : 'default'}>{s === 'active' ? 'Active' : 'Disable'}</Tag> },
  ];

  const expandedRowRender = (record: ServiceCustomer) => {
    const customerBranches = branches.filter(b => b.customerId === record.id);
    return (
      <div className="pl-8">
        <Space className="mb-2">
          <Button size="small" icon={<PlusOutlined />} onClick={() => { setSelectedCustomer(record); setBranchModal(true); }}>

            Add branch
          </Button>
        </Space>
        <Table
          rowKey="id" size="small" columns={branchColumns} dataSource={customerBranches}
          pagination={false}
        />
      </div>
    );
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: 'Customer name', dataIndex: 'name' },
    { title: 'Abbreviation', dataIndex: 'shortName' },
    { title: 'Alias', dataIndex: 'aliases', render: (v: string[]) => v?.length ? v.map(a => <Tag key={a}>{a}</Tag>) : null },
    { title: 'Number of branches', render: (_: unknown, r: ServiceCustomer) => branches.filter(b => b.customerId === r.id && b.status === 'active').length },
    { title: 'Status', dataIndex: 'status', render: (s: string) => <Tag color={s === 'active' ? 'green' : 'default'}>{s === 'active' ? 'Active' : 'Disable'}</Tag> },
    {
      title: 'Actions', width: 160, render: (_: unknown, r: ServiceCustomer) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => {
            setEditingCustomer(r);
            customerForm.setFieldsValue(r);
            setCustomerModal(true);
          }}>Edit</Button>
          {r.status === 'active' && (
            <Popconfirm title="Are you sure you want to deactivate this customer?" onConfirm={() => disableCustomer(r.id)}>
              <Button size="small" danger icon={<DeleteOutlined />}>Disable</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Card>
        <div className="flex items-center justify-between mb-4">
          <Title level={4} className="!mb-0">Customer data management</Title>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={load}>Refresh</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => {
              setEditingCustomer(null);
              customerForm.resetFields();
              setCustomerModal(true);
            }}>Add new customer</Button>
          </Space>
        </div>
        <Table
          rowKey="id" loading={loading} columns={columns} dataSource={customers}
          expandable={{ expandedRowRender }} pagination={{ pageSize: 20 }}
          expandRowByClick
        />
      </Card>

      <Modal
        title={editingCustomer ? 'Edit customer' : 'Add new customer'} open={customerModal}
        onOk={saveCustomer} onCancel={() => setCustomerModal(false)}
      >
        <Form form={customerForm} layout="vertical">
          <Form.Item name="name" label="Customer name" rules={[{ required: true }]}>
            <Input placeholder="For example: Beijing XX Technology Co., Ltd." />
          </Form.Item>
          <Form.Item name="shortName" label="Abbreviation">
            <Input placeholder="For example: XX Technology" />
          </Form.Item>
          <Form.Item name="aliases" label="Aliases (comma separated)">
            <Input placeholder="Alternative ways of writing customer name" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`Add branch - ${selectedCustomer?.name || ''}`} open={branchModal}
        onOk={saveBranch} onCancel={() => setBranchModal(false)}
      >
        <Form form={branchForm} layout="vertical">
          <Form.Item name="name" label="Branch name" rules={[{ required: true }]}>
            <Input placeholder="For example: Shanghai Branch" />
          </Form.Item>
          <Form.Item name="aliases" label="Aliases (comma separated)">
            <Input placeholder="Other ways to write branches" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
