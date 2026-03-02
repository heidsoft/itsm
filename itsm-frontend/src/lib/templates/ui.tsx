/**
 * UI 组件模板 - 常用 UI 组件封装
 */

'use client';

import React, { useState, useCallback } from 'react';
import {
  Card,
  Table,
  Button,
  Tag,
  Space,
  Modal,
  message,
  Popconfirm,
  Dropdown,
  type MenuProps,
  Tooltip,
  Badge,
  Avatar,
  Empty,
  Spin,
  Alert,
  Drawer,
  Descriptions,
  List,
  Rate,
  Progress,
  Tabs,
  Collapse,
  Divider,
  Typography,
} from 'antd';
import {
  ExclamationCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  MoreOutlined,
  ReloadOutlined,
  DownloadOutlined,
  UploadOutlined,
  SearchOutlined,
  PlusOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import type { FilterValue, SorterResult } from 'antd/es/table/interface';

// ============ 标准表格组件 ============

interface StandardTableProps<T> {
  columns: ColumnsType<T>;
  dataSource: T[];
  loading?: boolean;
  rowKey?: string | ((record: T) => string);
  pagination?: TablePaginationConfig | false;
  onChange?: (
    pagination: TablePaginationConfig,
    filters: Record<string, FilterValue | null>,
    sorter: SorterResult<T> | SorterResult<T>[]
  ) => void;
  onRowClick?: (record: T) => void;
  headerActions?: React.ReactNode;
  title?: React.ReactNode;
  emptyMessage?: string;
}

export function StandardTable<T extends { id: number | string }>({
  columns,
  dataSource,
  loading = false,
  rowKey = 'id',
  pagination = { pageSize: 10, showSizeChanger: true, showQuickJumper: true },
  onChange,
  onRowClick,
  headerActions,
  title,
  emptyMessage = '暂无数据',
}: StandardTableProps<T>) {
  return (
    <Card title={title} extra={<Space>{headerActions}</Space>}>
      <Table<T>
        columns={columns}
        dataSource={dataSource}
        loading={loading}
        rowKey={rowKey}
        pagination={pagination}
        onChange={onChange}
        onRow={record => ({
          onClick: () => onRowClick?.(record),
          style: onRowClick ? { cursor: 'pointer' } : undefined,
        })}
        locale={{ emptyText: <Empty description={emptyMessage} /> }}
      />
    </Card>
  );
}

// ============ 操作列组件 ============

interface ActionColumnProps {
  width?: number;
  maxVisible?: number;
  render?: (record: { id: number | string }) => React.ReactNode;
}

export function createActionColumn({
  width = 150,
  maxVisible = 3,
  render,
}: ActionColumnProps = {}) {
  return {
    title: '操作',
    width,
    fixed: 'right' as const,
    render: (_: unknown, record: { id: number | string }) => {
      const actions: { key: string; label: string; icon: React.ReactNode; danger?: boolean }[] = [];

      if (render) {
        const customActions = render(record);
        if (Array.isArray(customActions)) {
          actions.push(...customActions);
        } else {
          return customActions;
        }
      } else {
        // 默认操作
        actions.push(
          { key: 'view', label: '查看', icon: <EyeOutlined /> },
          { key: 'edit', label: '编辑', icon: <EditOutlined /> },
          { key: 'delete', label: '删除', icon: <DeleteOutlined />, danger: true }
        );
      }

      const visibleActions = maxVisible > 0 ? actions.slice(0, maxVisible) : actions;
      const moreActions =
        maxVisible > 0 && actions.length > maxVisible ? actions.slice(maxVisible) : [];

      return (
        <Space size="small">
          {visibleActions.map(action =>
            action.key === 'delete' ? (
              <Popconfirm key={action.key} title="确定要删除吗？" okText="确定" cancelText="取消">
                <Button type="link" danger size="small" icon={action.icon}>
                  {action.label}
                </Button>
              </Popconfirm>
            ) : (
              <Button
                key={action.key}
                type="link"
                size="small"
                icon={action.icon}
                danger={action.danger}
              >
                {action.label}
              </Button>
            )
          )}
          {moreActions.length > 0 && (
            <Dropdown
              menu={{ items: moreActions.map(a => ({ ...a, danger: undefined })) }}
              trigger={['click']}
            >
              <Button type="link" size="small" icon={<MoreOutlined />} />
            </Dropdown>
          )}
        </Space>
      );
    },
  };
}

// ============ 状态标签组件 ============

interface StatusTagProps {
  status: string;
  mapping: Record<string, { color: string; label: string }>;
}

export function StatusTag({ status, mapping }: StatusTagProps) {
  const config = mapping[status] || { color: 'default', label: status };

  return <Tag color={config.color}>{config.label}</Tag>;
}

// ============ 确认对话框组件 ============

interface ConfirmDialogProps {
  title?: string;
  content?: string;
  onConfirm?: () => void;
  onCancel?: () => void;
  okText?: string;
  cancelText?: string;
  danger?: boolean;
}

export function useConfirmDialog() {
  const [modal, contextHolder] = Modal.useModal();

  const confirm = useCallback(
    (options: ConfirmDialogProps) => {
      const {
        title = '确认操作',
        content = '确定要执行此操作吗？',
        onConfirm,
        onCancel,
        okText = '确定',
        cancelText = '取消',
        danger = false,
      } = options;

      modal.confirm({
        title,
        content,
        okText,
        cancelText,
        okButtonProps: { danger },
        onOk: onConfirm,
        onCancel,
        icon: <ExclamationCircleOutlined />,
      });
    },
    [modal]
  );

  const success = useCallback(
    (options: { title?: string; content?: string; onOk?: () => void }) => {
      const { title = '操作成功', content, onOk } = options;
      modal.success({ title, content, onOk });
    },
    [modal]
  );

  const error = useCallback(
    (options: { title?: string; content?: string; onOk?: () => void }) => {
      const { title = '操作失败', content, onOk } = options;
      modal.error({ title, content, onOk });
    },
    [modal]
  );

  const info = useCallback(
    (options: { title?: string; content?: string; onOk?: () => void }) => {
      const { title, content, onOk } = options;
      modal.info({ title, content, onOk });
    },
    [modal]
  );

  return {
    confirm,
    success,
    error,
    info,
    contextHolder,
  };
}

// ============ 详情抽屉组件 ============

interface DetailDrawerProps {
  open: boolean;
  onClose: () => void;
  title?: string;
  width?: number | string;
  children: React.ReactNode;
}

export function DetailDrawer({
  open,
  onClose,
  title = '详情',
  width = 600,
  children,
}: DetailDrawerProps) {
  return (
    <Drawer title={title} width={width} open={open} onClose={onClose} footer={null}>
      {children}
    </Drawer>
  );
}

// ============ 详情描述组件 ============

interface DetailDescriptionsProps {
  title?: string;
  items: {
    label: string;
    span?: number;
    children: React.ReactNode;
  }[];
  column?: number | { xs: number; sm: number; md: number; lg: number; xl: number };
}

export function DetailDescriptions({ title, items, column = 2 }: DetailDescriptionsProps) {
  return (
    <Descriptions title={title} column={column} bordered size="small">
      {items.map((item, index) => (
        <Descriptions.Item key={index} label={item.label} span={item.span}>
          {item.children}
        </Descriptions.Item>
      ))}
    </Descriptions>
  );
}

// ============ 空状态组件 ============

interface EmptyStateProps {
  description?: string;
  action?: React.ReactNode;
  image?: React.ReactNode;
}

export function EmptyState({ description = '暂无数据', action, image }: EmptyStateProps) {
  return (
    <Empty description={description} image={image}>
      {action}
    </Empty>
  );
}

// ============ 加载状态组件 ============

interface LoadingStateProps {
  tip?: string;
  children?: React.ReactNode;
}

export function LoadingState({ tip = '加载中...', children }: LoadingStateProps) {
  return (
    <Spin tip={tip} size="large">
      <div style={{ padding: 50 }}>{children}</div>
    </Spin>
  );
}

// ============ 错误状态组件 ============

interface ErrorStateProps {
  message?: string;
  description?: string;
  onRetry?: () => void;
}

export function ErrorState({ message = '出错了', description, onRetry }: ErrorStateProps) {
  return (
    <Alert
      type="error"
      showIcon
      message={message}
      description={
        description || (
          <span>
            加载失败，
            {onRetry && (
              <Button type="link" onClick={onRetry} size="small">
                重试
              </Button>
            )}
          </span>
        )
      }
      action={
        onRetry && (
          <Button size="small" danger onClick={onRetry}>
            重试
          </Button>
        )
      }
    />
  );
}

// ============ 评分组件 ============

interface RatingDisplayProps {
  value: number;
  allowHalf?: boolean;
  disabled?: boolean;
  onChange?: (value: number) => void;
}

export function RatingDisplay({
  value,
  allowHalf = true,
  disabled = true,
  onChange,
}: RatingDisplayProps) {
  return <Rate value={value} allowHalf={allowHalf} disabled={disabled} onChange={onChange} />;
}

// ============ 进度条组件 ============

interface ProgressBarProps {
  percent: number;
  status?: 'success' | 'exception' | 'active' | 'normal';
  showInfo?: boolean;
  strokeColor?: string;
  size?: 'small' | 'default';
}

export function ProgressBar({
  percent,
  status = 'active',
  showInfo = true,
  strokeColor,
  size = 'default',
}: ProgressBarProps) {
  return (
    <Progress
      percent={percent}
      status={status}
      showInfo={showInfo}
      strokeColor={strokeColor}
      size={size}
    />
  );
}

// ============ 头像组组件 ============

interface AvatarGroupProps {
  users: { name: string; avatar?: string }[];
  maxCount?: number;
  size?: number;
}

export function AvatarGroup({ users, maxCount = 5, size = 32 }: AvatarGroupProps) {
  const visibleUsers = users.slice(0, maxCount);
  const remainingCount = users.length - maxCount;

  return (
    <Avatar.Group maxCount={remainingCount > 0 ? remainingCount : undefined} size={size}>
      {visibleUsers.map((user, index) => (
        <Tooltip key={index} title={user.name}>
          <Avatar src={user.avatar} icon={<SettingOutlined />}>
            {user.name.charAt(0).toUpperCase()}
          </Avatar>
        </Tooltip>
      ))}
    </Avatar.Group>
  );
}

// ============ 标签列表组件 ============

interface TagListProps {
  tags: string[];
  colorMap?: Record<string, string>;
  closable?: boolean;
  onClose?: (tag: string) => void;
}

export function TagList({ tags, colorMap = {}, closable = false, onClose }: TagListProps) {
  return (
    <Space wrap>
      {tags.map(tag => (
        <Tag
          key={tag}
          color={colorMap[tag] || 'default'}
          closable={closable}
          onClose={() => onClose?.(tag)}
        >
          {tag}
        </Tag>
      ))}
    </Space>
  );
}

// ============ 时间线组件 ============

interface TimelineItem {
  time: string;
  title: string;
  description?: string;
  color?: 'blue' | 'red' | 'green' | 'gray';
}

interface TimelineListProps {
  items: TimelineItem[];
}

export function TimelineList({ items }: TimelineListProps) {
  return (
    <Collapse
      defaultActiveKey={[0]}
      expandIconPosition="end"
      items={items.map((item, index) => ({
        key: index,
        label: (
          <Space>
            <Badge color={item.color || 'blue'} />
            <Typography.Text strong>{item.title}</Typography.Text>
            <Typography.Text type="secondary">{item.time}</Typography.Text>
          </Space>
        ),
        children: <Typography.Paragraph type="secondary">{item.description}</Typography.Paragraph>,
      }))}
    />
  );
}

// ============ 分割线标签组件 ============

interface SectionProps {
  title?: string;
  children: React.ReactNode;
  extra?: React.ReactNode;
}

export function Section({ title, children, extra }: SectionProps) {
  return (
    <div style={{ marginBottom: 24 }}>
      {title && (
        <div style={{ marginBottom: 16 }}>
          <Typography.Title level={5} style={{ margin: 0 }}>
            {title}
          </Typography.Title>
          {extra && <span style={{ marginLeft: 16 }}>{extra}</span>}
        </div>
      )}
      {children}
      {title && <Divider style={{ margin: '12px 0' }} />}
    </div>
  );
}

// ============ 卡片列表组件 ============

interface CardListProps {
  title?: string;
  items: {
    title: string;
    description?: string;
    actions?: React.ReactNode[];
    extra?: React.ReactNode;
  }[];
  loading?: boolean;
  grid?: { column?: number; gutter?: number };
  onItemClick?: (item: unknown) => void;
}

export function CardList({
  title,
  items,
  loading = false,
  grid = { column: 3, gutter: 16 },
  onItemClick,
}: CardListProps) {
  return (
    <Card title={title}>
      <List
        loading={loading}
        grid={grid}
        dataSource={items}
        renderItem={item => (
          <List.Item
            onClick={() => onItemClick?.(item)}
            style={{ cursor: onItemClick ? 'pointer' : 'default' }}
            actions={item.actions}
          >
            <List.Item.Meta title={item.title} description={item.description} />
            {item.extra && <div style={{ marginLeft: 16 }}>{item.extra}</div>}
          </List.Item>
        )}
      />
    </Card>
  );
}

// ============ 使用示例 ============
/*
// 1. 标准表格
<StandardTable
  columns={columns}
  dataSource={data}
  loading={loading}
  onChange={handleTableChange}
  onRowClick={(record) => openDetail(record.id)}
  title="用户列表"
  headerActions={
    <Button type="primary" icon={<PlusOutlined />}>
      新增
    </Button>
  }
/>

// 2. 状态标签
<StatusTag
  status={record.status}
  mapping={{
    pending: { color: 'orange', label: '待处理' },
    done: { color: 'green', label: '已完成' },
    failed: { color: 'red', label: '失败' },
  }}
/>

// 3. 确认对话框
const { confirm, contextHolder } = useConfirmDialog();

const handleDelete = () => {
  confirm({
    title: '确认删除',
    content: '删除后无法恢复，是否继续？',
    danger: true,
    onConfirm: async () => {
      await deleteApi(id);
      message.success('删除成功');
    },
  });
};

// 4. 详情抽屉
<DetailDrawer
  open={drawerVisible}
  onClose={() => setDrawerVisible(false)}
  title="用户详情"
>
  <DetailDescriptions
    items={[
      { label: '姓名', children: user.name },
      { label: '邮箱', children: user.email },
      { label: '状态', children: <StatusTag {...} /> },
    ]}
  />
</DetailDrawer>

// 5. 空状态
<EmptyState
  description="暂无数据"
  action={
    <Button type="primary" icon={<PlusOutlined />}>
      新建
    </Button>
  }
/>

// 6. 错误状态
<ErrorState
  message="加载失败"
  description="请检查网络连接后重试"
  onRetry={refetch}
/>

// 7. 操作列
const columns: ColumnsType<User> = [
  { title: '姓名', dataIndex: 'name' },
  createActionColumn({
    maxVisible: 2,
    render: (record) => [
      { key: 'view', label: '查看', icon: <EyeOutlined /> },
      { key: 'edit', label: '编辑', icon: <EditOutlined /> },
      { key: 'delete', label: '删除', icon: <DeleteOutlined /> },
    ],
  }),
];
*/
