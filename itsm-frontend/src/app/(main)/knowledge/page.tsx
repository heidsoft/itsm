'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Row,
  Col,
  Statistic,
  Typography,
  Tabs,
  Button,
  Space,
  Tag,
  Table,
  message,
  Spin,
  Alert,
  Input,
  Empty,
} from 'antd';
import {
  BookOpen,
  FileText,
  Eye,
  CheckCircle,
  Plus,
  Clock,
  Star,
  MessageCircle,
  Search,
  Sparkles,
} from 'lucide-react';
import { useRouter } from 'next/navigation';
import ArticleList from '@/components/knowledge/ArticleList';
import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';
import { ArticleStatus } from '@/types/knowledge-base';
import { httpClient } from '@/lib/api/http-client';

const { Title, Text } = Typography;

export default function KnowledgePage() {
  const router = useRouter();
  const [activeTab, setActiveTab] = useState('list');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // AI 智能搜索
  const [aiSearchQuery, setAiSearchQuery] = useState('');
  const [aiSearchResults, setAiSearchResults] = useState<any[]>([]);
  const [aiSearchLoading, setAiSearchLoading] = useState(false);
  const [aiSearchError, setAiSearchError] = useState<string | null>(null);
  const [showAiSearch, setShowAiSearch] = useState(false);

  const [recentArticles, setRecentArticles] = useState<
    Array<{
      id: string;
      title: string;
      views: number;
      author: string;
      date: string;
      category: string;
    }>
  >([]);

  const [popularArticles, setPopularArticles] = useState<
    Array<{
      id: string;
      title: string;
      views: number;
      author: string;
      date: string;
      category: string;
      rating: number;
    }>
  >([]);

  const [stats, setStats] = useState<{
    total: number;
    published: number;
    draft: number;
    views: number;
    rating: number;
    categories: Array<{ name: string; count: number }>;
  }>({
    total: 0,
    published: 0,
    draft: 0,
    views: 0,
    rating: 0,
    categories: [],
  });

  // Fetch stats and articles
  const fetchStats = async () => {
    setLoading(true);
    setError(null);
    try {
      const [kbStats, articlesData] = await Promise.all([
        KnowledgeBaseApi.getStats(),
        KnowledgeBaseApi.getArticles({ page: 1, pageSize: 20, status: ArticleStatus.PUBLISHED }),
      ]);

      // Map backend stats to frontend state
      setStats({
        total: kbStats.totalArticles || 0,
        published: kbStats.publishedArticles || 0,
        draft: kbStats.draftArticles || 0,
        views: kbStats.totalViews || 0,
        rating: 0, // Rating not directly available in stats, could be calculated from topArticles
        categories: Object.entries(kbStats.articlesByCategory || {}).map(([name, count]) => ({
          name,
          count,
        })),
      });

      // Map articles - backend fields: id (int), title, views (int), author (string), published_at, category (string)
      const articles = articlesData.articles || [];
      const mappedArticles = articles.map((a: any) => ({
        id: String(a.id),
        title: a.title || '',
        views: a.views || 0,
        author: a.author || '-',
        date: a.published_at ? new Date(a.published_at).toLocaleDateString() : '-',
        category: a.category || '-',
      }));

      setRecentArticles(mappedArticles);

      // Popular articles: sorted by views, add rating placeholder
      const sortedByViews = [...mappedArticles].sort((a, b) => b.views - a.views);
      setPopularArticles(sortedByViews.slice(0, 10).map(a => ({ ...a, rating: 0 })));
    } catch (error) {
      console.error('Failed to fetch knowledge stats:', error);
      setError('加载知识库数据失败，请稍后重试');
      message.error('获取知识库统计数据失败');
    } finally {
      setLoading(false);
    }
  };

  // AI 智能搜索
  const handleAISearch = async () => {
    if (!aiSearchQuery.trim()) {
      message.warning('请输入搜索关键词');
      return;
    }

    setAiSearchLoading(true);
    setAiSearchError(null);

    try {
      const response = await httpClient.post<any[]>('/api/ai/knowledge/search', {
        query: aiSearchQuery,
        limit: 10,
        type: 'kb',
      });

      setAiSearchResults(response || []);
      setShowAiSearch(true);
    } catch (error) {
      console.error('AI search error:', error);
      setAiSearchError('智能搜索服务暂时不可用，请使用普通搜索');
    } finally {
      setAiSearchLoading(false);
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
      render: (_: any, __: any, index: number) => (
        <span className="font-bold text-blue-500">#{index + 1}</span>
      ),
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
          <Star size={14} className="text-yellow-500" fill="currentColor" /> {rating.toFixed(1)}
        </span>
      ),
    },
  ];

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <Spin size="large" description="加载知识库数据..." />
      </div>
    );
  }

  return (
    <div className="p-6 min-h-screen bg-gray-50">
      {/* 页面头部 */}
      <div className="mb-6 flex justify-between items-center">
        <div>
          <Title level={2} style={{ marginBottom: 4 }}>
            知识库
          </Title>
          <Text type="secondary">创建、维护和分享解决方案与最佳实践</Text>
        </div>
        <div className="flex items-center gap-2">
          {/* AI 智能搜索 */}
          <Input.Search
            placeholder="AI 智能搜索..."
            value={aiSearchQuery}
            onChange={e => setAiSearchQuery(e.target.value)}
            onSearch={handleAISearch}
            loading={aiSearchLoading}
            enterButton={
              <Button type="text" icon={<Sparkles className="w-4 h-4 text-yellow-500" />} />
            }
            style={{ width: 300 }}
            onPressEnter={handleAISearch}
          />
          <Button
            type="primary"
            icon={<Plus className="w-4 h-4" />}
            size="large"
            onClick={() => router.push('/knowledge/articles/new')}
          >
            新建文章
          </Button>
        </div>
      </div>

      {/* AI 搜索结果展示 */}
      {showAiSearch && (
        <Card
          className="mb-6"
          title={
            <span className="flex items-center gap-2">
              <Sparkles className="w-4 h-4 text-yellow-500" />
              智能搜索结果
            </span>
          }
        >
          {aiSearchError && (
            <Alert title={aiSearchError} type="warning" showIcon className="mb-4" />
          )}
          {aiSearchResults.length > 0 ? (
            <div className="space-y-3">
              {aiSearchResults.map((result: any, index: number) => (
                <Card
                  key={index}
                  size="small"
                  hoverable
                  onClick={() => router.push(`/knowledge/articles/${result.id}`)}
                >
                  <div className="flex justify-between items-start">
                    <div>
                      <Text strong>{result.title || '未命名'}</Text>
                      {result.category && <Tag className="ml-2">{result.category}</Tag>}
                    </div>
                    <Tag
                      color={result.score > 0.8 ? 'green' : result.score > 0.5 ? 'blue' : 'default'}
                    >
                      匹配度: {Math.round((result.score || 0) * 100)}%
                    </Tag>
                  </div>
                  {result.snippet && (
                    <Text type="secondary" className="block mt-2">
                      {result.snippet}
                    </Text>
                  )}
                </Card>
              ))}
            </div>
          ) : (
            <Empty description="未找到相关内容" />
          )}
          <Button type="link" onClick={() => setShowAiSearch(false)} className="mt-2">
            关闭搜索结果
          </Button>
        </Card>
      )}

      {/* 错误提示 */}
      {error && (
        <Alert
          message={error}
          description="请检查网络连接或稍后重试"
          type="error"
          showIcon
          className="mb-6"
          action={
            <Button size="small" onClick={fetchStats}>
              重试
            </Button>
          }
        />
      )}

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm">
            <Statistic
              title="文章总数"
              value={stats.total}
              prefix={<BookOpen className="text-blue-500 mr-2" />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm">
            <Statistic
              title="已发布"
              value={stats.published}
              prefix={<CheckCircle className="text-green-500 mr-2" />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm">
            <Statistic
              title="草稿"
              value={stats.draft}
              prefix={<FileText className="text-orange-500 mr-2" />}
              styles={{ content: { color: '#fa8c16' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm">
            <Statistic
              title="总浏览量"
              value={stats.views}
              prefix={<Eye className="text-purple-500 mr-2" />}
              styles={{ content: { color: '#722ed1' } }}
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
            children: <ArticleList showHeader={false} />,
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
                  locale={{ emptyText: '暂无文章' }}
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
                  locale={{ emptyText: '暂无热门文章' }}
                />
              </Card>
            ),
          },
        ]}
      />
    </div>
  );
}
