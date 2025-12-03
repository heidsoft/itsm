'use client';

import React from 'react';
import { Modal, Typography, List, Tag, Space, Divider } from 'antd';
import { ExclamationCircleOutlined, CheckCircleOutlined } from '@ant-design/icons';
import { Ticket } from '@/lib/services/ticket-service';

const { Text, Paragraph } = Typography;

export type BatchOperationType = 
  | 'delete' 
  | 'assign' 
  | 'updateStatus' 
  | 'updatePriority' 
  | 'export' 
  | 'archive';

export interface BatchOperationConfirmProps {
  visible: boolean;
  operationType: BatchOperationType;
  selectedCount: number;
  selectedTickets?: Ticket[];
  operationData?: {
    assignee?: string;
    status?: string;
    priority?: string;
    [key: string]: any;
  };
  onConfirm: () => Promise<void>;
  onCancel: () => void;
  loading?: boolean;
}

const OPERATION_CONFIG: Record<
  BatchOperationType,
  {
    title: string;
    icon: React.ReactNode;
    description: string;
    danger?: boolean;
    getContent: (props: BatchOperationConfirmProps) => React.ReactNode;
  }
> = {
  delete: {
    title: '确认批量删除',
    icon: <ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />,
    description: '此操作将永久删除选中的工单，且无法恢复',
    danger: true,
    getContent: (props) => (
      <div>
        <Paragraph>
          您即将删除 <Text strong>{props.selectedCount}</Text> 个工单，此操作不可撤销。
        </Paragraph>
        {props.selectedTickets && props.selectedTickets.length > 0 && (
          <div>
            <Text type="secondary" className="text-sm">待删除的工单：</Text>
            <List
              size="small"
              dataSource={props.selectedTickets.slice(0, 10)}
              renderItem={(ticket) => (
                <List.Item>
                  <Space>
                    <Text strong>#{ticket.id}</Text>
                    <Text>{ticket.title}</Text>
                    <Tag color="red">删除</Tag>
                  </Space>
                </List.Item>
              )}
              style={{ marginTop: 8, maxHeight: 200, overflow: 'auto' }}
            />
            {props.selectedTickets.length > 10 && (
              <Text type="secondary" className="text-xs">
                还有 {props.selectedTickets.length - 10} 个工单...
              </Text>
            )}
          </div>
        )}
      </div>
    ),
  },
  assign: {
    title: '确认批量分配',
    icon: <CheckCircleOutlined style={{ color: '#1890ff' }} />,
    description: '将选中的工单分配给指定处理人',
    getContent: (props) => (
      <div>
        <Paragraph>
          您即将将 <Text strong>{props.selectedCount}</Text> 个工单分配给：
          <Text strong style={{ color: '#1890ff' }}>
            {props.operationData?.assignee || '未指定'}
          </Text>
        </Paragraph>
        {props.selectedTickets && props.selectedTickets.length > 0 && (
          <div>
            <Text type="secondary" className="text-sm">待分配的工单：</Text>
            <List
              size="small"
              dataSource={props.selectedTickets.slice(0, 10)}
              renderItem={(ticket) => (
                <List.Item>
                  <Space>
                    <Text strong>#{ticket.id}</Text>
                    <Text>{ticket.title}</Text>
                    <Tag color="blue">分配</Tag>
                  </Space>
                </List.Item>
              )}
              style={{ marginTop: 8, maxHeight: 200, overflow: 'auto' }}
            />
            {props.selectedTickets.length > 10 && (
              <Text type="secondary" className="text-xs">
                还有 {props.selectedTickets.length - 10} 个工单...
              </Text>
            )}
          </div>
        )}
      </div>
    ),
  },
  updateStatus: {
    title: '确认批量更新状态',
    icon: <CheckCircleOutlined style={{ color: '#52c41a' }} />,
    description: '批量更新选中工单的状态',
    getContent: (props) => (
      <div>
        <Paragraph>
          您即将将 <Text strong>{props.selectedCount}</Text> 个工单的状态更新为：
          <Tag color="green" style={{ marginLeft: 8 }}>
            {props.operationData?.status || '未指定'}
          </Tag>
        </Paragraph>
        {props.selectedTickets && props.selectedTickets.length > 0 && (
          <div>
            <Text type="secondary" className="text-sm">待更新的工单：</Text>
            <List
              size="small"
              dataSource={props.selectedTickets.slice(0, 10)}
              renderItem={(ticket) => (
                <List.Item>
                  <Space>
                    <Text strong>#{ticket.id}</Text>
                    <Text>{ticket.title}</Text>
                    <Tag color="green">更新状态</Tag>
                  </Space>
                </List.Item>
              )}
              style={{ marginTop: 8, maxHeight: 200, overflow: 'auto' }}
            />
            {props.selectedTickets.length > 10 && (
              <Text type="secondary" className="text-xs">
                还有 {props.selectedTickets.length - 10} 个工单...
              </Text>
            )}
          </div>
        )}
      </div>
    ),
  },
  updatePriority: {
    title: '确认批量更新优先级',
    icon: <CheckCircleOutlined style={{ color: '#faad14' }} />,
    description: '批量更新选中工单的优先级',
    getContent: (props) => (
      <div>
        <Paragraph>
          您即将将 <Text strong>{props.selectedCount}</Text> 个工单的优先级更新为：
          <Tag color="orange" style={{ marginLeft: 8 }}>
            {props.operationData?.priority || '未指定'}
          </Tag>
        </Paragraph>
        {props.selectedTickets && props.selectedTickets.length > 0 && (
          <div>
            <Text type="secondary" className="text-sm">待更新的工单：</Text>
            <List
              size="small"
              dataSource={props.selectedTickets.slice(0, 10)}
              renderItem={(ticket) => (
                <List.Item>
                  <Space>
                    <Text strong>#{ticket.id}</Text>
                    <Text>{ticket.title}</Text>
                    <Tag color="orange">更新优先级</Tag>
                  </Space>
                </List.Item>
              )}
              style={{ marginTop: 8, maxHeight: 200, overflow: 'auto' }}
            />
            {props.selectedTickets.length > 10 && (
              <Text type="secondary" className="text-xs">
                还有 {props.selectedTickets.length - 10} 个工单...
              </Text>
            )}
          </div>
        )}
      </div>
    ),
  },
  export: {
    title: '确认导出',
    icon: <CheckCircleOutlined style={{ color: '#1890ff' }} />,
    description: '导出选中的工单数据',
    getContent: (props) => (
      <div>
        <Paragraph>
          您即将导出 <Text strong>{props.selectedCount}</Text> 个工单的数据。
        </Paragraph>
        <Paragraph type="secondary" className="text-sm">
          导出格式：Excel (.xlsx)
        </Paragraph>
      </div>
    ),
  },
  archive: {
    title: '确认批量归档',
    icon: <CheckCircleOutlined style={{ color: '#722ed1' }} />,
    description: '将选中的工单归档',
    getContent: (props) => (
      <div>
        <Paragraph>
          您即将归档 <Text strong>{props.selectedCount}</Text> 个工单。
        </Paragraph>
        <Paragraph type="secondary" className="text-sm">
          归档后的工单将移至归档列表，但仍可查看和恢复。
        </Paragraph>
      </div>
    ),
  },
};

export const BatchOperationConfirm: React.FC<BatchOperationConfirmProps> = ({
  visible,
  operationType,
  selectedCount,
  selectedTickets,
  operationData,
  onConfirm,
  onCancel,
  loading = false,
}) => {
  const config = OPERATION_CONFIG[operationType];

  if (!config) {
    return null;
  }

  return (
    <Modal
      open={visible}
      title={
        <Space>
          {config.icon}
          <span>{config.title}</span>
        </Space>
      }
      onOk={onConfirm}
      onCancel={onCancel}
      confirmLoading={loading}
      okText="确认"
      cancelText="取消"
      okButtonProps={{
        danger: config.danger,
        type: config.danger ? 'primary' : 'default',
      }}
      width={600}
      maskClosable={false}
    >
      <div style={{ marginTop: 16 }}>
        <Paragraph type="secondary">{config.description}</Paragraph>
        <Divider style={{ margin: '16px 0' }} />
        {config.getContent({
          visible,
          operationType,
          selectedCount,
          selectedTickets,
          operationData,
          onConfirm,
          onCancel,
          loading,
        })}
      </div>
    </Modal>
  );
};

