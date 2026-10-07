'use client';

import React, { useState, useEffect, useMemo } from 'react';
import {
  Card,
  Row,
  Col,
  Statistic,
  Typography,
  Tabs,
  Button,
  Tag,
  Table,
  App,
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
  Search,
  Sparkles,
} from 'lucide-react';
import { useRouter } from 'next/navigation';
import ArticleList from '@/components/knowledge/ArticleList';
import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';
import { ArticleStatus, type KnowledgeArticle } from '@/types/knowledge-base';
import { httpClient } from '@/lib/api/http-client';
import { useI18n } from '@/lib/i18n/useI18n';

const { Title, Text } = Typography;

// 可信 RAG：权威等级 → 文案
const authorityLabel = (level: number): string => {
  if (level >= 30) return '唯一真相源';
  if (level >= 20) return '官方标准';
  if (level >= 10) return '部门推荐';
  return '普通';
};

// 可信 RAG：时效窗口 → 文案（后端 L1 已过滤失效/未生效/逾期复核条目）
const freshnessLabel = (result: {
  validFrom?: string;
  validUntil?: string;
}): string => {
  if (result.validUntil) {
    const until = new Date(result.validUntil);
    if (!isNaN(until.getTime())) {
      return `有效至 ${until.toLocaleDateString('zh-CN')}`;
    }
  }
  return '长期有效';
};

