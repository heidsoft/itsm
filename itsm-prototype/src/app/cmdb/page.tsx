"use client";

import React, { useMemo, useState } from "react";
import {
  Card,
  Table,
  Button,
  Tag,
  Space,
  Row,
  Col,
  Select,
  Input,
  Tree,
  Tabs,
  Badge,
  Modal,
  Form,
  InputNumber,
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
  MoreHorizontal,
  X,
  GitBranch,
} from "lucide-react";
import { mockCIs, mockRelations } from "../lib/cmdb-relations";

const { Search: SearchInput } = Input;
const { Option } = Select;

// 模拟CI数据
// const mockCIs = []

const getCiTypeIcon = (type: string) => {
  switch (type) {
    case "Cloud Server":
      return <Cloud size={16} />;
    case "Physical Server":
      return <Server size={16} />;
    case "Relational Database":
      return <Database size={16} />;
    case "Storage Device":
      return <HardDrive size={16} />;
    default:
      return <Network size={16} />;
  }
};

const getCiTypeColor = (type: string) => {
  switch (type) {
    case "Cloud Server":
      return "blue";
    case "Physical Server":
      return "purple";
    case "Relational Database":
      return "green";
    case "Storage Device":
      return "orange";
    default:
      return "default";
  }
};

const getStatusConfig = (status: string) => {
  switch (status) {
    case "Running":
      return {
        color: "#52c41a",
        text: "Running",
        backgroundColor: "#f6ffed",
      };
    case "Maintenance":
      return {
        color: "#fa8c16",
        text: "Maintenance",
        backgroundColor: "#fff7e6",
      };
    case "Disabled":
      return {
        color: "#00000073",
        text: "Disabled",
        backgroundColor: "#fafafa",
      };
    default:
      return {
        color: "#00000073",
        text: status,
        backgroundColor: "#fafafa",
      };
  }
};

