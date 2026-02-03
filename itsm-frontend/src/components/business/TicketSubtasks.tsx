'use client';

import React, { useState, useCallback, useMemo } from 'react';
import {
  Card,
  Button,
  Space,
  Table,
  Modal,
  Form,
  Input,
  Select,
  DatePicker,
  Tag,
  Badge,
  Tooltip,
  message,
  App,
  Empty,
  Popconfirm,
  Timeline,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  LinkOutlined,
  BarChartOutlined,
  UnorderedListOutlined,
  ListOutlined,
  GanttChartOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { Ticket } from '@/lib/services/ticket-service';
import { getStatusConfig, getPriorityConfig } from '@/lib/constants/ticket-constants';
import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';

const { TextArea } = Input;
const { Option } = Select;

interface Subtask extends Ticket {
  parent_id: number;
  dependency_type?: 'blocks' | 'blocked_by' | 'depends_on' | 'relates_to';
  dependency_ticket_id?: number;
  due_date?: string;
  progress?: number;
}

interface TicketSubtasksProps {
  parentTicket: Ticket;
  subtasks?: Subtask[];
  loading?: boolean;
  onCreateSubtask?: (subtask: Partial<Subtask>) => Promise<void>;
  onUpdateSubtask?: (id: number, updates: Partial<Subtask>) => Promise<void>;
  onDeleteSubtask?: (id: number) => Promise<void>;
  onViewSubtask?: (subtask: Subtask) => void;
  canEdit?: boolean;
}

type ViewMode = 'list' | 'gantt' | 'timeline';

export const TicketSubtasks: React.FC<TicketSubtasksProps> = ({
  parentTicket,
  subtasks = [],
  loading = false,
  onCreateSubtask,
  onUpdateSubtask,
  onDeleteSubtask,
  onViewSubtask,
  canEdit = true,
}) => {
  const { message: antMessage } = App.useApp();
  const [modalVisible, setModalVisible] = useState(false);
  const [editingSubtask, setEditingSubtask] = useState<Subtask | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>('list');
  const [form] = Form.useForm();

  // 计算父工单进度
  const parentProgress = useMemo(() => {
    if (!subtasks || subtasks.length === 0) return 0;
    const completedCount = subtasks.filter(
      s => s.status === 'resolved' || s.status === 'closed'
    ).length;
    return Math.round((completedCount / subtasks.length) * 100);
  }, [subtasks]);

  // 计算父工单状态
  const parentStatus = useMemo(() => {
    if (!subtasks || subtasks.length === 0) return parentTicket.status;
    const allResolved = subtasks.every(
      s => s.status === 'resolved' || s.status === 'closed'
    );
    const anyInProgress = subtasks.some(s => s.status === 'in_progress');
    const anyOpen = subtasks.some(s => s.status === 'open' || s.status === 'new');

    if (allResolved) return 'resolved';
    if (anyInProgress) return 'in_progress';
    if (anyOpen) return 'open';
    return parentTicket.status;
  }, [subtasks, parentTicket.status]);

  // 处理创建/更新子任务
  const handleSubmit = useCallback(async () => {
    try {
      const values = await form.validateFields();
      const subtaskData: Partial<Subtask> = {
        ...values,
        parent_id: parentTicket.id,
        title: values.title,
        description: values.description,
        priority: values.priority || 'medium',
        status: values.status || 'open',
        assignee_id: values.assignee_id,
        due_date: values.due_date ? format(values.due_date, 'yyyy-MM-dd') : undefined,
      };

      if (editingSubtask) {
        await onUpdateSubtask?.(editingSubtask.id, subtaskData);
        antMessage.success('子任务已更新');
      } else {
        await onCreateSubtask?.(subtaskData);
        antMessage.success('子任务已创建');
      }

      setModalVisible(false);
      setEditingSubtask(null);
      form.resetFields();
    } catch (error) {
      console.error('Failed to save subtask:', error);
      antMessage.error('保存失败');
    }
  }, [form, editingSubtask, parentTicket.id, onCreateSubtask, onUpdateSubtask, antMessage]);

  // 处理删除子任务
  const handleDelete = useCallback(
    async (id: number) => {
      try {
        await onDeleteSubtask?.(id);
        antMessage.success('子任务已删除');
      } catch (error) {
        console.error('Failed to delete subtask:', error);
        antMessage.error('删除失败');
      }
    },
    [onDeleteSubtask, antMessage]
  );

  // 打开创建/编辑模态框
  const handleOpenModal = useCallback(
    (subtask?: Subtask) => {
      if (subtask) {
        setEditingSubtask(subtask);
        form.setFieldsValue({
          ...subtask,
          due_date: subtask.due_date ? new Date(subtask.due_date) : undefined,
        });
      } else {
        setEditingSubtask(null);
        form.resetFields();
      }
      setModalVisible(true);
    },
    [form]
  );

  // 表格列定义
  const columns: ColumnsType<Subtask> = [
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      render: (text: string, record: Subtask) => (
        <div>
          <div className='font-medium'>{text}</div>
          <div className='text-xs text-gray-500'>
            #{record.ticket_number || record.id}
          </div>
        </div>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => {
        const config = getStatusConfig(status);
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      render: (priority: string) => {
        const config = getPriorityConfig(priority);
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: '处理人',
      dataIndex: 'assignee',
      key: 'assignee',
      width: 120,
      render: (assignee: any) => assignee?.name || '未分配',
    },
    {
      title: '截止时间',
      dataIndex: 'due_date',
      key: 'due_date',
      width: 120,
      render: (date: string) =>
        date ? format(new Date(date), 'yyyy-MM-dd', { locale: zhCN }) : '-',
    },
    {
      title: '进度',
      dataIndex: 'progress',
      key: 'progress',
      width: 100,
      render: (progress: number) => (
        <div className='flex items-center gap-2'>
          <div className='flex-1 bg-gray-200 rounded-full h-2'>
            <div
              className='bg-blue-500 h-2 rounded-full'
              style={{ width: `${progress || 0}%` }}
            />
          </div>
          <span className='text-xs'>{progress || 0}%</span>
        </div>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 150,
      render: (_: any, record: Subtask) => (
        <Space size='small'>
          <Button
            type='link'
            size='small'
            onClick={() => onViewSubtask?.(record)}
          >
            查看
          </Button>
          {canEdit && (
            <>
              <Button
                type='link'
                size='small'
                icon={<EditOutlined />}
                onClick={() => handleOpenModal(record)}
              >
                编辑
              </Button>
              <Popconfirm
                title='确定要删除这个子任务吗？'
                onConfirm={() => handleDelete(record.id)}
              >
                <Button
                  type='link'
                  size='small'
                  danger
                  icon={<DeleteOutlined />}
                >
                  删除
                </Button>
              </Popconfirm>
            </>
          )}
        </Space>
      ),
    },
  ];

  // 甘特图视图
  const renderGanttView = () => {
    if (!subtasks || subtasks.length === 0) {
      return <Empty description='暂无子任务' />;
    }

    return (
      <div className='space-y-4'>
        {subtasks.map(subtask => {
          const startDate = subtask.created_at
            ? new Date(subtask.created_at)
            : new Date();
          const endDate = subtask.due_date
            ? new Date(subtask.due_date)
            : new Date(Date.now() + 7 * 24 * 60 * 60 * 1000);
          const days = Math.ceil(
            (endDate.getTime() - startDate.getTime()) / (24 * 60 * 60 * 1000)
          );
          const statusConfig = getStatusConfig(subtask.status || 'open');

          return (
            <div key={subtask.id} className='border rounded-lg p-4'>
              <div className='flex items-center justify-between mb-2'>
                <div className='flex items-center gap-2'>
                  <Tag color={statusConfig.color}>{statusConfig.text}</Tag>
                  <span className='font-medium'>{subtask.title}</span>
                </div>
                <div className='text-sm text-gray-500'>
                  {format(startDate, 'MM-dd')} ~ {format(endDate, 'MM-dd')}
                </div>
              </div>
              <div className='relative h-8 bg-gray-100 rounded'>
                <div
                  className='absolute h-full rounded'
                  style={{
                    width: `${subtask.progress || 0}%`,
                    backgroundColor: statusConfig.color,
                    opacity: 0.6,
                  }}
                />
                <div className='absolute inset-0 flex items-center justify-center text-xs text-gray-600'>
                  {days} 天
                </div>
              </div>
            </div>
          );
        })}
      </div>
    );
  };

  // 时间线视图
  const renderTimelineView = () => {
    if (!subtasks || subtasks.length === 0) {
      return <Empty description='暂无子任务' />;
    }

    return (
      <Timeline>
        {subtasks.map(subtask => {
          const statusConfig = getStatusConfig(subtask.status || 'open');
          return (
            <Timeline.Item
              key={subtask.id}
              color={statusConfig.color}
              dot={
                subtask.status === 'resolved' || subtask.status === 'closed' ? (
                  <CheckCircleOutlined />
                ) : (
                  <ClockCircleOutlined />
                )
              }
            >
              <div className='flex items-center justify-between'>
                <div>
                  <div className='font-medium'>{subtask.title}</div>
                  <div className='text-sm text-gray-500 mt-1'>
                    {subtask.description}
                  </div>
                  <div className='flex items-center gap-2 mt-2'>
                    <Tag color={statusConfig.color}>{statusConfig.text}</Tag>
                    {subtask.assignee && (
                      <span className='text-xs text-gray-500'>
                        处理人: {subtask.assignee.name}
                      </span>
                    )}
                  </div>
                </div>
                <div className='text-right'>
                  <div className='text-sm text-gray-500'>
                    {subtask.created_at &&
                      format(new Date(subtask.created_at), 'yyyy-MM-dd HH:mm')}
                  </div>
                  {subtask.due_date && (
                    <div className='text-xs text-gray-400'>
                      截止: {format(new Date(subtask.due_date), 'MM-dd')}
                    </div>
                  )}
                </div>
              </div>
            </Timeline.Item>
          );
        })}
      </Timeline>
    );
  };

  return (
    <div className='space-y-4'>
      {/* 父工单汇总信息 */}
      <Card
        title={
          <div className='flex items-center justify-between'>
            <span>子任务管理</span>
            <div className='flex items-center gap-4'>
              <div className='text-sm text-gray-600'>
                总任务: <span className='font-semibold'>{subtasks.length}</span>
              </div>
              <div className='text-sm text-gray-600'>
                完成进度: <span className='font-semibold'>{parentProgress}%</span>
              </div>
              <Badge
                status={
                  parentStatus === 'resolved' || parentStatus === 'closed'
                    ? 'success'
                    : parentStatus === 'in_progress'
                    ? 'processing'
                    : 'default'
                }
                text={
                  getStatusConfig(parentStatus).text
                }
              />
            </div>
          </div>
        }
        extra={
          <Space>
            <Button.Group>
              <Button
                type={viewMode === 'list' ? 'primary' : 'default'}
                icon={<ListOutlined />}
                onClick={() => setViewMode('list')}
              >
                列表
              </Button>
              <Button
                type={viewMode === 'gantt' ? 'primary' : 'default'}
                icon={<GanttChartOutlined />}
                onClick={() => setViewMode('gantt')}
              >
                甘特图
              </Button>
              <Button
                type={viewMode === 'timeline' ? 'primary' : 'default'}
                icon={<ClockCircleOutlined />}
                onClick={() => setViewMode('timeline')}
              >
                时间线
              </Button>
            </Button.Group>
            {canEdit && (
              <Button
                type='primary'
                icon={<PlusOutlined />}
                onClick={() => handleOpenModal()}
              >
                创建子任务
              </Button>
            )}
          </Space>
        }
      >
        {viewMode === 'list' && (
          <Table
            columns={columns}
            dataSource={subtasks}
            rowKey='id'
            loading={loading}
            pagination={false}
            size='small'
          />
        )}
        {viewMode === 'gantt' && renderGanttView()}
        {viewMode === 'timeline' && renderTimelineView()}
      </Card>

      {/* 创建/编辑子任务模态框 */}
      <Modal
        title={editingSubtask ? '编辑子任务' : '创建子任务'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => {
          setModalVisible(false);
          setEditingSubtask(null);
          form.resetFields();
        }}
        width={600}
      >
        <Form form={form} layout='vertical'>
          <Form.Item
            name='title'
            label='任务标题'
            rules={[{ required: true, message: '请输入任务标题' }]}
          >
            <Input placeholder='请输入任务标题' />
          </Form.Item>
          <Form.Item name='description' label='任务描述'>
            <TextArea rows={4} placeholder='请输入任务描述' />
          </Form.Item>
          <Form.Item name='priority' label='优先级' initialValue='medium'>
            <Select>
              <Option value='low'>低</Option>
              <Option value='medium'>中</Option>
              <Option value='high'>高</Option>
              <Option value='urgent'>紧急</Option>
            </Select>
          </Form.Item>
          <Form.Item name='status' label='状态' initialValue='open'>
            <Select>
              <Option value='new'>新建</Option>
              <Option value='open'>待处理</Option>
              <Option value='in_progress'>处理中</Option>
              <Option value='resolved'>已解决</Option>
              <Option value='closed'>已关闭</Option>
            </Select>
          </Form.Item>
          <Form.Item name='assignee_id' label='处理人'>
            <Select placeholder='请选择处理人' allowClear>
              {/* TODO: 从用户列表获取 */}
            </Select>
          </Form.Item>
          <Form.Item name='due_date' label='截止时间'>
            <DatePicker style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name='progress' label='进度' initialValue={0}>
            <Input type='number' min={0} max={100} addonAfter='%' />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

