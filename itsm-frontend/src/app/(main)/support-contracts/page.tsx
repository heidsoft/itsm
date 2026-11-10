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
const statusLabel: Record<string, string> = { active: 'valid', terminated: 'Terminated', expired: 'Expired', pending: 'To be effective' };

export default function ContractsPage() {
  const { message } = App.useApp();
  const [contracts, setContracts] = useState<SupportContract[]>([]);
  const [customers, setCustomers] = useState<ServiceCustomer[]>([]);
  const [branches, setBranches] = useState<CustomerBranch[]>([]);
  const [sources, setSources] = useState<SourceOrganization[]>([]);
  const [refs, setRefs] = useState<ExternalContractReference[]>([]);
  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染空表格。
  const [listLoading, setListLoading] = useState(true);
  const [modal, setModal] = useState(false);
  const [refModal, setRefModal] = useState(false);
  const [editingContract, setEditingContract] = useState<SupportContract | null>(null);
  const [form] = Form.useForm();
  const [refForm] = Form.useForm();
  const [selectedCustomer, setSelectedCustomer] = useState<number | undefined>();

  const load = async () => {
    setListLoading(true);
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
    } catch { message.error("Loading failed"); } finally { setListLoading(false); }
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
        message.success('Contract updated');
      } else {
        await emailIntakeService.createContract({
          customerId: v.customerId,
          branchId: v.branchId,
          contractNumber: v.contractNumber,
          status: v.status || 'active',
          startAt: v.range?.[0]?.toISOString(),
          endAt: v.range?.[1]?.toISOString(),
        });
        message.success('Contract has been created');
      }
      setModal(false); form.resetFields(); setSelectedCustomer(undefined); setEditingContract(null);
      load();
    } catch { message.error('SaveFailed'); }
  };

  const terminate = async (id: number) => {
    await emailIntakeService.terminateContract(id);
    message.success('Contract has been terminated'); load();
  };

  const saveRef = async () => {
    const v = await refForm.validateFields();
    try {
      await emailIntakeService.createExternalContractReference(v);
      message.success('External contract mapping added');
      setRefModal(false); refForm.resetFields();
      load();
    } catch { message.error('SaveFailed'); }
  };

  const deleteRef = async (id: number) => {
    await emailIntakeService.deleteExternalContractReference(id);
    message.success('Mapping deleted'); load();
  };

  const filteredBranches = useMemo(
    () => branches.filter(b => b.customerId === selectedCustomer),
    [branches, selectedCustomer]
  );

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: 'Contract number', dataIndex: 'contractNumber' },
    { title: 'Customer', render: (_: unknown, r: SupportContract) => customerName(r.customerId) },
    { title: 'Branch', render: (_: unknown, r: SupportContract) => branchName(r.branchId) },
    { title: 'Status', dataIndex: 'status', render: (s: string) => <Tag color={statusColor[s]}>{statusLabel[s] || s}</Tag> },
    { title: 'Validity date', render: (_: unknown, r: SupportContract) =>
      r.startAt || r.endAt ? `${r.startAt ? dayjs(r.startAt).format('YYYY-MM-DD') : '?'} ~ ${r.endAt ? dayjs(r.endAt).format('YYYY-MM-DD') : '?'}` : 'Permanent' },
    {
      title: 'Actions', width: 120,
      render: (_: unknown, r: SupportContract) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => openEditModal(r)}>Edit</Button>
          {r.status === 'active' && (
            <Popconfirm title="Are you sure you want to terminate this contract?" onConfirm={() => terminate(r.id)}>
              <Button size="small" danger>Termination</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  const refColumns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: 'Source organization', render: (_: unknown, r: ExternalContractReference) => sourceName(r.sourceOrganizationId) },
    { title: 'External contract number', dataIndex: 'externalContractNumber' },
    { title: 'Internal Contract', render: (_: unknown, r: ExternalContractReference) => {
        const c = contracts.find(x => x.id === r.supportContractId);
        return c?.contractNumber || `#${r.supportContractId}`;
      }},
    {
      title: 'Actions', width: 80,
      render: (_: unknown, r: ExternalContractReference) => (
        <Popconfirm title="Delete this mapping?" onConfirm={() => deleteRef(r.id)}>
          <Button size="small" danger>Delete</Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <div className="space-y-4">
      <Card>
        <div className="flex items-center justify-between mb-4">
          <Title level={4} className="!mb-0">Support contract management</Title>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={load}>Refresh</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => { form.resetFields(); setSelectedCustomer(undefined); setEditingContract(null); setModal(true); }}>

              New contract
            </Button>
          </Space>
        </div>
        <Table rowKey="id" loading={listLoading} columns={columns} dataSource={contracts} pagination={{ pageSize: 20 }} />
      </Card>

      <Card title="External contract number mapping">
        <div className="mb-3">
          <Button icon={<PlusOutlined />} onClick={() => { refForm.resetFields(); setRefModal(true); }}>

            Add mapping
          </Button>
        </div>
        <Table rowKey="id" size="small" columns={refColumns} dataSource={refs} pagination={false} />
      </Card>

      <Modal title={editingContract ? 'Edit contract' : 'New contract'} open={modal} onOk={saveContract} onCancel={() => setModal(false)} width={520}>
        <Form form={form} layout="vertical">
          <Form.Item name="customerId" label="Customer" rules={[{ required: true }]}>
            <Select placeholder="Select customer" showSearch optionFilterProp="label"
              options={customers.map(c => ({ value: c.id, label: c.name }))}
              onChange={(v) => { setSelectedCustomer(v); form.setFieldValue('branchId', undefined); }}
            />
          </Form.Item>
          <Form.Item name="branchId" label="Branch">
            <Select placeholder="Select branch (optional)" allowClear
              options={filteredBranches.map(b => ({ value: b.id, label: b.name }))}
            />
          </Form.Item>
          <Form.Item name="contractNumber" label="Contract number" rules={[{ required: true }]}>
            <Input placeholder="Internal contract number" />
          </Form.Item>
          <Form.Item name="range" label="Validity date">
            <DatePicker.RangePicker className="w-full" />
          </Form.Item>
          <Form.Item name="status" label="Status" initialValue="active">
            <Select options={[
              { value: 'active', label: 'valid' },
              { value: 'pending', label: 'To be effective' },
              { value: 'terminated', label: 'Terminated' },
            ]} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="Add external contract number mapping" open={refModal} onOk={saveRef} onCancel={() => setRefModal(false)}>
        <Form form={refForm} layout="vertical">
          <Form.Item name="sourceOrganizationId" label="Source organization" rules={[{ required: true }]}>
            <Select placeholder="Select source organization" showSearch optionFilterProp="label"
              options={sources.map(s => ({ value: s.id, label: s.name }))}
            />
          </Form.Item>
          <Form.Item name="supportContractId" label="Internal Contract" rules={[{ required: true }]}>
            <Select placeholder="Select Internal Contract" showSearch optionFilterProp="label"
              options={contracts.map(c => ({ value: c.id, label: `${c.contractNumber} (${customerName(c.customerId)})` }))}
            />
          </Form.Item>
          <Form.Item name="externalContractNumber" label="External contract number" rules={[{ required: true }]}>
            <Input placeholder="Contract number used by the customer" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
