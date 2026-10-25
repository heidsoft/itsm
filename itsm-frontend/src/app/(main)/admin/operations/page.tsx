'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Empty,
  Input,
  Popconfirm,
  Row,
  Select,
  Space,
  Statistic,
  Table,
  Tabs,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { AlertTriangle, MailWarning, RefreshCw, RotateCcw, Search } from 'lucide-react';
import { serverErrorMessage } from '@/lib/api/http-client';
import operationsService from '@/lib/services/operations-service';
import type {
  CommandStatus,
  CommandSummary,
  NotificationDelivery,
  OperationalCommand,
} from '@/lib/services/operations-service';

const { Title, Text } = Typography;

const COMMAND_STATUS_LABEL: Record<string, { text: string; color: string }> = {
  pending: { text: '待处理', color: 'default' },
  processing: { text: '处理中', color: 'processing' },
  succeeded: { text: '成功', color: 'success' },
  dead_letter: { text: '死信', color: 'error' },
  cancelled: { text: '已取消', color: 'warning' },
};

const COMMAND_STATUS_OPTIONS = [
  { value: 'dead_letter', label: '死信（默认）' },
  { value: 'pending', label: '待处理' },
  { value: 'processing', label: '处理中' },
  { value: 'cancelled', label: '已取消' },
];

const CHANNEL_OPTIONS = [
  { value: '', label: '全部渠道' },
  { value: 'email', label: '邮件' },
  { value: 'in_app', label: '站内' },
  { value: 'feishu', label: '飞书' },
  { value: 'wecom', label: '企业微信' },
  { value: 'dingtalk', label: '钉钉' },
  { value: 'sms', label: '短信' },
  { value: 'webhook', label: 'Webhook' },
];

const formatTime = (value?: string) => (value ? new Date(value).toLocaleString('zh-CN') : '—');

/**
 * 死信与投递重放的运维面。
 *
 * 存在理由：命令总线重放与投递重试两个入口此前只有 HTTP 端点，产品里唯一可点的
 * 「重放」在租户初始化抽屉里（只覆盖产品基线安装那一种命令）。BPMN 服务任务死信、
 * 通知投递命令死信、失败投递都无法从界面处理，只能 curl。
 *
 * 两个 Tab 的分工必须说清楚，否则运维会互相当成对方的替代：
 *   - 命令死信：命令根本没跑完（含从未进入业务处理的情况），重放保留原幂等键，
 *     不会与历史成功的投递重复；
 *   - 失败投递：命令跑过了但连接器发送失败，重试会真实再发一次消息，
 *     服务端对同一条投递做了防重复（在途/已成功一律拒绝）。
 */
