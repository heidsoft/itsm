'use client';

import React, { useEffect, useState } from 'react';
import {
  App, Button, Card, Form, Input, Modal, Popconfirm, Space, Table, Tag, Typography,
} from 'antd';
import { PlusOutlined, ReloadOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import {
  emailIntakeService, type SourceOrganization,
} from '@/lib/services/emailIntakeService';

const { Title, Text } = Typography;

export default function SourcesPage() {
  const { message } = App.useApp();
  const [items, setItems] = useState<SourceOrganization[]>([]);
  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染
  // 空表格，再被真实数据覆盖。
  const [listLoading, setListLoading] = useState(true);
  const [modal, setModal] = useState(false);
  const [editing, setEditing] = useState<SourceOrganization | null>(null);
  const [form] = Form.useForm();

  const load = async () => {
    setListLoading(true);
    try {
      const res = await emailIntakeService.sourceOrganizations();
      setItems(res.items);
    } catch { message.error("Loading failed"); } finally { setListLoading(false); }
  };
  useEffect(() => { load(); }, []);

  const save = async () => {
    const v = await form.validateFields();
    const payload = {
      ...v,
      emailAddresses: (v.emailAddresses as string || '').split(/[,，\n]/).map(s => s.trim()).filter(Boolean),
      emailDomains: (v.emailDomains as string || '').split(/[,，\n]/).map(s => s.trim().replace('@', '')).filter(Boolean),
      status: 'active',
    };
    try {
      if (editing) {
        await emailIntakeService.updateSourceOrganization(editing.id, payload);
        message.success('Updated');
      } else {
        await emailIntakeService.createSourceOrganization(payload);
        message.success('Created');
      }
      setModal(false); form.resetFields(); setEditing(null);
      load();
    } catch { message.error('SaveFailed'); }
  };

  const disable = async (id: number) => {
    await emailIntakeService.disableSourceOrganization(id);
    message.success('Disabled'); load();
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: 'Organization name', dataIndex: 'name' },
    { title: 'Email address', dataIndex: 'emailAddresses', render: (v: string[]) => v?.map(e => <Tag key={e}>{e}</Tag>) },
    { title: 'Email domain name', dataIndex: 'emailDomains', render: (v: string[]) => v?.map(d => <Tag key={d} color="blue">{d}</Tag>) },
    { title: 'Status', dataIndex: 'status', render: (s: string) => <Tag color={s === 'active' ? 'green' : 'default'}>{s === 'active' ? 'Active' : 'Disable'}</Tag> },
    {
      title: 'Actions', width: 160, render: (_: unknown, r: SourceOrganization) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => {
            setEditing(r);
            form.setFieldsValue({ ...r, emailAddresses: r.emailAddresses?.join(', '), emailDomains: r.emailDomains?.join(', ') });
            setModal(true);
          }}>Edit</Button>
          {r.status === 'active' && (
            <Popconfirm title="Are you sure to deactivate?" onConfirm={() => disable(r.id)}>
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
          <Title level={4} className="!mb-0">Source organization management</Title>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={load}>Refresh</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => {
              setEditing(null); form.resetFields(); setModal(true);
            }}>Add new organization</Button>
          </Space>
        </div>
        <Text type="secondary" className="block mb-3">

          Used by the source organization to automatically identify customers through the sender's email domain name or address. For example, if you add the domain name example.com, all emails from @example.com will be automatically associated.
        </Text>
        <Table rowKey="id" loading={listLoading} columns={columns} dataSource={items} pagination={false} />
      </Card>

      <Modal title={editing ? 'Edit source organization' : 'Add new source organization'} open={modal} onOk={save} onCancel={() => setModal(false)}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Organization name" rules={[{ required: true }]}>
            <Input placeholder="For example: XX technology IT department" />
          </Form.Item>
          <Form.Item name="emailAddresses" label="Email addresses (comma separated)">
            <Input.TextArea rows={2} placeholder="support@example.com, noc@example.com" />
          </Form.Item>
          <Form.Item name="emailDomains" label="Email domain names (comma separated)">
            <Input placeholder="example.com, example.org" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
