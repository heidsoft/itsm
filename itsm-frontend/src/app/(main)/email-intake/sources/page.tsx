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
  const [loading, setLoading] = useState(false);
  const [modal, setModal] = useState(false);
  const [editing, setEditing] = useState<SourceOrganization | null>(null);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const res = await emailIntakeService.sourceOrganizations();
      setItems(res.items);
    } catch { message.error('加载失败'); } finally { setLoading(false); }
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
        message.success('已更新');
      } else {
        await emailIntakeService.createSourceOrganization(payload);
        message.success('已创建');
      }
      setModal(false); form.resetFields(); setEditing(null);
      load();
    } catch { message.error('保存失败'); }
  };

  const disable = async (id: number) => {
    await emailIntakeService.disableSourceOrganization(id);
    message.success('已停用'); load();
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '组织名称', dataIndex: 'name' },
    { title: '邮箱地址', dataIndex: 'emailAddresses', render: (v: string[]) => v?.map(e => <Tag key={e}>{e}</Tag>) },
    { title: '邮箱域名', dataIndex: 'emailDomains', render: (v: string[]) => v?.map(d => <Tag key={d} color="blue">{d}</Tag>) },
    { title: '状态', dataIndex: 'status', render: (s: string) => <Tag color={s === 'active' ? 'green' : 'default'}>{s === 'active' ? '活跃' : '停用'}</Tag> },
    {
      title: '操作', width: 160, render: (_: unknown, r: SourceOrganization) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => {
            setEditing(r);
            form.setFieldsValue({ ...r, emailAddresses: r.emailAddresses?.join(', '), emailDomains: r.emailDomains?.join(', ') });
            setModal(true);
          }}>编辑</Button>
          {r.status === 'active' && (
            <Popconfirm title="确定停用？" onConfirm={() => disable(r.id)}>
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
          <Title level={4} className="!mb-0">来源组织管理</Title>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={load}>刷新</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => {
              setEditing(null); form.resetFields(); setModal(true);
            }}>新增组织</Button>
          </Space>
        </div>
        <Text type="secondary" className="block mb-3">
          来源组织用于通过发件人邮箱域名或地址自动识别客户身份。例如添加域名 example.com，所有来自 @example.com 的邮件都会自动关联。
        </Text>
        <Table rowKey="id" loading={loading} columns={columns} dataSource={items} pagination={false} />
      </Card>

      <Modal title={editing ? '编辑来源组织' : '新增来源组织'} open={modal} onOk={save} onCancel={() => setModal(false)}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="组织名称" rules={[{ required: true }]}>
            <Input placeholder="例如：某某科技IT部门" />
          </Form.Item>
          <Form.Item name="emailAddresses" label="邮箱地址（逗号分隔）">
            <Input.TextArea rows={2} placeholder="support@example.com, noc@example.com" />
          </Form.Item>
          <Form.Item name="emailDomains" label="邮箱域名（逗号分隔）">
            <Input placeholder="example.com, example.org" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
