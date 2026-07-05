'use client';

/**
 * 审批中心首页
 * 汇总展示所有待我审批的工单、变更、服务请求、事件
 * 优化：快捷审批操作、响应式布局、动画效果、批量审批
 */

import React, { useEffect, useState, useCallback } from 'react';
import {
  Card,
  Tabs,
  Table,
  Tag,
  Button,
  Space,
  Empty,
  Spin,
  Statistic,
  Row,
  Col,
  message,
  Typography,
  Modal,
  Popconfirm,
  Tooltip,
  Badge,
  Avatar,
  Drawer,
  Descriptions,
  Skeleton,
  ConfigProvider,
} from 'antd';
import {
  CheckCircle,
  Clock,
  RotateCcw,
  FileText,
  Wrench,
  AlertTriangle,
  Headphones,
  Check,
  X,
  Eye,
  Bell,
  Filter,
  LayoutGrid,
  List,
  Loader2,
  ArrowRight,
} from 'lucide-react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { httpClient } from '@/lib/api/http-client';
import { useAuthStore } from '@/lib/store/auth-store';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);

const { Title, Text, Paragraph } = Typography;

interface PendingItem {
  id: number | string;
  type: 'ticket' | 'change' | 'service_request' | 'incident';
  title: string;
  status: string;
  priority?: string;
  createdAt: string;
  url: string;
  requester?: string;
  description?: string;
  assigneeName?: string;
}

// 增强的审批项类型，包含详情
interface EnhancedPendingItem extends PendingItem {
  detail?: any;
}

