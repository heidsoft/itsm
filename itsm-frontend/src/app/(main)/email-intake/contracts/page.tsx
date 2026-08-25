'use client';

import React, { useEffect, useMemo, useState } from 'react';
import {
  App, Button, Card, DatePicker, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, Typography,
} from 'antd';
import { EditOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import {
  emailIntakeService,
  type SupportContract, type ServiceCustomer, type CustomerBranch,
  type SourceOrganization, type ExternalContractReference,
} from '@/lib/services/emailIntakeService';

const { Title } = Typography;

const statusColor: Record<string, string> = { active: 'green', terminated: 'red', expired: 'orange', pending: 'blue' };
const statusLabel: Record<string, string> = { active: '有效', terminated: '已终止', expired: '已过期', pending: '待生效' };

export default function ContractsPage() {
  const { message } = App.useApp();
  const [contracts, setContracts] = useState<SupportContract[]>([]);
  const [customers, setCustomers] = useState<ServiceCustomer[]>([]);
  const [branches, setBranches] = useState<CustomerBranch[]>([]);
  const [sources, setSources] = useState<SourceOrganization[]>([]);
  const [refs, setRefs] = useState<ExternalContractReference[]>([]);
  const [loading, setLoading] = useState(false);
  const [modal, setModal] = useState(false);
  const [refModal, setRefModal] = useState(false);
  const [editingContract, setEditingContract] = useState<SupportContract | null>(null);
  const [form] = Form.useForm();
  const [refForm] = Form.useForm();
  const [selectedCustomer, setSelectedCustomer] = useState<number | undefined>();

  const load = async () => {
    setLoading(true);
    try {
      const [c, b, s, r, k] = await Promise.all([
        emailIntakeService.contracts(),
        emailIntakeService.branches(),
        emailIntakeService.customers(),
        emailIntakeService.sourceOrganizations(),
        emailIntakeService.externalContractReferences(),
      ]);
      setContracts(c.items);
      setBranches(b.items);
      setCustomers(s.items);
      setSources(r.items);
      setRefs(k.items);
    } catch { message.error('加载失败'); } finally { setLoading(false); }
  };
  useEffect(() => { load(); }, []);

  const customerName = (id: number) => customers.find(c => c.id === id)?.name || `#${id}`;
  const branchName = (id?: number) => id ? branches.find(b => b.id === id)?.name || `#${id}` : '-';
  const sourceName = (id: number) => sources.find(s => s.id === id)?.name || `#${id}`;

  const openEditModal = (contract: SupportContract) => {
    setEditingContract(contract);
    setSelectedCustomer(contract.customerId);
    const range = contract.startAt || contract.endAt
      ? [contract.startAt ? dayjs(contract.startAt) : null, contract.endAt ? dayjs(contract.endAt) : null]
      : undefined;
    form.setFieldsValue({
      customerId: contract.customerId,
      branchId: contract.branchId,
      contractNumber: contract.contractNumber,
      status: contract.status,
      range,
    });
    setModal(true);
  };

  const saveContract = async () => {
    const v = await form.validateFields();
    try {
      if (editingContract) {
        await emailIntakeService.updateContract(editingContract.id, {
          customerId: v.customerId,
          branchId: v.branchId,
          contractNumber: v.contractNumber,
          status: v.status || editingContract.status,
          startAt: v.range?.[0]?.toISOString(),
          endAt: v.range?.[1]?.toISOString(),
        });
        message.success('合同已更新');
      } else {
        await emailIntakeService.createContract({
          customerId: v.customerId,
          branchId: v.branchId,
          contractNumber: v.contractNumber,
          status: v.status || 'active',
          startAt: v.range?.[0]?.toISOString(),
          endAt: v.range?.[1]?.toISOString(),
        });
        message.success('合同已创建');
      }
      setModal(false); form.resetFields(); setSelectedCustomer(undefined); setEditingContract(null);
      load();
    } catch { message.error('保存失败'); }
  };

  const terminate = async (id: number) => {
    await emailIntakeService.terminateContract(id);
    message.success('合同已终止'); load();
  };

  const saveRef = async () => {
    const v = await refForm.validateFields();
    try {
      await emailIntakeService.createExternalContractReference(v);
      message.success('外部合同映射已添加');
      setRefModal(false); refForm.resetFields();
      load();
    } catch { message.error('保存失败'); }
  };

  const deleteRef = async (id: number) => {
    await emailIntakeService.deleteExternalContractReference(id);
    message.success('映射已删除'); load();
  };

  const filteredBranches = useMemo(
    () => branches.filter(b => b.customerId === selectedCustomer),
    [branches, selectedCustomer]
  );

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '合同号', dataIndex: 'contractNumber' },
    { title: '客户', render: (_: unknown, r: SupportContract) => customerName(r.customerId) },
    { title: '分支', render: (_: unknown, r: SupportContract) => branchName(r.branchId) },
    { title: '状态', dataIndex: 'status', render: (s: string) => <Tag color={statusColor[s]}>{statusLabel[s] || s}</Tag> },
    { title: '有效期', render: (_: unknown, r: SupportContract) =>
      r.startAt || r.endAt ? `${r.startAt ? dayjs(r.startAt).format('YYYY-MM-DD') : '?'} ~ ${r.endAt ? dayjs(r.endAt).format('YYYY-MM-DD') : '?'}` : '永久' },
    {
      title: '操作', width: 120,
      render: (_: unknown, r: SupportContract) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => openEditModal(r)}>编辑</Button>
          {r.status === 'active' && (
            <Popconfirm title="确定终止该合同？" onConfirm={() => terminate(r.id)}>
              <Button size="small" danger>终止</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  const refColumns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '来源组织', render: (_: unknown, r: ExternalContractReference) => sourceName(r.sourceOrganizationId) },
    { title: '外部合同号', dataIndex: 'externalContractNumber' },
    { title: '内部合同', render: (_: unknown, r: ExternalContractReference) => {
        const c = contracts.find(x => x.id === r.supportContractId);
        return c?.contractNumber || `#${r.supportContractId}`;
      }},
    {
      title: '操作', width: 80,
      render: (_: unknown, r: ExternalContractReference) => (
        <Popconfirm title="删除该映射？" onConfirm={() => deleteRef(r.id)}>
          <Button size="small" danger>删除</Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <div className="space-y-4">
      <Card>
        <div className="flex items-center justify-between mb-4">
          <Title level={4} className="!mb-0">支持合同管理</Title>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={load}>刷新</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => { form.resetFields(); setSelectedCustomer(undefined); setEditingContract(null); setModal(true); }}>
              新增合同
            </Button>
          </Space>
        </div>
        <Table rowKey="id" loading={loading} columns={columns} dataSource={contracts} pagination={{ pageSize: 20 }} />
      </Card>

      <Card title="外部合同号映射">
        <div className="mb-3">
          <Button icon={<PlusOutlined />} onClick={() => { refForm.resetFields(); setRefModal(true); }}>
            添加映射
          </Button>
        </div>
        <Table rowKey="id" size="small" columns={refColumns} dataSource={refs} pagination={false} />
      </Card>

      <Modal title={editingContract ? '编辑合同' : '新增合同'} open={modal} onOk={saveContract} onCancel={() => setModal(false)} width={520}>
        <Form form={form} layout="vertical">
          <Form.Item name="customerId" label="客户" rules={[{ required: true }]}>
            <Select placeholder="选择客户" showSearch optionFilterProp="label"
              options={customers.map(c => ({ value: c.id, label: c.name }))}
              onChange={(v) => { setSelectedCustomer(v); form.setFieldValue('branchId', undefined); }}
            />
          </Form.Item>
          <Form.Item name="branchId" label="分支">
            <Select placeholder="选择分支（可选）" allowClear
              options={filteredBranches.map(b => ({ value: b.id, label: b.name }))}
            />
          </Form.Item>
          <Form.Item name="contractNumber" label="合同号" rules={[{ required: true }]}>
            <Input placeholder="内部合同编号" />
          </Form.Item>
          <Form.Item name="range" label="有效期">
            <DatePicker.RangePicker className="w-full" />
          </Form.Item>
          <Form.Item name="status" label="状态" initialValue="active">
            <Select options={[
              { value: 'active', label: '有效' },
              { value: 'pending', label: '待生效' },
              { value: 'terminated', label: '已终止' },
            ]} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="添加外部合同号映射" open={refModal} onOk={saveRef} onCancel={() => setRefModal(false)}>
        <Form form={refForm} layout="vertical">
          <Form.Item name="sourceOrganizationId" label="来源组织" rules={[{ required: true }]}>
            <Select placeholder="选择来源组织" showSearch optionFilterProp="label"
              options={sources.map(s => ({ value: s.id, label: s.name }))}
            />
          </Form.Item>
          <Form.Item name="supportContractId" label="内部合同" rules={[{ required: true }]}>
            <Select placeholder="选择内部合同" showSearch optionFilterProp="label"
              options={contracts.map(c => ({ value: c.id, label: `${c.contractNumber} (${customerName(c.customerId)})` }))}
            />
          </Form.Item>
          <Form.Item name="externalContractNumber" label="外部合同号" rules={[{ required: true }]}>
            <Input placeholder="客户方使用的合同编号" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
