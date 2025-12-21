'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Button,
  Input,
  Select,
  Space,
  Tag,
  List,
  Typography,
  Divider,
  message,
  Spin,
  Empty,
  Tooltip,
  Modal,
  Form,
  Rate,
  Progress,
} from 'antd';
import {
  BookOpen,
  Search,
  Link,
  Star,
  Eye,
  MessageSquare,
  Sparkles,
  Plus,
  CheckCircle,
  ClockCircle,
  AlertCircle,
  RobotOutlined,
} from 'lucide-react';

// 解决方案推荐接口
interface SolutionRecommendation {
  id: number;
  title: string;
  summary: string;
  relevance: number;
  category: string;
  tags: string[];
  viewCount: number;
  rating: number;
  lastUpdated: string;
}

// 知识库关联接口
interface KnowledgeAssociation {
  id: number;
  articleId: number;
  articleTitle: string;
  associationType: 'auto' | 'manual' | 'suggested';
  associatedAt: string;
  associatedBy: string;
}

// AI推荐接口
interface AIRecommendation {
  type: 'category' | 'priority' | 'tags' | 'assignee';
  value: string;
  confidence: number;
  reasoning: string;
}

// 知识库文章接口
interface KnowledgeArticle {
  id: number;
  title: string;
  summary: string;
  category: string;
  tags: string[];
  viewCount: number;
  rating: number;
  lastUpdated: string;
  content: string;
}

// 知识库集成组件属性
interface KnowledgeIntegrationProps {
  ticketId: number;
  ticketTitle: string;
  ticketDescription: string;
  ticketCategory: string;
}