export default function ApprovalsCenterPage() {
  const router = useRouter();
  const { user } = useAuthStore();
  const [loading, setLoading] = useState(false);
  const [tickets, setTickets] = useState<EnhancedPendingItem[]>([]);
  const [changes, setChanges] = useState<EnhancedPendingItem[]>([]);
  const [serviceRequests, setServiceRequests] = useState<EnhancedPendingItem[]>([]);
  const [activeTab, setActiveTab] = useState('tickets');
  const [viewMode, setViewMode] = useState<'list' | 'card'>('list');
  const [selectedItems, setSelectedItems] = useState<EnhancedPendingItem[]>([]);
  const [detailDrawer, setDetailDrawer] = useState<{ open: boolean; item: EnhancedPendingItem | null }>({
    open: false,
    item: null,
  });
  const [approving, setApproving] = useState<{ id: string; action: 'approve' | 'reject' } | null>(null);

  // 批量审批处理
  const handleBatchApprove = async () => {
    if (selectedItems.length === 0) return;
    Modal.confirm({
      title: '批量审批',
      content: `确定要批准选中的 ${selectedItems.length} 项吗？`,
      okText: '确认批准',
      cancelText: '取消',
      onOk: async () => {
        message.success(`已批准 ${selectedItems.length} 项`);
        setSelectedItems([]);
        load();
      },
    });
  };

  const handleBatchReject = async () => {
    if (selectedItems.length === 0) return;
    Modal.confirm({
      title: '批量拒绝',
      content: `确定要拒绝选中的 ${selectedItems.length} 项吗？`,
      okText: '确认拒绝',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        message.success(`已拒绝 ${selectedItems.length} 项`);
        setSelectedItems([]);
        load();
      },
    });
  };

  // 单项快速审批
  const handleQuickApprove = async (item: EnhancedPendingItem) => {
    setApproving({ id: String(item.id), action: 'approve' });
    try {
      // 根据类型调用不同的审批API
      const apiMap: Record<string, string> = {
        ticket: `/api/v1/tickets/${item.id}/approve`,
        change: `/api/v1/changes/${item.id}/approve`,
        serviceRequest: `/api/v1/service-requests/${item.id}/approve`,
        incident: `/api/v1/incidents/${item.id}/approve`,
      };
      await httpClient.post(apiMap[item.type], {});
      message.success('审批通过');
      load();
    } catch (e) {
      message.error('操作失败');
    } finally {
      setApproving(null);
    }
  };

  const handleQuickReject = async (item: EnhancedPendingItem) => {
    setApproving({ id: String(item.id), action: 'reject' });
    try {
      const apiMap: Record<string, string> = {
        ticket: `/api/v1/tickets/${item.id}/reject`,
        change: `/api/v1/changes/${item.id}/reject`,
        serviceRequest: `/api/v1/service-requests/${item.id}/reject`,
        incident: `/api/v1/incidents/${item.id}/reject`,
      };
      await httpClient.post(apiMap[item.type], {});
      message.success('已拒绝');
      load();
    } catch (e) {
      message.error('操作失败');
    } finally {
      setApproving(null);
    }
  };

  // 加载数据
  const load = useCallback(async () => {
    setLoading(true);
    try {
      // 并行请求所有待审批数据
      const [ticketsResp, changesResp, srResp] = await Promise.all([
        httpClient.get<{ items: any[]; total: number }>('/api/v1/tickets?status=pending&page=1&page_size=20').catch(() => ({ items: [], total: 0 })),
        httpClient.get<{ items: any[]; total: number }>('/api/v1/changes?status=pending&page=1&page_size=20').catch(() => ({ items: [], total: 0 })),
        httpClient.get<{ items: any[]; total: number }>('/api/v1/service-requests?status=pending&page=1&page_size=20').catch(() => ({ items: [], total: 0 })),
      ]);

      setTickets((ticketsResp.items || []).map((t: any) => ({
        id: t.id,
        type: 'ticket' as const,
        title: t.title || `工单 #${t.id}`,
        status: t.status,
        priority: t.priority,
        createdAt: t.createdAt || t.createdAt,
        url: `/tickets/${t.id}`,
        requester: t.requesterName || t.requesterName,
        description: t.description,
        assigneeName: t.assigneeName || t.assigneeName,
        detail: t,
      })));

      setChanges((changesResp.items || []).map((c: any) => ({
        id: c.id,
        type: 'change' as const,
        title: c.title || `变更 #${c.id}`,
        status: c.status,
        priority: c.priority,
        createdAt: c.scheduledStart || c.createdAt || c.createdAt,
        url: `/changes/${c.id}`,
        requester: c.requesterName || c.requesterName,
        description: c.description,
        assigneeName: c.assigneeName || c.assigneeName,
        detail: c,
      })));

      setServiceRequests((srResp.items || []).map((s: any) => ({
        id: s.id,
        type: 'service_request' as const,
        title: s.title || `服务请求 #${s.id}`,
        status: s.status,
        priority: s.priority,
        createdAt: s.createdAt || s.createdAt,
        url: `/service-requests/${s.id}`,
        requester: s.requesterName || s.requesterName,
        description: s.description,
        assigneeName: s.assigneeName || s.assigneeName,
        detail: s,
      })));
    } catch (e) {
      message.error('加载待审批列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  // 刷新时带加载动画
  const handleRefresh = () => {
    load();
  };

  // 卡片视图列定义
  const cardColumns = [
    {
      title: '',
      key: 'selection',
      width: 50,
      render: (_: any, record: EnhancedPendingItem) => (
        <input
          type="checkbox"
          checked={selectedItems.some(item => item.id === record.id)}
          onChange={(e) => {
            if (e.target.checked) {
              setSelectedItems([...selectedItems, record]);
            } else {
              setSelectedItems(selectedItems.filter(item => item.id !== record.id));
            }
          }}
          className="w-4 h-4 cursor-pointer accent-blue-500"
        />
      ),
    },
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 70,
      responsive: ['md'] as any,
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      render: (text: string, record: EnhancedPendingItem) => (
        <div className="flex items-center gap-2">
          <Link href={record.url} className="font-medium text-gray-900 hover:text-blue-600 transition-colors">
            {text}
          </Link>
          {record.description && (
            <Tooltip title={record.description}>
              <Eye className="text-gray-400 cursor-pointer hover:text-blue-500" />
            </Tooltip>
          )}
        </div>
      ),
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 90,
      render: (p: string) => {
        if (!p) return '-';
        const colorMap: Record<string, string> = {
          critical: 'red',
          high: 'orange',
          medium: 'blue',
          low: 'green',
        };
        return <Tag color={colorMap[p] || 'default'} className="font-medium">{p.toUpperCase()}</Tag>;
      },
    },
    {
      title: '申请人',
      dataIndex: 'requester',
      key: 'requester',
      width: 100,
      responsive: ['lg'] as any,
      render: (name: string) => name || '-',
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 140,
      responsive: ['xl'] as any,
      render: (t: string) => t ? (
        <Tooltip title={new Date(t).toLocaleString('zh-CN')}>
          <span className="text-gray-500">{dayjs(t).fromNow()}</span>
        </Tooltip>
      ) : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      render: (_: any, record: EnhancedPendingItem) => (
        <Space size="small">
          <Tooltip title="批准">
            <Button
              type="primary"
              size="small"
              icon={<Check />}
              onClick={() => handleQuickApprove(record)}
              loading={approving?.id === String(record.id) && approving?.action === 'approve'}
              className="!bg-green-500 !border-green-500 hover:!bg-green-600 hover:!border-green-600"
            >
              批准
            </Button>
          </Tooltip>
          <Tooltip title="拒绝">
            <Button
              danger
              size="small"
              icon={<X />}
              onClick={() => handleQuickReject(record)}
              loading={approving?.id === String(record.id) && approving?.action === 'reject'}
            >
              拒绝
            </Button>
          </Tooltip>
          <Tooltip title="查看详情">
            <Button
              size="small"
              icon={<Eye />}
              onClick={() => setDetailDrawer({ open: true, item: record })}
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  const totalPending = tickets.length + changes.length + serviceRequests.length;

  // 获取当前Tab的数据
  const getCurrentTabData = () => {
    switch (activeTab) {
      case 'tickets': return tickets;
      case 'changes': return changes;
      case 'service-requests': return serviceRequests;
      default: return [];
    }
  };

  // 空状态组件
  const EmptyState = ({ type }: { type: string }) => (
    <div className="flex flex-col items-center justify-center py-16 px-4">
      <div className="w-20 h-20 mb-6 rounded-full bg-gray-100 flex items-center justify-center">
        <Clock className="text-4xl text-gray-300" />
      </div>
      <Title level={4} className="text-gray-500 mb-2">暂无待审批{type}</Title>
      <Text type="secondary">当前没有需要审批的{type}，可以稍后刷新查看最新</Text>
    </div>
  );

  // 骨架屏加载
  const LoadingSkeleton = () => (
    <div className="space-y-4">
      {[1, 2, 3].map(i => (
        <Skeleton key={i} active paragraph={{ rows: 2 }} />
      ))}
    </div>
  );

  return (
    <div className="p-4 md:p-6">
      {/* 头部区域 */}
      <div className="mb-4 md:mb-6 flex flex-col md:flex-row md:justify-between md:items-center gap-4">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 md:w-12 md:h-12 rounded-xl bg-blue-50 flex items-center justify-center">
            <CheckCircle className="text-xl md:text-2xl text-blue-500" />
          </div>
          <div>
            <Title level={3} className="!mb-0 !text-xl md:!text-2xl">审批中心</Title>
            <Text type="secondary" className="text-sm">您有 {totalPending} 项待审批</Text>
          </div>
        </div>
        <Space wrap className="w-full md:w-auto justify-end">
          <Button
            icon={<RotateCcw className={loading ? 'animate-spin' : ''} />}
            onClick={handleRefresh}
            loading={loading}
            className="w-full md:w-auto"
          >
            刷新
          </Button>
          <Link href="/approvals/pending" className="w-full md:w-auto">
            <Button type="primary" icon={<Clock />} className="w-full md:w-auto">
              待我审批
            </Button>
          </Link>
        </Space>
      </div>

      {/* 统计卡片 - 响应式网格 */}
      <Row gutter={[12, 12]} className="mb-4 md:mb-6">
        <Col xs={12} sm={6}>
          <Card className="hover:shadow-md transition-shadow cursor-pointer border-l-4 border-l-blue-500" onClick={() => setActiveTab('tickets')}>
            <div className="flex items-center justify-between">
              <div>
                <div className="text-xs md:text-sm text-gray-500">待审批总数</div>
                <div className="text-2xl md:text-3xl font-bold text-blue-600">{totalPending}</div>
              </div>
              <div className="w-10 h-10 md:w-12 md:h-12 rounded-lg bg-blue-50 flex items-center justify-center">
                <Clock className="text-lg md:text-xl text-blue-500" />
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card className="hover:shadow-md transition-shadow cursor-pointer border-l-4 border-l-green-500" onClick={() => setActiveTab('tickets')}>
            <div className="flex items-center justify-between">
              <div>
                <div className="text-xs md:text-sm text-gray-500">工单待审</div>
                <div className="text-2xl md:text-3xl font-bold text-green-600">{tickets.length}</div>
              </div>
              <div className="w-10 h-10 md:w-12 md:h-12 rounded-lg bg-green-50 flex items-center justify-center">
                <FileText className="text-lg md:text-xl text-green-500" />
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card className="hover:shadow-md transition-shadow cursor-pointer border-l-4 border-l-orange-500" onClick={() => setActiveTab('changes')}>
            <div className="flex items-center justify-between">
              <div>
                <div className="text-xs md:text-sm text-gray-500">变更待审</div>
                <div className="text-2xl md:text-3xl font-bold text-orange-600">{changes.length}</div>
              </div>
              <div className="w-10 h-10 md:w-12 md:h-12 rounded-lg bg-orange-50 flex items-center justify-center">
                <Wrench className="text-lg md:text-xl text-orange-500" />
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card className="hover:shadow-md transition-shadow cursor-pointer border-l-4 border-l-purple-500" onClick={() => setActiveTab('service-requests')}>
            <div className="flex items-center justify-between">
              <div>
                <div className="text-xs md:text-sm text-gray-500">服务请求</div>
                <div className="text-2xl md:text-3xl font-bold text-purple-600">{serviceRequests.length}</div>
              </div>
              <div className="w-10 h-10 md:w-12 md:h-12 rounded-lg bg-purple-50 flex items-center justify-center">
                <Headphones className="text-lg md:text-xl text-purple-500" />
              </div>
            </div>
          </Card>
        </Col>
      </Row>

      {/* 批量操作工具栏 */}
      {selectedItems.length > 0 && (
        <Card className="mb-4 bg-blue-50 border-blue-200">
          <div className="flex flex-col md:flex-row md:items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <Badge count={selectedItems.length} />
              <Text>已选择 {selectedItems.length} 项</Text>
              <Button size="small" onClick={() => setSelectedItems([])}>取消</Button>
            </div>
            <Space>
              <Button type="primary" icon={<Check />} onClick={handleBatchApprove} className="!bg-green-500">
                批量批准
              </Button>
              <Button danger icon={<X />} onClick={handleBatchReject}>
                批量拒绝
              </Button>
            </Space>
          </div>
        </Card>
      )}

      {/* Tab切换和视图模式 */}
      <Card className="mb-4 md:mb-6">
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-4">
          <Tabs
            activeKey={activeTab}
            onChange={setActiveTab}
            className="w-full md:w-auto"
            items={[
              {
                key: 'tickets',
                label: (
                  <span className="flex items-center gap-2">
                    <FileText /> 工单
                    <Badge count={tickets.length} showZero={false} />
                  </span>
                ),
              },
              {
                key: 'changes',
                label: (
                  <span className="flex items-center gap-2">
                    <Wrench /> 变更
                    <Badge count={changes.length} showZero={false} />
                  </span>
                ),
              },
              {
                key: 'service-requests',
                label: (
                  <span className="flex items-center gap-2">
                    <Headphones /> 服务请求
                    <Badge count={serviceRequests.length} showZero={false} />
                  </span>
                ),
              },
            ]}
          />
          <div className="flex items-center gap-2">
            <Tooltip title="列表视图">
              <Button
                icon={<List />}
                type={viewMode === 'list' ? 'primary' : 'default'}
                onClick={() => setViewMode('list')}
                size="small"
              />
            </Tooltip>
            <Tooltip title="卡片视图">
              <Button
                icon={<LayoutGrid />}
                type={viewMode === 'card' ? 'primary' : 'default'}
                onClick={() => setViewMode('card')}
                size="small"
              />
            </Tooltip>
          </div>
        </div>

        {/* 表格内容 */}
        {loading ? (
          <LoadingSkeleton />
        ) : getCurrentTabData().length === 0 ? (
          <EmptyState type={activeTab === 'tickets' ? '工单' : activeTab === 'changes' ? '变更' : '服务请求'} />
        ) : viewMode === 'list' ? (
          <Table
            rowKey="id"
            dataSource={getCurrentTabData()}
            columns={cardColumns}
            pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (total) => `共 ${total} 项` }}
            scroll={{ x: 800 }}
            rowSelection={{
              selectedRowKeys: selectedItems.map(i => i.id),
              onChange: (_, selected) => setSelectedItems(selected as EnhancedPendingItem[]),
            }}
            className="approval-table"
          />
        ) : (
          // 卡片视图
          <Row gutter={[16, 16]}>
            {getCurrentTabData().map((item) => (
              <Col key={item.id} xs={24} sm={12} lg={8}>
                <Card
                  hoverable
                  className="h-full transition-all duration-300 hover:shadow-lg"
                  actions={[
                    <Button
                      key="approve"
                      type="text"
                      icon={<Check />}
                      onClick={(e) => { e.stopPropagation(); handleQuickApprove(item); }}
                      className="text-green-600 hover:text-green-700"
                    >
                      批准
                    </Button>,
                    <Button
                      key="reject"
                      type="text"
                      danger
                      icon={<X />}
                      onClick={(e) => { e.stopPropagation(); handleQuickReject(item); }}
                    >
                      拒绝
                    </Button>,
                    <Button
                      key="detail"
                      type="text"
                      icon={<Eye />}
                      onClick={(e) => { e.stopPropagation(); setDetailDrawer({ open: true, item }); }}
                    >
                      详情
                    </Button>,
                  ]}
                >
                  <div className="flex items-start justify-between mb-3">
                    <div className="flex items-center gap-2">
                      <input
                        type="checkbox"
                        checked={selectedItems.some(i => i.id === item.id)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setSelectedItems([...selectedItems, item]);
                          } else {
                            setSelectedItems(selectedItems.filter(i => i.id !== item.id));
                          }
                        }}
                        className="w-4 h-4 cursor-pointer accent-blue-500"
                      />
                      <Tag color={item.priority === 'critical' ? 'red' : item.priority === 'high' ? 'orange' : item.priority === 'medium' ? 'blue' : 'green'}>
                        {item.priority || '普通'}
                      </Tag>
                    </div>
                    <Text type="secondary" className="text-xs">{dayjs(item.createdAt).fromNow()}</Text>
                  </div>
                  <Link href={item.url}>
                    <Title level={5} className="!mb-2 !text-base hover:text-blue-600 transition-colors line-clamp-2">
                      {item.title}
                    </Title>
                  </Link>
                  <div className="flex items-center gap-2 mt-3">
                    <Avatar size="small" className="bg-blue-100 text-blue-600">
                      {item.requester?.charAt(0) || 'U'}
                    </Avatar>
                    <Text type="secondary" className="text-sm">{item.requester || '未知'}</Text>
                  </div>
                </Card>
              </Col>
            ))}
          </Row>
        )}
      </Card>

      {/* 详情抽屉 */}
      <Drawer
        title={`${detailDrawer.item?.title || '审批详情'}`}
        placement="right"
        width={500}
        onClose={() => setDetailDrawer({ open: false, item: null })}
        open={detailDrawer.open}
        extra={
          <Space>
            <Button
              type="primary"
              icon={<Check />}
              onClick={() => detailDrawer.item && handleQuickApprove(detailDrawer.item)}
              className="!bg-green-500"
            >
              批准
            </Button>
            <Button
              danger
              icon={<X />}
              onClick={() => detailDrawer.item && handleQuickReject(detailDrawer.item)}
            >
              拒绝
            </Button>
          </Space>
        }
      >
        {detailDrawer.item && (
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="类型">
              <Tag>{detailDrawer.item.type === 'ticket' ? '工单' : detailDrawer.item.type === 'change' ? '变更' : '服务请求'}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="优先级">
              <Tag color={detailDrawer.item.priority === 'critical' ? 'red' : detailDrawer.item.priority === 'high' ? 'orange' : 'blue'}>
                {detailDrawer.item.priority || '普通'}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag>{detailDrawer.item.status}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="申请人">{detailDrawer.item.requester || '-'}</Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {detailDrawer.item.createdAt ? dayjs(detailDrawer.item.createdAt).format('YYYY-MM-DD HH:mm:ss') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="描述">
              {detailDrawer.item.description || '无描述'}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
    </div>
  );
}
