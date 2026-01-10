'use client';

import React, { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Breadcrumb, Button, Card, Form, Input, Modal, Select, Space, Table, Tag, message } from 'antd';

import { CMDBApi } from '@/modules/cmdb/api';
import type { CloudAccount } from '@/modules/cmdb/types';

const { Option } = Select;

const providerOptions = [
  { value: 'aliyun', label: '阿里云' },
  { value: 'huawei', label: '华为云' },
  { value: 'tencent', label: '腾讯云' },
  { value: 'azure', label: 'Azure' },
  { value: 'onprem', label: '私有云' },
];

export default function CloudAccountPage() {
  const router = useRouter();
  const [form] = Form.useForm();
  const [createForm] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<CloudAccount[]>([]);
  const [createOpen, setCreateOpen] = useState(false);

  const loadData = async () => {
    setLoading(true);
    try {
      const values = form.getFieldsValue();
      const list = await CMDBApi.getCloudAccounts(values.provider);
      setData(list || []);
    } catch (error) {
      message.error('加载云账号失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleCreate = async () => {
    try {
      const values = await createForm.validateFields();
      await CMDBApi.createCloudAccount(values);
      message.success('云账号已创建');
      setCreateOpen(false);
      createForm.resetFields();
      loadData();
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message || '创建失败');
      }
    }
  };

  const columns = [
    {
      title: '厂商',
      dataIndex: 'provider',
      width: 110,
      render: (value: string) => providerOptions.find(p => p.value === value)?.label || value,
    },
    {
      title: '账号ID',
      dataIndex: 'account_id',
      width: 200,
    },
    {
      title: '账号名称',
      dataIndex: 'account_name',
      width: 180,
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      width: 80,
      render: (value: boolean) => (
        <Tag color={value ? 'green' : 'default'}>{value ? '启用' : '停用'}</Tag>
      ),
    },
  ];

  return (
    <Card variant="borderless">
      <Breadcrumb
        style={{ marginBottom: 16 }}
        items={[
          { title: '首页' },
          { title: '配置管理' },
          { title: <a onClick={() => router.push('/cmdb')}>配置项列表</a> },
          { title: '云账号管理' },
        ]}
      />

      <Form form={form} layout="inline" style={{ marginBottom: 24 }}>
        <Form.Item name="provider">
          <Select placeholder="云厂商" style={{ width: 160 }} allowClear>
            {providerOptions.map(item => (
              <Option key={item.value} value={item.value}>
                {item.label}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item>
          <Space>
            <Button onClick={loadData}>查询</Button>
            <Button type="primary" onClick={() => setCreateOpen(true)}>新增云账号</Button>
          </Space>
        </Form.Item>
      </Form>

      <Table
        rowKey="id"
        loading={loading}
        dataSource={data}
        columns={columns as any}
        pagination={{ pageSize: 10 }}
      />

      <Modal
        title="新增云账号"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={handleCreate}
        destroyOnClose
      >
        <Form form={createForm} layout="vertical">
          <Form.Item name="provider" label="云厂商" rules={[{ required: true, message: '请选择云厂商' }]}>
            <Select placeholder="请选择云厂商">
              {providerOptions.map(item => (
                <Option key={item.value} value={item.value}>
                  {item.label}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="account_id" label="账号ID" rules={[{ required: true, message: '请输入账号ID' }]}>
            <Input placeholder="例如 1234567890123456" />
          </Form.Item>
          <Form.Item name="account_name" label="账号名称" rules={[{ required: true, message: '请输入账号名称' }]}>
            <Input placeholder="例如 生产账号" />
          </Form.Item>
          <Form.Item name="credential_ref" label="凭据引用">
            <Input placeholder="例如 aliyun-prod-credential" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
