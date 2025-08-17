"use client";

import React, { useEffect, useMemo, useState } from "react";
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
  Tree,
  Tabs,
} from "antd";
import {
  Database,
  Server,
  Cloud,
  HardDrive,
  PlusCircle,
  Search,
  Eye,
  Edit,
  Network,
} from "lucide-react";
import { CMDBApi } from "../lib/cmdb-api";
// AppLayout is handled by layout.tsx

const { Search: SearchInput } = Input;
const { Option } = Select;

// 模拟CI数据
// const mockCIs = [
//   {
//     id: "CI-ECS-001",
//     name: "Web服务器-生产环境",
//     type: "云服务器",
//     status: "运行中",
//     business: "电商平台",
//     owner: "运维部",
//     location: "阿里云华东1",
//     ip: "192.168.1.100",
//     cpu: "4核",
//     memory: "8GB",
//     disk: "100GB SSD",
//   },
//   {
//     id: "CI-RDS-001",
//     name: "CRM数据库-生产环境",
//     type: "云数据库",
//     status: "运行中",
//     business: "销售管理",
//     owner: "DBA团队",
//     location: "阿里云华东1",
//     ip: "192.168.1.200",
//     version: "MySQL 8.0",
//     storage: "500GB",
//   },
//   {
//     id: "CI-APP-CRM",
//     name: "CRM应用系统",
//     type: "应用系统",
//     status: "运行中",
//     business: "销售管理",
//     owner: "开发部",
//     version: "v2.3.1",
//     environment: "生产",
//   },
//   {
//     id: "CI-STORAGE-001",
//     name: "对象存储-归档",
//     type: "存储",
//     status: "运行中",
//     business: "数据归档",
//     owner: "存储团队",
//     capacity: "10TB",
//     usage: "65%",
//   },
// ];

const getStatusColor = (status: string) => {
  switch (status) {
    case "active":
      return "success";
    case "inactive":
      return "error";
    case "maintenance":
      return "warning";
    case "故障":
      return "error";
    default:
      return "default";
  }
};

const getTypeIcon = (type: string) => {
  switch (type) {
    case "云服务器":
      return <Server size={16} />;
    case "云数据库":
      return <Database size={16} />;
    case "应用系统":
      return <Cloud size={16} />;
    case "存储":
      return <HardDrive size={16} />;
    default:
      return <Network size={16} />;
  }
};

