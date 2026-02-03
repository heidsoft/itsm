'use client';
// @ts-nocheck

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import {
  Card,
  Table,
  Button,
  Modal,
  Form,
  Input,
  Select,
  Space,
  Typography,
  Tag,
  Badge,
  Alert,
  Row,
  Col,
  Tooltip,
  Popconfirm,
  message,
  App,
  Tabs,
  Statistic,
  Tree,
  Empty,
  Divider,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  EyeOutlined,
  LinkOutlined,
  BranchesOutlined,
  WarningOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  ShareAltOutlined,
  BarChartOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import { Ticket } from '@/lib/services/ticket-service';
import { TicketRelationsApi } from '@/lib/api/ticket-relations-api';
import { TicketRelationType } from '@/types/ticket-relations';

const { Title, Text } = Typography;
const { TextArea } = Input;
const { Option } = Select;

interface TicketDependency {
  id: number;
  source_ticket_id: number;
  source_ticket_number: string;
  source_ticket_title: string;
  target_ticket_id: number;
  target_ticket_number: string;
  target_ticket_title: string;
  relation_type: TicketRelationType;
  dependency_type: 'hard' | 'soft';
  is_blocking: boolean;
  description?: string;
  created_at: string;
  created_by: number;
  created_by_name: string;
}

interface DependencyImpact {
  ticket_id: number;
  ticket_number: string;
  ticket_title: string;
  impact_level: 'high' | 'medium' | 'low';
  impact_type: 'blocked' | 'delayed' | 'affected';
  affected_fields: string[];
  estimated_delay_hours?: number;
}

interface TicketDependencyManagerProps {
  ticket: Ticket;
  onDependencyChange?: () => void;
  canManage?: boolean;
}

export const TicketDependencyManager: React.FC<TicketDependencyManagerProps> = ({
  ticket,
  onDependencyChange,
  canManage = true,
}) => {
  const { message: antMessage } = App.useApp();
  const [dependencies, setDependencies] = useState<TicketDependency[]>([]);
  const [impactAnalysis, setImpactAnalysis] = useState<DependencyImpact[]>([]);
  const [loading, setLoading] = useState(false);
  const [dependencyModalVisible, setDependencyModalVisible] = useState(false);
  const [editingDependency, setEditingDependency] = useState<TicketDependency | null>(null);
  const [graphModalVisible, setGraphModalVisible] = useState(false);
  const [activeTab, setActiveTab] = useState<'dependencies' | 'impact' | 'graph' | 'stats'>(
    'dependencies'
  );
  const [form] = Form.useForm();

  // 加载依赖关系
  const loadDependencies = useCallback(async () => {
    if (!ticket?.id) return;

    try {
      setLoading(true);
      // 调用实际API
      const relations = await TicketRelationsApi.getTicketRelations(ticket.id, {
        direction: 'outbound',
        includeDetails: true,
      });

      // 转换为依赖关系格式
      const deps: TicketDependency[] = relations.map(rel => ({
        id: parseInt(rel.id) || Date.now(),
        source_ticket_id: ticket.id,
        source_ticket_number: ticket.ticket_number || `T-${ticket.id}`,
        source_ticket_title: ticket.title,
        target_ticket_id: rel.targetTicket.id,
        target_ticket_number: rel.targetTicket.ticketNumber || `T-${rel.targetTicket.id}`,
        target_ticket_title: rel.targetTicket.title,
        relation_type: rel.relationType as TicketRelationType,
        dependency_type: rel.metadata?.dependency_type === 'hard' ? 'hard' : 'soft',
        is_blocking: rel.metadata?.is_blocking === true,
        description: rel.description,
        created_at: rel.createdAt,
        created_by: rel.createdBy?.id || 0,
        created_by_name: rel.createdBy?.name || '系统',
      }));

      setDependencies(deps);

      // 加载影响分析
      loadImpactAnalysis(deps);

      if (relations.length === 0) {
      }
    } catch (error) {
      console.error('Failed to load dependencies:', error);
      antMessage.error('加载依赖关系失败');
    } finally {
      setLoading(false);
    }
  }, [ticket, antMessage]);

  // 加载影响分析
  const loadImpactAnalysis = useCallback(
    async (deps: TicketDependency[] = dependencies) => {
      if (!ticket?.id) return;

      try {
        // 注意：影响分析API尚未实现
        // 未来可通过 TicketRelationsApi.analyzeImpact(ticket.id) 获取实际数据
        
        // 暂时返回空数据，禁止使用mock数据
        setImpactAnalysis([]);
      } catch (error) {
        console.error('Failed to load impact analysis:', error);
      }
    },
    [dependencies]
  );

  useEffect(() => {
    loadDependencies();
  }, [loadDependencies]);

  // 保存依赖关系
  const handleSaveDependency = useCallback(async () => {
    try {
      const values = await form.validateFields();

      if (editingDependency) {
        // 更新依赖关系
        await TicketRelationsApi.updateRelation(String(editingDependency.id), {
          relationType: values.relation_type as any,
          description: values.description,
          metadata: {
            dependency_type: values.dependency_type,
            is_blocking: values.is_blocking,
          },
        });
        antMessage.success('依赖关系已更新');
      } else {
        // 创建依赖关系
        await TicketRelationsApi.createRelation({
          sourceTicketId: ticket.id,
          targetTicketId: values.target_ticket_id,
          relationType: values.relation_type as any,
          description: values.description,
          metadata: {
            dependency_type: values.dependency_type,
            is_blocking: values.is_blocking,
          },
        });
        antMessage.success('依赖关系已创建');
      }

      setDependencyModalVisible(false);
      setEditingDependency(null);
      form.resetFields();
      loadDependencies();
      onDependencyChange?.();
    } catch (error) {
      console.error('Failed to save dependency:', error);
      antMessage.error('保存失败');
    }
  }, [form, editingDependency, ticket, antMessage, loadDependencies, onDependencyChange]);

  // 删除依赖关系
  const handleDeleteDependency = useCallback(
    async (id: number) => {
      try {
        // 调用删除API
        await TicketRelationsApi.deleteRelation(String(id), '用户删除');
        antMessage.success('依赖关系已删除');
        loadDependencies();
        onDependencyChange?.();
      } catch (error) {
        console.error('Failed to delete dependency:', error);
        antMessage.error('删除失败');
      }
    },
    [antMessage, loadDependencies, onDependencyChange]
  );

  // 获取关系类型文本
  const getRelationTypeText = (type: TicketRelationType) => {
    const typeMap: Record<TicketRelationType, string> = {
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
    return typeMap[type] || type;
  };

  // 获取关系类型颜色
  const getRelationTypeColor = (type: TicketRelationType) => {
    const colorMap: Record<TicketRelationType, string> = {
      [TicketRelationType.PARENT_CHILD]: 'blue',
      [TicketRelationType.BLOCKS]: 'red',
      [TicketRelationType.BLOCKED_BY]: 'orange',
      [TicketRelationType.DEPENDS_ON]: 'purple',
      [TicketRelationType.RELATES_TO]: 'cyan',
      [TicketRelationType.DUPLICATES]: 'default',
      [TicketRelationType.DUPLICATED_BY]: 'default',
      [TicketRelationType.CAUSES]: 'red',
      [TicketRelationType.CAUSED_BY]: 'orange',
      [TicketRelationType.REPLACES]: 'green',
      [TicketRelationType.REPLACED_BY]: 'green',
      [TicketRelationType.SPLITS_FROM]: 'geekblue',
      [TicketRelationType.MERGED_INTO]: 'geekblue',
    };
    return colorMap[type] || 'default';
  };

  // 获取影响级别颜色
  const getImpactLevelColor = (level: string) => {
    const colorMap: Record<string, string> = {
      high: 'red',
      medium: 'orange',
      low: 'green',
    };
    return colorMap[level] || 'default';
  };

  // 依赖关系表格列
  const dependencyColumns: ColumnsType<TicketDependency> = [
    {
      title: '关系类型',
      dataIndex: 'relation_type',
      key: 'relation_type',
      render: (type: TicketRelationType) => (
        <Tag color={getRelationTypeColor(type)}>{getRelationTypeText(type)}</Tag>
      ),
    },
    {
      title: '目标工单',
      key: 'target_ticket',
      render: (_: any, record: TicketDependency) => (
        <div>
          <Text strong style={{ color: '#1890ff' }}>
            {record.target_ticket_number}
          </Text>
          <br />
          <Text type='secondary' className='text-sm'>
            {record.target_ticket_title}
          </Text>
        </div>
      ),
    },
    {
      title: '依赖类型',
      dataIndex: 'dependency_type',
      key: 'dependency_type',
      render: (type: string) => (
        <Tag color={type === 'hard' ? 'red' : 'orange'}>
          {type === 'hard' ? '硬依赖' : '软依赖'}
        </Tag>
      ),
    },
    {
      title: '阻塞状态',
      dataIndex: 'is_blocking',
      key: 'is_blocking',
      render: (isBlocking: boolean) => (
        <Badge status={isBlocking ? 'error' : 'success'} text={isBlocking ? '阻塞中' : '未阻塞'} />
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (date: string) => format(new Date(date), 'yyyy-MM-dd HH:mm:ss', { locale: zhCN }),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: TicketDependency) => (
        <Space>
          <Tooltip title='查看工单详情'>
            <Button
              type='link'
              size='small'
              icon={<EyeOutlined />}
              onClick={() => {
                // 跳转到工单详情页
                window.open(`/tickets/${record.target_ticket_id}`, '_blank');
              }}
            >
              查看
            </Button>
          </Tooltip>
          {canManage && (
            <>
              <Button
                type='link'
                size='small'
                icon={<EditOutlined />}
                onClick={() => {
                  setEditingDependency(record);
                  form.setFieldsValue(record);
                  setDependencyModalVisible(true);
                }}
              >
                编辑
              </Button>
              <Popconfirm
                title='确定要删除这个依赖关系吗？'
                onConfirm={() => handleDeleteDependency(record.id)}
              >
                <Button type='link' size='small' danger icon={<DeleteOutlined />}>
                  删除
                </Button>
              </Popconfirm>
            </>
          )}
        </Space>
      ),
    },
  ];

  // 影响分析表格列
  const impactColumns: ColumnsType<DependencyImpact> = [
    {
      title: '受影响工单',
      key: 'ticket',
      render: (_: any, record: DependencyImpact) => (
        <div>
          <Text strong style={{ color: '#1890ff' }}>
            {record.ticket_number}
          </Text>
          <br />
          <Text type='secondary' className='text-sm'>
            {record.ticket_title}
          </Text>
        </div>
      ),
    },
    {
      title: '影响级别',
      dataIndex: 'impact_level',
      key: 'impact_level',
      render: (level: string) => (
        <Tag color={getImpactLevelColor(level)}>
          {level === 'high' ? '高' : level === 'medium' ? '中' : '低'}
        </Tag>
      ),
    },
    {
      title: '影响类型',
      dataIndex: 'impact_type',
      key: 'impact_type',
      render: (type: string) => {
        const typeMap: Record<string, { text: string; icon: React.ReactNode }> = {
          blocked: { text: '被阻塞', icon: <WarningOutlined /> },
          delayed: { text: '延迟', icon: <ClockCircleOutlined /> },
          affected: { text: '受影响', icon: <CheckCircleOutlined /> },
        };
        const config = typeMap[type] || { text: type, icon: null };
        return (
          <Tag
            icon={config.icon}
            color={type === 'blocked' ? 'red' : type === 'delayed' ? 'orange' : 'blue'}
          >
            {config.text}
          </Tag>
        );
      },
    },
    {
      title: '受影响字段',
      dataIndex: 'affected_fields',
      key: 'affected_fields',
      render: (fields: string[]) => (
        <Space>
          {fields.map(field => (
            <Tag key={field} color='blue'>
              {field === 'status'
                ? '状态'
                : field === 'progress'
                ? '进度'
                : field === 'priority'
                ? '优先级'
                : field}
            </Tag>
          ))}
        </Space>
      ),
    },
    {
      title: '预计延迟',
      dataIndex: 'estimated_delay_hours',
      key: 'estimated_delay_hours',
      render: (hours?: number) =>
        hours ? <Text type='warning'>{hours} 小时</Text> : <Text type='secondary'>-</Text>,
    },
  ];

  // 计算统计信息
  const stats = useMemo(() => {
    const total = dependencies.length;
    const blocking = dependencies.filter(d => d.is_blocking).length;
    const hardDeps = dependencies.filter(d => d.dependency_type === 'hard').length;
    const highImpact = impactAnalysis.filter(i => i.impact_level === 'high').length;
    return { total, blocking, hardDeps, highImpact };
  }, [dependencies, impactAnalysis]);

  // 构建依赖树数据
  const dependencyTreeData = useMemo(() => {
    if (dependencies.length === 0) return [];

    const treeData = [
      {
        title: (
          <div className='flex items-center gap-2'>
            <Text strong>{ticket.ticket_number || `T-${ticket.id}`}</Text>
            <Tag color='blue'>当前工单</Tag>
          </div>
        ),
        key: `ticket-${ticket.id}`,
        children: dependencies.map(dep => ({
          title: (
            <div className='flex items-center gap-2'>
              <Tag color={getRelationTypeColor(dep.relation_type)}>
                {getRelationTypeText(dep.relation_type)}
              </Tag>
              <Text>{dep.target_ticket_number}</Text>
              <Text type='secondary' className='text-sm'>
                {dep.target_ticket_title}
              </Text>
              {dep.is_blocking && <Badge status='error' text='阻塞中' />}
            </div>
          ),
          key: `dep-${dep.id}`,
        })),
      },
    ];

    return treeData;
  }, [dependencies, ticket]);

  return (
    <div className='space-y-4'>
      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          type='card'
          size='large'
          items={[
            {
              key: 'dependencies',
              label: (
                <span>
                  <LinkOutlined /> 依赖关系
                  {dependencies.length > 0 && (
                    <Badge count={dependencies.length} className='ml-2' />
                  )}
                </span>
              ),
              children: (
                <div className='space-y-4'>
                  <div className='flex items-center justify-between mb-4'>
                    <Title level={5} style={{ margin: 0 }}>
                      工单依赖关系
                    </Title>
                    {canManage && (
                      <Button
                        type='primary'
                        icon={<PlusOutlined />}
                        onClick={() => {
                          setEditingDependency(null);
                          form.resetFields();
                          form.setFieldsValue({
                            source_ticket_id: ticket.id,
                            dependency_type: 'soft',
                            is_blocking: false,
                          });
                          setDependencyModalVisible(true);
                        }}
                      >
                        添加依赖关系
                      </Button>
                    )}
                  </div>

                  {dependencies.length === 0 ? (
                    <Empty description='暂无依赖关系' image={Empty.PRESENTED_IMAGE_SIMPLE}>
                      {canManage && (
                        <Button
                          type='primary'
                          icon={<PlusOutlined />}
                          onClick={() => {
                            setEditingDependency(null);
                            form.resetFields();
                            form.setFieldsValue({
                              source_ticket_id: ticket.id,
                              dependency_type: 'soft',
                              is_blocking: false,
                            });
                            setDependencyModalVisible(true);
                          }}
                        >
                          添加依赖关系
                        </Button>
                      )}
                    </Empty>
                  ) : (
                    <>
                      <Alert
                        message='依赖关系说明'
                        description='依赖关系用于管理工单之间的关联和影响。硬依赖表示必须等待，软依赖表示建议等待。'
                        type='info'
                        showIcon
                        className='mb-4'
                      />
                      <Table
                        columns={dependencyColumns}
                        dataSource={dependencies}
                        rowKey='id'
                        loading={loading}
                        pagination={false}
                      />
                    </>
                  )}
                </div>
              ),
            },
            {
              key: 'impact',
              label: (
                <span>
                  <WarningOutlined /> 影响分析
                  {impactAnalysis.length > 0 && (
                    <Badge count={impactAnalysis.length} className='ml-2' />
                  )}
                </span>
              ),
              children: (
                <div className='space-y-4'>
                  <Alert
                    message='影响分析'
                    description='系统自动分析当前工单对其他工单的影响，包括阻塞、延迟和影响范围。'
                    type='warning'
                    showIcon
                    className='mb-4'
                  />
                  {impactAnalysis.length === 0 ? (
                    <Empty description='暂无影响分析数据' />
                  ) : (
                    <Table
                      columns={impactColumns}
                      dataSource={impactAnalysis}
                      rowKey='ticket_id'
                      pagination={false}
                    />
                  )}
                </div>
              ),
            },
            {
              key: 'graph',
              label: (
                <span>
                  <BranchesOutlined /> 依赖图谱
                </span>
              ),
              children: (
                <div className='space-y-4'>
                  {dependencyTreeData.length === 0 ? (
                    <Empty description='暂无依赖关系，无法显示图谱' />
                  ) : (
                    <Card>
                      <Tree
                        treeData={dependencyTreeData}
                        defaultExpandAll
                        showLine={{ showLeafIcon: false }}
                      />
                    </Card>
                  )}
                </div>
              ),
            },
            {
              key: 'stats',
              label: (
                <span>
                  <BarChartOutlined /> 统计信息
                </span>
              ),
              children: (
                <div className='space-y-4'>
                  <Row gutter={16}>
                    <Col xs={24} sm={12} md={6}>
                      <Card>
                        <Statistic title='总依赖数' value={stats.total} prefix={<LinkOutlined />} />
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={6}>
                      <Card>
                        <Statistic
                          title='阻塞中'
                          value={stats.blocking}
                          styles={{ content: { color: '#ff4d4f' } }}
                          prefix={<WarningOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={6}>
                      <Card>
                        <Statistic
                          title='硬依赖'
                          value={stats.hardDeps}
                          styles={{ content: { color: '#faad14' } }}
                          prefix={<LinkOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={6}>
                      <Card>
                        <Statistic
                          title='高影响'
                          value={stats.highImpact}
                          styles={{ content: { color: '#ff4d4f' } }}
                          prefix={<WarningOutlined />}
                        />
                      </Card>
                    </Col>
                  </Row>
                </div>
              ),
            },
          ]}
        />
      </Card>

      {/* 依赖关系编辑模态框 */}
      <Modal
        title={editingDependency ? '编辑依赖关系' : '添加依赖关系'}
        open={dependencyModalVisible}
        onOk={handleSaveDependency}
        onCancel={() => {
          setDependencyModalVisible(false);
          setEditingDependency(null);
          form.resetFields();
        }}
        width={700}
      >
        <Form form={form} layout='vertical'>
          <Form.Item
            name='target_ticket_id'
            label='目标工单ID'
            rules={[{ required: true, message: '请输入目标工单ID' }]}
          >
            <Input
              type='number'
              placeholder='请输入目标工单ID'
              addonAfter={
                <Button
                  type='link'
                  size='small'
                  onClick={() => {
                    // 工单选择器需要单独的Modal组件实现
                    antMessage.info('工单选择器功能开发中');
                  }}
                >
                  选择工单
                </Button>
              }
            />
          </Form.Item>
          <Form.Item
            name='relation_type'
            label='关系类型'
            rules={[{ required: true, message: '请选择关系类型' }]}
          >
            <Select placeholder='请选择关系类型'>
              <Option value={TicketRelationType.BLOCKS}>阻塞</Option>
              <Option value={TicketRelationType.BLOCKED_BY}>被阻塞</Option>
              <Option value={TicketRelationType.DEPENDS_ON}>依赖于</Option>
              <Option value={TicketRelationType.RELATES_TO}>相关</Option>
              <Option value={TicketRelationType.DUPLICATES}>重复</Option>
              <Option value={TicketRelationType.CAUSES}>导致</Option>
              <Option value={TicketRelationType.REPLACES}>替代</Option>
              <Option value={TicketRelationType.PARENT_CHILD}>父子关系</Option>
            </Select>
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name='dependency_type'
                label='依赖类型'
                rules={[{ required: true, message: '请选择依赖类型' }]}
              >
                <Select>
                  <Option value='hard'>硬依赖（必须等待）</Option>
                  <Option value='soft'>软依赖（建议等待）</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name='is_blocking' label='是否阻塞' valuePropName='checked'>
                <Select>
                  <Option value={true}>是</Option>
                  <Option value={false}>否</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name='description' label='描述'>
            <TextArea rows={3} placeholder='依赖关系描述（可选）' />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};