export default function KnowledgePage() {
  const router = useRouter();
  const { t } = useI18n();
  const { message } = App.useApp();
  const [activeTab, setActiveTab] = useState('list');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [statsTimedOut, setStatsTimedOut] = useState(false);

  // AI 智能搜索
  const [aiSearchQuery, setAiSearchQuery] = useState('');
  const [aiSearchResults, setAiSearchResults] = useState<Array<Record<string, unknown>>>([]);
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

  const fetchStats = async () => {
    setLoading(true);
    setError(null);
    try {
      const [kbStats, articlesData] = await Promise.all([
        KnowledgeBaseApi.getStats(),
        KnowledgeBaseApi.getArticles({ page: 1, pageSize: 20, status: ArticleStatus.PUBLISHED }),
      ]);

      setStats({
        total: kbStats.total || 0,
        published: kbStats.published || 0,
        draft: kbStats.draft || 0,
        views: kbStats.views || 0,
        rating: typeof kbStats.rating === 'number' ? kbStats.rating : 0,
        categories: Array.isArray(kbStats.categories)
          ? kbStats.categories.map((c: Record<string, unknown>) => ({ name: String(c.name ?? ''), count: Number(c.count ?? 0) }))
          : [],
      });

      const articles = articlesData.items ?? [];
      const mappedArticles = articles.map((a: KnowledgeArticle) => ({
        id: String(a.id ?? ''),
        title: a.title || '',
        views: a.viewCount ?? 0,
        author: a.authorName || '-',
        date: 'publishedAt' in a && a.publishedAt ? new Date(String(a.publishedAt)).toLocaleDateString() : '-',
        category: a.categoryName || '-',
      }));

      setRecentArticles(mappedArticles);

      const sortedByViews = [...mappedArticles].sort((a, b) => b.views - a.views);
      setPopularArticles(sortedByViews.slice(0, 10).map(a => ({ ...a, rating: 0 })));
    } catch (err) {
      console.error('Failed to fetch knowledge stats:', err);
      setError(t('knowledgeBase.loadFailed'));
      message.error(t('knowledgeBase.loadFailedShort'));
    } finally {
      setLoading(false);
    }
  };

  const handleAISearch = async () => {
    if (!aiSearchQuery.trim()) {
      message.warning(t('knowledgeBase.aiSearchPromptRequired'));
      return;
    }

    setAiSearchLoading(true);
    setAiSearchError(null);

    try {
      const response = await httpClient.post<{ results: Array<Record<string, unknown>>; degraded?: boolean }>(
        '/api/v1/ai/rag/search',
        {
          query: aiSearchQuery,
          limit: 10,
          type: 'kb',
        }
      );

      const answers = (response as { results?: Array<Record<string, unknown>> })?.results ?? [];
      setAiSearchResults(Array.isArray(answers) ? answers : []);
      setShowAiSearch(true);
      if (Array.isArray(answers) && answers.length === 0) {
        message.info(t('knowledgeBase.aiSearchNoResults'));
      }
    } catch (err) {
      console.error('AI search error:', err);
      setAiSearchError(t('knowledgeBase.aiSearchError'));
      setShowAiSearch(true);
    } finally {
      setAiSearchLoading(false);
    }
  };

  useEffect(() => {
    fetchStats();
  }, []);

  const recentColumns = useMemo(
    () => [
      {
        title: t('knowledgeBase.titleCol'),
        dataIndex: 'title',
        key: 'title',
        render: (text: string, record: Record<string, unknown>) => (
          <a onClick={() => router.push(`/knowledge/articles/${record.id}`)}>{text}</a>
        ),
      },
      {
        title: t('knowledgeBase.category'),
        dataIndex: 'category',
        key: 'category',
        render: (category: string) => <Tag>{category}</Tag>,
      },
      {
        title: t('knowledgeBase.author'),
        dataIndex: 'author',
        key: 'author',
      },
      {
        title: t('knowledgeBase.updatedAt'),
        dataIndex: 'date',
        key: 'date',
      },
      {
        title: t('knowledgeBase.views'),
        dataIndex: 'views',
        key: 'views',
        render: (views: number) => (
          <span className="flex items-center gap-1">
            <Eye size={14} /> {views}
          </span>
        ),
      },
    ],
    [t, router]
  );

  const popularColumns = useMemo(
    () => [
      {
        title: t('knowledgeBase.rank'),
        key: 'rank',
        width: 60,
        render: (_: unknown, __: unknown, index: number) => (
          <span className="font-bold text-blue-500">#{index + 1}</span>
        ),
      },
      {
        title: t('knowledgeBase.titleCol'),
        dataIndex: 'title',
        key: 'title',
        render: (text: string, record: Record<string, unknown>) => (
          <a onClick={() => router.push(`/knowledge/articles/${record.id}`)}>{text}</a>
        ),
      },
      {
        title: t('knowledgeBase.category'),
        dataIndex: 'category',
        key: 'category',
        render: (category: string) => <Tag color="blue">{category}</Tag>,
      },
      {
        title: t('knowledgeBase.views'),
        dataIndex: 'views',
        key: 'views',
        render: (views: number) => (
          <span className="flex items-center gap-1">
            <Eye size={14} /> {views}
          </span>
        ),
      },
      {
        title: t('knowledgeBase.rating'),
        dataIndex: 'rating',
        key: 'rating',
        render: (rating: number) => (
          <span className="flex items-center gap-1">
            <Star size={14} className="text-yellow-500" fill="currentColor" /> {rating.toFixed(1)}
          </span>
        ),
      },
    ],
    [t, router]
  );

  const tabItems = useMemo(
    () => [
      {
        key: 'list',
        label: (
          <span className="flex items-center gap-2">
            <FileText />
            {t('knowledgeBase.articleList')}
          </span>
        ),
        children: <ArticleList showHeader={false} />,
      },
      {
        key: 'recent',
        label: (
          <span className="flex items-center gap-2">
            <Clock />
            {t('knowledgeBase.recentArticles')}
          </span>
        ),
        children: (
          <Card>
            <Table
              columns={recentColumns}
              dataSource={recentArticles}
              rowKey="id"
              pagination={false}
              locale={{ emptyText: t('knowledgeBase.emptyRecent') }}
            />
          </Card>
        ),
      },
      {
        key: 'popular',
        label: (
          <span className="flex items-center gap-2">
            <Star />
            {t('knowledgeBase.popularArticles')}
          </span>
        ),
        children: (
          <Card>
            <Table
              columns={popularColumns}
              dataSource={popularArticles}
              rowKey="id"
              pagination={false}
              locale={{ emptyText: t('knowledgeBase.emptyPopular') }}
            />
          </Card>
        ),
      },
    ],
    [t, recentColumns, popularColumns, recentArticles, popularArticles]
  );

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <Spin size="large" description={t('knowledgeBase.loadingData')} />
      </div>
    );
  }

  return (
    <div className="p-6 min-h-screen bg-gray-50">
      <div className="mb-6 flex justify-between items-center">
        <div>
          <Title level={2} style={{ marginBottom: 4 }}>
            {t('knowledgeBase.title')}
          </Title>
          <Text type="secondary">{t('knowledgeBase.pageDescription')}</Text>
        </div>
        <div className="flex items-center gap-2">
          <Input.Search
            placeholder={t('knowledgeBase.aiSearchPlaceholder')}
            value={aiSearchQuery}
            onChange={e => setAiSearchQuery(e.target.value)}
            onSearch={handleAISearch}
            loading={aiSearchLoading}
            enterButton={
              <Button type="text" aria-label="AI 搜索" icon={<Sparkles className="w-4 h-4 text-yellow-500" />} />
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
            {t('knowledgeBase.createArticle')}
          </Button>
        </div>
      </div>

      {showAiSearch && (
        <Card
          className="mb-6"
          title={
            <span className="flex items-center gap-2">
              <Sparkles className="w-4 h-4 text-yellow-500" />
              {t('knowledgeBase.aiSearchResults')}
            </span>
          }
        >
          {aiSearchError && (
            <Alert title={aiSearchError} type="warning" showIcon className="mb-4" />
          )}
          {aiSearchResults.length > 0 ? (
            <div className="space-y-3">
              {aiSearchResults.map((result, index) => {
                const resultId = String(result.id ?? '');
                const resultTitle = String(result.title ?? t('knowledgeBase.aiSearchUnnamed'));
                const resultCategory = result.category ? String(result.category) : undefined;
                const authorityLevel = typeof result.authorityLevel === 'number' ? result.authorityLevel : undefined;
                const isRestricted = typeof result.isRestricted === 'boolean' ? result.isRestricted : undefined;
                const score = typeof result.score === 'number' ? result.score : 0;
                const snippet = result.snippet ? String(result.snippet) : undefined;

                return (
                <Card
                  key={index}
                  size="small"
                  hoverable
                  onClick={() => router.push(`/knowledge/articles/${resultId}`)}
                >
                  <div className="flex justify-between items-start">
                    <div>
                      <Text strong>{resultTitle}</Text>
                      {resultCategory && <Tag className="ml-2">{resultCategory}</Tag>}
                      {/* 可信 RAG：权威等级标签 */}
                      {authorityLevel !== undefined && authorityLevel > 0 && (
                        <Tag color="gold" className="ml-1">
                          {authorityLabel(authorityLevel)}
                        </Tag>
                      )}
                    </div>
                    <div className="flex items-center gap-1 flex-wrap justify-end">
                      {/* 可信 RAG：权限标签（分类级可见性守卫 L0） */}
                      {isRestricted !== undefined && (
                        <Tag color={isRestricted ? 'red' : 'green'}>
                          {isRestricted ? '受限' : '公开'}
                        </Tag>
                      )}
                      <Tag
                        color={score > 0.8 ? 'green' : score > 0.5 ? 'blue' : 'default'}
                      >
                        {t('knowledgeBase.aiSearchMatchScore', {
                          score: Math.round(score * 100),
                        })}
                      </Tag>
                    </div>
                  </div>
                  {snippet && (
                    <Text type="secondary" className="block mt-2">
                      {snippet}
                    </Text>
                  )}
                  {/* 可信 RAG：时效标签（L1 已过滤失效/未生效/逾期复核，此处展示有效期窗口） */}
                  {freshnessLabel(result as { validFrom?: string; validUntil?: string }) && (
                    <div className="mt-2">
                      <Tag color="cyan">{freshnessLabel(result as { validFrom?: string; validUntil?: string })}</Tag>
                    </div>
                  )}
                </Card>
                );
              })}
            </div>
          ) : (
            <Empty description={t('knowledgeBase.aiSearchEmpty')} />
          )}
          <Button type="link" onClick={() => setShowAiSearch(false)} className="mt-2">
            {t('knowledgeBase.closeSearchResults')}
          </Button>
        </Card>
      )}

      {error && (
        <Alert
          title={error}
          description={t('knowledgeBase.retryHint')}
          type="error"
          showIcon
          className="mb-6"
          action={
            <Button size="small" onClick={fetchStats}>
              {t('knowledgeBase.retry')}
            </Button>
          }
        />
      )}

      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm">
            <Statistic
              title={t('knowledgeBase.totalArticles')}
              value={stats.total}
              prefix={<BookOpen className="text-blue-500 mr-2" />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm">
            <Statistic
              title={t('knowledgeBase.publishedArticles')}
              value={stats.published}
              prefix={<CheckCircle className="text-green-500 mr-2" />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm">
            <Statistic
              title={t('knowledgeBase.draftArticles')}
              value={stats.draft}
              prefix={<FileText className="text-orange-500 mr-2" />}
              styles={{ content: { color: '#fa8c16' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm">
            <Statistic
              title={t('knowledgeBase.totalViews')}
              value={stats.views}
              prefix={<Eye className="text-purple-500 mr-2" />}
              styles={{ content: { color: '#722ed1' } }}
            />
          </Card>
        </Col>
      </Row>

      <Tabs activeKey={activeTab} onChange={setActiveTab} size="large" items={tabItems} />
    </div>
  );
}
