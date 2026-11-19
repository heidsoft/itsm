'use client';

// CI 数据认证任务页（ServiceNow CMDB Health / Compliance 维度）。
//
// 这个页面回答一个问题：**这批 CMDB 数据，还有哪些没人认领确认、逾期多久了？**
// 服务端的治理看板提供的是「整体质量分」，但运维真正要干的活是按任务清单逐条确认。
// 因此本页只做两件事：列出待办/逾期、就地完成确认，并把三个计数（待办/逾期/
// 已完成）随列表一并展示——口径来自后端同一份查询，不在前端二次统计。

import React, { useCallback, useMemo, useState } from 'react';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Empty,
  Form,
  Input,
  Modal,
  Row,
  Space,
  Statistic,
  Table,
  Tag,
  Tooltip,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';

import CISearchSelect from '@/components/cmdb/CISearchSelect';
import { ManagementPageHeader } from '@/components/ui/ManagementPageHeader';
import {
  CMDBApi,
  type CICertificationTask,
  type CICertificationTaskListResponse,
} from '@/lib/api/cmdb-api';
import { serverErrorMessage } from '@/lib/api/http-client';

const PAGE_SIZE = 20;

type TaskStatusFilter = 'all' | 'pending' | 'confirmed';

/**
 * 把后端状态 + dueAt 折成展示用的三态。
 *
 * 后端只存 pending/confirmed 两种状态，overdue 是「pending 且 dueAt 已过」的派生。
 * 前端不自己再造第三份判定逻辑去和统计数对不上——逾期数直接用后端下发的
 * overdue 计数，行上的逾期标记只用于视觉提示。
 */
function isOverdue(task: CICertificationTask): boolean {
  if (task.status !== 'pending' || !task.dueAt) return false;
  return dayjs(task.dueAt).isBefore(dayjs());
}

