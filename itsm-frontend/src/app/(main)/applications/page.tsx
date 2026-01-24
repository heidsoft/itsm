"use client";

import React, { useState, useEffect } from "react";
import { Table, Button, Tag, Space, Modal, Form, Input, Select, Tabs, message } from "antd";
import { PlusOutlined, EditOutlined, DeleteOutlined, AppstoreOutlined, ApiOutlined, SyncOutlined } from "@ant-design/icons";
import { PageContainer } from "@ant-design/pro-components";
import { applicationService, Application, Microservice } from "@/lib/services/application-service";

const { TabPane } = Tabs;

export default function ApplicationsPage() {
  const [activeTab, setActiveTab] = useState("applications");
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [modalType, setModalType] = useState<"application" | "microservice">("application");
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [applications, setApplications] = useState<Application[]>([]);
  const [microservices, setMicroservices] = useState<Microservice[]>([]);
  const [fetching, setFetching] = useState(false);

  const fetchData = async () => {
    setFetching(true);
    try {
      const apps = await applicationService.listApplications();
      setApplications(apps);
      
      // Flatten microservices from all apps
      const allMicroservices: Microservice[] = [];
      apps.forEach(app => {
        if (app.edges?.microservices) {
          allMicroservices.push(...app.edges.microservices);
        }
      });
      setMicroservices(allMicroservices);
    } catch (error) {
      console.error("Failed to fetch applications:", error);
      message.error("获取应用列表失败");
    } finally {
      setFetching(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const appColumns = [
    {
      title: "应用名称",
      dataIndex: "name",
      key: "name",
      render: (text: string) => (
        <Space>
          <AppstoreOutlined />
          <span className="font-medium">{text}</span>
        </Space>
      ),
    },
    { title: "应用代码", dataIndex: "code", key: "code" },
    { title: "所属项目", dataIndex: "project_id", key: "project", render: (id: number) => <span>ID: {id}</span> },
    // { title: "负责人", dataIndex: "owner", key: "owner" }, // Not in schema currently
    {
      title: "类型",
      dataIndex: "type",
      key: "type",
      render: (type: string) => <Tag color="blue">{(type || 'UNKNOWN').toUpperCase()}</Tag>,
    },
    {
      title: "微服务数",
      key: "microservices",
      render: (_: any, record: Application) => <Tag color="purple">{record.edges?.microservices?.length || 0}</Tag>,
    },
    {
      title: "操作",
      key: "action",
      render: (_: any, record: Application) => (
        <Space size="middle">
          <Button type="text" icon={<EditOutlined />} onClick={() => handleEdit(record, "application")} />
          <Button type="text" danger icon={<DeleteOutlined />} onClick={() => handleDelete(record)} />
        </Space>
      ),
    },
  ];

  const msColumns = [
    {
      title: "微服务名称",
      dataIndex: "name",
      key: "name",
      render: (text: string) => (
        <Space>
          <ApiOutlined />
          <span className="font-medium">{text}</span>
        </Space>
      ),
    },
    { title: "服务代码", dataIndex: "code", key: "code" },
    { title: "所属应用", dataIndex: "application_id", key: "application", render: (id: number) => <span>App ID: {id}</span> },
    {
      title: "技术栈",
      key: "tech",
      render: (_: any, record: Microservice) => (
        <Space>
          <Tag>{record.language || '-'}</Tag>
          <Tag>{record.framework || '-'}</Tag>
        </Space>
      ),
    },
    {
      title: "操作",
      key: "action",
      render: (_: any, record: Microservice) => (
        <Space size="middle">
          <Button type="text" icon={<EditOutlined />} onClick={() => handleEdit(record, "microservice")} />
          <Button type="text" danger icon={<DeleteOutlined />} onClick={() => handleDelete(record)} />
        </Space>
      ),
    },
  ];

  const handleCreate = () => {
    setModalType(activeTab === "applications" ? "application" : "microservice");
    form.resetFields();
    setIsModalVisible(true);
  };

  const handleEdit = (record: any, type: "application" | "microservice") => {
    setModalType(type);
    form.setFieldsValue(record);
    setIsModalVisible(true);
  };

  const handleDelete = (record: any) => {
    Modal.confirm({
      title: "确认删除",
      content: `确定要删除 "${record.name}" 吗？`,
      onOk: async () => {
        try {
          if (modalType === "application" || activeTab === "applications") {
            await applicationService.deleteApplication(record.id);
          } else {
            await applicationService.deleteMicroservice(record.id);
          }
          message.success("删除成功");
          fetchData();
        } catch (error) {
          message.error("删除失败");
        }
      },
    });
  };

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      
      if (modalType === "application") {
        await applicationService.createApplication(values);
      } else {
        await applicationService.createMicroservice(values);
      }
      
      message.success("保存成功");
      setIsModalVisible(false);
      form.resetFields();
      fetchData();
    } catch (error) {
      console.error("Operation Failed:", error);
      message.error("操作失败");
    } finally {
      setLoading(false);
    }
  };

  return (
    <PageContainer
      header={{
        title: "应用与服务管理",
        breadcrumb: { items: [{ title: "首页" }, { title: "应用管理" }] },
      }}
      extra={[
        <Button
          key="refresh"
          icon={<SyncOutlined />}
          onClick={fetchData}
          loading={fetching}
        >
          刷新
        </Button>,
        <Button key="create" type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
          新建{activeTab === "applications" ? "应用" : "微服务"}
        </Button>
      ]}
    >
      <Tabs activeKey={activeTab} onChange={setActiveTab} type="card">
        <TabPane tab="应用系统" key="applications">
          <Table 
            columns={appColumns} 
            dataSource={applications} 
            rowKey="id"
            loading={fetching}
          />
        </TabPane>
        <TabPane tab="微服务" key="microservices">
          <Table 
            columns={msColumns} 
            dataSource={microservices} 
            rowKey="id"
            loading={fetching}
          />
        </TabPane>
      </Tabs>

      <Modal
        title={modalType === "application" ? "新建/编辑应用" : "新建/编辑微服务"}
        open={isModalVisible}
        onOk={handleOk}
        onCancel={() => setIsModalVisible(false)}
        confirmLoading={loading}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="名称"
            rules={[{ required: true, message: "请输入名称" }]}
          >
            <Input placeholder="请输入名称" />
          </Form.Item>
          <Form.Item
            name="code"
            label="代码"
            rules={[{ required: true, message: "请输入代码" }]}
          >
            <Input placeholder="请输入代码" />
          </Form.Item>
          
          {modalType === "application" ? (
            <>
              <Form.Item name="project_id" label="所属项目">
                <Select placeholder="请选择项目">
                  <Select.Option value={1}>ITSM 系统重构</Select.Option>
                </Select>
              </Form.Item>
              <Form.Item name="type" label="应用类型">
                <Select placeholder="请选择类型">
                  <Select.Option value="web">Web应用</Select.Option>
                  <Select.Option value="mobile">移动应用</Select.Option>
                  <Select.Option value="backend">后端服务</Select.Option>
                </Select>
              </Form.Item>
            </>
          ) : (
            <>
              <Form.Item name="application_id" label="所属应用" rules={[{ required: true, message: '请选择所属应用' }]}>
                <Select placeholder="请选择应用">
                  {applications.map(app => (
                    <Select.Option key={app.id} value={app.id}>{app.name}</Select.Option>
                  ))}
                </Select>
              </Form.Item>
              <div className="grid grid-cols-2 gap-4">
                <Form.Item name="language" label="开发语言">
                  <Select placeholder="请选择语言">
                    <Select.Option value="go">Go</Select.Option>
                    <Select.Option value="java">Java</Select.Option>
                    <Select.Option value="python">Python</Select.Option>
                    <Select.Option value="nodejs">Node.js</Select.Option>
                  </Select>
                </Form.Item>
                <Form.Item name="framework" label="框架">
                  <Input placeholder="如: Gin, Spring Boot" />
                </Form.Item>
              </div>
            </>
          )}
        </Form>
      </Modal>
    </PageContainer>
  );
}