export default function OperationsPage() {
  const { message } = App.useApp();

  const [activeTab, setActiveTab] = useState('commands');

  const [summary, setSummary] = useState<CommandSummary | null>(null);
  const [commands, setCommands] = useState<OperationalCommand[]>([]);
  const [commandTotal, setCommandTotal] = useState(0);
  const [commandPage, setCommandPage] = useState(1);
  const [commandPageSize, setCommandPageSize] = useState(20);
  const [commandStatus, setCommandStatus] = useState<CommandStatus>('dead_letter');
  const [commandType, setCommandType] = useState('');
  const [commandTypeDraft, setCommandTypeDraft] = useState('');
  const [commandsLoading, setCommandsLoading] = useState(false);
  const [replayingId, setReplayingId] = useState<number | null>(null);
  const [bulkReplaying, setBulkReplaying] = useState(false);

  const [deliveries, setDeliveries] = useState<NotificationDelivery[]>([]);
  const [deliveryTotal, setDeliveryTotal] = useState(0);
  const [deliveryPage, setDeliveryPage] = useState(1);
  const [deliveryPageSize, setDeliveryPageSize] = useState(20);
  const [deliveryChannel, setDeliveryChannel] = useState('');
  const [deliveriesLoading, setDeliveriesLoading] = useState(false);
  const [retryingId, setRetryingId] = useState<number | null>(null);

  const loadCommands = useCallback(async () => {
    setCommandsLoading(true);
    try {
      const page = await operationsService.listCommands({
        status: commandStatus,
        commandType: commandType || undefined,
        page: commandPage,
        pageSize: commandPageSize,
      });
      setCommands(page.items ?? []);
      setCommandTotal(page.total ?? 0);
      setSummary(page.summary ?? null);
    } catch (error) {
      message.error(serverErrorMessage(error, '加载运维命令失败'));
    } finally {
      setCommandsLoading(false);
    }
  }, [commandStatus, commandType, commandPage, commandPageSize, message]);

  const loadDeliveries = useCallback(async () => {
    setDeliveriesLoading(true);
    try {
      const page = await operationsService.listDeliveries({
        status: 'failed',
        channel: deliveryChannel || undefined,
        page: deliveryPage,
        pageSize: deliveryPageSize,
      });
      setDeliveries(page.items ?? []);
      setDeliveryTotal(page.pagination?.total ?? 0);
    } catch (error) {
      message.error(serverErrorMessage(error, '加载失败投递失败'));
    } finally {
      setDeliveriesLoading(false);
    }
  }, [deliveryChannel, deliveryPage, deliveryPageSize, message]);

  useEffect(() => {
    void loadCommands();
  }, [loadCommands]);

  useEffect(() => {
    void loadDeliveries();
  }, [loadDeliveries]);

  const handleReplay = async (command: OperationalCommand) => {
    setReplayingId(command.id);
    try {
      await operationsService.replayCommand(command.id);
      message.success(`命令 #${command.id} 已重新入箱`);
      await loadCommands();
    } catch (error) {
      message.error(serverErrorMessage(error, `命令 #${command.id} 重放失败`));
    } finally {
      setReplayingId(null);
    }
  };

  const handleBulkReplay = async () => {
    setBulkReplaying(true);
    try {
      const result = await operationsService.bulkReplayCommands({
        status: commandStatus,
        commandType: commandType || undefined,
      });
      message.success(
        `已重放 ${result.updated} 条命令` +
          (result.skipped > 0 ? `（${result.skipped} 条因状态变化被跳过）` : '')
      );
      await loadCommands();
    } catch (error) {
      message.error(serverErrorMessage(error, '批量重放失败'));
    } finally {
      setBulkReplaying(false);
    }
  };

  const handleRetryDelivery = async (delivery: NotificationDelivery) => {
    setRetryingId(delivery.id);
    try {
      const result = await operationsService.retryDelivery(delivery.id);
      message.success(`投递 #${delivery.id} 已重新入队（命令 #${result.commandId}）`);
      await loadDeliveries();
    } catch (error) {
      // 409 是幂等守卫的正常输出（已有在途重试 / 已重试成功），必须原样透出。
      message.error(serverErrorMessage(error, `投递 #${delivery.id} 重试失败`));
    } finally {
      setRetryingId(null);
    }
  };

  const commandColumns = useMemo<ColumnsType<OperationalCommand>>(
    () => [
      {
        title: 'ID',
        dataIndex: 'id',
        width: 90,
        render: (id: number) => <Text strong>#{id}</Text>,
      },
      {
        title: '命令类型',
        dataIndex: 'commandType',
        render: (value: string) => <Text code>{value}</Text>,
      },
      {
        title: '聚合对象',
        dataIndex: 'aggregateType',
        render: (value: string, row) => (
          <Text>
            {value} / {row.aggregateId}
          </Text>
        ),
      },
      {
        title: '尝试',
        dataIndex: 'attempt',
        width: 100,
        render: (attempt: number, row) => (
          <Text type={attempt >= row.maxAttempts ? 'danger' : undefined}>
            {attempt}/{row.maxAttempts}
          </Text>
        ),
      },
      {
        title: '状态',
        dataIndex: 'status',
        width: 110,
        render: (status: string) => {
          const config = COMMAND_STATUS_LABEL[status] ?? { text: status, color: 'default' };
          return <Tag color={config.color}>{config.text}</Tag>;
        },
      },
      {
        title: '最后错误',
        dataIndex: 'lastError',
        render: (value?: string) =>
          value ? (
            <Tooltip title={value}>
              <Text type='danger' ellipsis style={{ maxWidth: 320, display: 'inline-block' }}>
                {value}
              </Text>
            </Tooltip>
          ) : (
            <Text type='secondary'>—</Text>
          ),
      },
      {
        title: '创建时间',
        dataIndex: 'createdAt',
        width: 180,
        render: (value: string) => formatTime(value),
      },
      {
        title: '操作',
        key: 'actions',
        width: 110,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Popconfirm
            title={
              row.commandType === 'ai.tool.execute' ? '确认重放 AI 写工具？' : '重新入箱这条命令？'
            }
            description={
              row.commandType === 'ai.tool.execute'
                ? `将恢复工具调用 #${row.aggregateId} 并使用原幂等键重跑。若中断前已写入业务数据，重放可能再次执行，请先核对对象与审计。`
                : '使用原始 payload 与幂等键重跑一次；是否重复副作用取决于具体命令的幂等保护。'
            }
            okText='重放'
            cancelText='取消'
            onConfirm={() => handleReplay(row)}
          >
            <Button
              size='small'
              type='link'
              icon={<RotateCcw size={14} />}
              loading={replayingId === row.id}
              disabled={row.status !== 'dead_letter' && row.status !== 'cancelled'}
            >
              重放
            </Button>
          </Popconfirm>
        ),
      },
    ],
    [replayingId]
  );

  const deliveryColumns = useMemo<ColumnsType<NotificationDelivery>>(
    () => [
      {
        title: 'ID',
        dataIndex: 'id',
        width: 90,
        render: (id: number) => <Text strong>#{id}</Text>,
      },
      {
        title: '渠道',
        dataIndex: 'channel',
        width: 110,
        render: (value: string) => <Tag>{value}</Tag>,
      },
      {
        title: '接收人',
        dataIndex: 'recipientId',
        width: 110,
        render: (value: number, row) => (
          <Tooltip title={`收件目标 ${row.targetMasked}`}>
            <Text>用户 #{value}</Text>
          </Tooltip>
        ),
      },
      {
        title: '尝试',
        dataIndex: 'attempt',
        width: 80,
      },
      {
        title: '失败原因',
        dataIndex: 'errorMessage',
        render: (value: string | undefined, row) =>
          value ? (
            <Tooltip title={value}>
              <Text type='danger' ellipsis style={{ maxWidth: 360, display: 'inline-block' }}>
                {row.errorCode ? `[${row.errorCode}] ${value}` : value}
              </Text>
            </Tooltip>
          ) : (
            <Text type='secondary'>—</Text>
          ),
      },
      {
        title: '失败时间',
        dataIndex: 'createdAt',
        width: 180,
        render: (value: string) => formatTime(value),
      },
      {
        title: '操作',
        key: 'actions',
        width: 110,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Popconfirm
            title='重新发送这条通知？'
            description='会真实调用连接器发送一次；若已有在途或已成功的重试，服务端会拒绝。'
            okText='重试'
            cancelText='取消'
            onConfirm={() => handleRetryDelivery(row)}
          >
            <Button
              size='small'
              type='link'
              icon={<RefreshCw size={14} />}
              loading={retryingId === row.id}
            >
              重试
            </Button>
          </Popconfirm>
        ),
      },
    ],
    [retryingId]
  );

  return (
    <div className='space-y-6'>
      <div>
        <Title level={3} style={{ marginBottom: 4 }}>
          死信与投递重放
        </Title>
        <Text type='secondary'>
          命令死信保留原幂等键重放，失败投递重试会真实再发一次消息；AI
          写工具中断后重放前需先核对业务结果。
        </Text>
      </div>

      <Row gutter={[16, 16]}>
        <Col xs={12} md={6}>
          <Card size='small'>
            <Statistic
              title='死信命令'
              value={summary?.deadLetter ?? 0}
              valueStyle={{ color: (summary?.deadLetter ?? 0) > 0 ? '#cf1322' : undefined }}
              prefix={<AlertTriangle size={16} />}
            />
          </Card>
        </Col>
        <Col xs={12} md={6}>
          <Card size='small'>
            <Statistic title='待处理' value={summary?.pending ?? 0} />
          </Card>
        </Col>
        <Col xs={12} md={6}>
          <Card size='small'>
            <Statistic title='处理中' value={summary?.processing ?? 0} />
          </Card>
        </Col>
        <Col xs={12} md={6}>
          <Card size='small'>
            <Statistic
              title='租约卡住'
              value={summary?.stuckLeases ?? 0}
              valueStyle={{ color: (summary?.stuckLeases ?? 0) > 0 ? '#d46b08' : undefined }}
            />
          </Card>
        </Col>
      </Row>

      <Alert
        type='info'
        showIcon
        message='重放的边界'
        description={
          <span>
            「命令死信」重放保留原幂等键，worker 仍按原身份去重；AI 写工具若在中断前已产生副作用，
            重放仍可能再次执行，必须先核对业务对象与审计。「失败投递」已有在途重试或上一次重试成功时返回
            409。
          </span>
        }
      />

      <Card size='small'>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'commands',
              label: '命令死信',
              children: (
                <div className='space-y-4'>
                  <Space wrap>
                    <Select
                      value={commandStatus}
                      style={{ width: 180 }}
                      options={COMMAND_STATUS_OPTIONS}
                      onChange={value => {
                        setCommandStatus(value as CommandStatus);
                        setCommandPage(1);
                      }}
                    />
                    <Input
                      allowClear
                      style={{ width: 240 }}
                      placeholder='按命令类型过滤，如 notification.deliver'
                      value={commandTypeDraft}
                      onChange={event => setCommandTypeDraft(event.target.value)}
                      onPressEnter={() => {
                        setCommandType(commandTypeDraft.trim());
                        setCommandPage(1);
                      }}
                    />
                    <Button
                      icon={<Search size={14} />}
                      onClick={() => {
                        setCommandType(commandTypeDraft.trim());
                        setCommandPage(1);
                      }}
                    >
                      查询
                    </Button>
                    <Button
                      icon={<RefreshCw size={14} />}
                      onClick={() => void loadCommands()}
                      loading={commandsLoading}
                    >
                      刷新
                    </Button>
                    <Popconfirm
                      title={`批量重放 ${commandTotal} 条命令？`}
                      description='按当前筛选条件匹配，单次最多 200 条并保留幂等键；包含 AI 写工具时，请先逐条核对中断前是否已写入业务数据。'
                      okText='全部重放'
                      cancelText='取消'
                      onConfirm={handleBulkReplay}
                      disabled={commandTotal === 0 || commandStatus !== 'dead_letter'}
                    >
                      <Button
                        type='primary'
                        danger
                        icon={<RotateCcw size={14} />}
                        loading={bulkReplaying}
                        disabled={commandTotal === 0 || commandStatus !== 'dead_letter'}
                      >
                        批量重放
                      </Button>
                    </Popconfirm>
                  </Space>

                  <Table<OperationalCommand>
                    rowKey='id'
                    size='small'
                    loading={commandsLoading}
                    columns={commandColumns}
                    dataSource={commands}
                    scroll={{ x: 1100 }}
                    locale={{ emptyText: <Empty description='没有匹配的命令' /> }}
                    pagination={{
                      current: commandPage,
                      pageSize: commandPageSize,
                      total: commandTotal,
                      showSizeChanger: true,
                      showTotal: total => `共 ${total} 条`,
                      onChange: (page, size) => {
                        setCommandPage(page);
                        setCommandPageSize(size);
                      },
                    }}
                  />
                </div>
              ),
            },
            {
              key: 'deliveries',
              label: (
                <Space size={4}>
                  <MailWarning size={14} />
                  失败投递
                </Space>
              ),
              children: (
                <div className='space-y-4'>
                  <Space wrap>
                    <Select
                      value={deliveryChannel}
                      style={{ width: 180 }}
                      options={CHANNEL_OPTIONS}
                      onChange={value => {
                        setDeliveryChannel(value);
                        setDeliveryPage(1);
                      }}
                    />
                    <Button
                      icon={<RefreshCw size={14} />}
                      onClick={() => void loadDeliveries()}
                      loading={deliveriesLoading}
                    >
                      刷新
                    </Button>
                    <Text type='secondary'>共 {deliveryTotal} 条失败记录</Text>
                  </Space>

                  <Table<NotificationDelivery>
                    rowKey='id'
                    size='small'
                    loading={deliveriesLoading}
                    columns={deliveryColumns}
                    dataSource={deliveries}
                    scroll={{ x: 1000 }}
                    locale={{ emptyText: <Empty description='没有失败的投递记录' /> }}
                    pagination={{
                      current: deliveryPage,
                      pageSize: deliveryPageSize,
                      total: deliveryTotal,
                      showSizeChanger: true,
                      showTotal: total => `共 ${total} 条`,
                      onChange: (page, size) => {
                        setDeliveryPage(page);
                        setDeliveryPageSize(size);
                      },
                    }}
                  />
                </div>
              ),
            },
          ]}
        />
      </Card>
    </div>
  );
}
