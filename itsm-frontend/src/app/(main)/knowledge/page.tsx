'use client';

import React, { useState, useEffect } from 'react';
import { Card, Row, Col, Statistic, Typography, Tabs, Button, Space, Tag, Table, List, Avatar } from 'antd';
import { BookOpen, FileText, Eye, CheckCircle, Plus, Clock, User, Star, MessageCircle } from 'lucide-react';
import { useRouter } from 'next/navigation';
import ArticleList from '@/components/knowledge/ArticleList';

const { Title, Text, Paragraph } = Typography;

// Mock recent articles
const recentArticles = [
  { id: 1, title: '如何申请云服务器', views: 1250, author: '张三', date: '2024-01-15', category: '云资源' },
  { id: 2, title: 'VPN连接配置指南', views: 980, author: '李四', date: '2024-01-14', category: '网络' },
  { id: 3, title: '账号权限申请流程', views: 856, author: '王五', date: '2024-01-13', category: '账号' },
  { id: 4, title: '数据库备份策略', views: 742, author: '赵六', date: '2024-01-12', category: '数据库' },
  { id: 5, title: '安全组配置最佳实践', views: 625, author: '钱七', date: '2024-01-11', category: '安全' },
];

// Mock popular articles
const popularArticles = [
  { id: 1, title: 'IT服务请求常见问题解答', views: 3560, rating: 4.8, category: 'FAQ' },
  { id: 2, title: '云服务器申请完全指南', views: 2890, rating: 4.9, category: '云资源' },
  { id: 3, title: '企业邮箱配置教程', views: 2450, rating: 4.7, category: '账号' },
  { id: 4, title: '网络故障排查手册', views: 2100, rating: 4.6, category: '网络' },
  { id: 5, title: '数据库连接问题汇总', views: 1850, rating: 4.5, category: '数据库' },
];

export default function KnowledgePage() {
  const router = useRouter();
  const [activeTab, setActiveTab] = useState('list');
  const [stats, setStats] = useState({
    totalArticles: 0,
    publishedArticles: 0,
    draftArticles: 0,
    totalViews: 0,
  });

  // Fetch stats
  const fetchStats = async () => {
    try {
      // Mock data - in real app would call API
      setStats({
        totalArticles: 156,
        publishedArticles: 128,
        draftArticles: 28,
        totalViews: 4562,
      });
    } catch (error) {
      console.error('Failed to fetch knowledge stats:', error);
    }
  };

  useEffect(() => {
    fetchStats();
  }, []);

  const recentColumns = [
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      render: (text: string, record: any) => (
        <a onClick={() => router.push(`/knowledge/articles/${record.id}`)}>{text}</a>
      ),
    },
    {
      title: '分类',
      dataIndex: 'category',
      key: 'category',
      render: (category: string) => <Tag>{category}</Tag>,
    },
    {
      title: '作者',
      dataIndex: 'author',
      key: 'author',
    },
    {
      title: '更新时间',
      dataIndex: 'date',
      key: 'date',
    },
    {
      title: '浏览',
      dataIndex: 'views',
      key: 'views',
      render: (views: number) => (
        <span className="flex items-center gap-1">
          <Eye size={14} /> {views}
        </span>
      ),
    },
  ];

  const popularColumns = [
    {
      title: '排名',
      key: 'rank',
      width: 60,
      render: (_: any, __: any, index: number) => <span className="font-bold text-blue-500">#{index + 1}</span>,
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      render: (text: string, record: any) => (
        <a onClick={() => router.push(`/knowledge/articles/${record.id}`)}>{text}</a>
      ),
    },
    {
      title: '分类',
      dataIndex: 'category',
      key: 'category',
      render: (category: string) => <Tag color="blue">{category}</Tag>,
    },
    {
      title: '浏览量',
      dataIndex: 'views',
      key: 'views',
      render: (views: number) => (
        <span className="flex items-center gap-1">
          <Eye size={14} /> {views}
        </span>
      ),
    },
    {
      title: '评分',
      dataIndex: 'rating',
      key: 'rating',
      render: (rating: number) => (
        <span className="flex items-center gap-1">
          <Star size={14} className="text-yellow-500" fill="currentColor" /> {rating}
        </span>
      ),
    },
  ];

  return (
    <div className="p-6 min-h-screen bg-gray-50">
      {/* 页面头部 */}
      <div className="mb-6 flex justify-between items-center">
        <div>
          <Title level={2} style={{ marginBottom: 4 }}>
            知识库
          </Title>
          <Text type="secondary">
            创建、维护和分享解决方案与最佳实践
          </Text>
        </div>
        <Button
          type="primary"
          icon={<Plus className="w-4 h-4" />}
          size="large"
          onClick={() => router.push('/knowledge/articles/new')}
        >
          新建文章
        </Button>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="文章总数"
              value={stats.totalArticles}
              prefix={<BookOpen className="text-blue-500 mr-2" />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="已发布"
              value={stats.publishedArticles}
              prefix={<CheckCircle className="text-green-500 mr-2" />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="草稿"
              value={stats.draftArticles}
              prefix={<FileText className="text-orange-500 mr-2" />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="总浏览量"
              value={stats.totalViews}
              prefix={<Eye className="text-purple-500 mr-2" />}
              valueStyle={{ color: '#722ed1' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 主要内容区域 */}
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        size="large"
        items={[
          {
            key: 'list',
            label: (
              <span className="flex items-center gap-2">
                <FileText />
                文章列表
              </span>
            ),
            children: <ArticleList />,
          },
          {
            key: 'recent',
            label: (
              <span className="flex items-center gap-2">
                <Clock />
                最新更新
              </span>
            ),
            children: (
              <Card>
                <Table
                  columns={recentColumns}
                  dataSource={recentArticles}
                  rowKey="id"
                  pagination={false}
                />
              </Card>
            ),
          },
          {
            key: 'popular',
            label: (
              <span className="flex items-center gap-2">
                <Star />
                热门文章
              </span>
            ),
            children: (
              <Card>
                <Table
                  columns={popularColumns}
                  dataSource={popularArticles}
                  rowKey="id"
                  pagination={false}
                />
              </Card>
            ),
          },
        ]}
      />
    </div>
  );
}
