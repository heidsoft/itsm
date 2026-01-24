"use client";

import React, { useState, useEffect } from "react";
import { Table, Button, Tag, Space, Modal, Form, Input, ColorPicker, message } from "antd";
import { PlusOutlined, EditOutlined, DeleteOutlined, TagOutlined, SyncOutlined } from "@ant-design/icons";
import { PageContainer } from "@ant-design/pro-components";
import { tagService, Tag as ITag } from "@/lib/services/tag-service";

export default function TagsPage() {
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [tags, setTags] = useState<ITag[]>([]);
  const [fetching, setFetching] = useState(false);

  const fetchTags = async () => {
    setFetching(true);
    try {
      const data = await tagService.listTags();
      setTags(data);
    } catch (error) {
      console.error("Failed to fetch tags:", error);
      message.error("获取标签列表失败");
    } finally {
      setFetching(false);
    }
  };

  useEffect(() => {
    fetchTags();
  }, []);

  const columns = [
    {
      title: "标签名称",
      dataIndex: "name",
      key: "name",
      render: (text: string, record: ITag) => (
        <Tag color={record.color}>
          <Space>
            <TagOutlined />
            {text}
          </Space>
        </Tag>
      ),
    },
    { title: "标签代码", dataIndex: "code", key: "code" },
    {
      title: "颜色",
      dataIndex: "color",
      key: "color",
      render: (color: string) => (
        <div className="flex items-center">
          <div className="w-4 h-4 rounded mr-2" style={{ backgroundColor: color }} />
          {color}
        </div>
      ),
    },
    // {
    //   title: "引用次数",
    //   dataIndex: "usageCount", // Not supported by backend yet
    //   key: "usageCount",
    // },
    { title: "描述", dataIndex: "description", key: "description" },
    {
      title: "操作",
      key: "action",
      render: (_: any, record: ITag) => (
        <Space size="middle">
          <Button type="text" icon={<EditOutlined />} onClick={() => handleEdit(record)} />
          <Button type="text" danger icon={<DeleteOutlined />} onClick={() => handleDelete(record)} />
        </Space>
      ),
    },
  ];

  const handleEdit = (record: ITag) => {
    form.setFieldsValue(record);
    setIsModalVisible(true);
  };

  const handleDelete = (record: ITag) => {
    Modal.confirm({
      title: "确认删除",
      content: `确定要删除标签 "${record.name}" 吗？`,
      onOk: async () => {
        try {
          await tagService.deleteTag(record.id);
          message.success("删除成功");
          fetchTags();
        } catch (error) {
          message.error("删除失败");
        }
      },
    });
  };

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      // Convert color object to hex string if needed
      if (typeof values.color === 'object' && values.color?.toHexString) {
        values.color = values.color.toHexString();
      }
      
      setLoading(true);
      await tagService.createTag(values);
      
      message.success("保存成功");
      setIsModalVisible(false);
      form.resetFields();
      fetchTags();
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
        title: "全局标签管理",
        breadcrumb: { items: [{ title: "首页" }, { title: "标签管理" }] },
      }}
      extra={[
        <Button
          key="refresh"
          icon={<SyncOutlined />}
          onClick={fetchTags}
          loading={fetching}
        >
          刷新
        </Button>,
        <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => setIsModalVisible(true)}>
          新建标签
        </Button>
      ]}
    >
      <Table 
        columns={columns} 
        dataSource={tags} 
        rowKey="id"
        loading={fetching}
      />

      <Modal
        title="新建/编辑标签"
        open={isModalVisible}
        onOk={handleOk}
        onCancel={() => setIsModalVisible(false)}
        confirmLoading={loading}
      >
        <Form form={form} layout="vertical" initialValues={{ color: "#1890ff" }}>
          <Form.Item
            name="name"
            label="标签名称"
            rules={[{ required: true, message: "请输入标签名称" }]}
          >
            <Input placeholder="请输入标签名称" />
          </Form.Item>
          <Form.Item
            name="code"
            label="标签代码"
            rules={[{ required: true, message: "请输入标签代码" }]}
          >
            <Input placeholder="请输入标签代码" />
          </Form.Item>
          <Form.Item name="color" label="颜色">
            <ColorPicker showText />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
}
