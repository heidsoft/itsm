'use client';

import React, { useState, useMemo, useCallback } from 'react';
import {
  Card,
  Space,
  Button,
  Input,
  Select,
  Dropdown,
  Badge,
  Tag,
  Avatar,
  Tooltip,
  Modal,
  Form,
  message,
  App,
} from 'antd';
import {
  SearchOutlined,
  FilterOutlined,
  SettingOutlined,
  PlusOutlined,
  EyeOutlined,
  EditOutlined,
  MoreOutlined,
  SaveOutlined,
  ShareAltOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import {
  DndContext,
  DragOverlay,
  closestCorners,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import {
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
  useSortable,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { Ticket } from '@/lib/services/ticket-service';
import { getStatusConfig, getPriorityConfig } from '@/lib/constants/ticket-constants';
import { formatDistanceToNow } from 'date-fns';
import { zhCN } from 'date-fns/locale';

const { Search } = Input;
const { Option } = Select;

interface TicketKanbanBoardProps {
  tickets: Ticket[];
  loading?: boolean;
  onTicketClick?: (ticket: Ticket) => void;
  onTicketEdit?: (ticket: Ticket) => void;
  onTicketStatusChange?: (ticketId: number, newStatus: string) => Promise<void>;
  onTicketMove?: (ticketId: number, fromStatus: string, toStatus: string) => Promise<void>;
  canEdit?: boolean;
}

interface KanbanColumn {
  id: string;
  title: string;
  status: string;
  tickets: Ticket[];
  color: string;
}

interface KanbanCardProps {
  ticket: Ticket;
  onClick?: () => void;
  onEdit?: () => void;
}

// 工单卡片组件
const KanbanCard: React.FC<KanbanCardProps> = ({ ticket, onClick, onEdit }) => {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: String(ticket.id),
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  const statusConfig = getStatusConfig(ticket.status || 'open');
  const priorityConfig = getPriorityConfig(ticket.priority || 'medium');

  const menuItems: MenuProps['items'] = [
    {
      key: 'view',
      label: '查看详情',
      icon: <EyeOutlined />,
      onClick: onClick,
    },
    {
      key: 'edit',
      label: '编辑',
      icon: <EditOutlined />,
      onClick: onEdit,
    },
  ];

  return (
    <div ref={setNodeRef} style={style} {...attributes} {...listeners} className='mb-3 cursor-move'>
      <Card
        size='small'
        className='hover:shadow-md transition-shadow duration-200'
        onClick={onClick}
        actions={[
          <Dropdown menu={{ items: menuItems }} trigger={['click']} key='more'>
            <MoreOutlined />
          </Dropdown>,
        ]}
      >
        <div className='space-y-2'>
          {/* 标题和编号 */}
          <div>
            <div className='font-medium text-sm text-gray-900 line-clamp-2'>{ticket.title}</div>
            <div className='text-xs text-gray-500 mt-1'>#{ticket.ticket_number || ticket.id}</div>
          </div>

          {/* 状态和优先级 */}
          <div className='flex items-center gap-2 flex-wrap'>
            <Tag color={statusConfig.color}>{statusConfig.text}</Tag>
            <Tag color={priorityConfig.color}>
              {priorityConfig.icon} {priorityConfig.text}
            </Tag>
          </div>

          {/* 处理人和时间 */}
          <div className='flex items-center justify-between text-xs text-gray-500'>
            <div className='flex items-center gap-1'>
              {ticket.assignee ? (
                <>
                  <Avatar size={16} src={(ticket.assignee as any).avatar}>
                    {ticket.assignee.name?.[0]}
                  </Avatar>
                  <span>{ticket.assignee.name}</span>
                </>
              ) : (
                <span>未分配</span>
              )}
            </div>
            <div>
              {ticket.created_at &&
                formatDistanceToNow(new Date(ticket.created_at), {
                  addSuffix: true,
                  locale: zhCN,
                })}
            </div>
          </div>

          {/* 分类和标签 */}
          {(ticket.category || (ticket.tags && ticket.tags.length > 0)) && (
            <div className='flex items-center gap-1 flex-wrap'>
              {ticket.category && <Tag color='blue'>{ticket.category}</Tag>}
              {ticket.tags &&
                ticket.tags.slice(0, 2).map((tag, index) => (
                  <Tag key={index} color='default'>
                    {tag}
                  </Tag>
                ))}
            </div>
          )}
        </div>
      </Card>
    </div>
  );
};

// 看板列组件
interface KanbanColumnProps {
  column: KanbanColumn;
  tickets: Ticket[];
  onTicketClick?: (ticket: Ticket) => void;
  onTicketEdit?: (ticket: Ticket) => void;
}

const KanbanColumnComponent: React.FC<KanbanColumnProps> = ({
  column,
  tickets,
  onTicketClick,
  onTicketEdit,
}) => {
  const sortableTickets = useMemo(() => tickets.map(t => String(t.id)), [tickets]);

  return (
    <div className='flex-1 min-w-[300px] bg-gray-50 rounded-lg p-4'>
      {/* 列标题 */}
      <div className='flex items-center justify-between mb-4'>
        <div className='flex items-center gap-2'>
          <div className='w-3 h-3 rounded-full' style={{ backgroundColor: column.color }} />
          <span className='font-semibold text-gray-900'>{column.title}</span>
          <Badge count={tickets.length} showZero className='ml-2' />
        </div>
      </div>

      {/* 工单列表 */}
      <div className='space-y-2 min-h-[200px]'>
        {tickets.map(ticket => (
          <KanbanCard
            key={ticket.id}
            ticket={ticket}
            onClick={() => onTicketClick?.(ticket)}
            onEdit={() => onTicketEdit?.(ticket)}
          />
        ))}
        {tickets.length === 0 && (
          <div className='text-center text-gray-400 py-8 text-sm'>暂无工单</div>
        )}
      </div>
    </div>
  );
};

// 主看板组件
export const TicketKanbanBoard: React.FC<TicketKanbanBoardProps> = ({
  tickets = [],
  loading = false,
  onTicketClick,
  onTicketEdit,
  onTicketStatusChange,
  onTicketMove,
  canEdit = true,
}) => {
  const { message: antMessage } = App.useApp();
  const [searchKeyword, setSearchKeyword] = useState('');
  const [groupBy, setGroupBy] = useState<'status' | 'assignee' | 'priority' | 'category'>('status');
  const [filterStatus, setFilterStatus] = useState<string>('all');
  const [filterPriority, setFilterPriority] = useState<string>('all');
  const [activeId, setActiveId] = useState<string | null>(null);
  const [saveViewModalVisible, setSaveViewModalVisible] = useState(false);
  const [form] = Form.useForm();

  // 拖拽传感器
  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  // 状态列配置
  const statusColumns: Omit<KanbanColumn, 'tickets'>[] = [
    { id: 'new', title: '新建', status: 'new', color: '#1890ff' },
    { id: 'open', title: '待处理', status: 'open', color: '#fa8c16' },
    { id: 'in_progress', title: '处理中', status: 'in_progress', color: '#13c2c2' },
    { id: 'pending_approval', title: '待审批', status: 'pending_approval', color: '#faad14' },
    { id: 'resolved', title: '已解决', status: 'resolved', color: '#52c41a' },
    { id: 'closed', title: '已关闭', status: 'closed', color: '#8c8c8c' },
  ];

  // 筛选后的工单
  const filteredTickets = useMemo(() => {
    let result = [...tickets];

    // 关键词搜索
    if (searchKeyword) {
      const keyword = searchKeyword.toLowerCase();
      result = result.filter(
        ticket =>
          ticket.title?.toLowerCase().includes(keyword) ||
          ticket.description?.toLowerCase().includes(keyword) ||
          ticket.ticket_number?.toLowerCase().includes(keyword)
      );
    }

    // 状态筛选
    if (filterStatus !== 'all') {
      result = result.filter(ticket => ticket.status === filterStatus);
    }

    // 优先级筛选
    if (filterPriority !== 'all') {
      result = result.filter(ticket => ticket.priority === filterPriority);
    }

    return result;
  }, [tickets, searchKeyword, filterStatus, filterPriority]);

  // 按状态分组的列
  const statusColumnsData = useMemo(() => {
    return statusColumns.map(column => ({
      ...column,
      tickets: filteredTickets.filter(ticket => ticket.status === column.status),
    }));
  }, [filteredTickets]);

  // 拖拽开始
  const handleDragStart = (event: any) => {
    setActiveId(event.active.id);
  };

  // 拖拽结束
  const handleDragEnd = async (event: any) => {
    const { active, over } = event;
    setActiveId(null);

    if (!over || !canEdit) {
      return;
    }

    const ticketId = Number(active.id);

    // 获取源工单状态
    const sourceTicket = tickets.find(t => t.id === ticketId);
    if (!sourceTicket) return;

    const sourceStatus = sourceTicket.status || 'open';

    // 获取目标状态（从列ID或数据中获取）
    let targetStatus = over.id;
    if (over.data?.current?.status) {
      targetStatus = over.data.current.status;
    } else {
      // 从列配置中查找
      const targetColumn = statusColumns.find(col => col.id === over.id);
      if (targetColumn) {
        targetStatus = targetColumn.status;
      }
    }

    if (sourceStatus === targetStatus) {
      return;
    }

    try {
      // 调用状态变更回调
      if (onTicketStatusChange) {
        await onTicketStatusChange(ticketId, targetStatus);
        antMessage.success('工单状态已更新');
      } else if (onTicketMove) {
        await onTicketMove(ticketId, sourceStatus, targetStatus);
        antMessage.success('工单已移动');
      }
    } catch (error) {
      console.error('Failed to move ticket:', error);
      antMessage.error('移动工单失败');
    }
  };

  // 保存视图
  const handleSaveView = async () => {
    try {
      const values = await form.validateFields();
      // 注意：保存视图API尚未实现
      // 未来可通过 ViewAPI.saveView() 保存视图配置
      antMessage.success('视图已保存（模拟）');
      setSaveViewModalVisible(false);
      form.resetFields();
    } catch (error) {
      console.error('Failed to save view:', error);
    }
  };

  const viewMenuItems: MenuProps['items'] = [
    {
      key: 'save',
      label: '保存当前视图',
      icon: <SaveOutlined />,
      onClick: () => setSaveViewModalVisible(true),
    },
    {
      key: 'share',
      label: '共享视图',
      icon: <ShareAltOutlined />,
      onClick: () => antMessage.info('共享功能开发中'),
    },
  ];

  if (loading) {
    return (
      <div className='flex items-center justify-center h-96'>
        <div className='text-gray-500'>加载中...</div>
      </div>
    );
  }

  return (
    <div className='space-y-4'>
      {/* 工具栏 */}
      <div className='flex items-center justify-between flex-wrap gap-4 bg-white p-4 rounded-lg shadow-sm'>
        <div className='flex items-center gap-3 flex-1 min-w-[300px]'>
          <Search
            placeholder='搜索工单...'
            allowClear
            value={searchKeyword}
            onChange={e => setSearchKeyword(e.target.value)}
            style={{ width: 250 }}
            prefix={<SearchOutlined />}
          />
          <Select
            value={filterStatus}
            onChange={setFilterStatus}
            style={{ width: 120 }}
            placeholder='状态'
          >
            <Option value='all'>全部状态</Option>
            {statusColumns.map(col => (
              <Option key={col.id} value={col.status}>
                {col.title}
              </Option>
            ))}
          </Select>
          <Select
            value={filterPriority}
            onChange={setFilterPriority}
            style={{ width: 120 }}
            placeholder='优先级'
          >
            <Option value='all'>全部优先级</Option>
            <Option value='low'>低</Option>
            <Option value='medium'>中</Option>
            <Option value='high'>高</Option>
            <Option value='urgent'>紧急</Option>
          </Select>
        </div>
        <div className='flex items-center gap-2'>
          <Dropdown menu={{ items: viewMenuItems }} trigger={['click']}>
            <Button icon={<SettingOutlined />}>视图设置</Button>
          </Dropdown>
        </div>
      </div>

      {/* 看板区域 */}
      <DndContext
        sensors={sensors}
        collisionDetection={closestCorners}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
      >
        <div className='flex gap-4 overflow-x-auto pb-4' style={{ minHeight: '600px' }}>
          {statusColumnsData.map(column => (
            <div key={column.id} data-status={column.status} className='flex-shrink-0'>
              <SortableContext
                items={column.tickets.map(t => String(t.id))}
                strategy={verticalListSortingStrategy}
              >
                <div data-status={column.status}>
                  <KanbanColumnComponent
                    column={column}
                    tickets={column.tickets}
                    onTicketClick={onTicketClick}
                    onTicketEdit={onTicketEdit}
                  />
                </div>
              </SortableContext>
            </div>
          ))}
        </div>

        <DragOverlay>
          {activeId ? (
            <div className='opacity-50'>
              <Card size='small' style={{ width: 280 }}>
                <div className='text-sm font-medium'>拖拽中...</div>
              </Card>
            </div>
          ) : null}
        </DragOverlay>
      </DndContext>

      {/* 保存视图模态框 */}
      <Modal
        title='保存看板视图'
        open={saveViewModalVisible}
        onOk={handleSaveView}
        onCancel={() => {
          setSaveViewModalVisible(false);
          form.resetFields();
        }}
      >
        <Form form={form} layout='vertical'>
          <Form.Item
            name='name'
            label='视图名称'
            rules={[{ required: true, message: '请输入视图名称' }]}
          >
            <Input placeholder='例如：我的看板视图' />
          </Form.Item>
          <Form.Item name='description' label='描述'>
            <Input.TextArea rows={3} placeholder='视图描述（可选）' />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};
