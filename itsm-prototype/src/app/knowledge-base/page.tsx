"use client";

import { Search, Edit, Eye, BookOpen, PlusCircle } from "lucide-react";

import React, { useEffect, useState } from "react";
import Link from "next/link";
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
} from "antd";
import { KnowledgeApi } from "../lib/knowledge-api";
// AppLayout is handled by layout.tsx

const { Search: SearchInput } = Input;
const { Option } = Select;
const { Text } = Typography;

// 模拟知识文章数据
// const mockArticles = [] as any[];

const getCategoryColor = (category: string) => {
  const colors: Record<string, string> = {
    账号管理: "blue",
    故障排除: "red",
    网络连接: "green",
    流程指南: "purple",
    系统配置: "orange",
  };
  return colors[category] || "default";
};

interface Article {
  id: string;
  title: string;
  summary: string;
  category: string;
  views: number;
  content?: string;
}

const KnowledgeBaseListPage = () => {
  const [filter, setFilter] = useState("全部");
  const [searchText, setSearchText] = useState("");
  const [viewMode, setViewMode] = useState("table"); // table or card
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [articles, setArticles] = useState<Article[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [categories, setCategories] = useState<string[]>(["全部"]);

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      setError(null);
      try {
        const resp = await KnowledgeApi.list({
          page,
          page_size: pageSize,
          category: filter === "全部" ? undefined : filter,
          search: searchText || undefined,
        });
        setArticles(resp.articles as unknown as Article[]);
        setTotal(resp.total);
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : "加载失败");
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [page, pageSize, filter, searchText]);

  useEffect(() => {
    KnowledgeApi.categories()
      .then((cats) => setCategories(["全部", ...cats]))
      .catch(() => {});
  }, []);

  const stats = {
    total,
    published: total,
    totalViews: 0,
    totalLikes: 0,
  };

  const popularArticles = articles.slice(0, 5);

  const columns = [
    {
      title: "标题",
      key: "title",
      render: (_: unknown, record: Article) => (
        <div>
          <Link href={`/knowledge-base/${record.id}`}>
            <Text
              strong
              className="text-blue-600 hover:text-blue-800 cursor-pointer"
            >
              {record.title}
            </Text>
          </Link>
        </div>
      ),
    },
    {
      title: "分类",
      dataIndex: "category",
      key: "category",
      render: (category: string) => (
        <Tag color={getCategoryColor(category)}>{category}</Tag>
      ),
    },
    {
      title: "浏览量",
      dataIndex: "views",
      key: "views",
      render: (views: number) => (
        <span>
          <Eye size={14} style={{ marginRight: 4 }} />
          {views}
        </span>
      ),
    },
    {
      title: "操作",
      key: "actions",
      render: (_: unknown, record: Article) => (
        <Space size="small">
          <Tooltip title="查看详情">
            <Link href={`/knowledge-base/${record.id}`}>
              <Button type="text" size="small" icon={<Eye size={14} />} />
            </Link>
          </Tooltip>
          <Tooltip title="编辑">
            <Button type="text" size="small" icon={<Edit size={14} />} />
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <>
      {/* 页面头部操作 */}
      <div className="mb-6 flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">知识库</h1>
          <p className="text-gray-600 mt-1">查找和分享IT知识，提高问题解决效率</p>
        </div>
        <Link href="/knowledge-base/new">
          <Button type="primary" icon={<PlusCircle size={16} />}>
            新建文章
          </Button>
        </Link>
      </div>
      <Row gutter={16}>
        {/* 左侧主要内容 */}
        <Col span={18}>
          {/* 统计卡片 */}
          <Row gutter={16} style={{ marginBottom: 24 }}>
            <Col span={6}>
              <Card>
                <Statistic
                  title="总文章数"
                  value={stats.total}
                  prefix={<BookOpen size={16} style={{ color: "#1890ff" }} />}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="已发布"
                  value={stats.published}
                  valueStyle={{ color: "#52c41a" }}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="总浏览量"
                  value={stats.totalViews}
                  valueStyle={{ color: "#1890ff" }}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="总点赞数"
                  value={stats.totalLikes}
                  valueStyle={{ color: "#faad14" }}
                />
              </Card>
            </Col>
          </Row>

          {/* 搜索和筛选 */}
          <Card style={{ marginBottom: 24 }}>
            <Row gutter={16} align="middle">
              <Col span={12}>
                <SearchInput
                  placeholder="搜索文章标题或标签"
                  allowClear
                  value={searchText}
                  onChange={(e) => setSearchText(e.target.value)}
                  prefix={<Search size={16} />}
                />
              </Col>
              <Col span={6}>
                <Select
                  value={filter}
                  onChange={setFilter}
                  style={{ width: "100%" }}
                  placeholder="选择分类"
                >
                  {categories.map((category) => (
                    <Option key={category} value={category}>
                      {category}
                    </Option>
                  ))}
                </Select>
              </Col>
              <Col span={6}>
                <Select
                  value={viewMode}
                  onChange={setViewMode}
                  style={{ width: "100%" }}
                >
                  <Option value="table">表格视图</Option>
                  <Option value="card">卡片视图</Option>
                </Select>
              </Col>
            </Row>
          </Card>

          {/* 文章列表 */}
          <Card>
            {error ? (
              <div style={{ padding: 24, color: "#ff4d4f" }}>
                加载失败：{error}
              </div>
            ) : viewMode === "table" ? (
              <Table
                columns={columns as any}
                dataSource={articles}
                rowKey="id"
                loading={loading}
                pagination={{
                  current: page,
                  pageSize,
                  total,
                  showSizeChanger: true,
                  showQuickJumper: true,
                  showTotal: (total, range) =>
                    `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
                  onChange: (p, ps) => {
                    setPage(p);
                    setPageSize(ps);
                  },
                }}
              />
            ) : (
              <List
                grid={{ gutter: 16, xs: 1, sm: 2, md: 2, lg: 2, xl: 2, xxl: 3 }}
                dataSource={articles}
                renderItem={(article: Article) => (
                  <List.Item>
                    <Card
                      hoverable
                      actions={[
                        <Link key="view" href={`/knowledge-base/${article.id}`}>
                          <Button
                            type="text"
                            size="small"
                            icon={<Eye size={14} />}
                          >
                            查看
                          </Button>
                        </Link>,
                        <Button
                          key="edit"
                          type="text"
                          size="small"
                          icon={<Edit size={14} />}
                        >
                          编辑
                        </Button>,
                      ]}
                    >
                      <Card.Meta
                        title={
                          <Link href={`/knowledge-base/${article.id}`}>
                            <Text
                              strong
                              className="text-blue-600 hover:text-blue-800"
                            >
                              {article.title}
                            </Text>
                          </Link>
                        }
                        description={
                          <div>
                            <Text
                              type="secondary"
                              style={{ display: "block", marginBottom: 8 }}
                            >
                              {article.summary}
                            </Text>
                            <div
                              style={{
                                display: "flex",
                                justifyContent: "space-between",
                                alignItems: "center",
                              }}
                            >
                              <Tag color={getCategoryColor(article.category)}>
                                {article.category}
                              </Tag>
                            </div>
                          </div>
                        }
                      />
                    </Card>
                  </List.Item>
                )}
              />
            )}
          </Card>
        </Col>

        {/* 右侧边栏 */}
        <Col span={6}>
          <Card title="热门文章" size="small">
            <List
              size="small"
              dataSource={popularArticles}
              renderItem={(article, index) => (
                <List.Item>
                  <div style={{ width: "100%" }}>
                    <div style={{ display: "flex", alignItems: "center" }}>
                      <div
                        style={{
                          minWidth: 20,
                          height: 20,
                          borderRadius: "50%",
                          backgroundColor: index < 3 ? "#ff7a45" : "#d9d9d9",
                          color: "white",
                          fontSize: 12,
                          display: "flex",
                          alignItems: "center",
                          justifyContent: "center",
                          marginRight: 8,
                        }}
                      >
                        {index + 1}
                      </div>
                      <div style={{ flex: 1 }}>
                        <Link href={`/knowledge-base/${article.id}`}>
                          <Text
                            style={{
                              fontSize: 13,
                              display: "block",
                              marginBottom: 4,
                            }}
                            className="text-blue-600 hover:text-blue-800"
                          >
                            {article.title}
                          </Text>
                        </Link>
                        <Text type="secondary" style={{ fontSize: 11 }}>
                          <Eye size={10} /> {article.views} 次浏览
                        </Text>
                      </div>
                    </div>
                  </div>
                </List.Item>
              )}
            />
          </Card>

          <Card title="分类统计" size="small" style={{ marginTop: 16 }}>
            <List
              size="small"
              dataSource={categories.filter((c) => c !== "全部")}
              renderItem={(category) => {
                const count = articles.filter(
                  (a) => a.category === category
                ).length;
                return (
                  <List.Item style={{ padding: "8px 0" }}>
                    <div
                      style={{
                        display: "flex",
                        justifyContent: "space-between",
                        width: "100%",
                      }}
                    >
                      <Tag color={getCategoryColor(category)}>{category}</Tag>
                      <span style={{ fontSize: 12, color: "#999" }}>
                        {count} 篇
                      </span>
                    </div>
                  </List.Item>
                );
              }}
            />
          </Card>
        </Col>
      </Row>
    </>
  );
};

export default KnowledgeBaseListPage;
