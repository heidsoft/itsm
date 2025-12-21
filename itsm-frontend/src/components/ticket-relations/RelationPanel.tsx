/**
 * 工单关联管理面板
 * 提供工单关联的查看、创建、管理功能
 */

'use client';

import React, { useState } from 'react';
import {
  Card,
  Tabs,
  List,
  Button,
  Space,
  Tag,
  Empty,
  Spin,
  Modal,
  Form,
  Select,
  Input,
  Tooltip,
  Popconfirm,
  Badge,
  message,
} from 'antd';
import {
  LinkOutlined,
  PlusOutlined,
  DeleteOutlined,
  BranchesOutlined,
  SwapOutlined,
  ExclamationCircleOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import {
  useTicketRelationsQuery,
  useCreateRelationMutation,
  useDeleteRelationMutation,
  useRelationStatsQuery,
} from '@/lib/hooks/useTicketRelations';
import { TicketRelationType } from '@/types/ticket-relations';

const { Option } = Select;
const { TextArea } = Input;

export interface RelationPanelProps {
  ticketId: number;
  ticketNumber: string;
  onRelationClick?: (ticketId: number) => void;
}

export const RelationPanel: React.FC<RelationPanelProps> = ({
  ticketId,
  ticketNumber,
  onRelationClick,
}) => {
  const [modalVisible, setModalVisible] = useState(false);
  const [form] = Form.useForm();

  // Queries
  const { data: relations, isLoading } = useTicketRelationsQuery(ticketId, {
    includeDetails: true,
  });
  const { data: stats } = useRelationStatsQuery(ticketId);

  // Mutations
  const createMutation = useCreateRelationMutation();
  const deleteMutation = useDeleteRelationMutation();

  const getRelationTypeLabel = (type: TicketRelationType) => {
    const labels: Record<TicketRelationType, string> = {
      [TicketRelationType.PARENT_CHILD]: '父子关系',
      [TicketRelationType.BLOCKS]: '阻塞',
      [TicketRelationType.BLOCKED_BY]: '被阻塞',
      [TicketRelationType.DEPENDS_ON]: '依赖于',
      [TicketRelationType.RELATES_TO]: '相关',
      [TicketRelationType.DUPLICATES]: '重复',
      [TicketRelationType.DUPLICATED_BY]: '被重复',
      [TicketRelationType.CAUSES]: '导致',
      [TicketRelationType.CAUSED_BY]: '由...导致',
      [TicketRelationType.REPLACES]: '替代',
      [TicketRelationType.REPLACED_BY]: '被替代',
      [TicketRelationType.SPLITS_FROM]: '分离自',
      [TicketRelationType.MERGED_INTO]: '合并到',
    };
    return labels[type] || type;
  };

  const getRelationTypeColor = (type: TicketRelationType) => {
    const colors: Partial<Record<TicketRelationType, string>> = {
      [TicketRelationType.PARENT_CHILD]: 'blue',
      [TicketRelationType.BLOCKS]: 'red',
      [TicketRelationType.BLOCKED_BY]: 'orange',
      [TicketRelationType.DEPENDS_ON]: 'purple',
      [TicketRelationType.RELATES_TO]: 'cyan',
      [TicketRelationType.DUPLICATES]: 'volcano',
    };
    return colors[type] || 'default';
  };

  const handleCreateRelation = async () => {
    try {
      const values = await form.validateFields();
      await createMutation.mutateAsync({
        sourceTicketId: ticketId,
        targetTicketId: values.targetTicketId,
        relationType: values.relationType,
        description: values.description,
      });
      setModalVisible(false);
      form.resetFields();
    } catch (error) {
      console.error('创建关联失败:', error);
    }
  };

  const handleDeleteRelation = async (relationId: string) => {
    try {
      await deleteMutation.mutateAsync({ relationId });
    } catch (error) {
      console.error('删除关联失败:', error);
    }
  };

  const renderRelationsList = () => {
    if (isLoading) {
      return (
        <div className='flex justify-center items-center py-12'>
          <Spin size='large' />
        </div>
      );
    }

    if (!relations || relations.length === 0) {
      return (
        <Empty
          description='暂无关联工单'
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        >
          <Button
            type='primary'
            icon={<PlusOutlined />}
            onClick={() => setModalVisible(true)}
          >
            添加关联
          </Button>
        </Empty>
      );
    }

    return (
      <List
        dataSource={relations}
        renderItem={(relation) => {
          const isSource = relation.sourceTicketId === ticketId;
          const relatedTicket = isSource
            ? relation.targetTicket
            : relation.sourceTicket;

          return (
            <List.Item
              actions={[
                <Tooltip key='view' title='查看工单'>
                  <Button
                    type='link'
                    size='small'
                    onClick={() =>
                      onRelationClick?.(
                        isSource
                          ? relation.targetTicketId
                          : relation.sourceTicketId
                      )
                    }
                  >
                    查看
                  </Button>
                </Tooltip>,
                <Popconfirm
                  key='delete'
                  title='确认删除此关联？'
                  onConfirm={() => handleDeleteRelation(relation.id)}
                  okText='确认'
                  cancelText='取消'
                >
                  <Button
                    type='link'
                    danger
                    size='small'
                    icon={<DeleteOutlined />}
                  >
                    删除
                  </Button>
                </Popconfirm>,
              ]}
            >
              <List.Item.Meta
                avatar={
                  <Tag color={getRelationTypeColor(relation.relationType)}>
                    {getRelationTypeLabel(relation.relationType)}
                  </Tag>
                }
                title={
                  <Space>
                    <span>
                      #{isSource ? relation.targetTicketNumber : relation.sourceTicketNumber}
                    </span>
                    <span>{relatedTicket?.title}</span>
                    {relatedTicket?.status && (
                      <Tag>{relatedTicket.status}</Tag>
                    )}
                  </Space>
                }
                description={relation.description}
              />
            </List.Item>
          );
        }}
      />
    );
  };

  const tabItems = [
    {
      key: 'all',
      label: (
        <Badge count={stats?.totalRelations || 0} showZero>
          <span>全部关联</span>
        </Badge>
      ),
      children: renderRelationsList(),
    },
    {
      key: 'hierarchy',
      label: (
        <Badge count={stats?.childrenCount || 0} showZero>
          <span>父子关系</span>
        </Badge>
      ),
      children: <div>父子关系视图</div>,
    },
    {
      key: 'dependencies',
      label: (
        <Badge count={stats?.blockedByCount || 0} showZero>
          <span>依赖关系</span>
        </Badge>
      ),
      children: <div>依赖关系视图</div>,
    },
  ];

  return (
    <Card
      title={
        <Space>
          <LinkOutlined />
          <span>工单关联</span>
          {stats && stats.totalRelations > 0 && (
            <Badge count={stats.totalRelations} showZero />
          )}
        </Space>
      }
      extra={
        <Button
          type='primary'
          icon={<PlusOutlined />}
          onClick={() => setModalVisible(true)}
          size='small'
        >
          添加关联
        </Button>
      }
    >
      <Tabs items={tabItems} />

      {/* 创建关联模态框 */}
      <Modal
        title='添加工单关联'
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          form.resetFields();
        }}
        onOk={handleCreateRelation}
        confirmLoading={createMutation.isPending}
      >
        <Form form={form} layout='vertical'>
          <Form.Item
            label='关联类型'
            name='relationType'
            rules={[{ required: true, message: '请选择关联类型' }]}
          >
            <Select placeholder='选择关联类型'>
              <Option value={TicketRelationType.RELATES_TO}>相关</Option>
              <Option value={TicketRelationType.BLOCKS}>阻塞</Option>
              <Option value={TicketRelationType.DEPENDS_ON}>依赖于</Option>
              <Option value={TicketRelationType.DUPLICATES}>重复</Option>
              <Option value={TicketRelationType.PARENT_CHILD}>父子关系</Option>
            </Select>
          </Form.Item>

          <Form.Item
            label='目标工单'
            name='targetTicketId'
            rules={[{ required: true, message: '请输入目标工单ID' }]}
          >
            <Input
              type='number'
              placeholder='输入工单ID'
              addonBefore='#'
            />
          </Form.Item>

          <Form.Item label='描述' name='description'>
            <TextArea rows={3} placeholder='添加关联描述（可选）' />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
};

export default RelationPanel;

