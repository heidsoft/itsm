'use client';

import React, { useState, useCallback, useMemo } from 'react';
import {
  Card,
  Typography,
  Tag,
  Avatar,
  Space,
  Button,
  Dropdown,
  Tooltip,
  Badge,
  Input,
  Select,
  Row,
  Col,
  Modal,
  Form,
  App,
} from 'antd';
import {
  MoreOutlined,
  EditOutlined,
  DeleteOutlined,
  UserOutlined,
  CalendarOutlined,
  ClockCircleOutlined,
  PlusOutlined,
  FilterOutlined,
  SortAscendingOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import 'dayjs/locale/zh-cn';

import { useRouter } from 'next/navigation';
import type { Ticket } from '@/lib/api/types';
import { useTickets } from '@/lib/hooks/useTickets';

dayjs.extend(relativeTime);
dayjs.locale('zh-cn');

const { Title, Text, Paragraph } = Typography;
const { Search } = Input;

interface TicketKanbanProps {
  onTicketSelect?: (ticket: Ticket) => void;
}

// 状态配置
const KANBAN_STATUS_CONFIG = [
  { key: 'new', title: '新建', color: '#1890ff' },
  { key: 'open', title: '待处理', color: '#1890ff' },
  { key: 'in_progress', title: '处理中', color: '#fa8c16' },
  { key: 'pending', title: '等待中', color: '#faad14' },
  { key: 'resolved', title: '已解决', color: '#52c41a' },
  { key: 'closed', title: '已关闭', color: '#d9d9d9' },
];

// 优先级配置
const PRIORITY_CONFIG = {
  low: { color: 'green', text: '低', icon: '↓' },
  medium: { color: 'orange', text: '中', icon: '→' },
  high: { color: 'red', text: '高', icon: '↑' },
  urgent: { color: 'purple', text: '紧急', icon: '⚡' },
  critical: { color: 'red', text: '严重', icon: '🚨' },
};

const TicketKanban: React.FC<TicketKanbanProps> = ({ onTicketSelect }) => {
  const router = useRouter();
  const { tickets, loading, fetchTickets, updateTicket, deleteTicket } = useTickets();
  const { message, modal } = App.useApp();

  const [searchValue, setSearchValue] = useState('');
  const [selectedStatus, setSelectedStatus] = useState<string>('');
  const [selectedPriority, setSelectedPriority] = useState<string>('');
  const [sortBy, setSortBy] = useState<string>('created_at');
  const [viewModalVisible, setViewModalVisible] = useState(false);
  const [selectedTicket, setSelectedTicket] = useState<Ticket | null>(null);

  // 过滤和排序工单
  const filteredTickets = useMemo(() => {
    let filtered = [...tickets];

    // 搜索过滤
    if (searchValue) {
      filtered = filtered.filter(
        ticket =>
          ticket.title.toLowerCase().includes(searchValue.toLowerCase()) ||
          ticket.description.toLowerCase().includes(searchValue.toLowerCase()) ||
          ticket.ticketNumber.toLowerCase().includes(searchValue.toLowerCase())
      );
    }

    // 状态过滤
    if (selectedStatus) {
      filtered = filtered.filter(ticket => ticket.status === selectedStatus);
    }

    // 优先级过滤
    if (selectedPriority) {
      filtered = filtered.filter(ticket => ticket.priority === selectedPriority);
    }

    // 排序
    filtered.sort((a, b) => {
      switch (sortBy) {
        case 'priority':
          const priorityOrder = { critical: 4, urgent: 3, high: 2, medium: 1, low: 0 };
          return (
            (priorityOrder[b.priority as keyof typeof priorityOrder] || 0) -
            (priorityOrder[a.priority as keyof typeof priorityOrder] || 0)
          );
        case 'title':
          return a.title.localeCompare(b.title);
        case 'created_at':
        default:
          return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
      }
    });

    return filtered;
  }, [tickets, searchValue, selectedStatus, selectedPriority, sortBy]);

  // 按状态分组工单
  const ticketsByStatus = useMemo(() => {
    const grouped: Record<string, Ticket[]> = {};

    KANBAN_STATUS_CONFIG.forEach(status => {
      grouped[status.key] = [];
    });

    filteredTickets.forEach(ticket => {
      if (grouped[ticket.status]) {
        grouped[ticket.status].push(ticket);
      }
    });

    return grouped;
  }, [filteredTickets]);

  // 工单操作菜单
  const getTicketMenu = (ticket: Ticket): MenuProps['items'] => [
    {
      key: 'view',
      icon: <EditOutlined />,
      label: '查看详情',
      onClick: () => {
        if (onTicketSelect) {
          onTicketSelect(ticket);
        } else {
          router.push(`/tickets/${ticket.id}`);
        }
      },
    },
    {
      key: 'edit',
      icon: <EditOutlined />,
      label: '编辑',
      onClick: () => router.push(`/tickets/${ticket.id}/edit`),
    },
    {
      type: 'divider',
    },
    {
      key: 'delete',
      icon: <DeleteOutlined />,
      label: '删除',
      danger: true,
      onClick: () => handleDeleteTicket(ticket),
    },
  ];

  // 删除工单
  const handleDeleteTicket = useCallback(
    async (ticket: Ticket) => {
      Modal.confirm({
        title: '确认删除',
        content: `确定要删除工单 ${ticket.ticketNumber || '-'} 吗？此操作不可撤销。`,
        okText: '确认删除',
        cancelText: '取消',
        okType: 'danger',
        onOk: async () => {
          try {
            await deleteTicket(ticket.id);
            message.success('删除成功');
          } catch (error) {
            message.error('删除失败');
          }
        },
      });
    },
    [deleteTicket]
  );

  // 拖拽状态更新
  const handleStatusChange = useCallback(
    async (ticket: Ticket, newStatus: string) => {
      try {
        await updateTicket(ticket.id, { status: newStatus });
        message.success('状态更新成功');
      } catch (error) {
        message.error('状态更新失败');
      }
    },
    [updateTicket]
  );

  // 工单卡片组件
  const TicketCard = ({ ticket }: { ticket: Ticket }) => {
    const priorityConfig = PRIORITY_CONFIG[ticket.priority as keyof typeof PRIORITY_CONFIG];

    return (
      <Card
        size='small'
        hoverable
        className='mb-3 cursor-move'
        style={{
          borderLeft: `4px solid ${priorityConfig.color}`,
          margin: '0 0 12px 0',
        }}
        actions={[
          <Dropdown key='more' menu={{ items: getTicketMenu(ticket) }} trigger={['click']}>
            <Button type='text' icon={<MoreOutlined />} size='small' />
          </Dropdown>,
        ]}
      >
        <div className='space-y-2'>
          {/* 工单标题 */}
          <div className='flex items-start justify-between'>
            <Text strong className='text-sm flex-1 mr-2'>
              {ticket.title}
            </Text>
            <Badge
              count={priorityConfig.icon}
              style={{
                backgroundColor: priorityConfig.color,
                fontSize: '12px',
                lineHeight: '16px',
              }}
            />
          </div>

          {/* 工单号和类型 */}
          <div className='flex items-center justify-between'>
            <Text code className='text-xs'>
              {ticket.ticketNumber || '-'}
            </Text>
            <Tag color='blue'>{ticket.type}</Tag>
          </div>

          {/* 工单描述 */}
          <Paragraph
            ellipsis={{ rows: 2, expandable: false }}
            className='text-xs text-gray-500 mb-2'
          >
            {ticket.description}
          </Paragraph>

          {/* 时间信息 */}
          <div className='flex items-center text-xs text-gray-400'>
            <ClockCircleOutlined className='mr-1' />
            {dayjs(ticket.createdAt).fromNow()}
          </div>

          {/* 分配人信息 */}
          {ticket.assignee && (
            <div className='flex items-center mt-2'>
              <Avatar size='small' icon={<UserOutlined />} className='mr-2' />
              <Text className='text-xs'>{ticket.assignee.name || ticket.assignee.username}</Text>
            </div>
          )}

          {/* 截止时间 */}
          {ticket.dueTime && (
            <div className='flex items-center mt-1'>
              <CalendarOutlined className='mr-1 text-xs text-red-500' />
              <Text className='text-xs text-red-500'>
                截止: {dayjs(ticket.dueTime).format('MM-DD HH:mm')}
              </Text>
            </div>
          )}
        </div>
      </Card>
    );
  };

  return (
    <div className='ticket-kanban p-4'>
      {/* 工具栏 */}
      <Card className='mb-4'>
        <Row gutter={[16, 16]} align='middle'>
          <Col flex='auto'>
            <Space size='middle'>
              <Search
                placeholder='搜索工单...'
                value={searchValue}
                onChange={e => setSearchValue(e.target.value)}
                style={{ width: 300 }}
                allowClear
              />
              <Select
                placeholder='状态'
                value={selectedStatus}
                onChange={setSelectedStatus}
                allowClear
                style={{ width: 120 }}
                options={KANBAN_STATUS_CONFIG.map(status => ({
                  value: status.key,
                  label: status.title,
                }))}
              />
              <Select
                placeholder='优先级'
                value={selectedPriority}
                onChange={setSelectedPriority}
                allowClear
                style={{ width: 120 }}
                options={Object.entries(PRIORITY_CONFIG).map(([key, config]) => ({
                  value: key,
                  label: (
                    <>
                      <Tag color={config.color}>{config.text}</Tag>
                    </>
                  ),
                }))}
              />
              <Select
                value={sortBy}
                onChange={setSortBy}
                style={{ width: 140 }}
                options={[
                  { value: 'created_at', label: '创建时间' },
                  { value: 'priority', label: '优先级' },
                  { value: 'title', label: '标题' },
                ]}
              />
            </Space>
          </Col>
          <Col>
            <Space>
              <Button
                type='primary'
                icon={<PlusOutlined />}
                onClick={() => router.push('/tickets/create')}
              >
                新建工单
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 看板列 */}
      <Row gutter={[16, 0]}>
        {KANBAN_STATUS_CONFIG.map(status => (
          <Col span={4} key={status.key}>
            <Card
              title={
                <div className='flex items-center justify-between'>
                  <div className='flex items-center'>
                    <div
                      className='w-2 h-2 rounded-full mr-2'
                      style={{ backgroundColor: status.color }}
                    />
                    <Text strong>{status.title}</Text>
                  </div>
                  <Badge
                    count={ticketsByStatus[status.key]?.length || 0}
                    style={{ backgroundColor: status.color }}
                  />
                </div>
              }
              size='small'
              className='h-full'
              style={{ minHeight: '600px' }}
            >
              <div className='space-y-2'>
                {ticketsByStatus[status.key]?.map(ticket => (
                  <TicketCard key={ticket.id} ticket={ticket} />
                ))}
                {(!ticketsByStatus[status.key] || ticketsByStatus[status.key].length === 0) && (
                  <div className='text-center text-gray-400 py-8'>
                    <Text type='secondary'>暂无工单</Text>
                  </div>
                )}
              </div>
            </Card>
          </Col>
        ))}
      </Row>

      {/* 工单详情模态框 */}
      <Modal
        title='工单详情'
        open={viewModalVisible}
        onCancel={() => setViewModalVisible(false)}
        footer={[
          <Button key='close' onClick={() => setViewModalVisible(false)}>
            关闭
          </Button>,
          <Button
            key='view'
            type='primary'
            onClick={() => {
              if (selectedTicket) {
                router.push(`/tickets/${selectedTicket.id}`);
                setViewModalVisible(false);
              }
            }}
          >
            查看详情
          </Button>,
        ]}
        width={800}
      >
        {selectedTicket && (
          <div className='space-y-4'>
            <div>
              <Text strong>{selectedTicket.ticketNumber || '-'}</Text>
              <Title level={4}>{selectedTicket.title}</Title>
            </div>
            <div>
              <Text>描述:</Text>
              <Paragraph>{selectedTicket.description}</Paragraph>
            </div>
            <div className='grid grid-cols-2 gap-4'>
              <div>
                <Text>状态:</Text>
                <div>{selectedTicket.status}</div>
              </div>
              <div>
                <Text>优先级:</Text>
                <div>{selectedTicket.priority}</div>
              </div>
              <div>
                <Text>创建时间:</Text>
                <div>{dayjs(selectedTicket.createdAt).format('YYYY-MM-DD HH:mm')}</div>
              </div>
              <div>
                <Text>更新时间:</Text>
                <div>{dayjs(selectedTicket.updatedAt).format('YYYY-MM-DD HH:mm')}</div>
              </div>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default TicketKanban;
