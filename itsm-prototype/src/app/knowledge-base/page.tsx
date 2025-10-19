'use client';

import { Search, Plus, BookOpen, FileText, Tag as TagIcon, Calendar, User } from 'lucide-react';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import {
  Card,
  Table,
  Button,
  Tag,
  Space,
  Row,
  Col,
  Statistic,
  Select,
  Input,
  Tooltip,
  Typography,
  List,
  Badge,
} from 'antd';
import { KnowledgeApi } from '../lib/knowledge-api';
// AppLayout is handled by layout.tsx

const { Search: SearchInput } = Input;
const { Option } = Select;
const { Text } = Typography;

// 模拟知识文章数据
// const mockArticles = [] as unknown[];

const getCategoryColor = (category: string) => {
  const colors: Record<string, string> = {
    账号管理: 'blue',
    故障排除: 'red',
    网络连接: 'green',
    流程指南: 'purple',
    系统配置: 'orange',
  };
  return colors[category] || 'default';
};

interface Article {
  id: string;
  title: string;
  summary: string;
  category: string;
  views: number;
  content?: string;
}

const KnowledgeBasePage = () => {
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [searchText, setSearchText] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('');

  // 统计数据
  const [stats, setStats] = useState({
    total: 0,
    accountManagement: 0,
    troubleshooting: 0,
    network: 0,
  });

  useEffect(() => {
    loadArticles();
    loadStats();
  }, []);

  const loadArticles = async () => {
    setLoading(true);
    try {
      // 模拟数据
      const mockArticles: Article[] = [
        {
          id: '1',
          title: '如何重置企业邮箱密码',
          summary: '本文介绍了如何通过自助服务重置企业邮箱密码的详细步骤。',
          category: '账号管理',
          views: 128,
        },
        {
          id: '2',
          title: 'VPN连接常见问题及解决方案',
          summary: '整理了VPN连接过程中可能遇到的问题及其解决方案。',
          category: '故障排除',
          views: 96,
        },
        {
          id: '3',
          title: '内部系统访问权限申请流程',
          summary: '详细说明了如何申请和审批内部系统的访问权限。',
          category: '流程指南',
          views: 75,
        },
      ];
      setArticles(mockArticles);
    } catch (error) {
      console.error('加载知识库文章失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadStats = async () => {
    try {
      // 模拟统计数据
      setStats({
        total: 42,
        accountManagement: 12,
        troubleshooting: 18,
        network: 8,
      });
    } catch (error) {
      console.error('加载统计数据失败:', error);
    }
  };

  const handleCreateArticle = () => {
    window.location.href = '/knowledge-base/new';
  };

  // 渲染统计卡片
  const renderStatsCards = () => (
    <div style={{ marginBottom: 24 }}>
      <Row gutter={16}>
        <Col span={6}>
          <Card>
            <Statistic
              title='文章总数'
              value={stats.total}
              prefix={<BookOpen size={16} style={{ color: '#1890ff' }} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title='账号管理'
              value={stats.accountManagement}
              valueStyle={{ color: '#1890ff' }}
              prefix={<BookOpen size={16} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title='故障排除'
              value={stats.troubleshooting}
              valueStyle={{ color: '#ff4d4f' }}
              prefix={<BookOpen size={16} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title='网络连接'
              value={stats.network}
              valueStyle={{ color: '#52c41a' }}
              prefix={<BookOpen size={16} />}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );

  // 渲染筛选器
  const renderFilters = () => (
    <Card style={{ marginBottom: 24 }}>
      <Row gutter={20} align='middle'>
        <Col xs={24} sm={12} md={8}>
          <SearchInput
            placeholder='搜索文章标题或摘要...'
            allowClear
            onSearch={value => setSearchText(value)}
            size='large'
            enterButton
          />
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Select
            placeholder='分类筛选'
            size='large'
            allowClear
            value={categoryFilter}
            onChange={value => setCategoryFilter(value)}
            style={{ width: '100%' }}
          >
            <Option value='账号管理'>账号管理</Option>
            <Option value='故障排除'>故障排除</Option>
            <Option value='网络连接'>网络连接</Option>
            <Option value='流程指南'>流程指南</Option>
            <Option value='系统配置'>系统配置</Option>
          </Select>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Button
            icon={<Search size={20} />}
            onClick={() => {}}
            loading={loading}
            size='large'
            style={{ width: '100%' }}
          >
            刷新
          </Button>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Button
            type='primary'
            icon={<PlusCircle size={20} />}
            size='large'
            style={{ width: '100%' }}
            onClick={handleCreateArticle}
          >
            新建文章
          </Button>
        </Col>
      </Row>
    </Card>
  );

  // 渲染文章列表
  const renderArticleList = () => (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <div>
          {selectedRowKeys.length > 0 && (
            <Badge count={selectedRowKeys.length} showZero style={{ backgroundColor: '#1890ff' }} />
          )}
        </div>
      </div>

      <Table
        rowSelection={{
          selectedRowKeys,
          onChange: setSelectedRowKeys,
        }}
        columns={columns}
        dataSource={articles}
        rowKey='id'
        loading={loading}
        pagination={{
          pageSize: 10,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
        }}
        scroll={{ x: 1000 }}
      />
    </div>
  );

  // 表格列定义
  const columns = [
    {
      title: '文章信息',
      key: 'article_info',
      width: 400,
      render: (_: unknown, record: Article) => (
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <div
            style={{
              width: 40,
              height: 40,
              backgroundColor: '#e6f7ff',
              borderRadius: 8,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              marginRight: 12,
            }}
          >
            <BookOpen size={20} style={{ color: '#1890ff' }} />
          </div>
          <div>
            <div style={{ fontWeight: 'medium', color: '#000', marginBottom: 4 }}>
              <Link
                href={`/knowledge-base/${record.id}`}
                style={{ textDecoration: 'none', color: 'inherit' }}
              >
                {record.title}
              </Link>
            </div>
            <div style={{ fontSize: 'small', color: '#666' }}>{record.summary}</div>
          </div>
        </div>
      ),
    },
    {
      title: '分类',
      dataIndex: 'category',
      key: 'category',
      width: 120,
      render: (category: string) => <Tag color={getCategoryColor(category)}>{category}</Tag>,
    },
    {
      title: '浏览量',
      dataIndex: 'views',
      key: 'views',
      width: 100,
      render: (views: number) => <div style={{ fontSize: 'small' }}>{views}</div>,
    },
    {
      title: '操作',
      key: 'actions',
      width: 150,
      render: (_: unknown, record: Article) => (
        <Space size='small'>
          <Button
            type='text'
            size='small'
            icon={<Eye size={16} />}
            onClick={() => window.open(`/knowledge-base/${record.id}`)}
          />
          <Button
            type='text'
            size='small'
            icon={<Edit size={16} />}
            onClick={() => window.open(`/knowledge-base/${record.id}/edit`)}
          />
          <Button type='text' size='small' icon={<MoreHorizontal size={16} />} />
        </Space>
      ),
    },
  ];

  return (
    <div>
      {renderStatsCards()}
      {renderFilters()}
      {renderArticleList()}
    </div>
  );
};

export default KnowledgeBasePage;
