'use client';

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Alert,
  App,
  Button,
  Card,
  DatePicker,
  Descriptions,
  Drawer,
  Form,
  Input,
  InputNumber,
  Modal,
  Progress,
  Select,
  Space,
  Table,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import { RefreshCw, ShieldAlert } from 'lucide-react';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';

import {
  emailIntakeService,
  type EmailConversation,
  type EmailConversationDetail,
  type ExternalContractReference,
  type ServiceCustomer,
  type SourceOrganization,
  type SupportContract,
  type OnCallSchedule,
} from '@/lib/services/emailIntakeService';
import { UserApi } from '@/lib/api/user-api';
import { BACKEND_MAX_PAGE_SIZE } from '@/lib/api/pagination';
import { formatDateTime } from '@/lib/formatters';

interface User {
  id: number;
  name: string;
  email?: string;
}

const { Title, Text, Paragraph } = Typography;

const statusMeta: Record<string, { color: string; label: string }> = {
  PROCESSING: { color: 'processing', label: 'In Progress' },
  NEED_INFORMATION: { color: 'warning', label: 'Missing data' },
  WAITING_CUSTOMER: { color: 'warning', label: 'Waiting for customers' },
  MANUAL_REVIEW: { color: 'orange', label: 'Manual review' },
  VERIFIED: { color: 'success', label: 'Verification passed' },
  INCIDENT_CREATED: { color: 'blue', label: 'Event created' },
  REJECTED: { color: 'error', label: 'Rejected' },
};

// 客户状态：后端 service_customer.status（启用/停用，停用由 DisableCustomer 置为 inactive）
const customerStatusMeta: Record<string, { color: string; label: string }> = {
  active: { color: 'green', label: 'Enabled' },
  inactive: { color: 'default', label: 'Disabled' },
};

// 支持合同状态：后端 support_contract.status（active/terminated/expired/pending）
const contractStatusMeta: Record<string, { color: string; label: string }> = {
  active: { color: 'success', label: 'In effect' },
  terminated: { color: 'error', label: 'Terminated' },
  expired: { color: 'warning', label: 'Expired' },
  pending: { color: 'default', label: 'To be effective' },
};

const formatTime = (value?: string): string => formatDateTime(value) || '-';