const CMDBListPage = () => {
  const [filter, setFilter] = useState("全部");
  const [searchText, setSearchText] = useState("");
  const [activeTab, setActiveTab] = useState("list");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [items, setItems] = useState<any[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      setError(null);
      try {
        const resp = await CMDBApi.list({
          limit: pageSize,
          offset: (page - 1) * pageSize,
          // 简化：类型筛选需要 CITypeID，这里先仅做前端展示筛选
        });
        setItems(resp.cis as any[]);
        setTotal(resp.total);
      } catch (e: any) {
        setError(e?.message || "加载失败");
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [page, pageSize]);

  const filteredCIs = useMemo(() => {
    const typeMap: Record<number, string> = {};
    // TODO: 可从后端加载 CIType 映射；当前仅做无类型映射的回退
    return (items || [])
      .filter((ci) => {
        const typeName = typeMap[ci.ci_type_id] || "";
        const matchesType = filter === "全部" || typeName === filter;
        const matchesSearch =
          ci.name.toLowerCase().includes(searchText.toLowerCase()) ||
          String(ci.id).toLowerCase().includes(searchText.toLowerCase());
        return matchesType && matchesSearch;
      })
      .map((ci) => ({
        id: ci.id,
        name: ci.name,
        type: typeMap[ci.ci_type_id] || "",
        status: ci.status,
      }));
  }, [items, filter, searchText]);

  const statusCounts = useMemo(
    () => ({
      total: total,
      running: filteredCIs.filter((ci) => ci.status === "active").length,
      stopped: filteredCIs.filter((ci) => ci.status === "inactive").length,
      maintenance: filteredCIs.filter((ci) => ci.status === "maintenance")
        .length,
    }),
    [filteredCIs, total]
  );

  const typeCounts = useMemo(
    () => ({
      servers: 0,
      databases: 0,
      applications: 0,
      storage: 0,
    }),
    []
  );

  const columns = [
    {
      title: "配置项",
      key: "ci",
      width: 250,
      render: (_, record) => (
        <div className="flex items-center space-x-3">
          <div className="text-blue-500">{getTypeIcon(record.type)}</div>
          <div>
            <Link href={`/cmdb/${record.id}`}>
              <div className="font-medium text-blue-600 hover:text-blue-800 cursor-pointer">
                {record.name}
              </div>
            </Link>
            <div className="text-gray-500 text-sm">{record.id}</div>
          </div>
        </div>
      ),
    },
    {
      title: "类型",
      dataIndex: "type",
      key: "type",
      width: 120,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>{status}</Tag>
      ),
    },
    {
      title: "业务系统",
      dataIndex: "business",
      key: "business",
      width: 120,
    },
    {
      title: "负责人",
      dataIndex: "owner",
      key: "owner",
      width: 120,
    },
    {
      title: "详细信息",
      key: "details",
      width: 200,
      render: (_, record) => (
        <div className="text-sm text-gray-600">
          {record.ip && <div>IP: {record.ip}</div>}
          {record.version && <div>版本: {record.version}</div>}
          {record.capacity && <div>容量: {record.capacity}</div>}
        </div>
      ),
    },
    {
      title: "操作",
      key: "actions",
      width: 120,
      render: (_, record) => (
        <Space size="small">
          <Tooltip title="查看详情">
            <Link href={`/cmdb/${record.id}`}>
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

  // 关系图数据
  const relationshipTreeData = [
    {
      title: "电商平台",
      key: "business-1",
      children: [
        {
          title: "Web服务器-生产环境",
          key: "CI-ECS-001",
          children: [
            { title: "CRM数据库-生产环境", key: "CI-RDS-001" },
            { title: "对象存储-归档", key: "CI-STORAGE-001" },
          ],
        },
        { title: "CRM应用系统", key: "CI-APP-CRM" },
      ],
    },
  ];

  return (
    <>
      {/* 页面头部操作 */}
      <div className="mb-6 flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">配置管理数据库</h1>
          <p className="text-gray-600 mt-1">管理IT基础设施和服务的配置项及其关系</p>
        </div>
        <Space>
          <Button icon={<Network size={16} />}>关系视图</Button>
          <Button type="primary" icon={<PlusCircle size={16} />}>
            新增配置项
          </Button>
        </Space>
      </div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="总配置项"
              value={statusCounts.total}
              prefix={<Database size={16} style={{ color: "#1890ff" }} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="运行中"
              value={statusCounts.running}
              valueStyle={{ color: "#52c41a" }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="维护中"
              value={statusCounts.maintenance}
              valueStyle={{ color: "#faad14" }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="已停止"
              value={statusCounts.stopped}
              valueStyle={{ color: "#f5222d" }}
            />
          </Card>
        </Col>
      </Row>

      {/* 类型统计 */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="云服务器"
              value={typeCounts.servers}
              prefix={<Server size={14} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="云数据库"
              value={typeCounts.databases}
              prefix={<Database size={14} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="应用系统"
              value={typeCounts.applications}
              prefix={<Cloud size={14} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="存储设备"
              value={typeCounts.storage}
              prefix={<HardDrive size={14} />}
            />
          </Card>
        </Col>
      </Row>

      {/* 主要内容区域 */}
      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: "list",
              label: "配置项列表",
              children: (
                <>
                  {/* 筛选和搜索 */}
                  <div style={{ marginBottom: 16 }}>
                    <Row gutter={16} align="middle">
                      <Col span={8}>
                        <SearchInput
                          placeholder="搜索配置项名称或ID"
                          allowClear
                          value={searchText}
                          onChange={(e) => setSearchText(e.target.value)}
                          prefix={<Search size={16} />}
                        />
                      </Col>
                      <Col span={4}>
                        <Select
                          value={filter}
                          onChange={setFilter}
                          style={{ width: "100%" }}
                          placeholder="类型筛选"
                        >
                          <Option value="全部">全部类型</Option>
                          <Option value="云服务器">云服务器</Option>
                          <Option value="云数据库">云数据库</Option>
                          <Option value="应用系统">应用系统</Option>
                          <Option value="存储">存储</Option>
                        </Select>
                      </Col>
                    </Row>
                  </div>

                  {/* 配置项表格 */}
                  <Table
                    columns={columns as any}
                    dataSource={filteredCIs}
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
                    scroll={{ x: 1000 }}
                  />
                  {error && (
                    <div style={{ padding: 16, color: "#ff4d4f" }}>
                      加载失败：{error}
                    </div>
                  )}
                </>
              ),
            },
            {
              key: "relationship",
              label: "关系图",
              children: (
                <div style={{ padding: 16 }}>
                  <h3
                    style={{
                      fontSize: "18px",
                      fontWeight: "500",
                      marginBottom: 16,
                    }}
                  >
                    配置项关系图
                  </h3>
                  <Tree
                    showLine
                    showIcon
                    defaultExpandedKeys={["business-1"]}
                    treeData={relationshipTreeData}
                  />
                </div>
              ),
            },
          ]}
        />
      </Card>
    </>
  );
};

export default CMDBListPage;
