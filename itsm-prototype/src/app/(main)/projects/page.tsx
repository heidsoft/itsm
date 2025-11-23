"use client";

import React, { useState, useEffect } from "react";
import { Table, Button, Tag, Space, Modal, Form, Input, Select, DatePicker, Progress, message } from "antd";
import { PlusOutlined, EditOutlined, DeleteOutlined, ProjectOutlined, SyncOutlined } from "@ant-design/icons";
import { PageContainer } from "@ant-design/pro-components";
import { projectService, Project } from "@/lib/services/project-service";

const { RangePicker } = DatePicker;

export default function ProjectsPage() {
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [projects, setProjects] = useState<Project[]>([]);
  const [fetching, setFetching] = useState(false);

  const fetchProjects = async () => {
    setFetching(true);
    try {
      const data = await projectService.listProjects();
      setProjects(data);
    } catch (error) {
      console.error("Failed to fetch projects:", error);
      message.error("获取项目列表失败");
    } finally {
      setFetching(false);
    }
  };

  useEffect(() => {
    fetchProjects();
  }, []);

  const columns = [
    {
      title: "项目名称",
      dataIndex: "name",
      key: "name",
      render: (text: string) => (
        <Space>
          <ProjectOutlined />
          <span className="font-medium">{text}</span>
        </Space>
      ),
    },
    {
      title: "项目代码",
      dataIndex: "code",
      key: "code",
    },
    {
      title: "所属部门",
      dataIndex: "department_id", // TODO: Resolve department name
      key: "department",
      render: (id: number) => <span>ID: {id}</span>,
    },
    {
      title: "负责人",
      dataIndex: "manager_id", // TODO: Resolve user name
      key: "manager",
      render: (id: number) => <span>ID: {id}</span>,
    },
    {
      title: "进度",
      dataIndex: "progress", // Not in API yet
      key: "progress",
      render: (percent: number) => <Progress percent={percent || 0} size="small" />,
      width: 200,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      render: (status: string) => {
        // TODO: Map status
        return <Tag>{status || 'Unknown'}</Tag>;
      },
    },
    {
      title: "操作",
      key: "action",
      render: (_: any, record: Project) => (
        <Space size="middle">
          <Button type="text" icon={<EditOutlined />} onClick={() => handleEdit(record)} />
          <Button type="text" danger icon={<DeleteOutlined />} onClick={() => handleDelete(record)} />
        </Space>
      ),
    },
  ];

  const handleEdit = (record: Project) => {
    form.setFieldsValue(record);
    setIsModalVisible(true);
  };

  const handleDelete = (record: Project) => {
    Modal.confirm({
      title: "确认删除",
      content: `确定要删除项目 "${record.name}" 吗？`,
      onOk: () => {
        message.success("删除功能暂未实现");
      },
    });
  };

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      
      await projectService.createProject(values);
      
      message.success("保存成功");
      setIsModalVisible(false);
      form.resetFields();
      fetchProjects();
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
        title: "项目管理",
        breadcrumb: { items: [{ title: "首页" }, { title: "项目管理" }] },
      }}
      extra={[
        <Button
          key="refresh"
          icon={<SyncOutlined />}
          onClick={fetchProjects}
          loading={fetching}
        >
          刷新
        </Button>,
        <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => setIsModalVisible(true)}>
          新建项目
        </Button>
      ]}
    >
      <Table 
        columns={columns} 
        dataSource={projects} 
        rowKey="id"
        loading={fetching}
      />

      <Modal
        title="新建/编辑项目"
        open={isModalVisible}
        onOk={handleOk}
        onCancel={() => setIsModalVisible(false)}
        confirmLoading={loading}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="项目名称"
            rules={[{ required: true, message: "请输入项目名称" }]}
          >
            <Input placeholder="请输入项目名称" />
          </Form.Item>
          <Form.Item
            name="code"
            label="项目代码"
            rules={[{ required: true, message: "请输入项目代码" }]}
          >
            <Input placeholder="请输入项目代码" />
          </Form.Item>
          <div className="grid grid-cols-2 gap-4">
            <Form.Item name="department_id" label="所属部门">
              <Select placeholder="请选择部门">
                <Select.Option value={1}>研发中心</Select.Option>
              </Select>
            </Form.Item>
            <Form.Item name="manager_id" label="负责人">
              <Select placeholder="请选择负责人">
                <Select.Option value={1}>管理员</Select.Option>
              </Select>
            </Form.Item>
          </div>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
}
