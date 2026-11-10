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
import { Plus, Trash2, Link, AlertCircle, CheckCircle, XCircle, GitBranch, ArrowLeftRight } from 'lucide-react';
import {
  useTicketRelationsQuery,
  useCreateRelationMutation,
  useDeleteRelationMutation,
  useRelationStatsQuery,
} from '@/lib/hooks/useTicketRelations';
import { TicketRelationType } from '@/types/ticket-relations';
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
  const { data: rawRelations, isLoading } = useTicketRelationsQuery(ticketId, {
    includeDetails: true,
  });
  const { data: stats } = useRelationStatsQuery(ticketId);

  // Defensive: ensure relations is always an array
  const relations = Array.isArray(rawRelations) ? rawRelations : [];

  // Mutations
  const createMutation = useCreateRelationMutation();
  const deleteMutation = useDeleteRelationMutation();

  const getRelationTypeLabel = (type: TicketRelationType) => {
    const labels: Record<TicketRelationType, string> = {
      [TicketRelationType.PARENT_CHILD]: 'Parent-child relationship',
      [TicketRelationType.BLOCKS]: 'blocking',
      [TicketRelationType.BLOCKED_BY]: 'blocked',
      [TicketRelationType.DEPENDS_ON]: 'Depends on',
      [TicketRelationType.RELATES_TO]: 'Related',
      [TicketRelationType.DUPLICATES]: 'Duplicate',
      [TicketRelationType.DUPLICATED_BY]: 'repeated',
      [TicketRelationType.CAUSES]: 'causes',
      [TicketRelationType.CAUSED_BY]: 'Caused by',
      [TicketRelationType.REPLACES]: 'Replacement',
      [TicketRelationType.REPLACED_BY]: 'replaced',
      [TicketRelationType.SPLITS_FROM]: 'separated from',
      [TicketRelationType.MERGED_INTO]: 'Merge into',
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
      message.error('Failed to create association');
    }
  };

  const handleDeleteRelation = async (relationId: string) => {
    try {
      await deleteMutation.mutateAsync({ relationId });
    } catch (error) {
      message.error('Failed to delete association');
    }
  };

  const renderRelationsList = () => {
    if (isLoading) {
      return (
        <div className="flex justify-center items-center py-12">
          <Spin size="large" />
        </div>
      );
    }

    if (!relations || relations.length === 0) {
      return (
        <Empty description="No associated work orders yet" image={Empty.PRESENTED_IMAGE_SIMPLE}>
          <Button type="primary" icon={<Plus />} onClick={() => setModalVisible(true)}>

            Add association
          </Button>
        </Empty>
      );
    }

    return (
      <List
        dataSource={relations}
        renderItem={relation => {
          const isSource = relation.sourceTicketId === ticketId;
          const relatedTicket = isSource ? relation.targetTicket : relation.sourceTicket;

          return (
            <List.Item
              actions={[
                <Tooltip key="view" title="View work order">
                  <Button
                    type="link"
                    size="small"
                    onClick={() =>
                      onRelationClick?.(
                        isSource ? relation.targetTicketId : relation.sourceTicketId
                      )
                    }
                  >

                    View
                  </Button>
                </Tooltip>,
                <Popconfirm
                  key="delete"
                  title="Confirm to delete this association?"
                  onConfirm={() => handleDeleteRelation(relation.id)}
                  okText="Confirm"
                  cancelText="Cancel"
                >
                  <Button type="link" danger size="small" icon={<Trash2 />}>

                    Delete
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
                    {relatedTicket?.status && <Tag>{relatedTicket.status}</Tag>}
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
          <span>All related</span>
        </Badge>
      ),
      children: renderRelationsList(),
    },
    {
      key: 'hierarchy',
      label: (
        <Badge count={stats?.childrenCount || 0} showZero>
          <span>Parent-child relationship</span>
        </Badge>
      ),
      children: <div>Parent-child relationship view</div>,
    },
    {
      key: 'dependencies',
      label: (
        <Badge count={stats?.blockedByCount || 0} showZero>
          <span>Dependencies</span>
        </Badge>
      ),
      children: <div>Dependency view</div>,
    },
  ];

  return (
    <Card
      title={
        <Space>
          <Link />
          <span>Association</span>
          {stats && stats.totalRelations > 0 && <Badge count={stats.totalRelations} showZero />}
        </Space>
      }
      extra={
        <Button
          type="primary"
          icon={<Plus />}
          onClick={() => setModalVisible(true)}
          size="small"
        >

          Add association
        </Button>
      }
    >
      <Tabs items={tabItems} />

      {/* 创建关联模态框 */}
      <Modal
        title="Add work order association"
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          form.resetFields();
        }}
        onOk={handleCreateRelation}
        confirmLoading={createMutation.isPending}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            label="Relationship Type"
            name="relationType"
            rules={[{ required: true, message: 'Please select the association type' }]}
          >
            <Select placeholder="Select association type" options={[{ value: TicketRelationType.RELATES_TO, label: "Related" }, { value: TicketRelationType.BLOCKS, label: "blocking" }, { value: TicketRelationType.DEPENDS_ON, label: "Depends on" }, { value: TicketRelationType.DUPLICATES, label: "Duplicate" }, { value: TicketRelationType.PARENT_CHILD, label: "Parent-child relationship" }]} />
          </Form.Item>

          <Form.Item
            label="Target work order"
            name="targetTicketId"
            rules={[{ required: true, message: 'Please enter the target work order ID' }]}
          >
            <Input type="number" placeholder="Enter the work order ID" addonBefore="#" />
          </Form.Item>

          <Form.Item label="Description" name="description">
            <TextArea rows={3} placeholder="Add association description (optional)" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
};

export default RelationPanel;