export default function CICertificationPage() {
  const { message } = App.useApp();

  const [data, setData] = useState<CICertificationTaskListResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [errorText, setErrorText] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState<TaskStatusFilter>('all');

  const [createOpen, setCreateOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [createForm] = Form.useForm<{ ciId: number; dueAt?: string; evidence?: string }>();
  const [confirmTarget, setConfirmTarget] = useState<CICertificationTask | null>(null);
  const [confirming, setConfirming] = useState(false);
  const [confirmEvidence, setConfirmEvidence] = useState('');

  const loadData = useCallback(
    async (targetPage: number) => {
      setLoading(true);
      setErrorText(null);
      try {
        const res = await CMDBApi.listCICertificationTasks({
          page: targetPage,
          pageSize: PAGE_SIZE,
        });
        setData(res);
      } catch (error) {
        // 失败必须显式呈现：只吞掉异常会让「待办 0」看起来像「全都确认完了」，
        // 而实际上是一次 403/500。这正是认证体系最危险的假绿。
        setErrorText(serverErrorMessage(error, '加载认证任务失败，请重试'));
        setData(null);
      } finally {
        setLoading(false);
      }
    },
    []
  );

  React.useEffect(() => {
    void loadData(page);
  }, [loadData, page]);

  const handleCreate = useCallback(async () => {
    try {
      const values = await createForm.validateFields();
      setCreating(true);
      await CMDBApi.createCICertificationTask({
        ciId: Number(values.ciId),
        dueAt: values.dueAt ? dayjs(values.dueAt).toISOString() : undefined,
        evidence: values.evidence,
      });
      message.success('认证任务已创建');
      setCreateOpen(false);
      createForm.resetFields();
      setPage(1);
      await loadData(1);
    } catch (error) {
      if (error && typeof error === 'object' && 'errorFields' in error) return; // 表单校验失败
      message.error(serverErrorMessage(error, '创建认证任务失败'));
    } finally {
      setCreating(false);
    }
  }, [createForm, loadData, message]);

  const handleConfirm = useCallback(async () => {
    if (!confirmTarget) return;
    setConfirming(true);
    try {
      await CMDBApi.confirmCICertificationTask(confirmTarget.id, {
        evidence: confirmEvidence || undefined,
      });
      message.success('已完成认证');
      setConfirmTarget(null);
      setConfirmEvidence('');
      await loadData(page);
    } catch (error) {
      message.error(serverErrorMessage(error, '完成认证失败'));
    } finally {
      setConfirming(false);
    }
  }, [confirmEvidence, confirmTarget, loadData, message, page]);

  // 状态筛选在前端做（后端只返回全量分页）；三个统计数始终取后端字段，
  // 筛选时不清零——否则用户在「已确认」页签里会看到「已完成 0」。
  const visibleItems = useMemo(() => {
    const items = data?.items ?? [];
    if (statusFilter === 'all') return items;
    return items.filter(t => t.status === statusFilter);
  }, [data, statusFilter]);

  const columns: ColumnsType<CICertificationTask> = [
    {
      title: '配置项',
      dataIndex: 'ciName',
      render: (_: unknown, record) =>
        record.ciName ? (
          <a href={`/cmdb/cis/${record.ciId}`}>{record.ciName}</a>
        ) : (
          <span className='text-slate-500'>CI #{record.ciId}</span>
        ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 120,
      render: (_: unknown, record) => {
        if (record.status === 'confirmed') return <Tag color='green'>已确认</Tag>;
        if (isOverdue(record)) return <Tag color='red'>已逾期</Tag>;
        return <Tag color='blue'>待确认</Tag>;
      },
    },
    {
      title: '截止时间',
      dataIndex: 'dueAt',
      width: 180,
      render: (value: string | undefined, record) => {
        if (!value) return <span className='text-slate-400'>—</span>;
        return (
          <span className={isOverdue(record) ? 'font-medium text-red-600' : undefined}>
            {dayjs(value).format('YYYY-MM-DD HH:mm')}
          </span>
        );
      },
    },
    {
      title: '确认人',
      dataIndex: 'confirmedBy',
      width: 120,
      render: (value: number | undefined) =>
        value ? `用户 #${value}` : <span className='text-slate-400'>—</span>,
    },
    {
      title: '确认时间',
      dataIndex: 'confirmedAt',
      width: 180,
      render: (value: string | undefined) =>
        value ? dayjs(value).format('YYYY-MM-DD HH:mm') : <span className='text-slate-400'>—</span>,
    },
    {
      title: '依据',
      dataIndex: 'evidence',
      ellipsis: { showTitle: false },
      render: (value: string | undefined) =>
        value ? (
          <Tooltip title={value}>
            <span>{value}</span>
          </Tooltip>
        ) : (
          <span className='text-slate-400'>—</span>
        ),
    },
    {
      title: '操作',
      width: 120,
      render: (_: unknown, record) =>
        record.status === 'pending' ? (
          <Button type='link' onClick={() => setConfirmTarget(record)}>
            完成认证
          </Button>
        ) : (
          <span className='text-slate-400'>—</span>
        ),
    },
  ];

  return (
    <div className='space-y-6 p-6'>
      <ManagementPageHeader
        title='数据认证'
        description='按任务清单核对配置项是否仍然准确：谁确认、什么时候确认、依据是什么。逾期的待办会持续提醒。'
        actions={
          <Space wrap>
            <Button onClick={() => void loadData(page)} loading={loading}>
              刷新
            </Button>
            <Button type='primary' onClick={() => setCreateOpen(true)}>
              发起认证
            </Button>
          </Space>
        }
      />

      {errorText ? (
        <Alert
          type='error'
          showIcon
          title='认证任务加载失败'
          description={errorText}
          action={
            <Button size='small' onClick={() => void loadData(page)}>
              重试
            </Button>
          }
        />
      ) : null}

      <Row gutter={[16, 16]}>
        <Col xs={12} md={6}>
          <Card size='small'>
            <Statistic title='待确认' value={data?.pending ?? 0} loading={loading} />
          </Card>
        </Col>
        <Col xs={12} md={6}>
          <Card size='small'>
            <Statistic
              title='已逾期'
              value={data?.overdue ?? 0}
              loading={loading}
              valueStyle={data && data.overdue > 0 ? { color: '#cf1322' } : undefined}
            />
          </Card>
        </Col>
        <Col xs={12} md={6}>
          <Card size='small'>
            <Statistic title='已确认' value={data?.confirmed ?? 0} loading={loading} />
          </Card>
        </Col>
        <Col xs={12} md={6}>
          <Card size='small'>
            <Statistic title='任务总数' value={data?.total ?? 0} loading={loading} />
          </Card>
        </Col>
      </Row>

      <Card className='shadow-sm'>
        <Space className='mb-4' wrap>
          <Button
            type={statusFilter === 'all' ? 'primary' : 'default'}
            onClick={() => setStatusFilter('all')}
          >
            全部
          </Button>
          <Button
            type={statusFilter === 'pending' ? 'primary' : 'default'}
            onClick={() => setStatusFilter('pending')}
          >
            待确认
          </Button>
          <Button
            type={statusFilter === 'confirmed' ? 'primary' : 'default'}
            onClick={() => setStatusFilter('confirmed')}
          >
            已确认
          </Button>
        </Space>

        <Table<CICertificationTask>
          rowKey='id'
          columns={columns}
          dataSource={visibleItems}
          loading={loading}
          locale={{
            emptyText: errorText ? (
              <Empty description='数据加载失败，请重试' />
            ) : (
              <Empty description='暂无认证任务。发起一条认证，让资产有人负责。' />
            ),
          }}
          pagination={{
            current: page,
            pageSize: PAGE_SIZE,
            total: data?.total ?? 0,
            showSizeChanger: false,
            onChange: next => setPage(next),
          }}
        />
      </Card>

      <Modal
        open={createOpen}
        title='发起数据认证'
        onCancel={() => setCreateOpen(false)}
        onOk={() => void handleCreate()}
        confirmLoading={creating}
        destroyOnHidden
      >
        <Alert
          type='info'
          showIcon
          className='mb-4'
          title='同一配置项同时只能有一条待办任务'
          description='未指定截止时间时，系统默认给 30 天。逾期后会在上方「已逾期」计数里持续暴露。'
        />
        <Form form={createForm} layout='vertical'>
          <Form.Item
            label='配置项'
            name='ciId'
            rules={[{ required: true, message: '请选择要认证的配置项' }]}
          >
            <CISearchSelect placeholder='搜索配置项名称或编号' />
          </Form.Item>
          <Form.Item label='截止时间' name='dueAt'>
            <Input type='datetime-local' />
          </Form.Item>
          <Form.Item label='认证要求/说明' name='evidence'>
            <Input.TextArea rows={3} placeholder='例如：请核对机房位置与资产编号' />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={Boolean(confirmTarget)}
        title='完成数据认证'
        onCancel={() => setConfirmTarget(null)}
        onOk={() => void handleConfirm()}
        confirmLoading={confirming}
        destroyOnHidden
      >
        <Alert
          type='warning'
          showIcon
          className='mb-4'
          title='确认即声明「这条数据现在仍然准确」'
          description='确认会记入认证历史并更新该配置项的最近确认时间；治理看板的确认新鲜度读的就是它。'
        />
        <Input.TextArea
          rows={3}
          value={confirmEvidence}
          onChange={event => setConfirmEvidence(event.target.value)}
          placeholder='核对依据（可选）：例如已比对 CMDB 与云上实际配置一致'
        />
      </Modal>
    </div>
  );
}