const CMDBPage = () => {
  const [cis, setCis] = useState(mockCIs);
  const [relations] = useState(mockRelations);
  const [loading] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [, setSearchText] = useState("");
  const [filterType, setFilterType] = useState("");
  const [activeTab, setActiveTab] = useState("list");
  const [selectedCI, setSelectedCI] = useState<typeof mockCIs[0] | null>(null);
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [relationModalVisible, setRelationModalVisible] = useState(false);

  const [createForm] = Form.useForm();

  // 树形数据
  const treeData = useMemo(() => [
    {
      title: "Infrastructure",
      key: "infrastructure",
      children: [
        {
          title: "Cloud Resources",
          key: "cloud",
          children: [
            { title: "Alibaba Cloud", key: "aliyun" },
            { title: "Tencent Cloud", key: "tencent" },
          ],
        },
        {
          title: "Physical Devices",
          key: "physical",
          children: [
            { title: "Servers", key: "servers" },
            { title: "Network Devices", key: "network" },
          ],
        },
      ],
    },
    {
      title: "Application Systems",
      key: "applications",
      children: [
        { title: "E-commerce Platform", key: "ecommerce" },
        { title: "Customer Relationship Management", key: "crm" },
      ],
    },
  ], []);

  const handleCreateCI = () => {
    setCreateModalVisible(true);
  };

  const handleCreateCIConfirm = async () => {
    try {
      const values = await createForm.validateFields();
      // 模拟创建CI
      const newCI = {
        id: `CI-${Math.floor(Math.random() * 10000)}`,
        ...values,
        status: "Running",
        business: "New Business System",
        owner: "Operations Team",
        location: "Data Center",
        ip: "192.168.1.102",
        cpu: `${values.cpu || 4} Cores`,
        memory: `${values.memory || 8}GB`,
        disk: `${values.disk || 100}GB SSD`,
      };
      setCis([...cis, newCI]);
      setCreateModalVisible(false);
      createForm.resetFields();
    } catch (error) {
      console.error("Failed to create CI:", error);
    }
  };

  const handleViewRelations = (ci: typeof mockCIs[0]) => {
    setSelectedCI(ci);
    setRelationModalVisible(true);
  };

  // 渲染统计卡片
  const renderStatsCards = () => (
    <div style={{ marginBottom: 24 }}>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card 
            style={{ 
              background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
              border: 'none',
              borderRadius: 16,
              boxShadow: '0 8px 32px rgba(102, 126, 234, 0.3)',
              transition: 'all 0.3s ease',
              cursor: 'pointer'
            }}
            bodyStyle={{ padding: '24px' }}
            onMouseEnter={(e) => {
              e.currentTarget.style.transform = 'translateY(-4px)';
              e.currentTarget.style.boxShadow = '0 12px 40px rgba(102, 126, 234, 0.4)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.transform = 'translateY(0)';
              e.currentTarget.style.boxShadow = '0 8px 32px rgba(102, 126, 234, 0.3)';
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div>
                <div style={{ color: 'rgba(255, 255, 255, 0.8)', fontSize: 14, marginBottom: 8 }}>Total CIs</div>
                <div style={{ color: '#fff', fontSize: 32, fontWeight: 'bold', lineHeight: 1 }}>{cis.length}</div>
              </div>
              <div style={{ 
                width: 56, 
                height: 56, 
                backgroundColor: 'rgba(255, 255, 255, 0.2)', 
                borderRadius: 16, 
                display: 'flex', 
                alignItems: 'center', 
                justifyContent: 'center' 
              }}>
                <Database size={24} style={{ color: '#fff' }} />
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card 
            style={{ 
              background: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)',
              border: 'none',
              borderRadius: 16,
              boxShadow: '0 8px 32px rgba(240, 147, 251, 0.3)',
              transition: 'all 0.3s ease',
              cursor: 'pointer'
            }}
            bodyStyle={{ padding: '24px' }}
            onMouseEnter={(e) => {
              e.currentTarget.style.transform = 'translateY(-4px)';
              e.currentTarget.style.boxShadow = '0 12px 40px rgba(240, 147, 251, 0.4)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.transform = 'translateY(0)';
              e.currentTarget.style.boxShadow = '0 8px 32px rgba(240, 147, 251, 0.3)';
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div>
                <div style={{ color: 'rgba(255, 255, 255, 0.8)', fontSize: 14, marginBottom: 8 }}>Cloud Servers</div>
                <div style={{ color: '#fff', fontSize: 32, fontWeight: 'bold', lineHeight: 1 }}>{cis.filter(ci => ci.type === "Cloud Server").length}</div>
              </div>
              <div style={{ 
                width: 56, 
                height: 56, 
                backgroundColor: 'rgba(255, 255, 255, 0.2)', 
                borderRadius: 16, 
                display: 'flex', 
                alignItems: 'center', 
                justifyContent: 'center' 
              }}>
                <Cloud size={24} style={{ color: '#fff' }} />
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card 
            style={{ 
              background: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
              border: 'none',
              borderRadius: 16,
              boxShadow: '0 8px 32px rgba(79, 172, 254, 0.3)',
              transition: 'all 0.3s ease',
              cursor: 'pointer'
            }}
            bodyStyle={{ padding: '24px' }}
            onMouseEnter={(e) => {
              e.currentTarget.style.transform = 'translateY(-4px)';
              e.currentTarget.style.boxShadow = '0 12px 40px rgba(79, 172, 254, 0.4)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.transform = 'translateY(0)';
              e.currentTarget.style.boxShadow = '0 8px 32px rgba(79, 172, 254, 0.3)';
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div>
                <div style={{ color: 'rgba(255, 255, 255, 0.8)', fontSize: 14, marginBottom: 8 }}>Running</div>
                <div style={{ color: '#fff', fontSize: 32, fontWeight: 'bold', lineHeight: 1 }}>{cis.filter(ci => ci.status === "Running").length}</div>
              </div>
              <div style={{ 
                width: 56, 
                height: 56, 
                backgroundColor: 'rgba(255, 255, 255, 0.2)', 
                borderRadius: 16, 
                display: 'flex', 
                alignItems: 'center', 
                justifyContent: 'center' 
              }}>
                <Server size={24} style={{ color: '#fff' }} />
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card 
            style={{ 
              background: 'linear-gradient(135deg, #fa709a 0%, #fee140 100%)',
              border: 'none',
              borderRadius: 16,
              boxShadow: '0 8px 32px rgba(250, 112, 154, 0.3)',
              transition: 'all 0.3s ease',
              cursor: 'pointer'
            }}
            bodyStyle={{ padding: '24px' }}
            onMouseEnter={(e) => {
              e.currentTarget.style.transform = 'translateY(-4px)';
              e.currentTarget.style.boxShadow = '0 12px 40px rgba(250, 112, 154, 0.4)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.transform = 'translateY(0)';
              e.currentTarget.style.boxShadow = '0 8px 32px rgba(250, 112, 154, 0.3)';
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div>
                <div style={{ color: 'rgba(255, 255, 255, 0.8)', fontSize: 14, marginBottom: 8 }}>Maintenance</div>
                <div style={{ color: '#fff', fontSize: 32, fontWeight: 'bold', lineHeight: 1 }}>{cis.filter(ci => ci.status === "Maintenance").length}</div>
              </div>
              <div style={{ 
                width: 56, 
                height: 56, 
                backgroundColor: 'rgba(255, 255, 255, 0.2)', 
                borderRadius: 16, 
                display: 'flex', 
                alignItems: 'center', 
                justifyContent: 'center' 
              }}>
                <HardDrive size={24} style={{ color: '#fff' }} />
              </div>
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  );

  // 渲染筛选器
  const renderFilters = () => (
    <Card 
      style={{ 
        marginBottom: 24,
        borderRadius: 16,
        border: '1px solid #f0f0f0',
        boxShadow: '0 4px 20px rgba(0, 0, 0, 0.08)'
      }}
      bodyStyle={{ padding: '24px' }}
    >
      <Row gutter={[20, 16]} align="middle">
        <Col xs={24} sm={12} md={8}>
          <div style={{ position: 'relative' }}>
            <SearchInput
              placeholder="搜索配置项名称、ID或IP..."
              allowClear
              onSearch={(value) => setSearchText(value)}
              size="large"
              enterButton={{
                style: {
                  background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                  border: 'none',
                  borderRadius: '0 8px 8px 0'
                }
              }}
              style={{
                borderRadius: 8,
                boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)'
              }}
            />
          </div>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Select
            placeholder="类型筛选"
            size="large"
            allowClear
            value={filterType}
            onChange={(value) => setFilterType(value)}
            style={{ 
              width: "100%",
              borderRadius: 8,
              boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)'
            }}
            dropdownStyle={{
              borderRadius: 8,
              boxShadow: '0 8px 32px rgba(0, 0, 0, 0.12)'
            }}
          >
            <Option value="云服务器">☁️ 云服务器</Option>
            <Option value="物理服务器">🖥️ 物理服务器</Option>
            <Option value="关系型数据库">🗄️ 关系型数据库</Option>
            <Option value="存储设备">💾 存储设备</Option>
            <Option value="网络设备">🌐 网络设备</Option>
            <Option value="缓存数据库">⚡ 缓存数据库</Option>
          </Select>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Select
            placeholder="状态筛选"
            size="large"
            allowClear
            style={{ 
              width: "100%",
              borderRadius: 8,
              boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)'
            }}
            dropdownStyle={{
              borderRadius: 8,
              boxShadow: '0 8px 32px rgba(0, 0, 0, 0.12)'
            }}
          >
            <Option value="运行中">🟢 运行中</Option>
            <Option value="维护中">🟡 维护中</Option>
            <Option value="已停用">⚫ 已停用</Option>
          </Select>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Space size={12} style={{ width: '100%', justifyContent: 'flex-end' }}>
            <Button
              icon={<Search size={18} />}
              onClick={() => {}}
              loading={loading}
              size="large"
              style={{
                borderRadius: 8,
                border: '1px solid #d9d9d9',
                boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)',
                transition: 'all 0.3s ease'
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.borderColor = '#667eea';
                e.currentTarget.style.color = '#667eea';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.borderColor = '#d9d9d9';
                e.currentTarget.style.color = 'rgba(0, 0, 0, 0.88)';
              }}
            >
              刷新
            </Button>
            <Button
              type="primary"
              icon={<PlusCircle size={18} />}
              onClick={handleCreateCI}
              size="large"
              style={{
                background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                border: 'none',
                borderRadius: 8,
                boxShadow: '0 4px 16px rgba(102, 126, 234, 0.3)',
                transition: 'all 0.3s ease'
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.transform = 'translateY(-2px)';
                e.currentTarget.style.boxShadow = '0 6px 20px rgba(102, 126, 234, 0.4)';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = 'translateY(0)';
                e.currentTarget.style.boxShadow = '0 4px 16px rgba(102, 126, 234, 0.3)';
              }}
            >
              新建配置项
            </Button>
          </Space>
        </Col>
      </Row>
    </Card>
  );

  // 渲染CI列表
  const renderCIList = () => (
    <Card
      style={{
        borderRadius: 16,
        border: '1px solid #f0f0f0',
        boxShadow: '0 4px 20px rgba(0, 0, 0, 0.08)'
      }}
      bodyStyle={{ padding: 0 }}
    >
      <div style={{ padding: '24px 24px 0 24px', display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <h3 style={{ margin: 0, fontSize: 18, fontWeight: 600 }}>Configuration Items List</h3>
          {selectedRowKeys.length > 0 && (
            <Badge 
              count={selectedRowKeys.length} 
              showZero 
              style={{ 
                backgroundColor: "#667eea",
                boxShadow: '0 2px 8px rgba(102, 126, 234, 0.3)'
              }} 
            />
          )}
        </div>
        <div style={{ fontSize: 14, color: '#666' }}>
          共 {cis.length} 个配置项
        </div>
      </div>

      <Table
        rowSelection={{
          selectedRowKeys,
          onChange: setSelectedRowKeys,
          getCheckboxProps: () => ({
            style: { borderRadius: 4 }
          })
        }}
        columns={columns}
        dataSource={cis}
        rowKey="id"
        loading={loading}
        pagination={{
          pageSize: 10,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total, range) =>
            `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
          style: {
            padding: '16px 24px',
            borderTop: '1px solid #f0f0f0'
          }
        }}
        scroll={{ x: 1200 }}
        style={{ 
          borderRadius: '0 0 16px 16px',
          overflow: 'hidden'
        }}
        rowClassName={(record, index) => 
          index % 2 === 0 ? 'table-row-light' : 'table-row-dark'
        }
        onRow={() => ({
          style: {
            transition: 'all 0.2s ease',
            cursor: 'pointer'
          },
          onMouseEnter: (e) => {
            e.currentTarget.style.backgroundColor = '#f8f9ff';
            e.currentTarget.style.transform = 'scale(1.01)';
          },
          onMouseLeave: (e) => {
            e.currentTarget.style.backgroundColor = '';
            e.currentTarget.style.transform = 'scale(1)';
          }
        })}
      />
      
      <style jsx global>{`
        .table-row-light {
          background-color: #ffffff;
        }
        .table-row-dark {
          background-color: #fafbff;
        }
        .ant-table-thead > tr > th {
          background: linear-gradient(135deg, #f8f9ff 0%, #f0f2ff 100%) !important;
          border-bottom: 2px solid #e6f0ff !important;
          font-weight: 600 !important;
          color: #1a1a1a !important;
          padding: 16px 12px !important;
        }
        .ant-table-tbody > tr > td {
          padding: 16px 12px !important;
          border-bottom: 1px solid #f5f5f5 !important;
        }
        .ant-table-tbody > tr:hover > td {
          background-color: #f8f9ff !important;
        }
      `}</style>
    </Card>
  );

  // 表格列定义
  const columns = [
    {
      title: "配置项信息",
      key: "ci_info",
      width: 300,
      render: (_: unknown, record: typeof mockCIs[0]) => (
        <div style={{ display: "flex", alignItems: "center" }}>
          <div style={{ width: 40, height: 40, backgroundColor: "#e6f7ff", borderRadius: 8, display: "flex", alignItems: "center", justifyContent: "center", marginRight: 12, color: "#1890ff" }}>
            {getCiTypeIcon(record.type)}
          </div>
          <div>
            <div style={{ fontWeight: "medium", color: "#000", marginBottom: 4 }}>{record.name}</div>
            <div style={{ fontSize: "small", color: "#666" }}>
              {record.id} • {record.ip}
            </div>
          </div>
        </div>
      ),
    },
    {
      title: "类型",
      dataIndex: "type",
      key: "type",
      width: 120,
      render: (type: string) => (
        <Tag color={getCiTypeColor(type)} icon={getCiTypeIcon(type)} style={{ marginRight: 0 }}>
          {type}
        </Tag>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 120,
      render: (status: string) => {
        const config = getStatusConfig(status);
        return (
          <span
            style={{
              padding: "4px 12px",
              borderRadius: 16,
              fontSize: "small",
              fontWeight: 500,
              color: config.color,
              backgroundColor: config.backgroundColor,
            }}
          >
            {config.text}
          </span>
        );
      },
    },
    {
      title: "所属业务",
      dataIndex: "business",
      key: "business",
      width: 150,
      render: (business: string) => (
        <div style={{ fontSize: "small" }}>{business}</div>
      ),
    },
    {
      title: "负责人",
      dataIndex: "owner",
      key: "owner",
      width: 120,
      render: (owner: string) => (
        <div style={{ fontSize: "small" }}>{owner}</div>
      ),
    },
    {
      title: "位置",
      dataIndex: "location",
      key: "location",
      width: 150,
      render: (location: string) => (
        <div style={{ fontSize: "small" }}>{location}</div>
      ),
    },
    {
      title: "配置",
      key: "config",
      width: 150,
      render: (_: unknown, record: typeof mockCIs[0]) => (
        <div style={{ fontSize: "small" }}>
          <div>{record.cpu} / {record.memory}</div>
          <div style={{ color: "#666" }}>{record.disk}</div>
        </div>
      ),
    },
    {
      title: "操作",
      key: "actions",
      width: 150,
      render: (_: unknown, record: typeof mockCIs[0]) => (
        <Space size="small">
          <Button
            type="text"
            size="small"
            icon={<Eye size={16} />}
            onClick={() => window.open(`/cmdb/${record.id}`)}
          />
          <Button
            type="text"
            size="small"
            icon={<Edit size={16} />}
            onClick={() => window.open(`/cmdb/${record.id}/edit`)}
          />
          <Button
            type="text"
            size="small"
            icon={<Network size={16} />}
            onClick={() => handleViewRelations(record)}
          />
          <Button
            type="text"
            size="small"
            icon={<MoreHorizontal size={16} />}
          />
        </Space>
      ),
    },
  ];

  // 渲染关系图
  const renderRelationGraph = () => {
    if (!selectedCI) {
      return (
        <div style={{ textAlign: "center", padding: "100px 0", color: "#666" }}>
          <Network size={48} style={{ margin: "0 auto 16px", color: "#1890ff" }} />
          <p>请在配置项列表中选择一个配置项查看其关系图</p>
        </div>
      );
    }

    // 获取与选中CI相关的所有关系
    const relatedRelations = relations.filter(
      rel => rel.source === selectedCI.id || rel.target === selectedCI.id
    );

    // 获取相关的CI
    const relatedCIs = cis.filter(ci => 
      relatedRelations.some(rel => rel.source === ci.id || rel.target === ci.id)
    );

    return (
      <div style={{ padding: 24 }}>
        <div style={{ marginBottom: 24, display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <h3 style={{ margin: 0 }}>
            {selectedCI.name} 的关系图
          </h3>
          <Button 
            icon={<X size={16} />} 
            onClick={() => setSelectedCI(null)}
          >
            关闭
          </Button>
        </div>
        
        <div style={{ 
          height: 500, 
          border: "1px solid #f0f0f0", 
          borderRadius: 8,
          backgroundColor: "#fafafa",
          position: "relative",
          overflow: "hidden"
        }}>
          {/* 简化的关系图可视化 */}
          <div style={{ 
            position: "absolute", 
            top: "50%", 
            left: "50%", 
            transform: "translate(-50%, -50%)",
            textAlign: "center"
          }}>
            <div style={{
              width: 120,
              height: 120,
              borderRadius: 8,
              backgroundColor: "#e6f7ff",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              margin: "0 auto 20px",
              color: "#1890ff",
              fontWeight: "bold"
            }}>
              {getCiTypeIcon(selectedCI.type)}
              <div style={{ marginTop: 8 }}>{selectedCI.name}</div>
            </div>
            
            <div style={{ display: "flex", justifyContent: "center", gap: 40 }}>
              {relatedCIs.slice(0, 3).map((ci) => (
                <div key={ci.id} style={{ textAlign: "center" }}>
                  <div style={{
                    width: 80,
                    height: 80,
                    borderRadius: 8,
                    backgroundColor: "#f6ffed",
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                    margin: "0 auto",
                    color: "#52c41a"
                  }}>
                    {getCiTypeIcon(ci.type)}
                  </div>
                  <div style={{ marginTop: 8, fontSize: 12, width: 80, overflow: "hidden", textOverflow: "ellipsis" }}>{ci.name}</div>
                </div>
              ))}
            </div>
            
            <div style={{ marginTop: 20, fontSize: 14, color: "#666" }}>
              {relatedRelations.length} 个关联关系
            </div>
          </div>
        </div>
        
        <div style={{ marginTop: 24 }}>
          <h4>关联关系详情</h4>
          <Table
            columns={[
              {
                title: "源配置项",
                dataIndex: "source",
                key: "source",
                render: (source: string) => {
                  const ci = cis.find(c => c.id === source);
                  return ci ? ci.name : source;
                }
              },
              {
                title: "目标配置项",
                dataIndex: "target",
                key: "target",
                render: (target: string) => {
                  const ci = cis.find(c => c.id === target);
                  return ci ? ci.name : target;
                }
              },
              {
                title: "关系类型",
                dataIndex: "type",
                key: "type",
              },
              {
                title: "描述",
                dataIndex: "description",
                key: "description",
              },
            ]}
            dataSource={relatedRelations}
            pagination={false}
            rowKey="id"
          />
        </div>
      </div>
    );
  };

  // 渲染拓扑视图
  const renderTopologyView = () => (
    <div style={{ padding: 24 }}>
      <div style={{ 
        height: 600, 
        border: "1px solid #f0f0f0", 
        borderRadius: 8,
        backgroundColor: "#fafafa",
        position: "relative",
        overflow: "hidden"
      }}>
        <div style={{ 
          position: "absolute", 
          top: "50%", 
          left: "50%", 
          transform: "translate(-50%, -50%)",
          textAlign: "center"
        }}>
          <Network size={48} style={{ margin: "0 auto 16px", color: "#1890ff" }} />
          <h3>配置项拓扑视图</h3>
          <p style={{ color: "#666", maxWidth: 400 }}>
            此视图展示了所有配置项之间的连接关系。通过可视化的方式，您可以快速了解整个IT基础设施的架构和依赖关系。
          </p>
          <div style={{ 
            display: "flex", 
            justifyContent: "center", 
            gap: 20, 
            marginTop: 30 
          }}>
            {cis.slice(0, 5).map((ci) => (
              <div key={ci.id} style={{ textAlign: "center" }}>
                <div style={{
                  width: 60,
                  height: 60,
                  borderRadius: 8,
                  backgroundColor: "#e6f7ff",
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  margin: "0 auto",
                  color: "#1890ff"
                }}>
                  {getCiTypeIcon(ci.type)}
                </div>
                <div style={{ marginTop: 8, fontSize: 12, width: 80, overflow: "hidden", textOverflow: "ellipsis" }}>{ci.name}</div>
              </div>
            ))}
          </div>
          <div style={{ marginTop: 20, fontSize: 14, color: "#666" }}>
            共 {cis.length} 个配置项，{relations.length} 个关系连接
          </div>
        </div>
      </div>
    </div>
  );

  return (
    <div>
      {/* 页面头部 */}
      <div style={{ 
        display: 'flex', 
        justifyContent: 'space-between', 
        alignItems: 'center', 
        marginBottom: 24,
        padding: '20px 0'
      }}>
        <div>
          <h2 style={{ 
            margin: 0, 
            fontSize: 24, 
            fontWeight: 700, 
            color: '#1a1a1a',
            marginBottom: 8
          }}>
            配置管理数据库 (CMDB)
          </h2>
          <p style={{ 
            margin: 0, 
            color: '#666', 
            fontSize: 14 
          }}>
            管理和维护IT基础设施配置项及其关系
          </p>
        </div>
        <Button 
          icon={<PlusCircle size={18} />} 
          type="primary" 
          size="large"
          onClick={handleCreateCI}
          style={{
            borderRadius: 12,
            background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
            border: 'none',
            fontWeight: 500,
            height: 44,
            padding: '0 24px',
            transition: 'all 0.3s ease',
            boxShadow: '0 4px 15px rgba(102, 126, 234, 0.4)'
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.transform = 'translateY(-2px)';
            e.currentTarget.style.boxShadow = '0 6px 20px rgba(102, 126, 234, 0.6)';
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.transform = 'translateY(0)';
            e.currentTarget.style.boxShadow = '0 4px 15px rgba(102, 126, 234, 0.4)';
          }}
        >
          新建配置项
        </Button>
      </div>

      <Tabs 
         activeKey={activeTab} 
         onChange={setActiveTab} 
         style={{ 
           marginBottom: 24
         }}
         size="large"
         tabBarStyle={{
           background: 'white',
           borderRadius: '12px 12px 0 0',
           padding: '0 24px',
           margin: 0,
           border: '1px solid #f0f0f0',
           borderBottom: 'none'
         }}
         items={[
           {
             key: 'list',
             label: (
               <div style={{ 
                 display: 'flex', 
                 alignItems: 'center', 
                 gap: 8,
                 padding: '8px 16px',
                 borderRadius: 8,
                 transition: 'all 0.2s ease'
               }}>
                 <Database size={16} />
                 配置项列表
               </div>
             )
           },
           {
             key: 'relations',
             label: (
               <div style={{ 
                 display: 'flex', 
                 alignItems: 'center', 
                 gap: 8,
                 padding: '8px 16px',
                 borderRadius: 8,
                 transition: 'all 0.2s ease'
               }}>
                 <Network size={16} />
                 关系图
               </div>
             )
           },
           {
             key: 'topology',
             label: (
               <div style={{ 
                 display: 'flex', 
                 alignItems: 'center', 
                 gap: 8,
                 padding: '8px 16px',
                 borderRadius: 8,
                 transition: 'all 0.2s ease'
               }}>
                 <GitBranch size={16} />
                 拓扑视图
               </div>
             )
           }
         ]}
       />

      {activeTab === "list" && (
        <>
          {renderStatsCards()}
          {renderFilters()}
          {renderCIList()}
        </>
      )}

      {activeTab === "relation" && (
        <Card>
          <div style={{ display: "flex", height: "100%" }}>
            <div style={{ width: 300, borderRight: "1px solid #f0f0f0", padding: "16px 0" }}>
              <Tree
                treeData={treeData}
                defaultExpandedKeys={["infrastructure", "cloud", "physical", "applications"]}
              />
            </div>
            <div style={{ flex: 1 }}>
              {renderRelationGraph()}
            </div>
          </div>
        </Card>
      )}

      {activeTab === "topology" && (
        <Card>
          {renderTopologyView()}
        </Card>
      )}

      {/* 创建配置项模态框 */}
      <Modal
        title={
          <div style={{ 
            fontSize: 18, 
            fontWeight: 600, 
            color: '#1a1a1a',
            display: 'flex',
            alignItems: 'center',
            gap: 12
          }}>
            <div style={{
              width: 40,
              height: 40,
              borderRadius: 12,
              background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'white'
            }}>
              <PlusCircle size={20} />
            </div>
            新建配置项
          </div>
        }
        open={createModalVisible}
        onCancel={() => {
          setCreateModalVisible(false);
          createForm.resetFields();
        }}
        onOk={handleCreateCIConfirm}
        width={700}
        centered
        maskClosable={false}
        destroyOnClose
        styles={{
          header: {
            borderBottom: '1px solid #f0f0f0',
            paddingBottom: 16,
            marginBottom: 24
          },
          body: {
            padding: '24px 0'
          },
          footer: {
            borderTop: '1px solid #f0f0f0',
            paddingTop: 16,
            marginTop: 24
          }
        }}
        okButtonProps={{
          style: {
            background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
            border: 'none',
            borderRadius: 8,
            height: 40,
            fontWeight: 500
          }
        }}
        cancelButtonProps={{
          style: {
            borderRadius: 8,
            height: 40,
            fontWeight: 500
          }
        }}
      >
        <Form form={createForm} layout="vertical" style={{ marginTop: 20 }}>
          <Row gutter={24}>
            <Col span={12}>
              <Form.Item
                label="配置项名称"
                name="name"
                rules={[{ required: true, message: "请输入配置项名称" }]}
              >
                <Input placeholder="请输入配置项名称" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="类型"
                name="type"
                rules={[{ required: true, message: "请选择类型" }]}
              >
                <Select placeholder="请选择类型">
                  <Option value="云服务器">云服务器</Option>
                  <Option value="物理服务器">物理服务器</Option>
                  <Option value="关系型数据库">关系型数据库</Option>
                  <Option value="存储设备">存储设备</Option>
                  <Option value="网络设备">网络设备</Option>
                  <Option value="缓存数据库">缓存数据库</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={12}>
              <Form.Item
                label="CPU"
                name="cpu"
              >
                <InputNumber
                  placeholder="请输入CPU核心数"
                  addonAfter="核"
                  min={1}
                  style={{ width: "100%" }}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="内存"
                name="memory"
              >
                <InputNumber
                  placeholder="请输入内存大小"
                  addonAfter="GB"
                  min={1}
                  style={{ width: "100%" }}
                />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={12}>
              <Form.Item
                label="磁盘"
                name="disk"
              >
                <InputNumber
                  placeholder="请输入磁盘大小"
                  addonAfter="GB"
                  min={1}
                  style={{ width: "100%" }}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="IP地址"
                name="ip"
              >
                <Input placeholder="请输入IP地址" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            label="描述"
            name="description"
          >
            <Input.TextArea
              placeholder="请输入配置项描述"
              rows={3}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 关系查看模态框 */}
      <Modal
        title={
          <div style={{ 
            fontSize: 18, 
            fontWeight: 600, 
            color: '#1a1a1a',
            display: 'flex',
            alignItems: 'center',
            gap: 12
          }}>
            <div style={{
              width: 40,
              height: 40,
              borderRadius: 12,
              background: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'white'
            }}>
              <Network size={20} />
            </div>
            配置项关系图
          </div>
        }
        open={relationModalVisible}
        onCancel={() => setRelationModalVisible(false)}
        footer={[
          <Button 
            key="close" 
            onClick={() => setRelationModalVisible(false)}
            style={{
              borderRadius: 8,
              height: 40,
              fontWeight: 500,
              background: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
              border: 'none',
              color: 'white'
            }}
          >
            关闭
          </Button>
        ]}
        width={1000}
        centered
        maskClosable={false}
        destroyOnClose
        styles={{
          header: {
            borderBottom: '1px solid #f0f0f0',
            paddingBottom: 16,
            marginBottom: 24
          },
          body: {
            padding: '24px 0',
            maxHeight: '70vh',
            overflowY: 'auto'
          },
          footer: {
            borderTop: '1px solid #f0f0f0',
            paddingTop: 16,
            marginTop: 24
          }
        }}
      >
        {selectedCI && (
          <div style={{ padding: "24px 0" }}>
            <div style={{ marginBottom: 24 }}>
              <h3 style={{ margin: 0 }}>
                {selectedCI.name} 的关系图
              </h3>
              <p style={{ color: "#666", margin: "8px 0 0 0" }}>
                查看此配置项与其他配置项之间的关联关系
              </p>
            </div>
            
            <div style={{ 
              height: 400, 
              border: "1px solid #f0f0f0", 
              borderRadius: 8,
              backgroundColor: "#fafafa",
              position: "relative",
              overflow: "hidden",
              marginBottom: 24
            }}>
              {/* 简化的关系图可视化 */}
              <div style={{ 
                position: "absolute", 
                top: "50%", 
                left: "50%", 
                transform: "translate(-50%, -50%)",
                textAlign: "center"
              }}>
                <div style={{
                  width: 100,
                  height: 100,
                  borderRadius: 8,
                  backgroundColor: "#e6f7ff",
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  margin: "0 auto 20px",
                  color: "#1890ff",
                  fontWeight: "bold"
                }}>
                  {getCiTypeIcon(selectedCI.type)}
                  <div style={{ marginTop: 8, fontSize: 12 }}>{selectedCI.name}</div>
                </div>
                
                <div style={{ display: "flex", justifyContent: "center", gap: 30 }}>
                  {cis.filter(ci => 
                    relations.some(rel => 
                      (rel.source === selectedCI.id && rel.target === ci.id) ||
                      (rel.target === selectedCI.id && rel.source === ci.id)
                    )
                  ).slice(0, 3).map((ci) => (
                    <div key={ci.id} style={{ textAlign: "center" }}>
                      <div style={{
                        width: 70,
                        height: 70,
                        borderRadius: 8,
                        backgroundColor: "#f6ffed",
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "center",
                        margin: "0 auto",
                        color: "#52c41a"
                      }}>
                        {getCiTypeIcon(ci.type)}
                      </div>
                      <div style={{ marginTop: 8, fontSize: 12, width: 80, overflow: "hidden", textOverflow: "ellipsis" }}>{ci.name}</div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
            
            <div>
              <h4>关联关系详情</h4>
              <Table
                columns={[
                  {
                    title: "源配置项",
                    dataIndex: "source",
                    key: "source",
                    render: (source: string) => {
                      const ci = cis.find(c => c.id === source);
                      return ci ? ci.name : source;
                    }
                  },
                  {
                    title: "目标配置项",
                    dataIndex: "target",
                    key: "target",
                    render: (target: string) => {
                      const ci = cis.find(c => c.id === target);
                      return ci ? ci.name : target;
                    }
                  },
                  {
                    title: "关系类型",
                    dataIndex: "type",
                    key: "type",
                  },
                  {
                    title: "描述",
                    dataIndex: "description",
                    key: "description",
                  },
                ]}
                dataSource={relations.filter(
                  rel => rel.source === selectedCI.id || rel.target === selectedCI.id
                )}
                pagination={false}
                rowKey="id"
              />
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default CMDBPage;