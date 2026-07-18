'use client';

import React, { useState, useEffect } from 'react';
import {
  Table,
  Button,
  Tag,
  Space,
  Modal,
  Form,
  Input,
  Select,
  DatePicker,
  Progress,
  message,
} from 'antd';
import { Plus, Pencil, Trash2, RefreshCw, Briefcase } from 'lucide-react';
import { PageContainer } from '@/app/components/PageContainer';
import type { Project } from '@/lib/services/project-service';
import { projectService } from '@/lib/services/project-service';
import { departmentService } from '@/lib/services/department-service';
import type { Department } from '@/lib/services/department-service';
import { UserApi } from '@/lib/api/user-api';
import { useI18n } from '@/lib/i18n';

const { RangePicker } = DatePicker;

export default function ProjectsPage() {
  const { t } = useI18n();
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [editingProject, setEditingProject] = useState<Project | null>(null);
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [projects, setProjects] = useState<Project[]>([]);
  const [fetching, setFetching] = useState(false);
  const [departmentOptions, setDepartmentOptions] = useState<{ label: string; value: number }[]>([]);
  const [userOptions, setUserOptions] = useState<{ label: string; value: number }[]>([]);

  const fetchProjects = async () => {
    setFetching(true);
    try {
      const data = await projectService.listProjects();
      setProjects(data);
    } catch (error) {
      console.error('Failed to fetch projects:', error);
      message.error(t('common.getFailed'));
    } finally {
      setFetching(false);
    }
  };

  const flattenDepartments = (list: Department[]): { label: string; value: number }[] => {
    const result: { label: string; value: number }[] = [];
    list.forEach(d => {
      result.push({ label: d.name, value: d.id });
      if (d.children) result.push(...flattenDepartments(d.children));
    });
    return result;
  };

  const fetchDepartmentOptions = async () => {
    try {
      const tree = await departmentService.getDepartmentTree();
      setDepartmentOptions(flattenDepartments(tree));
    } catch (error) {
      console.error('Failed to fetch departments:', error);
    }
  };

  const fetchUserOptions = async () => {
    try {
      const response = await UserApi.getUsers({ page: 1, pageSize: 100 });
      setUserOptions(
        response.users.map(user => ({
          label: user.name || user.username,
          value: user.id,
        }))
      );
    } catch (error) {
      console.error('Failed to fetch users:', error);
    }
  };

  useEffect(() => {
    fetchProjects();
    fetchDepartmentOptions();
    fetchUserOptions();
  }, []);

  const columns = [
    {
      title: '项目名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => (
        <Space>
          <Briefcase />
          <span className="font-medium">{text}</span>
        </Space>
      ),
    },
    {
      title: '项目代码',
      dataIndex: 'code',
      key: 'code',
    },
    {
      title: '所属部门',
      dataIndex:'departmentName',
      key: 'department',
      render: (id: number) => <span>ID: {id}</span>,
    },
    {
      title: '负责人',
      dataIndex:'managerName',
      key: 'manager',
      render: (id: number) => <span>ID: {id}</span>,
    },
    {
      title: '进度',
      dataIndex: 'progress', // Not in API yet
      key: 'progress',
      render: (percent: number) => <Progress percent={percent || 0} size="small" />,
      width: 200,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        // Map status
        return <Tag>{status || 'Unknown'}</Tag>;
      },
    },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: Project) => (
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

  const handleEdit = (record: Project) => {
    setEditingProject(record);
    form.setFieldsValue(record);
    setIsModalVisible(true);
  };

  const handleDelete = (record: Project) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除项目 "${record.name}" 吗？`,
      onOk: async () => {
        try {
          await projectService.deleteProject(record.id);
          message.success(t('common.deleteSuccess'));
          fetchProjects();
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
      if (editingProject) {
        await projectService.updateProject(editingProject.id, values);
      } else {
        await projectService.createProject(values);
      }

      message.success(t('common.saveSuccess'));
      setIsModalVisible(false);
      setEditingProject(null);
      form.resetFields();
      fetchProjects();
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
        title: '项目管理',
        breadcrumb: { items: [{ title: '首页' }, { title: '项目管理' }] },
      }}
      extra={[
        <Button key="refresh" icon={<RefreshCw />} onClick={fetchProjects} loading={fetching}>
          刷新
        </Button>,
        <Button
          key="create"
          type="primary"
          icon={<Plus />}
          onClick={() => {
            setEditingProject(null);
            form.resetFields();
            setIsModalVisible(true);
          }}
        >
          新建项目
        </Button>,
      ]}
    >
      <Table columns={columns} dataSource={projects} rowKey="id" loading={fetching} />

      <Modal
        title="新建/编辑项目"
        open={isModalVisible}
        onOk={handleOk}
        onCancel={() => {
          setIsModalVisible(false);
          setEditingProject(null);
        }}
        confirmLoading={loading}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="项目名称"
            rules={[{ required: true, message: '请输入项目名称' }]}
          >
            <Input placeholder="请输入项目名称" />
          </Form.Item>
          <Form.Item
            name="code"
            label="项目代码"
            rules={[{ required: true, message: '请输入项目代码' }]}
          >
            <Input placeholder="请输入项目代码" />
          </Form.Item>
          <div className="grid grid-cols-2 gap-4">
            <Form.Item name="departmentId" label="所属部门">
              <Select placeholder="请选择部门" options={departmentOptions} />
            </Form.Item>
            <Form.Item name="managerId" label="负责人">
              <Select placeholder="请选择负责人" options={userOptions} />
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
