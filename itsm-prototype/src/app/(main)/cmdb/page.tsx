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
  DatabaseOutlined,
  CloudServerOutlined,
  DesktopOutlined,
  HddOutlined,
  PlusCircleOutlined,
  SearchOutlined,
  EyeOutlined,
  EditOutlined,
  ClusterOutlined,
  MoreOutlined,
  CloseOutlined,
  BranchesOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import { mockCIs, mockRelations } from "@/app/lib/cmdb-relations";

const { Search: SearchInput } = Input;
const { Option } = Select;

const getCiTypeIcon = (type: string) => {
  switch (type) {
    case "Cloud Server":
      return <CloudServerOutlined />;
    case "Physical Server":
      return <DesktopOutlined />;
    case "Relational Database":
      return <DatabaseOutlined />;
    case "Storage Device":
      return <HddOutlined />;
    default:
      return <ClusterOutlined />;
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
        text: "运行中",
        backgroundColor: "#f6ffed",
      };
    case "Maintenance":
      return {
        color: "#fa8c16",
        text: "维护中",
        backgroundColor: "#fff7e6",
      };
    case "Disabled":
      return {
        color: "#00000073",
        text: "已停用",
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

  const treeData = useMemo(() => [
    {
      title: "基础设施",
      key: "infrastructure",
      icon: <DesktopOutlined />,
      children: [
        {
          title: "云资源",
          key: "cloud",
          icon: <CloudServerOutlined />,
          children: [
            { title: "阿里云", key: "aliyun" },
            { title: "腾讯云", key: "tencent" },
          ],
        },
        {
          title: "物理设备",
          key: "physical",
          icon: <DesktopOutlined />,
          children: [
            { title: "服务器", key: "servers" },
            { title: "网络设备", key: "network" },
          ],
        },
      ],
    },
    {
      title: "应用系统",
      key: "applications",
      icon: <ClusterOutlined />,
      children: [
        { title: "电商平台", key: "ecommerce" },
        { title: "客户关系管理", key: "crm" },
      ],
    },
  ], []);

  const handleCreateCI = () => {
    setCreateModalVisible(true);
  };

  const handleCreateCIConfirm = async () => {
    try {
      const values = await createForm.validateFields();
      const newCI = {
        id: `CI-${Math.floor(Math.random() * 10000)}`,
        ...values,
        status: "Running",
        business: "新业务系统",
        owner: "运维团队",
        location: "数据中心",
        ip: "192.168.1.102",
        cpu: `${values.cpu || 4} 核`,
        memory: `${values.memory || 8}GB`,
        disk: `${values.disk || 100}GB SSD`,
      };
      setCis([...cis, newCI]);
      setCreateModalVisible(false);
      createForm.resetFields();
    } catch (error) {
      console.error("创建失败:", error);
    }
  };

  const handleViewRelations = (ci: typeof mockCIs[0]) => {
    setSelectedCI(ci);
    setRelationModalVisible(true);
  };

  const renderStatsCards = () => (
    <div style={{ marginBottom: 16 }}>
      <Row gutter={[12, 12]}>
        <Col xs={24} sm={12} md={6} lg={6}>
          <Card 
            style={{ 
              background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
              border: 'none',
              borderRadius: 12,
            }}
            styles={{ body: { padding: '16px' } }}
          >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div>
                <div style={{ color: 'rgba(255, 255, 255, 0.8)', fontSize: 14, marginBottom: 8 }}>配置项总数</div>
                <div style={{ color: '#fff', fontSize: 32, fontWeight: 'bold', lineHeight: 1 }}>{cis.length}</div>
              </div>
              <div style={{ 
                width: 56, 
                height: 56, 
                backgroundColor: 'rgba(255, 255, 255, 0.2)', 
                borderRadius: 16, 
                display: 'flex', 
                alignItems: 'center', 
                justifyContent: 'center',
                color: '#fff',
                fontSize: 24
              }}>
                <DatabaseOutlined />
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6} lg={6}>
          <Card 
            style={{ 
              background: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)',
              border: 'none',
              borderRadius: 12,
            }}
            styles={{ body: { padding: '16px' } }}
          >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div>
                <div style={{ color: 'rgba(255, 255, 255, 0.8)', fontSize: 14, marginBottom: 8 }}>云服务器</div>
                <div style={{ color: '#fff', fontSize: 32, fontWeight: 'bold', lineHeight: 1 }}>
                  {cis.filter(ci => ci.type === "Cloud Server").length}
                </div>
              </div>
              <div style={{ 
                width: 56, 
                height: 56, 
                backgroundColor: 'rgba(255, 255, 255, 0.2)', 
                borderRadius: 16, 
                display: 'flex', 
                alignItems: 'center', 
                justifyContent: 'center',
                color: '#fff',
                fontSize: 24
              }}>
                <CloudServerOutlined />
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6} lg={6}>
          <Card 
            style={{ 
              background: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
              border: 'none',
              borderRadius: 12,
            }}
            styles={{ body: { padding: '16px' } }}
          >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div>
                <div style={{ color: 'rgba(255, 255, 255, 0.8)', fontSize: 14, marginBottom: 8 }}>运行中</div>
                <div style={{ color: '#fff', fontSize: 32, fontWeight: 'bold', lineHeight: 1 }}>
                  {cis.filter(ci => ci.status === "Running").length}
                </div>
              </div>
              <div style={{ 
                width: 56, 
                height: 56, 
                backgroundColor: 'rgba(255, 255, 255, 0.2)', 
                borderRadius: 16, 
                display: 'flex', 
                alignItems: 'center', 
                justifyContent: 'center',
                color: '#fff',
                fontSize: 24
              }}>
                <DesktopOutlined />
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6} lg={6}>
          <Card 
            style={{ 
              background: 'linear-gradient(135deg, #fa709a 0%, #fee140 100%)',
              border: 'none',
              borderRadius: 12,
            }}
            styles={{ body: { padding: '16px' } }}
          >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div>
                <div style={{ color: 'rgba(255, 255, 255, 0.8)', fontSize: 14, marginBottom: 8 }}>维护中</div>
                <div style={{ color: '#fff', fontSize: 32, fontWeight: 'bold', lineHeight: 1 }}>
                  {cis.filter(ci => ci.status === "Maintenance").length}
                </div>
              </div>
              <div style={{ 
                width: 56, 
                height: 56, 
                backgroundColor: 'rgba(255, 255, 255, 0.2)', 
                borderRadius: 16, 
                display: 'flex', 
                alignItems: 'center', 
                justifyContent: 'center',
                color: '#fff',
                fontSize: 24
              }}>
                <HddOutlined />
              </div>
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  );

  const renderFilters = () => (
    <Card 
      style={{ 
        marginBottom: 16,
        borderRadius: 12,
      }}
      styles={{ body: { padding: '16px' } }}
    >
      <Row gutter={[20, 16]} align="middle">
        <Col xs={24} sm={12} md={8}>
          <SearchInput
            placeholder="搜索配置项名称、ID或IP..."
            allowClear
            onSearch={(value) => setSearchText(value)}
            size="large"
          />
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Select
            placeholder="类型筛选"
            size="large"
            allowClear
            value={filterType}
            onChange={(value) => setFilterType(value)}
            style={{ width: "100%" }}
          >
            <Option value="云服务器">☁️ 云服务器</Option>
            <Option value="物理服务器">🖥️ 物理服务器</Option>
            <Option value="关系型数据库">🗄️ 关系型数据库</Option>
            <Option value="存储设备">💾 存储设备</Option>
            <Option value="网络设备">🌐 网络设备</Option>
          </Select>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Select
            placeholder="状态筛选"
            size="large"
            allowClear
            style={{ width: "100%" }}
          >
            <Option value="运行中">🟢 运行中</Option>
            <Option value="维护中">🟡 维护中</Option>
            <Option value="已停用">⚫ 已停用</Option>
          </Select>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Space size={12} style={{ width: '100%', justifyContent: 'flex-end' }}>
            <Button
              icon={<ReloadOutlined />}
              onClick={() => {}}
              loading={loading}
              size="large"
            >
              刷新
            </Button>
            <Button
              type="primary"
              icon={<PlusCircleOutlined />}
              onClick={handleCreateCI}
              size="large"
            >
              新建配置项
            </Button>
          </Space>
        </Col>
      </Row>
    </Card>
  );

  const columns = [
    {
      title: "配置项信息",
      key: "ci_info",
      width: 300,
      render: (_: unknown, record: typeof mockCIs[0]) => (
        <div style={{ display: "flex", alignItems: "center" }}>
          <div style={{ 
            width: 40, 
            height: 40, 
            backgroundColor: "#e6f7ff", 
            borderRadius: 8, 
            display: "flex", 
            alignItems: "center", 
            justifyContent: "center", 
            marginRight: 12, 
            color: "#1890ff",
            fontSize: 18
          }}>
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
        <Tag color={getCiTypeColor(type)} icon={getCiTypeIcon(type)}>
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
    },
    {
      title: "负责人",
      dataIndex: "owner",
      key: "owner",
      width: 120,
    },
    {
      title: "位置",
      dataIndex: "location",
      key: "location",
      width: 150,
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
            icon={<EyeOutlined />}
            onClick={() => window.open(`/cmdb/${record.id}`)}
          />
          <Button
            type="text"
            size="small"
            icon={<EditOutlined />}
            onClick={() => window.open(`/cmdb/${record.id}/edit`)}
          />
          <Button
            type="text"
            size="small"
            icon={<ClusterOutlined />}
            onClick={() => handleViewRelations(record)}
          />
          <Button
            type="text"
            size="small"
            icon={<MoreOutlined />}
          />
        </Space>
      ),
    },
  ];

  const renderCIList = () => (
    <Card
      style={{ borderRadius: 16 }}
      styles={{ body: { padding: 0 } }}
    >
      <div style={{ padding: '24px 24px 0 24px', display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <h3 style={{ margin: 0, fontSize: 18, fontWeight: 600 }}>配置项列表</h3>
          {selectedRowKeys.length > 0 && (
            <Badge 
              count={selectedRowKeys.length} 
              showZero 
              style={{ backgroundColor: "#667eea" }} 
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
        }}
        columns={columns}
        dataSource={cis}
        rowKey="id"
        loading={loading}
        pagination={{
          pageSize: 10,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
        }}
        scroll={{ x: 1200 }}
      />
    </Card>
  );

  const renderRelationGraph = () => {
    if (!selectedCI) {
      return (
        <div style={{ textAlign: "center", padding: "100px 0", color: "#666" }}>
          <div style={{ margin: "0 auto 16px", color: "#1890ff", fontSize: 48 }}>
            <ClusterOutlined />
          </div>
          <p>请在配置项列表中选择一个配置项查看其关系图</p>
        </div>
      );
    }

    const relatedRelations = relations.filter(
      rel => rel.source === selectedCI.id || rel.target === selectedCI.id
    );

    return (
      <div style={{ padding: 24 }}>
        <div style={{ marginBottom: 24, display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <h3 style={{ margin: 0 }}>{selectedCI.name} 的关系图</h3>
          <Button 
            icon={<CloseOutlined />} 
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
          display: "flex",
          alignItems: "center",
          justifyContent: "center"
        }}>
          <div style={{ textAlign: "center" }}>
            <div style={{ fontSize: 48, color: "#1890ff", marginBottom: 16 }}>
              <ClusterOutlined />
            </div>
            <div style={{ fontSize: 16, color: "#666" }}>
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

  const renderTopologyView = () => (
    <div style={{ padding: 24 }}>
      <div style={{ 
        height: 600, 
        border: "1px solid #f0f0f0", 
        borderRadius: 8,
        backgroundColor: "#fafafa",
        display: "flex",
        alignItems: "center",
        justifyContent: "center"
      }}>
        <div style={{ textAlign: "center" }}>
          <div style={{ fontSize: 48, color: "#1890ff", marginBottom: 16 }}>
            <BranchesOutlined />
          </div>
          <h3>配置项拓扑视图</h3>
          <p style={{ color: "#666", maxWidth: 400, margin: "0 auto" }}>
            此视图展示了所有配置项之间的连接关系。通过可视化的方式，您可以快速了解整个IT基础设施的架构和依赖关系。
          </p>
          <div style={{ marginTop: 20, fontSize: 14, color: "#666" }}>
            共 {cis.length} 个配置项，{relations.length} 个关系连接
          </div>
        </div>
      </div>
    </div>
  );

  return (
    <div>
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
          icon={<PlusCircleOutlined />} 
          type="primary" 
          size="large"
          onClick={handleCreateCI}
        >
          新建配置项
        </Button>
      </div>

      <Tabs 
         activeKey={activeTab} 
         onChange={setActiveTab} 
         style={{ marginBottom: 24 }}
         size="large"
         items={[
           {
             key: 'list',
             label: (
               <span>
                 <DatabaseOutlined style={{ marginRight: 8 }} />
                 配置项列表
               </span>
             )
           },
           {
             key: 'relations',
             label: (
               <span>
                 <ClusterOutlined style={{ marginRight: 8 }} />
                 关系图
               </span>
             )
           },
           {
             key: 'topology',
             label: (
               <span>
                 <BranchesOutlined style={{ marginRight: 8 }} />
                 拓扑视图
               </span>
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

      {activeTab === "relations" && (
        <Card>
          <div style={{ display: "flex", height: "100%" }}>
            <div style={{ width: 300, borderRight: "1px solid #f0f0f0", padding: "16px 0" }}>
              <Tree
                showIcon
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

      <Modal
        title="新建配置项"
        open={createModalVisible}
        onCancel={() => {
          setCreateModalVisible(false);
          createForm.resetFields();
        }}
        onOk={handleCreateCIConfirm}
        width={700}
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
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={12}>
              <Form.Item label="CPU" name="cpu">
                <InputNumber
                  placeholder="请输入CPU核心数"
                  addonAfter="核"
                  min={1}
                  style={{ width: "100%" }}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="内存" name="memory">
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
              <Form.Item label="磁盘" name="disk">
                <InputNumber
                  placeholder="请输入磁盘大小"
                  addonAfter="GB"
                  min={1}
                  style={{ width: "100%" }}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="IP地址" name="ip">
                <Input placeholder="请输入IP地址" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item label="描述" name="description">
            <Input.TextArea placeholder="请输入配置项描述" rows={3} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="配置项关系图"
        open={relationModalVisible}
        onCancel={() => setRelationModalVisible(false)}
        footer={[
          <Button 
            key="close" 
            type="primary"
            onClick={() => setRelationModalVisible(false)}
          >
            关闭
          </Button>
        ]}
        width={1000}
      >
        {selectedCI && renderRelationGraph()}
      </Modal>
    </div>
  );
};

export default CMDBPage;
