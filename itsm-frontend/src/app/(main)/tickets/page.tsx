'use client';

import React, { useState } from 'react';
import { Card, Typography, Space, Button, Tabs, Badge, Alert } from 'antd';
import { 
  PlusOutlined, 
  TableOutlined, 
  AppstoreOutlined, 
  BarChartOutlined,
  SearchOutlined,
  BellOutlined 
} from '@ant-design/icons';
import { useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import TicketList from '@/components/ticket/TicketList';
import TicketKanban from '@/components/ticket/TicketKanban';
import TicketAdvancedSearch from '@/components/ticket/TicketAdvancedSearch';

const { Title, Text } = Typography;

export default function TicketsPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [activeTab, setActiveTab] = useState('list');
  const [showAdvancedSearch, setShowAdvancedSearch] = useState(false);
  const [ticketStats, setTicketStats] = useState({
    total: 0,
    open: 0,
    overdue: 0,
    today: 0,
  });

  // 从URL参数获取当前标签页
  React.useEffect(() => {
    const tab = searchParams.get('tab');
    if (tab && ['list', 'kanban', 'analytics', 'search'].includes(tab)) {
      setActiveTab(tab);
    }
  }, [searchParams]);

  // 模拟获取工单统计数据
  React.useEffect(() => {
    // 这里应该调用API获取实际统计数据
    setTicketStats({
      total: 847,
      open: 124,
      overdue: 18,
      today: 23,
    });
  }, []);

  // 处理标签页切换
  const handleTabChange = (tab: string) => {
    setActiveTab(tab);
    const newParams = new URLSearchParams(searchParams.toString());
    newParams.set('tab', tab);
    router.push(`/tickets?${newParams.toString()}`, { scroll: false });
  };

  // 高级搜索处理
  const handleAdvancedSearch = (filters: any) => {
    console.log('Advanced search filters:', filters);
    // 这里应该调用实际的搜索API
    setActiveTab('list');
  };

  const handleSearchReset = () => {
    console.log('Search reset');
    // 重置搜索条件
  };

  return (
    <div className='min-h-screen bg-gray-50'>
      {/* 页面头部 */}
      <div className='bg-white border-b border-gray-200'>
        <div className='w-full px-6 py-4'>
          <div className='flex items-center justify-between'>
            <div>
              <Title level={2} style={{ marginBottom: 0 }}>
                工单管理
              </Title>
              <Text type='secondary'>
                全功能工单管理系统 - 支持列表、看板、分析多种视图
              </Text>
            </div>
            <Space>
              <Button 
                icon={<SearchOutlined />} 
                onClick={() => setShowAdvancedSearch(!showAdvancedSearch)}
              >
                高级搜索
              </Button>
              <Badge count={ticketStats.overdue} size="small">
                <Button icon={<BellOutlined />}>
                  SLA预警
                </Button>
              </Badge>
              <Link href='/tickets/create'>
                <Button type='primary' icon={<PlusOutlined />} size="large">
                  新建工单
                </Button>
              </Link>
            </Space>
          </div>

          {/* 统计数据栏 */}
          <div className='grid grid-cols-1 md:grid-cols-4 gap-4 mt-4'>
            <Card size="small">
              <div className='flex items-center justify-between'>
                <div>
                  <Text type='secondary'>总工单</Text>
                  <div className='text-2xl font-bold'>{ticketStats.total}</div>
                </div>
                <TableOutlined className='text-2xl text-blue-500' />
              </div>
            </Card>
            <Card size="small">
              <div className='flex items-center justify-between'>
                <div>
                  <Text type='secondary'>待处理</Text>
                  <div className='text-2xl font-bold text-orange-500'>{ticketStats.open}</div>
                </div>
                <BellOutlined className='text-2xl text-orange-500' />
              </div>
            </Card>
            <Card size="small">
              <div className='flex items-center justify-between'>
                <div>
                  <Text type='secondary'>超时工单</Text>
                  <div className='text-2xl font-bold text-red-500'>{ticketStats.overdue}</div>
                </div>
                <BellOutlined className='text-2xl text-red-500' />
              </div>
            </Card>
            <Card size="small">
              <div className='flex items-center justify-between'>
                <div>
                  <Text type='secondary'>今日新增</Text>
                  <div className='text-2xl font-bold text-green-500'>{ticketStats.today}</div>
                </div>
                <PlusOutlined className='text-2xl text-green-500' />
              </div>
            </Card>
          </div>
        </div>
      </div>

      {/* 高级搜索面板 */}
      {showAdvancedSearch && (
        <div className='bg-gray-50 border-b border-gray-200'>
          <div className='w-full px-6 py-4'>
            <TicketAdvancedSearch
              onSearch={handleAdvancedSearch}
              onReset={handleSearchReset}
            />
          </div>
        </div>
      )}

      {/* 主内容区域 */}
      <div className='w-full px-6 py-6'>
        {/* 功能提示 */}
        <Alert
          message="🎉 工单管理功能已全面升级"
          description="现在支持列表视图、看板视图、高级搜索、统计分析等完整功能，提供更高效的工单管理体验。"
          type="success"
          showIcon
          closable
          className="mb-4"
        />

        {/* 标签页导航 */}
        <Tabs
          activeKey={activeTab}
          onChange={handleTabChange}
          size="large"
          className="mb-6"
          items={[
            {
              key: 'list',
              label: (
                <span>
                  <TableOutlined />
                  列表视图
                </span>
              ),
            },
            {
              key: 'kanban',
              label: (
                <span>
                  <AppstoreOutlined />
                  看板视图
                </span>
              ),
            },
            {
              key: 'analytics',
              label: (
                <span>
                  <BarChartOutlined />
                  数据分析
                </span>
              ),
            },
          ]}
        />

        {/* 标签页内容 */}
        {activeTab === 'list' && (
          <TicketList 
            showHeader={false}
            pageSize={20}
          />
        )}
        
        {activeTab === 'kanban' && (
          <TicketKanban 
            onTicketSelect={(ticket) => router.push(`/tickets/${ticket.id}`)}
          />
        )}
        
        {activeTab === 'analytics' && (
          <div className="bg-white rounded-lg p-4">
            <div className="text-center py-8">
              <BarChartOutlined className="text-4xl text-gray-400 mb-4" />
              <Title level={4} type="secondary">数据分析功能</Title>
              <Text type="secondary">
                完整的数据分析功能已迁移至专门的统计页面
              </Text>
              <div className="mt-4">
                <Button 
                  type="primary" 
                  size="large"
                  onClick={() => router.push('/tickets/analytics')}
                >
                  查看详细分析
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* 快捷操作浮动按钮 */}
      <div className="fixed bottom-6 right-6 z-50">
        <Space direction="vertical" size="middle">
          <Button
            type="primary"
            shape="circle"
            size="large"
            icon={<PlusOutlined />}
            onClick={() => router.push('/tickets/create')}
            style={{
              boxShadow: '0 4px 12px rgba(24, 144, 255, 0.4)',
            }}
          />
        </Space>
      </div>
    </div>
  );
}
