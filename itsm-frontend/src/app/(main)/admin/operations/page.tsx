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
import { formatDateTime } from '@/lib/formatters';
import type {
  CommandStatus,
  CommandSummary,
  NotificationDelivery,
  OperationalCommand,
} from '@/lib/services/operations-service';

const { Title, Text } = Typography;

const COMMAND_STATUS_LABEL: Record<string, { text: string; color: string }> = {
  pending: { text: 'Pending', color: 'default' },
  processing: { text: 'In Progress', color: 'processing' },
  succeeded: { text: 'Success', color: 'success' },
  dead_letter: { text: 'Dead letter', color: 'error' },
  cancelled: { text: 'Cancelled', color: 'warning' },
};

const COMMAND_STATUS_OPTIONS = [
  { value: 'dead_letter', label: 'Dead letter (default)' },
  { value: 'pending', label: 'Pending' },
  { value: 'processing', label: 'In Progress' },
  { value: 'cancelled', label: 'Cancelled' },
];

const CHANNEL_OPTIONS = [
  { value: '', label: 'All Channels' },
  { value: 'email', label: 'Email' },
  { value: 'in_app', label: 'Within the site' },
  { value: 'feishu', label: 'Feishu' },
  { value: 'wecom', label: 'WeCom' },
  { value: 'dingtalk', label: 'DingTalk' },
  { value: 'sms', label: 'SMS' },
  { value: 'webhook', label: 'Webhook' },
];

