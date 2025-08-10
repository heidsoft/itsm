"use client";

import React, { useState, useEffect } from "react";
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
} from "antd";
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
} from "@ant-design/icons";

const { Text, Title, Paragraph } = Typography;
const { TextArea } = Input;
const { Option } = Select;

// 接口定义
interface SolutionRecommendation {
  article_id: number;
  title: string;
  content: string;
  relevance_score: number;
  category: string;
  tags: string[];
  last_updated: string;
}

interface KnowledgeAssociation {
  ticket_id: number;
  article_id: number;
  association_type: string;
  relevance_score: number;
  created_at: string;
}

interface AIRecommendation {
  ticket_id: number;
  suggested_category: string;
  suggested_priority: string;
  suggested_tags: string[];
  suggested_assignee?: number;
  confidence: number;
  reason: string;
}

interface KnowledgeArticle {
  id: number;
  title: string;
  content: string;
  category: string;
  tags: string;
  created_at: string;
  updated_at: string;
  view_count: number;
}

interface KnowledgeIntegrationProps {
  ticketId: number;
  ticketTitle: string;
  ticketDescription: string;
  ticketCategory?: string;
}

const KnowledgeIntegration: React.FC<KnowledgeIntegrationProps> = ({
  ticketId,
  ticketTitle,
  ticketDescription,
  ticketCategory,
}) => {
  // 状态管理
  const [recommendations, setRecommendations] = useState<
    SolutionRecommendation[]
  >([]);
  const [associations, setAssociations] = useState<KnowledgeAssociation[]>([]);
  const [aiRecommendation, setAiRecommendation] =
    useState<AIRecommendation | null>(null);
  const [relatedArticles, setRelatedArticles] = useState<KnowledgeArticle[]>(
    []
  );
  const [loading, setLoading] = useState(false);
  const [searchModalVisible, setSearchModalVisible] = useState(false);
  const [searchResults, setSearchResults] = useState<KnowledgeArticle[]>([]);
  const [searchLoading, setSearchLoading] = useState(false);
  const [associateModalVisible, setAssociateModalVisible] = useState(false);
  const [selectedArticle, setSelectedArticle] =
    useState<KnowledgeArticle | null>(null);

  // 表单状态
  const [searchForm] = Form.useForm();
  const [associateForm] = Form.useForm();

  // 加载数据
  useEffect(() => {
    loadKnowledgeData();
  }, [ticketId]);

  const loadKnowledgeData = async () => {
    setLoading(true);
    try {
      // 并行加载所有数据
      const [recRes, assocRes, aiRes, relatedRes] = await Promise.allSettled([
        fetchRecommendations(),
        fetchAssociations(),
        fetchAIRecommendations(),
        fetchRelatedArticles(),
      ]);

      if (recRes.status === "fulfilled") {
        setRecommendations(recRes.value || []);
      }
      if (assocRes.status === "fulfilled") {
        setAssociations(assocRes.value || []);
      }
      if (aiRes.status === "fulfilled") {
        setAiRecommendation(aiRes.value || null);
      }
      if (relatedRes.status === "fulfilled") {
        setRelatedArticles(relatedRes.value || []);
      }
    } catch (error) {
      console.error("加载知识库数据失败:", error);
      message.error("加载知识库数据失败");
    } finally {
      setLoading(false);
    }
  };

  // API调用函数
  const fetchRecommendations = async (): Promise<SolutionRecommendation[]> => {
    try {
      const response = await fetch(
        `/api/v1/tickets/${ticketId}/knowledge/recommendations?limit=5`
      );
      if (response.ok) {
        const data = await response.json();
        return data.data || [];
      }
    } catch (error) {
      console.error("获取推荐解决方案失败:", error);
    }
    return [];
  };

  const fetchAssociations = async (): Promise<KnowledgeAssociation[]> => {
    try {
      const response = await fetch(
        `/api/v1/tickets/${ticketId}/knowledge/associations`
      );
      if (response.ok) {
        const data = await response.json();
        return data.data || [];
      }
    } catch (error) {
      console.error("获取知识库关联失败:", error);
    }
    return [];
  };

  const fetchAIRecommendations = async (): Promise<AIRecommendation | null> => {
    try {
      const response = await fetch(
        `/api/v1/tickets/${ticketId}/knowledge/ai-recommendations`
      );
      if (response.ok) {
        const data = await response.json();
        return data.data || null;
      }
    } catch (error) {
      console.error("获取AI建议失败:", error);
    }
    return null;
  };

  const fetchRelatedArticles = async (): Promise<KnowledgeArticle[]> => {
    try {
      const response = await fetch(
        `/api/v1/tickets/${ticketId}/knowledge/related-articles?limit=10`
      );
      if (response.ok) {
        const data = await response.json();
        return data.data || [];
      }
    } catch (error) {
      console.error("获取相关文章失败:", error);
    }
    return [];
  };

  // 搜索知识库
  const handleSearch = async (values: {
    keywords: string[];
    limit: number;
  }) => {
    setSearchLoading(true);
    try {
      const response = await fetch("/api/v1/knowledge/search", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          keywords: values.keywords,
          limit: values.limit || 10,
        }),
      });

      if (response.ok) {
        const data = await response.json();
        setSearchResults(data.data || []);
      } else {
        message.error("搜索失败");
      }
    } catch (error) {
      console.error("搜索知识库失败:", error);
      message.error("搜索失败");
    } finally {
      setSearchLoading(false);
    }
  };

  // 关联知识库文章
  const handleAssociate = async (values: { association_type: string }) => {
    if (!selectedArticle) return;

    try {
      const response = await fetch(
        `/api/v1/tickets/${ticketId}/knowledge/associate`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            article_id: selectedArticle.id,
            association_type: values.association_type,
          }),
        }
      );

      if (response.ok) {
        message.success("关联成功");
        setAssociateModalVisible(false);
        loadKnowledgeData(); // 重新加载数据
      } else {
        message.error("关联失败");
      }
    } catch (error) {
      console.error("关联知识库文章失败:", error);
      message.error("关联失败");
    }
  };

  // 渲染推荐解决方案
  const renderRecommendations = () => (
    <Card
      title={
        <Space>
          <BookOpen className="text-blue-500" />
          <span>推荐解决方案</span>
          <Button
            size="small"
            icon={<Sparkles />}
            onClick={loadKnowledgeData}
            loading={loading}
          >
            刷新推荐
          </Button>
        </Space>
      }
      className="mb-4"
    >
      {loading ? (
        <div className="text-center py-8">
          <Spin size="large" />
        </div>
      ) : recommendations.length > 0 ? (
        <List
          dataSource={recommendations}
          renderItem={(item) => (
            <List.Item
              actions={[
                <Button
                  key="view"
                  type="link"
                  icon={<Eye />}
                  onClick={() =>
                    window.open(`/knowledge-base/${item.article_id}`, "_blank")
                  }
                >
                  查看
                </Button>,
                <Button
                  key="associate"
                  type="link"
                  icon={<Link />}
                  onClick={() => {
                    setSelectedArticle({
                      id: item.article_id,
                      title: item.title,
                      content: item.content,
                      category: item.category,
                      tags: item.tags.join(","),
                      created_at: item.last_updated,
                      updated_at: item.last_updated,
                      view_count: 0,
                    });
                    setAssociateModalVisible(true);
                  }}
                >
                  关联
                </Button>,
              ]}
            >
              <List.Item.Meta
                title={
                  <Space>
                    <span>{item.title}</span>
                    <Tag color="blue">{item.category}</Tag>
                    <Progress
                      percent={Math.round(item.relevance_score * 100)}
                      size="small"
                      showInfo={false}
                      strokeColor="#52c41a"
                    />
                  </Space>
                }
                description={
                  <div>
                    <Paragraph ellipsis={{ rows: 2 }} className="mb-2">
                      {item.content}
                    </Paragraph>
                    <Space wrap>
                      {item.tags.map((tag) => (
                        <Tag key={tag} color="green">
                          {tag}
                        </Tag>
                      ))}
                    </Space>
                    <div className="text-xs text-gray-500 mt-2">
                      相关性: {Math.round(item.relevance_score * 100)}% |
                      更新时间:{" "}
                      {new Date(item.last_updated).toLocaleDateString()}
                    </div>
                  </div>
                }
              />
            </List.Item>
          )}
        />
      ) : (
        <Empty description="暂无推荐解决方案" />
      )}
    </Card>
  );

  // 渲染AI建议
  const renderAIRecommendations = () => (
    <Card
      title={
        <Space>
          <RobotOutlined className="text-purple-500" />
          <span>AI 智能建议</span>
        </Space>
      }
      className="mb-4"
    >
      {aiRecommendation ? (
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Text strong>建议分类:</Text>
              <div className="mt-1">
                <Tag color="blue">{aiRecommendation.suggested_category}</Tag>
              </div>
            </div>
            <div>
              <Text strong>建议优先级:</Text>
              <div className="mt-1">
                <Tag
                  color={
                    aiRecommendation.suggested_priority === "high"
                      ? "red"
                      : "orange"
                  }
                >
                  {aiRecommendation.suggested_priority}
                </Tag>
              </div>
            </div>
          </div>

          <div>
            <Text strong>建议标签:</Text>
            <div className="mt-2">
              {aiRecommendation.suggested_tags.map((tag) => (
                <Tag key={tag} color="green" className="mb-1">
                  {tag}
                </Tag>
              ))}
            </div>
          </div>

          <div>
            <Text strong>置信度:</Text>
            <div className="mt-1">
              <Progress
                percent={Math.round(aiRecommendation.confidence * 100)}
                size="small"
                strokeColor="#52c41a"
              />
            </div>
          </div>

          <div>
            <Text strong>建议理由:</Text>
            <Paragraph className="mt-1 text-gray-600">
              {aiRecommendation.reason}
            </Paragraph>
          </div>
        </div>
      ) : (
        <Empty description="暂无AI建议" />
      )}
    </Card>
  );

  // 渲染知识库关联
  const renderAssociations = () => (
    <Card
      title={
        <Space>
          <Link className="text-green-500" />
          <span>知识库关联</span>
          <Button
            size="small"
            icon={<Plus />}
            onClick={() => setSearchModalVisible(true)}
          >
            添加关联
          </Button>
        </Space>
      }
      className="mb-4"
    >
      {associations.length > 0 ? (
        <List
          dataSource={associations}
          renderItem={(item) => (
            <List.Item
              actions={[
                <Button
                  key="view"
                  type="link"
                  icon={<Eye />}
                  onClick={() =>
                    window.open(`/knowledge-base/${item.article_id}`, "_blank")
                  }
                >
                  查看
                </Button>,
              ]}
            >
              <List.Item.Meta
                title={
                  <Space>
                    <span>文章 #{item.article_id}</span>
                    <Tag
                      color={
                        item.association_type === "auto"
                          ? "blue"
                          : item.association_type === "manual"
                          ? "green"
                          : "orange"
                      }
                    >
                      {item.association_type === "auto"
                        ? "自动关联"
                        : item.association_type === "manual"
                        ? "手动关联"
                        : "建议关联"}
                    </Tag>
                  </Space>
                }
                description={
                  <div>
                    <div className="text-sm text-gray-600">
                      关联时间: {new Date(item.created_at).toLocaleString()}
                    </div>
                    <div className="text-xs text-gray-500">
                      相关性: {Math.round(item.relevance_score * 100)}%
                    </div>
                  </div>
                }
              />
            </List.Item>
          )}
        />
      ) : (
        <Empty description="暂无知识库关联" />
      )}
    </Card>
  );

  // 渲染相关文章
  const renderRelatedArticles = () => (
    <Card
      title={
        <Space>
          <MessageSquare className="text-orange-500" />
          <span>相关文章</span>
        </Space>
      }
      className="mb-4"
    >
      {relatedArticles.length > 0 ? (
        <List
          dataSource={relatedArticles}
          renderItem={(item) => (
            <List.Item
              actions={[
                <Button
                  key="view"
                  type="link"
                  icon={<Eye />}
                  onClick={() =>
                    window.open(`/knowledge-base/${item.id}`, "_blank")
                  }
                >
                  查看
                </Button>,
                <Button
                  key="associate"
                  type="link"
                  icon={<Link />}
                  onClick={() => {
                    setSelectedArticle(item);
                    setAssociateModalVisible(true);
                  }}
                >
                  关联
                </Button>,
              ]}
            >
              <List.Item.Meta
                title={
                  <Space>
                    <span>{item.title}</span>
                    <Tag color="blue">{item.category}</Tag>
                  </Space>
                }
                description={
                  <div>
                    <Paragraph ellipsis={{ rows: 2 }} className="mb-2">
                      {item.content}
                    </Paragraph>
                    <div className="text-xs text-gray-500">
                      更新时间: {new Date(item.updated_at).toLocaleDateString()}{" "}
                      | 浏览次数: {item.view_count}
                    </div>
                  </div>
                }
              />
            </List.Item>
          )}
        />
      ) : (
        <Empty description="暂无相关文章" />
      )}
    </Card>
  );

  return (
    <div className="space-y-4">
      {/* 快速操作栏 */}
      <Card className="bg-blue-50">
        <div className="flex items-center justify-between">
          <div>
            <Title level={5} className="mb-2">
              <BookOpen className="mr-2" />
              知识库助手
            </Title>
            <Text type="secondary">基于工单内容智能推荐解决方案和相关知识</Text>
          </div>
          <Space>
            <Button
              icon={<Search />}
              onClick={() => setSearchModalVisible(true)}
            >
              搜索知识库
            </Button>
            <Button
              type="primary"
              icon={<Sparkles />}
              onClick={loadKnowledgeData}
              loading={loading}
            >
              智能分析
            </Button>
          </Space>
        </div>
      </Card>

      {/* 主要内容区域 */}
      {renderRecommendations()}
      {renderAIRecommendations()}
      {renderAssociations()}
      {renderRelatedArticles()}

      {/* 搜索知识库模态框 */}
      <Modal
        title="搜索知识库"
        open={searchModalVisible}
        onCancel={() => setSearchModalVisible(false)}
        footer={null}
        width={800}
      >
        <Form form={searchForm} onFinish={handleSearch} layout="vertical">
          <Form.Item
            name="keywords"
            label="搜索关键词"
            rules={[{ required: true, message: "请输入搜索关键词" }]}
          >
            <Select
              mode="tags"
              placeholder="输入关键词，按回车添加"
              style={{ width: "100%" }}
            />
          </Form.Item>
          <Form.Item name="limit" label="结果数量限制">
            <Select defaultValue={10}>
              <Option value={5}>5条</Option>
              <Option value={10}>10条</Option>
              <Option value={20}>20条</Option>
            </Select>
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={searchLoading}>
              搜索
            </Button>
          </Form.Item>
        </Form>

        {searchResults.length > 0 && (
          <div className="mt-4">
            <Divider>搜索结果</Divider>
            <List
              dataSource={searchResults}
              renderItem={(item) => (
                <List.Item
                  actions={[
                    <Button
                      key="view"
                      type="link"
                      icon={<Eye />}
                      onClick={() =>
                        window.open(`/knowledge-base/${item.id}`, "_blank")
                      }
                    >
                      查看
                    </Button>,
                    <Button
                      key="associate"
                      type="link"
                      icon={<Link />}
                      onClick={() => {
                        setSelectedArticle(item);
                        setAssociateModalVisible(true);
                        setSearchModalVisible(false);
                      }}
                    >
                      关联
                    </Button>,
                  ]}
                >
                  <List.Item.Meta
                    title={
                      <Space>
                        <span>{item.title}</span>
                        <Tag color="blue">{item.category}</Tag>
                      </Space>
                    }
                    description={
                      <Paragraph ellipsis={{ rows: 2 }}>
                        {item.content}
                      </Paragraph>
                    }
                  />
                </List.Item>
              )}
            />
          </div>
        )}
      </Modal>

      {/* 关联知识库文章模态框 */}
      <Modal
        title="关联知识库文章"
        open={associateModalVisible}
        onCancel={() => setAssociateModalVisible(false)}
        footer={null}
      >
        {selectedArticle && (
          <div>
            <div className="mb-4">
              <Text strong>文章标题:</Text>
              <div className="mt-1">{selectedArticle.title}</div>
            </div>
            <div className="mb-4">
              <Text strong>文章分类:</Text>
              <div className="mt-1">
                <Tag color="blue">{selectedArticle.category}</Tag>
              </div>
            </div>
            <Form
              form={associateForm}
              onFinish={handleAssociate}
              layout="vertical"
            >
              <Form.Item
                name="association_type"
                label="关联类型"
                rules={[{ required: true, message: "请选择关联类型" }]}
              >
                <Select placeholder="选择关联类型">
                  <Option value="manual">手动关联</Option>
                  <Option value="suggested">建议关联</Option>
                </Select>
              </Form.Item>
              <Form.Item>
                <Space>
                  <Button type="primary" htmlType="submit">
                    确认关联
                  </Button>
                  <Button onClick={() => setAssociateModalVisible(false)}>
                    取消
                  </Button>
                </Space>
              </Form.Item>
            </Form>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default KnowledgeIntegration;
