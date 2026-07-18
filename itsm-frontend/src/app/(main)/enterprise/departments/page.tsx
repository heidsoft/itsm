'use client';

import React, { useState, useEffect } from 'react';
import { Table, Button, Tag, Space, Modal, Form, Input, Select, TreeSelect, App } from 'antd';
import { Plus, Pencil, Trash2, Users, RefreshCw } from 'lucide-react';
import { PageContainer } from '@/app/components/PageContainer';
import type { Department } from '@/lib/services/department-service';
import { departmentService } from '@/lib/services/department-service';

import { UserApi } from '@/lib/api/user-api';
import { useI18n } from '@/lib/i18n';

export default function DepartmentsPage() {
  const { message, modal } = App.useApp();
  const { t } = useI18n();
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [editingDepartment, setEditingDepartment] = useState<Department | null>(null);
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [treeData, setTreeData] = useState<Department[]>([]);
  const [fetching, setFetching] = useState(false);
  const [users, setUsers] = useState<{ label: string; value: number }[]>([]);

  const fetchUsers = async () => {
    try {
      const response = await UserApi.getUsers({ page: 1, pageSize: 100 });
      setUsers(
        response.users.map(user => ({
          label: user.name || user.username,
          value: user.id,
        }))
      );
    } catch (error) {
      console.error('Failed to fetch users:', error);
    }
  };

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
      message.error(t('common.getFailed'));
    } finally {
      setFetching(false);
    }
  };

  useEffect(() => {
    fetchDepartments();
    fetchUsers();
  }, []);

  const columns = [
    {
      title: t('departments.departmentName'),
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: t('departments.departmentCode'),
      dataIndex: 'code',
      key: 'code',
      render: (text: string) => <Tag color="blue">{text}</Tag>,
    },
    {
      title: t('departments.manager'),
      dataIndex:'managerId',
      key: 'manager',
      render: (text: string) => (
        <Space>
          <Users />
          {text || '-'}
        </Space>
      ),
    },
    {
      title: t('common.description'),
      dataIndex: 'description',
      key: 'description',
    },
    {
      title: t('common.action'),
      key: 'action',
      render: (_: unknown, record: Department) => (
        <Space size="middle">
          <Button type="text" icon={<Pencil />} onClick={() => handleEdit(record)} />
          <Button
            type="text"
            danger
            icon={<Trash2 />}
            onClick={() => handleDelete(record)}
          />
        </Space>
      ),
    },
  ];

  const handleEdit = (record: Department) => {
    setEditingDepartment(record);
    form.setFieldsValue(record);
    setIsModalVisible(true);
  };

  const handleDelete = (record: Department) => {
    modal.confirm({
      title: t('common.confirmDelete'),
      content: t('departments.deleteConfirm', { name: record.name }),
      onOk: async () => {
        try {
          await departmentService.deleteDepartment(record.id);
          message.success(t('common.deleteSuccess'));
          fetchDepartments();
        } catch (error) {
          message.error(t('common.deleteFailed'));
        }
      },
    });
  };

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      // 编辑/新建分支，避免修改时走 create 导致重复记录
      if (editingDepartment) {
        await departmentService.updateDepartment(editingDepartment.id, values);
      } else {
        await departmentService.createDepartment(values);
      }

      message.success(t('common.saveSuccess'));
      setIsModalVisible(false);
      setEditingDepartment(null);
      form.resetFields();
      fetchDepartments();
    } catch (error) {
      console.error('Operation Failed:', error);
      message.error(t('common.operationFailed'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <PageContainer
      header={{
        title: t('departments.title'),
        breadcrumb: {
          items: [
            { title: t('common.back') },
            { title: '企业管理' },
            { title: t('departments.title') },
          ],
        },
      }}
      extra={[
        <Button key="refresh" icon={<RefreshCw />} onClick={fetchDepartments} loading={fetching}>
          {t('common.refresh')}
        </Button>,
        <Button
          key="create"
          type="primary"
          icon={<Plus />}
          onClick={() => {
            setEditingDepartment(null);
            form.resetFields();
            setIsModalVisible(true);
          }}
        >
          {t('departments.create')}
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
        title={editingDepartment ? t('departments.editTitle') : t('departments.createOrEdit')}
        open={isModalVisible}
        onOk={handleOk}
        onCancel={() => {
          setIsModalVisible(false);
          setEditingDepartment(null);
          form.resetFields();
        }}
        confirmLoading={loading}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label={t('departments.departmentName')}
            rules={[
              {
                required: true,
                message: t('common.inputPlaceholder') + t('departments.departmentName'),
              },
            ]}
          >
            <Input placeholder={t('common.inputPlaceholder') + t('departments.departmentName')} />
          </Form.Item>
          <Form.Item
            name="code"
            label={t('departments.departmentCode')}
            rules={[
              {
                required: true,
                message: t('common.inputPlaceholder') + t('departments.departmentCode'),
              },
            ]}
          >
            <Input placeholder={t('common.inputPlaceholder') + t('departments.departmentCode')} />
          </Form.Item>
          <Form.Item name="parentId" label={t('departments.parentDepartment')}>
            <TreeSelect
              treeData={treeData}
              placeholder={t('common.selectPlaceholder') + t('departments.parentDepartment')}
              fieldNames={{ label: 'name', value: 'id', children: 'children' }}
              allowClear
              treeDefaultExpandAll
            />
          </Form.Item>
          <Form.Item name="managerId" label={t('departments.manager')}>
            <Select
              placeholder={t('common.selectPlaceholder') + t('departments.manager')}
              options={users}
              showSearch
              filterOption={(input, option) =>
                (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
              }
            />
          </Form.Item>
          <Form.Item name="description" label={t('common.description')}>
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
}