const formatTime = (value?: string) => formatDateTime(value) || '—';

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
      message.error(serverErrorMessage(error, 'Failed to load operation and maintenance commands'));
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
      message.error(serverErrorMessage(error, 'Loading failed and delivery failed.'));
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
      message.success(`Command #${command.id} Reboxed`);
      await loadCommands();
    } catch (error) {
      message.error(serverErrorMessage(error, `Command #${command.id} Replay failed`));
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
        `Replayed ${result.updated} command` +
          (result.skipped > 0 ? `（${result.skipped} Entry skipped due to status change)` : '')
      );
      await loadCommands();
    } catch (error) {
      message.error(serverErrorMessage(error, 'Batch replay failed'));
    } finally {
      setBulkReplaying(false);
    }
  };

  const handleRetryDelivery = async (delivery: NotificationDelivery) => {
    setRetryingId(delivery.id);
    try {
      const result = await operationsService.retryDelivery(delivery.id);
      message.success(`Delivery #${delivery.id} Requeued (command #${result.commandId}）`);
      await loadDeliveries();
    } catch (error) {
      // 409 是幂等守卫的正常输出（已有在途重试 / 已重试成功），必须原样透出。
      message.error(serverErrorMessage(error, `Delivery #${delivery.id} Retry failed`));
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
        title: 'Command type',
        dataIndex: 'commandType',
        render: (value: string) => <Text code>{value}</Text>,
      },
      {
        title: 'Aggregation object',
        dataIndex: 'aggregateType',
        render: (value: string, row) => (
          <Text>
            {value} / {row.aggregateId}
          </Text>
        ),
      },
      {
        title: 'Try',
        dataIndex: 'attempt',
        width: 100,
        render: (attempt: number, row) => (
          <Text type={attempt >= row.maxAttempts ? 'danger' : undefined}>
            {attempt}/{row.maxAttempts}
          </Text>
        ),
      },
      {
        title: 'Status',
        dataIndex: 'status',
        width: 110,
        render: (status: string) => {
          const config = COMMAND_STATUS_LABEL[status] ?? { text: status, color: 'default' };
          return <Tag color={config.color}>{config.text}</Tag>;
        },
      },
      {
        title: 'Last error',
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
        title: 'Created',
        dataIndex: 'createdAt',
        width: 180,
        render: (value: string) => formatTime(value),
      },
      {
        title: 'Actions',
        key: 'actions',
        width: 110,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Popconfirm
            title={
              row.commandType === 'ai.tool.execute' ? 'Confirm replay AI writing tool?' : 'Rebox this command?'
            }
            description={
              row.commandType === 'ai.tool.execute'
                ? `Invoking the recovery tool #${row.aggregateId} And rerun using the original idempotent keys. If business data has been written before the interruption, replay may be executed again. Please check the object and audit first.`
                : 'Rerun with original payload and idempotent keys; whether to repeat side effects depends on the idempotent protection of the specific command.'
            }
            okText="Replay"
            cancelText="Cancel"
            onConfirm={() => handleReplay(row)}
          >
            <Button
              size='small'
              type='link'
              icon={<RotateCcw size={14} />}
              loading={replayingId === row.id}
              disabled={row.status !== 'dead_letter' && row.status !== 'cancelled'}
            >

              Replay
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
        title: 'Channel',
        dataIndex: 'channel',
        width: 110,
        render: (value: string) => <Tag>{value}</Tag>,
      },
      {
        title: 'Recipient',
        dataIndex: 'recipientId',
        width: 110,
        render: (value: number, row) => (
          <Tooltip title={`Receiving destination ${row.targetMasked}`}>
            <Text>User #{value}</Text>
          </Tooltip>
        ),
      },
      {
        title: 'Try',
        dataIndex: 'attempt',
        width: 80,
      },
      {
        title: 'Failure reason',
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
        title: 'Failure time',
        dataIndex: 'createdAt',
        width: 180,
        render: (value: string) => formatTime(value),
      },
      {
        title: 'Actions',
        key: 'actions',
        width: 110,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Popconfirm
            title="Resend this notification?"
            description="The connector will actually be called to send once; if there is already a retry in progress or has been successful, the server will reject it."
            okText="Retry"
            cancelText="Cancel"
            onConfirm={() => handleRetryDelivery(row)}
          >
            <Button
              size='small'
              type='link'
              icon={<RefreshCw size={14} />}
              loading={retryingId === row.id}
            >

              Retry
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

          Dead Letter and Delivery Replay
        </Title>
        <Text type='secondary'>

          The command dead letter retains the original idempotent key for replay, and the failed delivery retry will actually send the message again; AI After the
writing tool is interrupted, the business results must be checked before replaying.
        </Text>
      </div>

      <Row gutter={[16, 16]}>
        <Col xs={12} md={6}>
          <Card size='small'>
            <Statistic
              title="Dead letter command"
              value={summary?.deadLetter ?? 0}
              styles={{ content: { color: (summary?.deadLetter ?? 0) > 0 ? '#cf1322' : undefined } }}
              prefix={<AlertTriangle size={16} />}
            />
          </Card>
        </Col>
        <Col xs={12} md={6}>
          <Card size='small'>
            <Statistic title="Pending" value={summary?.pending ?? 0} />
          </Card>
        </Col>
        <Col xs={12} md={6}>
          <Card size='small'>
            <Statistic title="In Progress" value={summary?.processing ?? 0} />
          </Card>
        </Col>
        <Col xs={12} md={6}>
          <Card size='small'>
            <Statistic
              title="Lease stuck"
              value={summary?.stuckLeases ?? 0}
              styles={{ content: { color: (summary?.stuckLeases ?? 0) > 0 ? '#d46b08' : undefined } }}
            />
          </Card>
        </Col>
      </Row>

      <Alert
        type='info'
        showIcon
        title="Replay boundary"
        description={
          <span>

            "Command Dead Letter" replay retains the original idempotent key, and the worker still deduplicates according to the original identity; if the AI writing tool has produced side effects before being interrupted,
replay may still be executed again, and the business objects and audits must be checked first. "Failed delivery" has been retried in transit or returned when the last retry was successful.
409.
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
              label: 'Command dead letter',
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
                      placeholder="Filter by command type, such as notification.deliver"
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

                      Query
                    </Button>
                    <Button
                      icon={<RefreshCw size={14} />}
                      onClick={() => void loadCommands()}
                      loading={commandsLoading}
                    >

                      Refresh
                    </Button>
                    <Popconfirm
                      title={`Batch replay ${commandTotal} A command?`}
                      description="Match according to the current filtering conditions, with a maximum of 200 entries at a time and retaining idempotent keys; when including the AI writing tool, please first check one by one whether the business data has been written before the interruption."
                      okText="Replay all"
                      cancelText="Cancel"
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

                        Batch replay
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
                    locale={{ emptyText: <Empty description="No matching command" /> }}
                    pagination={{
                      current: commandPage,
                      pageSize: commandPageSize,
                      total: commandTotal,
                      showSizeChanger: true,
                      showTotal: total => `Total ${total} items`,
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

                  Delivery failed
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

                      Refresh
                    </Button>
                    <Text type='secondary'>Total {deliveryTotal}  failed records</Text>
                  </Space>

                  <Table<NotificationDelivery>
                    rowKey='id'
                    size='small'
                    loading={deliveriesLoading}
                    columns={deliveryColumns}
                    dataSource={deliveries}
                    scroll={{ x: 1000 }}
                    locale={{ emptyText: <Empty description="No failed delivery record" /> }}
                    pagination={{
                      current: deliveryPage,
                      pageSize: deliveryPageSize,
                      total: deliveryTotal,
                      showSizeChanger: true,
                      showTotal: total => `Total ${total} items`,
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
