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

interface User {
  id: number;
  name: string;
  email?: string;
}

const { Title, Text, Paragraph } = Typography;

const statusMeta: Record<string, { color: string; label: string }> = {
  PROCESSING: { color: 'processing', label: '处理中' },
  NEED_INFORMATION: { color: 'warning', label: '缺少资料' },
  WAITING_CUSTOMER: { color: 'warning', label: '等待客户' },
  MANUAL_REVIEW: { color: 'orange', label: '人工复核' },
  VERIFIED: { color: 'success', label: '核验通过' },
  INCIDENT_CREATED: { color: 'blue', label: '已创建事件' },
  REJECTED: { color: 'error', label: '已拒绝' },
};

const formatTime = (value?: string): string =>
  value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-';

export default function EmailIntakePage() {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState<string>();
  const [conversations, setConversations] = useState<EmailConversation[]>([]);
  const [detail, setDetail] = useState<EmailConversationDetail>();
  const [detailLoading, setDetailLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);
  const detailRequest = useRef(0);
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
    setLoading(true);
    try {
      const response = await emailIntakeService.conversations(status);
      setConversations(response.items ?? []);
    } catch (error) {
      message.error(`加载邮件处理队列失败：${(error as Error).message}`);
    } finally {
      setLoading(false);
    }
  }, [message, status]);

  const loadMasterData = useCallback(async () => {
    try {
      const [customerResult, contractResult, sourceResult, externalReferenceResult, scheduleResult, userResult] =
        await Promise.allSettled([
          emailIntakeService.customers(),
          emailIntakeService.contracts(),
          emailIntakeService.sourceOrganizations(),
          emailIntakeService.externalContractReferences(),
          emailIntakeService.schedules(),
          fetch('/api/v1/users?pageSize=200').then(r => r.json()),
        ]);
      if (customerResult.status === 'fulfilled') setCustomers(customerResult.value.items ?? []);
      if (contractResult.status === 'fulfilled') setContracts(contractResult.value.items ?? []);
      if (sourceResult.status === 'fulfilled') setSources(sourceResult.value.items ?? []);
      if (externalReferenceResult.status === 'fulfilled')
        setExternalReferences(externalReferenceResult.value.items ?? []);
      if (scheduleResult.status === 'fulfilled') setSchedules(scheduleResult.value.items ?? []);
      if (userResult.status === 'fulfilled')
        setUsers(userResult.value.items || userResult.value.data?.items || []);
    } catch (error) {
      message.error(`加载主数据失败：${(error as Error).message}`);
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
      message.error(`加载会话详情失败：${(error as Error).message}`);
    } finally {
      if (requestId === detailRequest.current) setDetailLoading(false);
    }
  };

  const runAction = async (action: 'revalidate' | 'confirm' | 'reject' | 'retry') => {
    if (!detail) return;
    setActionLoading(true);
    try {
      const updated = await emailIntakeService[action](detail.id, detail.version);
      message.success('操作已完成');
      await Promise.all([loadConversations(), openDetail(updated)]);
    } catch (error) {
      message.error((error as Error).message);
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
      message.success('修正已保存并重新核验');
      await Promise.all([loadConversations(), openDetail(updated)]);
    } catch (error) {
      if (error instanceof Error) message.error(error.message);
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
      message.success('已记录强制开单原因并创建事件');
      await Promise.all([loadConversations(), openDetail(updated)]);
    } catch (error) {
      if (error instanceof Error) message.error(error.message);
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
      message.success('保存成功');
      setEntityModal(undefined);
      setEditingCustomer(undefined);
      setEditingContract(undefined);
      form.resetFields();
      await loadMasterData();
    } catch (error) {
      if (error instanceof Error) message.error(error.message);
    }
  };

  const columns = useMemo(
    () => [
      {
        title: '状态',
        dataIndex: 'status',
        width: 110,
        render: (value: string) => (
          <Tag color={statusMeta[value]?.color}>{statusMeta[value]?.label ?? value}</Tag>
        ),
      },
      {
        title: '客户',
        dataIndex: 'customerName',
        ellipsis: true,
        render: (value: string) => value || '-',
      },
      {
        title: '分部',
        dataIndex: 'branchName',
        width: 140,
        render: (value: string) => value || '-',
      },
      {
        title: '合同',
        dataIndex: 'contractNumber',
        width: 150,
        render: (value: string) => value || '-',
      },
      {
        title: '可信度',
        dataIndex: 'confidence',
        width: 130,
        render: (value: number) => (
          <Progress percent={Math.round((value ?? 0) * 100)} size='small' />
        ),
      },
      {
        title: '事件',
        dataIndex: 'incidentNumber',
        width: 150,
        render: (value: string, record: EmailConversation) =>
          value ? <a href={`/incidents/${record.incidentId}`}>{value}</a> : '-',
      },
      { title: '最后邮件', dataIndex: 'lastMessageAt', width: 180, render: formatTime },
      {
        title: '操作',
        width: 80,
        fixed: 'right' as const,
        render: (_: unknown, record: EmailConversation) => (
          <Button type='link' onClick={() => openDetail(record)}>
            处理
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
          placeholder='全部状态'
          value={status}
          onChange={setStatus}
          style={{ width: 180 }}
          options={Object.entries(statusMeta).map(([value, meta]) => ({
            value,
            label: meta.label,
          }))}
        />
        <Button icon={<RefreshCw size={14} />} onClick={loadConversations}>
          刷新
        </Button>
      </Space>
      <Table
        rowKey='id'
        size='middle'
        loading={loading}
        columns={columns}
        dataSource={conversations}
        scroll={{ x: 1100 }}
        locale={{ emptyText: '暂无邮件报障记录' }}
      />
    </Card>
  );

  const masterData = (
    <div className='space-y-4'>
      <Space wrap>
        <Button type='primary' onClick={() => setEntityModal('customer')}>
          新增客户
        </Button>
        <Button onClick={() => setEntityModal('branch')}>新增分部</Button>
        <Button onClick={() => setEntityModal('contract')}>新增支持合同</Button>
        <Button onClick={() => setEntityModal('source')}>新增来源组织</Button>
        <Button onClick={() => setEntityModal('externalReference')}>新增外部合同映射</Button>
      </Space>
      <Card title='服务客户'>
        <Table
          rowKey='id'
          size='small'
          pagination={false}
          dataSource={customers}
          columns={[
            { title: '名称', dataIndex: 'name' },
            { title: '简称', dataIndex: 'shortName' },
            { title: 'Alias', dataIndex: 'aliases', render: (v: string[]) => v?.join('、') || '-' },
            {
              title: '状态',
              dataIndex: 'status',
              render: (v: string) => <Tag color={v === 'active' ? 'green' : 'default'}>{v}</Tag>,
            },
            {
              title: '操作',
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
                    编辑
                  </Button>
                  <Button
                    type='link'
                    danger
                    disabled={record.status !== 'active'}
                    onClick={() =>
                      Modal.confirm({
                        title: '停用该客户？',
                        onOk: async () => {
                          await emailIntakeService.disableCustomer(record.id);
                          await loadMasterData();
                        },
                      })
                    }
                  >
                    停用
                  </Button>
                </Space>
              ),
            },
          ]}
        />
      </Card>
      <Card title='支持合同'>
        <Table
          rowKey='id'
          size='small'
          pagination={false}
          dataSource={contracts}
          columns={[
            { title: '合同号', dataIndex: 'contractNumber' },
            {
              title: '客户',
              dataIndex: 'customerId',
              render: (id: number) => customers.find(item => item.id === id)?.name ?? id,
            },
            {
              title: '状态',
              dataIndex: 'status',
              render: (v: string) => <Tag color={v === 'active' ? 'success' : 'error'}>{v}</Tag>,
            },
            {
              title: '有效期',
              render: (_: unknown, record: SupportContract) =>
                `${formatTime(record.startAt)} — ${formatTime(record.endAt)}`,
            },
            {
              title: '操作',
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
                    编辑
                  </Button>
                  <Button
                    type='link'
                    danger
                    disabled={record.status !== 'active'}
                    onClick={() =>
                      Modal.confirm({
                        title: '终止该合同？',
                        content: '终止后自动开单将立即被阻止。',
                        onOk: async () => {
                          await emailIntakeService.terminateContract(record.id);
                          await loadMasterData();
                        },
                      })
                    }
                  >
                    终止
                  </Button>
                </Space>
              ),
            },
          ]}
        />
      </Card>
      <Card title='来源组织'>
        <Table
          rowKey='id'
          size='small'
          pagination={false}
          dataSource={sources}
          columns={[
            { title: '组织', dataIndex: 'name' },
            {
              title: '邮箱',
              dataIndex: 'emailAddresses',
              render: (v: string[]) => v?.join('、') || '-',
            },
            {
              title: '域名',
              dataIndex: 'emailDomains',
              render: (v: string[]) => v?.join('、') || '-',
            },
          ]}
        />
      </Card>
      <Card title='外部合同号映射'>
        <Table
          rowKey='id'
          size='small'
          pagination={false}
          dataSource={externalReferences}
          columns={[
            {
              title: '来源组织',
              dataIndex: 'sourceOrganizationId',
              render: (id: number) => sources.find(item => item.id === id)?.name ?? id,
            },
            { title: '外部合同号', dataIndex: 'externalContractNumber' },
            {
              title: '内部支持合同',
              dataIndex: 'supportContractId',
              render: (id: number) => contracts.find(item => item.id === id)?.contractNumber ?? id,
            },
            {
              title: '操作',
              render: (_: unknown, record: ExternalContractReference) => (
                <Button
                  type='link'
                  danger
                  onClick={() =>
                    Modal.confirm({
                      title: '删除该映射？',
                      onOk: async () => {
                        await emailIntakeService.deleteExternalContractReference(record.id);
                        await loadMasterData();
                      },
                    })
                  }
                >
                  删除
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
          邮件智能报障
        </Title>
        <Text type='secondary'>AI 负责提取，客户、分部和合同规则负责授权开单。</Text>
      </div>
      <Alert
        type='info'
        showIcon
        message='自动开单默认关闭'
        description='建议先使用 observeOnly 或 manualConfirm 验证 Golden Set，再按租户启用 autoCreate。'
      />
      <Tabs
        items={[
          { key: 'queue', label: 'NOC 处理队列', children: queue },
          { key: 'master', label: '客户与合同主数据', children: masterData },
          {
            key: 'oncall',
            label: '值班配置',
            children: (
              <Card>
                <Space>
                  <Button type='primary' onClick={() => setEntityModal('schedule')}>
                    新增排班
                  </Button>
                  <Button onClick={() => setEntityModal('shift')}>新增班次</Button>
                </Space>
                <Paragraph type='secondary' className='!mt-4'>
                  PoC 使用显式起止时间班次；值班用户必须是对应支持组的启用成员。
                </Paragraph>
              </Card>
            ),
          },
        ]}
      />

      <Drawer
        width={760}
        title={`会话 ${detail?.conversationToken ?? ''}`}
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
                { key: 'customer', label: '客户', children: detail.customerName || '-' },
                { key: 'branch', label: '分部', children: detail.branchName || '-' },
                { key: 'contract', label: '合同', children: detail.contractNumber || '-' },
                {
                  key: 'confidence',
                  label: 'AI 可信度',
                  children: `${Math.round(detail.confidence * 100)}%`,
                },
                {
                  key: 'missing',
                  label: '缺失字段',
                  span: 2,
                  children: detail.missingFields?.join('、') || '无',
                },
              ]}
            />
            <Card size='small' title='人工修正'>
              <Form form={correctionForm} layout='vertical'>
                <Form.Item name='customerName' label='客户名称'>
                  <Input />
                </Form.Item>
                <Form.Item name='branchName' label='分部名称'>
                  <Input />
                </Form.Item>
                <Form.Item name='reportedContractNumber' label='报障合同号'>
                  <Input />
                </Form.Item>
                <Form.Item name='title' label='标题'>
                  <Input maxLength={500} />
                </Form.Item>
                <Form.Item name='description' label='故障描述'>
                  <Input.TextArea rows={4} maxLength={5000} />
                </Form.Item>
                <Space wrap>
                  <Button
                    loading={actionLoading}
                    disabled={actionLoading}
                    onClick={saveCorrections}
                  >
                    保存并重新核验
                  </Button>
                  <Button disabled={actionLoading} onClick={() => runAction('revalidate')}>
                    仅重新核验
                  </Button>
                  <Button
                    type='primary'
                    loading={actionLoading}
                    disabled={actionLoading || detail.status !== 'VERIFIED'}
                    onClick={() => runAction('confirm')}
                  >
                    确认开单
                  </Button>
                  <Button danger disabled={actionLoading} onClick={() => runAction('reject')}>
                    拒绝
                  </Button>
                  <Button disabled={actionLoading} onClick={() => runAction('retry')}>
                    重试
                  </Button>
                  <Button
                    danger
                    disabled={actionLoading}
                    icon={<ShieldAlert size={14} />}
                    onClick={() => setOverrideOpen(true)}
                  >
                    强制开单
                  </Button>
                </Space>
              </Form>
            </Card>
            <Card size='small' title='邮件原文安全预览'>
              {detail.messages.length ? (
                <>
                  <Text strong>{detail.messages[detail.messages.length - 1].subject}</Text>
                  <Paragraph type='secondary'>
                    来自：{detail.messages[detail.messages.length - 1].fromAddress} ·{' '}
                    {formatTime(detail.messages[detail.messages.length - 1].receivedAt)}
                  </Paragraph>
                  <pre className='max-h-80 overflow-auto whitespace-pre-wrap rounded bg-slate-50 p-3'>
                    {detail.messages[detail.messages.length - 1].plainText}
                  </pre>
                </>
              ) : (
                '暂无邮件'
              )}
            </Card>
            <Card size='small' title='AI 与发送审计'>
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
        title='高风险：强制开单'
        open={overrideOpen}
        confirmLoading={actionLoading}
        onCancel={() => setOverrideOpen(false)}
        onOk={submitOverride}
        okButtonProps={{ danger: true }}
        okText='确认并开单'
      >
        <Alert
          type='warning'
          showIcon
          message='此操作会绕过合同状态限制，并记录操作者、原因及输入快照。'
          className='mb-4'
        />
        <Form form={overrideForm} layout='vertical'>
          <Form.Item
            name='reason'
            label='强制原因'
            rules={[{ required: true, min: 5, message: '请填写至少 5 个字符的原因' }]}
          >
            <Input.TextArea rows={4} maxLength={1000} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editingCustomer || editingContract ? '编辑配置' : '新增配置'}
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
            <Form.Item name='name' label='名称' rules={[{ required: true }]}>
              <Input />
            </Form.Item>
          )}
          {entityModal === 'customer' && (
            <>
              <Form.Item name='shortName' label='简称'>
                <Input />
              </Form.Item>
              <Form.Item name='aliases' label='Alias'>
                <Select mode='tags' />
              </Form.Item>
              <Form.Item name='historicalNames' label='历史名称'>
                <Select mode='tags' />
              </Form.Item>
            </>
          )}
          {['branch', 'contract'].includes(entityModal ?? '') && (
            <Form.Item name='customerId' label='客户' rules={[{ required: true }]}>
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
              <Form.Item name='contractNumber' label='合同号' rules={[{ required: true }]}>
                <Input />
              </Form.Item>
              <Form.Item name='branchId' label='分部 ID'>
                <InputNumber className='w-full' min={1} />
              </Form.Item>
              <Form.Item name='status' label='状态' rules={[{ required: true }]}>
                <Select
                  options={['active', 'terminated', 'expired', 'pending'].map(value => ({ value }))}
                />
              </Form.Item>
              <Form.Item name='period' label='有效期'>
                <DatePicker.RangePicker showTime allowEmpty={[true, true]} className='w-full' />
              </Form.Item>
            </>
          )}
          {entityModal === 'source' && (
            <>
              <Form.Item name='emailAddresses' label='发件邮箱'>
                <Select mode='tags' />
              </Form.Item>
              <Form.Item name='emailDomains' label='发件域名'>
                <Select mode='tags' />
              </Form.Item>
            </>
          )}
          {entityModal === 'externalReference' && (
            <>
              <Form.Item name='sourceOrganizationId' label='来源组织' rules={[{ required: true }]}>
                <Select options={sources.map(item => ({ value: item.id, label: item.name }))} />
              </Form.Item>
              <Form.Item name='supportContractId' label='内部支持合同' rules={[{ required: true }]}>
                <Select
                  options={contracts.map(item => ({ value: item.id, label: item.contractNumber }))}
                />
              </Form.Item>
              <Form.Item
                name='externalContractNumber'
                label='外部合同号'
                rules={[{ required: true }]}
              >
                <Input />
              </Form.Item>
            </>
          )}
          {entityModal === 'schedule' && (
            <>
              <Form.Item name='groupId' label='支持组 ID' rules={[{ required: true }]}>
                <InputNumber className='w-full' min={1} />
              </Form.Item>
              <Form.Item name='timezone' label='时区'>
                <Input />
              </Form.Item>
            </>
          )}
          {entityModal === 'shift' && (
            <>
              <Form.Item name='scheduleId' label='排班' rules={[{ required: true }]}>
                <Select
                  placeholder='选择排班'
                  showSearch
                  optionFilterProp='label'
                  options={schedules.map(s => ({ value: s.id, label: s.name }))}
                />
              </Form.Item>
              <Form.Item name='userId' label='值班工程师' rules={[{ required: true }]}>
                <Select
                  placeholder='选择工程师'
                  showSearch
                  optionFilterProp='label'
                  options={users.map(u => ({
                    value: u.id,
                    label: u.name || u.email || `用户#${u.id}`,
                  }))}
                />
              </Form.Item>
              <Form.Item name='period' label='班次时间' rules={[{ required: true }]}>
                <DatePicker.RangePicker showTime className='w-full' />
              </Form.Item>
            </>
          )}
        </Form>
      </Modal>
    </div>
  );
}