export default function EmailIntakePage() {
  const { message, modal } = App.useApp();
  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染空表格。
  const [listLoading, setListLoading] = useState(true);
  const [status, setStatus] = useState<string>();
  const [conversations, setConversations] = useState<EmailConversation[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [detail, setDetail] = useState<EmailConversationDetail>();
  const [detailLoading, setDetailLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);
  const detailRequest = useRef(0);
  const listRequest = useRef(0);
  const [customers, setCustomers] = useState<ServiceCustomer[]>([]);
  const [contracts, setContracts] = useState<SupportContract[]>([]);
  const [sources, setSources] = useState<SourceOrganization[]>([]);
  const [externalReferences, setExternalReferences] = useState<ExternalContractReference[]>([]);
  const [schedules, setSchedules] = useState<OnCallSchedule[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [entityModal, setEntityModal] = useState<
    'customer' | 'branch' | 'contract' | 'source' | 'externalReference' | 'schedule' | 'shift'
  >();
  const [overrideOpen, setOverrideOpen] = useState(false);
  const [editingCustomer, setEditingCustomer] = useState<ServiceCustomer>();
  const [editingContract, setEditingContract] = useState<SupportContract>();
  const [form] = Form.useForm();
  const [correctionForm] = Form.useForm();
  const [overrideForm] = Form.useForm();

  const loadConversations = useCallback(async () => {
    const requestId = ++listRequest.current;
    setListLoading(true);
    try {
      const response = await emailIntakeService.conversations({ page, pageSize, status });
      if (requestId !== listRequest.current) return;
      setConversations(response.items);
      setTotal(response.total);
    } catch (error) {
      if (requestId === listRequest.current) {
        message.error(`Failed to load mail processing queue:${(error as Error).message}`);
      }
    } finally {
      if (requestId === listRequest.current) setListLoading(false);
    }
  }, [message, page, pageSize, status]);

  const loadMasterData = useCallback(async () => {
    try {
      const [customerResult, contractResult, sourceResult, externalReferenceResult, scheduleResult, userResult] =
        await Promise.allSettled([
          emailIntakeService.customers(),
          emailIntakeService.contracts(),
          emailIntakeService.sourceOrganizations(),
          emailIntakeService.externalContractReferences(),
          emailIntakeService.schedules(),
          UserApi.getUsers({ pageSize: BACKEND_MAX_PAGE_SIZE }),
        ]);
      if (customerResult.status === 'fulfilled') setCustomers(customerResult.value.items ?? []);
      if (contractResult.status === 'fulfilled') setContracts(contractResult.value.items ?? []);
      if (sourceResult.status === 'fulfilled') setSources(sourceResult.value.items ?? []);
      if (externalReferenceResult.status === 'fulfilled')
        setExternalReferences(externalReferenceResult.value.items ?? []);
      if (scheduleResult.status === 'fulfilled') setSchedules(scheduleResult.value.items ?? []);
      if (userResult.status === 'fulfilled')
        setUsers(
          (userResult.value.items || []).map((u) => ({
            id: u.id,
            name: u.name || u.username,
            email: u.email,
          }))
        );
    } catch (error) {
      message.error(`Failed to load master data:${(error as Error).message}`);
    }
  }, [message]);

  useEffect(() => {
    loadConversations();
  }, [loadConversations]);
  useEffect(() => {
    loadMasterData();
  }, [loadMasterData]);

  const openDetail = async (record: EmailConversation) => {
    const requestId = ++detailRequest.current;
    setDetailLoading(true);
    try {
      const result = await emailIntakeService.conversation(record.id);
      if (requestId !== detailRequest.current) return;
      setDetail(result);
      correctionForm.setFieldsValue(result.canonicalData ?? {});
    } catch (error) {
      message.error(`Failed to load session details:${(error as Error).message}`);
    } finally {
      if (requestId === detailRequest.current) setDetailLoading(false);
    }
  };

  const runAction = async (action: 'revalidate' | 'confirm' | 'reject' | 'retry') => {
    if (!detail) return;
    setActionLoading(true);
    try {
      const updated = await emailIntakeService[action](detail.id, detail.version);
      message.success('Operation completed');
      await Promise.all([loadConversations(), openDetail(updated)]);
    } catch (error) {
      message.error(error instanceof Error && error.message ? error.message : 'Operation failed');
    } finally {
      setActionLoading(false);
    }
  };

  const saveCorrections = async () => {
    if (!detail) return;
    setActionLoading(true);
    try {
      const fields = await correctionForm.validateFields();
      const updated = await emailIntakeService.correct(detail.id, detail.version, fields);
      message.success('Correction saved and rechecked');
      await Promise.all([loadConversations(), openDetail(updated)]);
    } catch (error) {
      if (error instanceof Error) message.error(error.message ? error.message : 'Failed to save fixes');
    } finally {
      setActionLoading(false);
    }
  };

  const submitOverride = async () => {
    if (!detail) return;
    setActionLoading(true);
    try {
      const values = await overrideForm.validateFields();
      const updated = await emailIntakeService.override(detail.id, detail.version, values.reason);
      setOverrideOpen(false);
      overrideForm.resetFields();
      message.success('Forced billing reason has been recorded and event created');
      await Promise.all([loadConversations(), openDetail(updated)]);
    } catch (error) {
      if (error instanceof Error) message.error(error.message ? error.message : 'Forced order opening failed');
    } finally {
      setActionLoading(false);
    }
  };

  const submitEntity = async () => {
    if (!entityModal) return;
    try {
      const values = await form.validateFields();
      if (entityModal === 'customer') {
        const payload = {
          name: values.name,
          shortName: values.shortName ?? '',
          aliases: values.aliases ?? [],
          historicalNames: values.historicalNames ?? [],
          status: 'active',
        };
        if (editingCustomer) await emailIntakeService.updateCustomer(editingCustomer.id, payload);
        else await emailIntakeService.createCustomer(payload);
      } else if (entityModal === 'branch') {
        await emailIntakeService.createBranch({
          customerId: values.customerId,
          name: values.name,
          aliases: values.aliases ?? [],
          status: 'active',
        });
      } else if (entityModal === 'contract') {
        const period = values.period as [Dayjs | null, Dayjs | null] | undefined;
        const payload = {
          customerId: values.customerId,
          branchId: values.branchId,
          contractNumber: values.contractNumber,
          status: values.status,
          startAt: period?.[0]?.toISOString() ?? editingContract?.startAt,
          endAt: period?.[1]?.toISOString() ?? editingContract?.endAt,
        };
        if (editingContract) await emailIntakeService.updateContract(editingContract.id, payload);
        else await emailIntakeService.createContract(payload);
      } else if (entityModal === 'source') {
        await emailIntakeService.createSourceOrganization({
          name: values.name,
          emailAddresses: values.emailAddresses ?? [],
          emailDomains: values.emailDomains ?? [],
          status: 'active',
        });
      } else if (entityModal === 'externalReference') {
        await emailIntakeService.createExternalContractReference({
          sourceOrganizationId: values.sourceOrganizationId,
          supportContractId: values.supportContractId,
          externalContractNumber: values.externalContractNumber,
        });
      } else if (entityModal === 'schedule') {
        await emailIntakeService.createSchedule({
          groupId: values.groupId,
          name: values.name,
          timezone: values.timezone,
          status: 'active',
        });
      } else {
        await emailIntakeService.createShift({
          scheduleId: values.scheduleId,
          userId: values.userId,
          startAt: values.period[0].toISOString(),
          endAt: values.period[1].toISOString(),
        });
      }
      message.success('SaveSuccess');
      setEntityModal(undefined);
      setEditingCustomer(undefined);
      setEditingContract(undefined);
      form.resetFields();
      await loadMasterData();
    } catch (error) {
      if (error instanceof Error) message.error(error.message ? error.message : 'SaveFailed');
    }
  };

  const columns = useMemo(
    () => [
      {
        title: 'Status',
        dataIndex: 'status',
        width: 110,
        render: (value: string) => (
          <Tag color={statusMeta[value]?.color}>{statusMeta[value]?.label ?? value}</Tag>
        ),
      },
      {
        title: 'Customer',
        dataIndex: 'customerName',
        ellipsis: true,
        render: (value: string) => value || '-',
      },
      {
        title: 'Branch',
        dataIndex: 'branchName',
        width: 140,
        render: (value: string) => value || '-',
      },
      {
        title: 'Contract',
        dataIndex: 'contractNumber',
        width: 150,
        render: (value: string) => value || '-',
      },
      {
        title: 'Credibility',
        dataIndex: 'confidence',
        width: 130,
        render: (value: number) => (
          <Progress percent={Math.round((value ?? 0) * 100)} size='small' />
        ),
      },
      {
        title: 'Incident',
        dataIndex: 'incidentNumber',
        width: 150,
        render: (value: string, record: EmailConversation) =>
          value ? <a href={`/incidents/${record.incidentId}`}>{value}</a> : '-',
      },
      { title: 'Last email', dataIndex: 'lastMessageAt', width: 180, render: formatTime },
      {
        title: 'Actions',
        width: 80,
        fixed: 'right' as const,
        render: (_: unknown, record: EmailConversation) => (
          <Button type='link' onClick={() => openDetail(record)}>

            Process
          </Button>
        ),
      },
    ],
    []
  );

  const queue = (
    <Card>
      <Space className='mb-4' wrap>
        <Select
          allowClear
          placeholder="All Status"
          value={status}
          onChange={(value?: string) => {
            setStatus(value);
            setPage(1);
          }}
          style={{ width: 180 }}
          options={Object.entries(statusMeta).map(([value, meta]) => ({
            value,
            label: meta.label,
          }))}
        />
        <Button icon={<RefreshCw size={14} />} onClick={loadConversations}>

          Refresh
        </Button>
      </Space>
      <Table
        rowKey='id'
        size='middle'
        loading={listLoading}
        columns={columns}
        dataSource={conversations}
        scroll={{ x: 1100 }}
        locale={{ emptyText: 'No email fault report record yet' }}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          showTotal: (value) => `Total ${value} conversations`,
          onChange: (nextPage, nextSize) => {
            setPage(nextPage);
            setPageSize(nextSize);
          },
        }}
      />
    </Card>
  );

  const masterData = (
    <div className='space-y-4'>
      <Space wrap>
        <Button type='primary' onClick={() => setEntityModal('customer')}>

          Add new customer
        </Button>
        <Button onClick={() => setEntityModal('branch')}>Add new branch</Button>
        <Button onClick={() => setEntityModal('contract')}>Add support contract</Button>
        <Button onClick={() => setEntityModal('source')}>Add new source organization</Button>
        <Button onClick={() => setEntityModal('externalReference')}>Add external contract mapping</Button>
      </Space>
      <Card title="Serving Customers">
        <Table
          rowKey='id'
          size='small'
          pagination={false}
          dataSource={customers}
          columns={[
            { title: 'Name', dataIndex: 'name' },
            { title: 'Abbreviation', dataIndex: 'shortName' },
            { title: 'Alias', dataIndex: 'aliases', render: (v: string[]) => v?.join('、') || '-' },
            {
              title: 'Status',
              dataIndex: 'status',
              render: (v: string) => {
                const meta = customerStatusMeta[v];
                return <Tag color={meta?.color ?? 'default'}>{meta?.label ?? (v || '-')}</Tag>;
              },
            },
            {
              title: 'Actions',
              render: (_: unknown, record: ServiceCustomer) => (
                <Space>
                  <Button
                    type='link'
                    onClick={() => {
                      setEditingCustomer(record);
                      setEntityModal('customer');
                      form.setFieldsValue(record);
                    }}
                  >

                    Edit
                  </Button>
                  <Button
                    type='link'
                    danger
                    disabled={record.status !== 'active'}
                    onClick={() =>
                      modal.confirm({
                        title: 'Deactivate this customer?',
                        onOk: async () => {
                          await emailIntakeService.disableCustomer(record.id);
                          await loadMasterData();
                        },
                      })
                    }
                  >

                    Disable
                  </Button>
                </Space>
              ),
            },
          ]}
        />
      </Card>
      <Card title="Support Contract">
        <Table
          rowKey='id'
          size='small'
          pagination={false}
          dataSource={contracts}
          columns={[
            { title: 'Contract number', dataIndex: 'contractNumber' },
            {
              title: 'Customer',
              dataIndex: 'customerId',
              render: (id: number) => customers.find(item => item.id === id)?.name ?? id,
            },
            {
              title: 'Status',
              dataIndex: 'status',
              render: (v: string) => {
                const meta = contractStatusMeta[v];
                return <Tag color={meta?.color ?? 'default'}>{meta?.label ?? (v || '-')}</Tag>;
              },
            },
            {
              title: 'Validity date',
              render: (_: unknown, record: SupportContract) =>
                `${formatTime(record.startAt)} — ${formatTime(record.endAt)}`,
            },
            {
              title: 'Actions',
              render: (_: unknown, record: SupportContract) => (
                <Space>
                  <Button
                    type='link'
                    onClick={() => {
                      setEditingContract(record);
                      setEntityModal('contract');
                      form.setFieldsValue({
                        ...record,
                        period:
                          record.startAt || record.endAt
                            ? [
                                record.startAt ? dayjs(record.startAt) : null,
                                record.endAt ? dayjs(record.endAt) : null,
                              ]
                            : undefined,
                      });
                    }}
                  >

                    Edit
                  </Button>
                  <Button
                    type='link'
                    danger
                    disabled={record.status !== 'active'}
                    onClick={() =>
                      modal.confirm({
                        title: 'Terminate this contract?',
                        content: 'Automatic order opening will be blocked immediately after termination.',
                        onOk: async () => {
                          await emailIntakeService.terminateContract(record.id);
                          await loadMasterData();
                        },
                      })
                    }
                  >

                    Termination
                  </Button>
                </Space>
              ),
            },
          ]}
        />
      </Card>
      <Card title="Source organization">
        <Table
          rowKey='id'
          size='small'
          pagination={false}
          dataSource={sources}
          columns={[
            { title: 'Organization', dataIndex: 'name' },
            {
              title: 'Email',
              dataIndex: 'emailAddresses',
              render: (v: string[]) => v?.join('、') || '-',
            },
            {
              title: 'Domain',
              dataIndex: 'emailDomains',
              render: (v: string[]) => v?.join('、') || '-',
            },
          ]}
        />
      </Card>
      <Card title="External contract number mapping">
        <Table
          rowKey='id'
          size='small'
          pagination={false}
          dataSource={externalReferences}
          columns={[
            {
              title: 'Source organization',
              dataIndex: 'sourceOrganizationId',
              render: (id: number) => sources.find(item => item.id === id)?.name ?? id,
            },
            { title: 'External contract number', dataIndex: 'externalContractNumber' },
            {
              title: 'Internal Support Contract',
              dataIndex: 'supportContractId',
              render: (id: number) => contracts.find(item => item.id === id)?.contractNumber ?? id,
            },
            {
              title: 'Actions',
              render: (_: unknown, record: ExternalContractReference) => (
                <Button
                  type='link'
                  danger
                  onClick={() =>
                    modal.confirm({
                      title: 'Delete this mapping?',
                      onOk: async () => {
                        await emailIntakeService.deleteExternalContractReference(record.id);
                        await loadMasterData();
                      },
                    })
                  }
                >

                  Delete
                </Button>
              ),
            },
          ]}
        />
      </Card>
    </div>
  );

  return (
    <div className='space-y-5'>
      <div>
        <Title level={2} className='!mb-1'>

          Customers and Contracts
        </Title>
        <Text type='secondary'>

          AI email fault reporting workbench: customer, support contract and source organization master data, AI is responsible for extraction, customer and contract rules are responsible for authorizing billing.
        </Text>
      </div>
      <Alert
        type='info'
        showIcon
        title="Automatic order opening is turned off by default. It is recommended to use observation mode to verify before enabling it."
      />
      <Tabs
        items={[
          { key: 'queue', label: 'NOC processing queue', children: queue },
          { key: 'master', label: 'Customer and Contract Master Data', children: masterData },
          {
            key: 'oncall',
            label: 'Duty configuration',
            children: (
              <Card>
                <Space>
                  <Button type='primary' onClick={() => setEntityModal('schedule')}>

                    Add new shift schedule
                  </Button>
                  <Button onClick={() => setEntityModal('shift')}>Add new flight</Button>
                </Space>
                <Paragraph type='secondary' className='!mt-4'>

                  PoC uses explicit start and end time shifts; the shift user must be an enabled member of the corresponding support group.
                </Paragraph>
              </Card>
            ),
          },
        ]}
      />

      <Drawer
        width={760}
        title={`Session ${detail?.conversationToken ?? ''}`}
        open={Boolean(detail)}
        loading={detailLoading}
        onClose={() => {
          detailRequest.current++;
          setDetail(undefined);
        }}
        extra={
          detail && (
            <Tag color={statusMeta[detail.status]?.color}>{statusMeta[detail.status]?.label}</Tag>
          )
        }
      >
        {detail && (
          <Space orientation='vertical' size='large' className='w-full'>
            <Descriptions
              bordered
              size='small'
              column={2}
              items={[
                { key: 'customer', label: 'Customer', children: detail.customerName || '-' },
                { key: 'branch', label: 'Branch', children: detail.branchName || '-' },
                { key: 'contract', label: 'Contract', children: detail.contractNumber || '-' },
                {
                  key: 'confidence',
                  label: 'AI credibility',
                  children: `${Math.round(detail.confidence * 100)}%`,
                },
                {
                  key: 'missing',
                  label: 'Missing field',
                  span: 2,
                  children: detail.missingFields?.join('、') || 'None',
                },
              ]}
            />
            <Card size='small' title="Manual correction">
              <Form form={correctionForm} layout='vertical'>
                <Form.Item name='customerName' label="Customer name">
                  <Input />
                </Form.Item>
                <Form.Item name='branchName' label="Branch name">
                  <Input />
                </Form.Item>
                <Form.Item name='reportedContractNumber' label="Failure report contract number">
                  <Input />
                </Form.Item>
                <Form.Item name='title' label="Title">
                  <Input maxLength={500} />
                </Form.Item>
                <Form.Item name='description' label="Fault description">
                  <Input.TextArea rows={4} maxLength={5000} />
                </Form.Item>
                <Space wrap>
                  <Button
                    loading={actionLoading}
                    disabled={actionLoading}
                    onClick={saveCorrections}
                  >

                    Save and recheck
                  </Button>
                  <Button disabled={actionLoading} onClick={() => runAction('revalidate')}>

                    Recheck only
                  </Button>
                  <Button
                    type='primary'
                    loading={actionLoading}
                    disabled={actionLoading || detail.status !== 'VERIFIED'}
                    onClick={() => runAction('confirm')}
                  >

                    Confirm billing
                  </Button>
                  <Button danger disabled={actionLoading} onClick={() => runAction('reject')}>

                    Rejected
                  </Button>
                  <Button disabled={actionLoading} onClick={() => runAction('retry')}>

                    Retry
                  </Button>
                  <Button
                    danger
                    disabled={actionLoading}
                    icon={<ShieldAlert size={14} />}
                    onClick={() => setOverrideOpen(true)}
                  >

                    Forced order opening
                  </Button>
                </Space>
              </Form>
            </Card>
            <Card size='small' title="Security preview of original email text">
              {detail.messages.length ? (
                <>
                  <Text strong>{detail.messages[detail.messages.length - 1].subject}</Text>
                  <Paragraph type='secondary'>

                    From:{detail.messages[detail.messages.length - 1].fromAddress} ·{' '}
                    {formatTime(detail.messages[detail.messages.length - 1].receivedAt)}
                  </Paragraph>
                  <pre className='max-h-80 overflow-auto whitespace-pre-wrap rounded bg-slate-50 p-3'>
                    {detail.messages[detail.messages.length - 1].plainText}
                  </pre>
                </>
              ) : (
                'No email yet'
              )}
            </Card>
            <Card size='small' title="AI and sending audit">
              <pre className='max-h-72 overflow-auto rounded bg-slate-50 p-3 text-xs'>
                {JSON.stringify(
                  { analyses: detail.analyses, outboundMessages: detail.outboundMessages },
                  null,
                  2
                )}
              </pre>
            </Card>
          </Space>
        )}
      </Drawer>

      <Modal
        title="High risk: forced opening of orders"
        open={overrideOpen}
        confirmLoading={actionLoading}
        onCancel={() => setOverrideOpen(false)}
        onOk={submitOverride}
        okButtonProps={{ danger: true }}
        okText="Confirm and issue order"
      >
        <Alert
          type='warning'
          showIcon
          title="This operation bypasses contract status restrictions and records the operator, reason, and input snapshot."
          className='mb-4'
        />
        <Form form={overrideForm} layout='vertical'>
          <Form.Item
            name='reason'
            label="Forced reason"
            rules={[{ required: true, min: 5, message: 'Please enter a reason of at least 5 characters' }]}
          >
            <Input.TextArea rows={4} maxLength={1000} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editingCustomer || editingContract ? 'Edit configuration' : 'New configuration'}
        open={Boolean(entityModal)}
        onCancel={() => {
          setEntityModal(undefined);
          setEditingCustomer(undefined);
          setEditingContract(undefined);
          form.resetFields();
        }}
        onOk={submitEntity}
        destroyOnHidden
      >
        <Form
          form={form}
          layout='vertical'
          initialValues={{ status: 'active', timezone: 'Asia/Shanghai' }}
        >
          {['customer', 'branch', 'source', 'schedule'].includes(entityModal ?? '') && (
            <Form.Item name='name' label="Name" rules={[{ required: true }]}>
              <Input />
            </Form.Item>
          )}
          {entityModal === 'customer' && (
            <>
              <Form.Item name='shortName' label="Abbreviation">
                <Input />
              </Form.Item>
              <Form.Item name='aliases' label='Alias'>
                <Select mode='tags' />
              </Form.Item>
              <Form.Item name='historicalNames' label="Historical name">
                <Select mode='tags' />
              </Form.Item>
            </>
          )}
          {['branch', 'contract'].includes(entityModal ?? '') && (
            <Form.Item name='customerId' label="Customer" rules={[{ required: true }]}>
              <Select options={customers.map(item => ({ value: item.id, label: item.name }))} />
            </Form.Item>
          )}
          {entityModal === 'branch' && (
            <Form.Item name='aliases' label='Alias'>
              <Select mode='tags' />
            </Form.Item>
          )}
          {entityModal === 'contract' && (
            <>
              <Form.Item name='contractNumber' label="Contract number" rules={[{ required: true }]}>
                <Input />
              </Form.Item>
              <Form.Item name='branchId' label="Division ID">
                <InputNumber className='w-full' min={1} />
              </Form.Item>
              <Form.Item name='status' label="Status" rules={[{ required: true }]}>
                <Select
                  options={['active', 'terminated', 'expired', 'pending'].map(value => ({ value }))}
                />
              </Form.Item>
              <Form.Item name='period' label="Validity date">
                <DatePicker.RangePicker showTime allowEmpty={[true, true]} className='w-full' />
              </Form.Item>
            </>
          )}
          {entityModal === 'source' && (
            <>
              <Form.Item name='emailAddresses' label="Sending email">
                <Select mode='tags' />
              </Form.Item>
              <Form.Item name='emailDomains' label="Sending domain name">
                <Select mode='tags' />
              </Form.Item>
            </>
          )}
          {entityModal === 'externalReference' && (
            <>
              <Form.Item name='sourceOrganizationId' label="Source organization" rules={[{ required: true }]}>
                <Select options={sources.map(item => ({ value: item.id, label: item.name }))} />
              </Form.Item>
              <Form.Item name='supportContractId' label="Internal Support Contract" rules={[{ required: true }]}>
                <Select
                  options={contracts.map(item => ({ value: item.id, label: item.contractNumber }))}
                />
              </Form.Item>
              <Form.Item
                name='externalContractNumber'
                label="External contract number"
                rules={[{ required: true }]}
              >
                <Input />
              </Form.Item>
            </>
          )}
          {entityModal === 'schedule' && (
            <>
              <Form.Item name='groupId' label="Support group ID" rules={[{ required: true }]}>
                <InputNumber className='w-full' min={1} />
              </Form.Item>
              <Form.Item name='timezone' label="Time zone">
                <Input />
              </Form.Item>
            </>
          )}
          {entityModal === 'shift' && (
            <>
              <Form.Item name='scheduleId' label="Scheduling" rules={[{ required: true }]}>
                <Select
                  placeholder="Select shift"
                  showSearch
                  optionFilterProp='label'
                  options={schedules.map(s => ({ value: s.id, label: s.name }))}
                />
              </Form.Item>
              <Form.Item name='userId' label="Engineer on duty" rules={[{ required: true }]}>
                <Select
                  placeholder="Select Engineer"
                  showSearch
                  optionFilterProp='label'
                  options={users.map(u => ({
                    value: u.id,
                    label: u.name || u.email || `User#${u.id}`,
                  }))}
                />
              </Form.Item>
              <Form.Item name='period' label="Shift time" rules={[{ required: true }]}>
                <DatePicker.RangePicker showTime className='w-full' />
              </Form.Item>
            </>
          )}
        </Form>
      </Modal>
    </div>
  );
}
