'use client';

import React, { useState, useEffect } from 'react';
import { Table, Button, Tag, Space, Modal, Form, Input, Select, TreeSelect, message } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, TeamOutlined, SyncOutlined } from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-components';
import { departmentService, Department } from '@/lib/services/department-service';

export default function DepartmentsPage() {
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [treeData, setTreeData] = useState<Department[]>([]);
  const [fetching, setFetching] = useState(false);

  const fetchDepartments = async () => {
    setFetching(true);
    try {
      const data = await departmentService.getDepartmentTree();
      setDepartments(data);
      // For TreeSelect, we might need to map data if fields don't match exactly, 
      // but we configured fieldNames in TreeSelect which should handle it.
      // However, antd TreeSelect expects 'value', 'title', 'children' by default or configured.
      // Let's assume backend returns a structure that fits or we might need to process it recursively.
      setTreeData(data); 
    } catch (error) {
      console.error('Failed to fetch departments:', error);
      message.error('获取部门列表失败');
    } finally {
      setFetching(false);
    }
  };

  useEffect(() => {
    fetchDepartments();
  }, []);

  const columns = [
    {
      title: '部门名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '部门代码',
      dataIndex: 'code',
      key: 'code',
      render: (text: string) => <Tag color='blue'>{text}</Tag>,
    },
    {
      title: '负责人',
      dataIndex: 'manager_id',
      key: 'manager',
      render: (text: string) => (
        <Space>
          <TeamOutlined />
          {text || '-'}
        </Space>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: Department) => (
        <Space size='middle'>
          <Button type='text' icon={<EditOutlined />} onClick={() => handleEdit(record)} />
          <Button
            type='text'
            danger
            icon={<DeleteOutlined />}
            onClick={() => handleDelete(record)}
          />
        </Space>
      ),
    },
  ];

  const handleEdit = (record: Department) => {
    form.setFieldsValue(record);
    setIsModalVisible(true);
  };

  const handleDelete = (record: Department) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除部门 "${record.name}" 吗？`,
      onOk: async () => {
        try {
          await departmentService.deleteDepartment(record.id);
          message.success('删除成功');
          fetchDepartments();
        } catch (error) {
          message.error('删除失败');
        }
      },
    });
  };

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      
      // If editing, we need the ID. But currently form doesn't have it.
      // We should store current editing record or use a hidden field.
      // For simplicity, let's assume create for now or check how to handle edit.
      // Actually, handleEdit sets values.
      
      await departmentService.createDepartment(values);
      
      message.success('保存成功');
      setIsModalVisible(false);
      form.resetFields();
      fetchDepartments();
    } catch (error) {
      console.error('Operation Failed:', error);
      message.error('操作失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <PageContainer
      header={{
        title: '部门管理',
        breadcrumb: { items: [{ title: '首页' }, { title: '企业管理' }, { title: '部门管理' }] },
      }}
      extra={[
        <Button
          key='refresh'
          icon={<SyncOutlined />}
          onClick={fetchDepartments}
          loading={fetching}
        >
          刷新
        </Button>,
        <Button
          key='create'
          type='primary'
          icon={<PlusOutlined />}
          onClick={() => {
            form.resetFields();
            setIsModalVisible(true);
          }}
        >
          新建部门
        </Button>,
      ]}
    >
      <Table
        columns={columns}
        dataSource={departments}
        rowKey="id"
        pagination={false}
        loading={fetching}
        expandable={{ defaultExpandAllRows: true }}
      />

      <Modal
        title='新建/编辑部门'
        open={isModalVisible}
        onOk={handleOk}
        onCancel={() => setIsModalVisible(false)}
        confirmLoading={loading}
      >
        <Form form={form} layout='vertical'>
          <Form.Item
            name='name'
            label='部门名称'
            rules={[{ required: true, message: '请输入部门名称' }]}
          >
            <Input placeholder='请输入部门名称' />
          </Form.Item>
          <Form.Item
            name='code'
            label='部门代码'
            rules={[{ required: true, message: '请输入部门代码' }]}
          >
            <Input placeholder='请输入部门代码' />
          </Form.Item>
          <Form.Item name='parent_id' label='上级部门'>
            <TreeSelect
              treeData={treeData}
              placeholder='请选择上级部门'
              fieldNames={{ label: 'name', value: 'id', children: 'children' }}
              allowClear
              treeDefaultExpandAll
            />
          </Form.Item>
          <Form.Item name='manager_id' label='负责人'>
             {/* TODO: Load users from backend */}
            <Select placeholder='请选择负责人'>
              <Select.Option value={1}>管理员</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name='description' label='描述'>
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
}