const KnowledgeIntegration: React.FC<KnowledgeIntegrationProps> = ({
  ticketId,
  ticketTitle,
  ticketDescription,
  ticketCategory,
}) => {
  // 状态管理
  const [recommendations, setRecommendations] = useState<SolutionRecommendation[]>([]);
  const [associations, setAssociations] = useState<KnowledgeAssociation[]>([]);
  const [aiRecommendations, setAiRecommendations] = useState<AIRecommendation[]>([]);
  const [relatedArticles, setRelatedArticles] = useState<KnowledgeArticle[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchModalVisible, setSearchModalVisible] = useState(false);
  const [associateModalVisible, setAssociateModalVisible] = useState(false);
  const [searchKeywords, setSearchKeywords] = useState<string[]>([]);
  const [searchResults, setSearchResults] = useState<KnowledgeArticle[]>([]);
  const [selectedArticle, setSelectedArticle] = useState<KnowledgeArticle | null>(null);
  const [associationType, setAssociationType] = useState<'auto' | 'manual' | 'suggested'>('manual');

  // 模拟数据加载
  useEffect(() => {
    loadData();
  }, [ticketId]);

  const loadData = async () => {
    setLoading(true);
    try {
      // 模拟API调用
      await Promise.all([
        loadRecommendations(),
        loadAssociations(),
        loadAIRecommendations(),
        loadRelatedArticles(),
      ]);
    } catch (error) {
      message.error('加载数据失败');
    } finally {
      setLoading(false);
    }
  };

  const loadRecommendations = async () => {
    // 模拟推荐数据
    const mockRecommendations: SolutionRecommendation[] = [
      {
        id: 1,
        title: '常见网络连接问题解决方案',
        summary: '针对网络连接异常的详细排查步骤和解决方案',
        relevance: 95,
        category: '网络故障',
        tags: ['网络', '连接', '故障排除'],
        viewCount: 1250,
        rating: 4.8,
        lastUpdated: '2024-01-15',
      },
      {
        id: 2,
        title: '系统性能优化指南',
        summary: '系统运行缓慢时的性能优化方法和工具使用',
        relevance: 87,
        category: '系统优化',
        tags: ['性能', '优化', '系统'],
        viewCount: 890,
        rating: 4.6,
        lastUpdated: '2024-01-10',
      },
    ];
    setRecommendations(mockRecommendations);
  };

  const loadAssociations = async () => {
    // 模拟关联数据
    const mockAssociations: KnowledgeAssociation[] = [
      {
        id: 1,
        articleId: 1,
        articleTitle: '常见网络连接问题解决方案',
        associationType: 'auto',
        associatedAt: '2024-01-20',
        associatedBy: '系统',
      },
    ];
    setAssociations(mockAssociations);
  };

  const loadAIRecommendations = async () => {
    // 模拟AI推荐数据
    const mockAIRecommendations: AIRecommendation[] = [
      {
        type: 'category',
        value: '网络故障',
        confidence: 92,
        reasoning: '基于工单描述中的网络连接问题关键词',
      },
      {
        type: 'priority',
        value: '高',
        confidence: 85,
        reasoning: '网络问题影响用户正常工作',
      },
      {
        type: 'tags',
        value: '网络,连接,故障',
        confidence: 88,
        reasoning: '工单内容与网络连接问题高度相关',
      },
    ];
    setAiRecommendations(mockAIRecommendations);
  };

  const loadRelatedArticles = async () => {
    // 模拟相关文章数据
    const mockRelatedArticles: KnowledgeArticle[] = [
      {
        id: 3,
        title: '网络设备配置检查清单',
        summary: '网络故障时的设备配置检查步骤',
        category: '网络故障',
        tags: ['网络', '配置', '检查'],
        viewCount: 650,
        rating: 4.5,
        lastUpdated: '2024-01-08',
        content: '详细的网络设备配置检查步骤...',
      },
      {
        id: 4,
        title: '故障排除最佳实践',
        summary: 'IT故障排除的标准流程和方法',
        category: '故障排除',
        tags: ['故障排除', '流程', '方法'],
        viewCount: 1200,
        rating: 4.7,
        lastUpdated: '2024-01-12',
        content: '故障排除的标准流程...',
      },
    ];
    setRelatedArticles(mockRelatedArticles);
  };

  // 搜索知识库
  const handleSearch = async () => {
    if (!searchKeywords.length) {
      message.warning('请输入搜索关键词');
      return;
    }

    try {
      // 模拟搜索API调用
      const mockSearchResults: KnowledgeArticle[] = [
        {
          id: 5,
          title: '网络故障诊断工具使用指南',
          summary: '各种网络诊断工具的使用方法和技巧',
          category: '网络故障',
          tags: ['网络', '诊断', '工具'],
          viewCount: 800,
          rating: 4.6,
          lastUpdated: '2024-01-05',
          content: '网络诊断工具的详细使用说明...',
        },
      ];
      setSearchResults(mockSearchResults);
      message.success('搜索完成');
    } catch (error) {
      message.error('搜索失败');
    }
  };

  // 关联文章
  const handleAssociate = async () => {
    if (!selectedArticle) {
      message.warning('请选择要关联的文章');
      return;
    }

    try {
      // 模拟关联API调用
      const newAssociation: KnowledgeAssociation = {
        id: Date.now(),
        articleId: selectedArticle.id,
        articleTitle: selectedArticle.title,
        associationType: associationType,
        associatedAt: new Date().toISOString().split('T')[0],
        associatedBy: '当前用户',
      };

      setAssociations([...associations, newAssociation]);
      setAssociateModalVisible(false);
      setSelectedArticle(null);
      message.success('文章关联成功');
    } catch (error) {
      message.error('关联失败');
    }
  };

  // 移除关联
  const handleRemoveAssociation = async (associationId: number) => {
    try {
      setAssociations(associations.filter(a => a.id !== associationId));
      message.success('关联已移除');
    } catch (error) {
      message.error('移除失败');
    }
  };

  if (loading) {
    return (
      <div className='flex justify-center items-center h-32'>
        <Spin size='large' />
      </div>
    );
  }

  return (
    <div className='space-y-4'>
      {/* 推荐解决方案 */}
      <Card
        title={
          <span className='flex items-center space-x-2'>
            <BookOpen className='w-5 h-5 text-blue-500' />
            <span>推荐解决方案</span>
          </span>
        }
        className='shadow-sm'
      >
        {recommendations.length > 0 ? (
          <List
            dataSource={recommendations}
            renderItem={item => (
              <List.Item
                actions={[
                  <Button key='view' type='link' icon={<Eye />}>
                    查看
                  </Button>,
                  <Button key='associate' type='link' icon={<Link />}>
                    关联
                  </Button>,
                ]}
              >
                <List.Item.Meta
                  title={
                    <div className='flex items-center space-x-2'>
                      <span>{item.title}</span>
                      <Tag color='blue'>{item.category}</Tag>
                      <div className='flex items-center space-x-1'>
                        <Rate disabled defaultValue={item.rating} />
                        <span className='text-sm text-gray-500'>({item.rating})</span>
                      </div>
                    </div>
                  }
                  description={
                    <div className='space-y-2'>
                      <p className='text-gray-600'>{item.summary}</p>
                      <div className='flex items-center space-x-2 text-sm text-gray-500'>
                        <span>相关度: {item.relevance}%</span>
                        <span>浏览量: {item.viewCount}</span>
                        <span>更新: {item.lastUpdated}</span>
                      </div>
                      <div className='flex flex-wrap gap-1'>
                        {item.tags.map(tag => (
                          <Tag key={tag}>{tag}</Tag>
                        ))}
                      </div>
                    </div>
                  }
                />
              </List.Item>
            )}
          />
        ) : (
          <Empty description='暂无推荐解决方案' />
        )}
      </Card>

      {/* AI智能推荐 */}
      <Card
        title={
          <span className='flex items-center space-x-2'>
            <RobotOutlined className='w-5 h-5 text-purple-500' />
            <span>AI智能推荐</span>
          </span>
        }
        className='shadow-sm'
      >
        {aiRecommendations.length > 0 ? (
          <div className='space-y-4'>
            {aiRecommendations.map(rec => (
              <div
                key={`${rec.type}-${rec.value}`}
                className='flex items-center justify-between p-3 bg-purple-50 rounded-md'
              >
                <div className='flex items-center space-x-3'>
                  <div className='w-8 h-8 bg-purple-100 rounded-full flex items-center justify-center'>
                    <Sparkles className='w-4 h-4 text-purple-600' />
                  </div>
                  <div>
                    <div className='font-medium'>
                      {rec.type === 'category' && '分类建议'}
                      {rec.type === 'priority' && '优先级建议'}
                      {rec.type === 'tags' && '标签建议'}
                      {rec.type === 'assignee' && '处理人建议'}
                    </div>
                    <div className='text-sm text-gray-600'>{rec.reasoning}</div>
                  </div>
                </div>
                <div className='text-right'>
                  <div className='font-medium text-purple-600'>{rec.value}</div>
                  <div className='text-sm text-gray-500'>置信度: {rec.confidence}%</div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <Empty description='暂无AI推荐' />
        )}
      </Card>

      {/* 知识库关联 */}
      <Card
        title={
          <span className='flex items-center space-x-2'>
            <Link className='w-5 h-5 text-green-500' />
            <span>知识库关联</span>
            <Button
              type='primary'
              size='small'
              icon={<Plus />}
              onClick={() => setAssociateModalVisible(true)}
            >
              关联文章
            </Button>
          </span>
        }
        className='shadow-sm'
      >
        {associations.length > 0 ? (
          <List
            dataSource={associations}
            renderItem={item => (
              <List.Item
                actions={[
                  <Button key='view' type='link' icon={<Eye />}>
                    查看
                  </Button>,
                  <Button
                    key='remove'
                    type='link'
                    danger
                    icon={<AlertCircle />}
                    onClick={() => handleRemoveAssociation(item.id)}
                  >
                    移除
                  </Button>,
                ]}
              >
                <List.Item.Meta
                  title={
                    <div className='flex items-center space-x-2'>
                      <span>{item.articleTitle}</span>
                      <Tag
                        color={
                          item.associationType === 'auto'
                            ? 'blue'
                            : item.associationType === 'manual'
                            ? 'green'
                            : 'orange'
                        }
                      >
                        {item.associationType === 'auto'
                          ? '自动关联'
                          : item.associationType === 'manual'
                          ? '手动关联'
                          : '建议关联'}
                      </Tag>
                    </div>
                  }
                  description={
                    <div className='text-sm text-gray-500'>
                      关联时间: {item.associatedAt} | 关联人: {item.associatedBy}
                    </div>
                  }
                />
              </List.Item>
            )}
          />
        ) : (
          <Empty description='暂无关联文章' />
        )}
      </Card>

      {/* 相关文章 */}
      <Card
        title={
          <span className='flex items-center space-x-2'>
            <MessageSquare className='w-5 h-5 text-orange-500' />
            <span>相关文章</span>
            <Button
              type='default'
              size='small'
              icon={<Search />}
              onClick={() => setSearchModalVisible(true)}
            >
              搜索更多
            </Button>
          </span>
        }
        className='shadow-sm'
      >
        {relatedArticles.length > 0 ? (
          <List
            dataSource={relatedArticles}
            renderItem={item => (
              <List.Item
                actions={[
                  <Button key='view' type='link' icon={<Eye />}>
                    查看
                  </Button>,
                  <Button key='associate' type='link' icon={<Link />}>
                    关联
                  </Button>,
                ]}
              >
                <List.Item.Meta
                  title={
                    <div className='flex items-center space-x-2'>
                      <span>{item.title}</span>
                      <Tag color='blue'>{item.category}</Tag>
                      <div className='flex items-center space-x-1'>
                        <Rate disabled defaultValue={item.rating} />
                        <span className='text-sm text-gray-500'>({item.rating})</span>
                      </div>
                    </div>
                  }
                  description={
                    <div className='space-y-2'>
                      <p className='text-gray-600'>{item.summary}</p>
                      <div className='flex items-center space-x-2 text-sm text-gray-500'>
                        <span>浏览量: {item.viewCount}</span>
                        <span>更新: {item.lastUpdated}</span>
                      </div>
                      <div className='flex flex-wrap gap-1'>
                        {item.tags.map(tag => (
                          <Tag key={tag}>{tag}</Tag>
                        ))}
                      </div>
                    </div>
                  }
                />
              </List.Item>
            )}
          />
        ) : (
          <Empty description='暂无相关文章' />
        )}
      </Card>

      {/* 搜索知识库模态框 */}
      <Modal
        title='搜索知识库'
        open={searchModalVisible}
        onCancel={() => setSearchModalVisible(false)}
        footer={null}
        width={800}
      >
        <div className='space-y-4'>
          <div className='flex space-x-2'>
            <Input
              placeholder='输入关键词，用逗号分隔'
              value={searchKeywords.join(', ')}
              onChange={e =>
                setSearchKeywords(
                  e.target.value
                    .split(',')
                    .map(k => k.trim())
                    .filter(k => k)
                )
              }
              onPressEnter={handleSearch}
            />
            <Button type='primary' icon={<Search />} onClick={handleSearch}>
              搜索
            </Button>
          </div>

          {searchResults.length > 0 && (
            <div className='space-y-2'>
              <Divider>搜索结果</Divider>
              <List
                dataSource={searchResults}
                renderItem={item => (
                  <List.Item
                    actions={[
                      <Button key='view' type='link' icon={<Eye />}>
                        查看
                      </Button>,
                      <Button
                        key='select'
                        type='primary'
                        size='small'
                        onClick={() => {
                          setSelectedArticle(item);
                          setAssociateModalVisible(true);
                          setSearchModalVisible(false);
                        }}
                      >
                        选择关联
                      </Button>,
                    ]}
                  >
                    <List.Item.Meta
                      title={
                        <div className='flex items-center space-x-2'>
                          <span>{item.title}</span>
                          <Tag color='blue'>{item.category}</Tag>
                        </div>
                      }
                      description={
                        <div className='space-y-2'>
                          <p className='text-gray-600'>{item.summary}</p>
                          <div className='flex items-center space-x-2 text-sm text-gray-500'>
                            <span>浏览量: {item.viewCount}</span>
                            <span>评分: {item.rating}</span>
                          </div>
                        </div>
                      }
                    />
                  </List.Item>
                )}
              />
            </div>
          )}
        </div>
      </Modal>

      {/* 关联文章模态框 */}
      <Modal
        title='关联知识库文章'
        open={associateModalVisible}
        onCancel={() => {
          setAssociateModalVisible(false);
          setSelectedArticle(null);
        }}
        onOk={handleAssociate}
        okText='确认关联'
        cancelText='取消'
      >
        <div className='space-y-4'>
          {selectedArticle && (
            <div className='p-3 bg-gray-50 rounded-md'>
              <div className='font-medium'>{selectedArticle.title}</div>
              <div className='text-sm text-gray-600'>{selectedArticle.summary}</div>
            </div>
          )}

          <div>
            <label className='block text-sm font-medium text-gray-700 mb-2'>关联类型</label>
            <Select value={associationType} onChange={setAssociationType} style={{ width: '100%' }}>
              <Select.Option value='manual'>手动关联</Select.Option>
              <Select.Option value='suggested'>建议关联</Select.Option>
            </Select>
          </div>

          <div className='text-sm text-gray-500'>
            关联后，该文章将显示在工单的知识库关联列表中，帮助处理人员快速找到相关解决方案。
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default KnowledgeIntegration;
