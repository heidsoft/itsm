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
      message.error('加载数据失败');
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
        message.success('客户已更新');
      } else {
        await emailIntakeService.createCustomer({ ...values, status: 'active' });
        message.success('客户已创建');
      }
      setCustomerModal(false);
      customerForm.resetFields();
      setEditingCustomer(null);
      load();
    } catch { message.error('保存失败'); }
  };

  const disableCustomer = async (id: number) => {
    await emailIntakeService.disableCustomer(id);
    message.success('客户已停用');
    load();
  };

  const saveBranch = async () => {
    const values = await branchForm.validateFields();
    try {
      await emailIntakeService.createBranch({ ...values, customerId: selectedCustomer!.id, status: 'active' });
      message.success('分支已添加');
      setBranchModal(false);
      branchForm.resetFields();
      load();
    } catch { message.error('保存失败'); }
  };

  const branchColumns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '名称', dataIndex: 'name' },
    { title: '状态', dataIndex: 'status', render: (s: string) => <Tag color={s === 'active' ? 'green' : 'default'}>{s === 'active' ? '活跃' : '停用'}</Tag> },
  ];

  const expandedRowRender = (record: ServiceCustomer) => {
    const customerBranches = branches.filter(b => b.customerId === record.id);
    return (
      <div className="pl-8">
        <Space className="mb-2">
          <Button size="small" icon={<PlusOutlined />} onClick={() => { setSelectedCustomer(record); setBranchModal(true); }}>
            添加分支
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
    { title: '客户名称', dataIndex: 'name' },
    { title: '简称', dataIndex: 'shortName' },
    { title: '别名', dataIndex: 'aliases', render: (v: string[]) => v?.length ? v.map(a => <Tag key={a}>{a}</Tag>) : null },
    { title: '分支数', render: (_: unknown, r: ServiceCustomer) => branches.filter(b => b.customerId === r.id && b.status === 'active').length },
    { title: '状态', dataIndex: 'status', render: (s: string) => <Tag color={s === 'active' ? 'green' : 'default'}>{s === 'active' ? '活跃' : '停用'}</Tag> },
    {
      title: '操作', width: 160, render: (_: unknown, r: ServiceCustomer) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => {
            setEditingCustomer(r);
            customerForm.setFieldsValue(r);
            setCustomerModal(true);
          }}>编辑</Button>
          {r.status === 'active' && (
            <Popconfirm title="确定停用该客户？" onConfirm={() => disableCustomer(r.id)}>
              <Button size="small" danger icon={<DeleteOutlined />}>停用</Button>
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
          <Title level={4} className="!mb-0">客户资料管理</Title>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={load}>刷新</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => {
              setEditingCustomer(null);
              customerForm.resetFields();
              setCustomerModal(true);
            }}>新增客户</Button>
          </Space>
        </div>
        <Table
          rowKey="id" loading={loading} columns={columns} dataSource={customers}
          expandable={{ expandedRowRender }} pagination={{ pageSize: 20 }}
          expandRowByClick
        />
      </Card>

      <Modal
        title={editingCustomer ? '编辑客户' : '新增客户'} open={customerModal}
        onOk={saveCustomer} onCancel={() => setCustomerModal(false)}
      >
        <Form form={customerForm} layout="vertical">
          <Form.Item name="name" label="客户名称" rules={[{ required: true }]}>
            <Input placeholder="例如：北京某某科技有限公司" />
          </Form.Item>
          <Form.Item name="shortName" label="简称">
            <Input placeholder="例如：某某科技" />
          </Form.Item>
          <Form.Item name="aliases" label="别名（逗号分隔）">
            <Input placeholder="客户名称的其他写法" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`添加分支 - ${selectedCustomer?.name || ''}`} open={branchModal}
        onOk={saveBranch} onCancel={() => setBranchModal(false)}
      >
        <Form form={branchForm} layout="vertical">
          <Form.Item name="name" label="分支名称" rules={[{ required: true }]}>
            <Input placeholder="例如：上海分公司" />
          </Form.Item>
          <Form.Item name="aliases" label="别名（逗号分隔）">
            <Input placeholder="分支的其他写法" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
