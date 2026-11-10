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
  App,
} from 'antd';
import { Plus, Pencil, Trash2, RefreshCw, Briefcase } from 'lucide-react';
import { PageContainer } from '@/app/components/PageContainer';
import type { Project } from '@/lib/services/project-service';
import { projectService } from '@/lib/services/project-service';
import { departmentService } from '@/lib/services/department-service';
import type { Department } from '@/lib/services/department-service';
import { UserApi } from '@/lib/api/user-api';
import { BACKEND_MAX_PAGE_SIZE } from '@/lib/api/pagination';
import { useI18n } from '@/lib/i18n';

const { RangePicker } = DatePicker;

// 项目状态枚举与后端 ent/schema/project.go 保持一致（active/completed/suspended）
const projectStatusLabels: Record<string, string> = {
  active: 'In Progress',
  completed: 'Completed',
  suspended: 'Paused',
};

export default function ProjectsPage() {
  const { t } = useI18n();
  const { message, modal } = App.useApp();
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
      const response = await UserApi.getUsers({ page: 1, pageSize: BACKEND_MAX_PAGE_SIZE });
      setUserOptions(
        response.items.map(user => ({
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
      title: 'Project Name',
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
      title: 'Project code',
      dataIndex: 'code',
      key: 'code',
    },
    {
      title: 'Department',
      dataIndex:'departmentName',
      key: 'department',
      render: (id: number) => <span>ID: {id}</span>,
    },
    {
      title: 'Assignee',
      dataIndex:'managerName',
      key: 'manager',
      render: (id: number) => <span>ID: {id}</span>,
    },
    {
      title: 'Progress',
      dataIndex: 'progress', // Not in API yet
      key: 'progress',
      render: (percent: number) => <Progress percent={percent || 0} size="small" />,
      width: 200,
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => <Tag>{projectStatusLabels[status] || status || '-'}</Tag>,
    },
    {
      title: 'Actions',
      key: 'action',
      render: (_: unknown, record: Project) => (
        <Space size="middle">
          <Button aria-label="Edit Project" type="text" icon={<Pencil />} onClick={() => handleEdit(record)} />
          <Button
            type="text"
            danger
            aria-label="Delete project"
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
    modal.confirm({
      title: 'Confirm Delete',
      content: `Are you sure you want to delete the project"${record.name}"?`,
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
        title: 'Projects',
        breadcrumb: { items: [{ title: 'Home page' }, { title: 'Projects' }] },
      }}
      extra={[
        <Button key="refresh" icon={<RefreshCw />} onClick={fetchProjects} loading={fetching}>

          Refresh
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

          Create Project
        </Button>,
      ]}
    >
      <Table columns={columns} dataSource={projects} rowKey="id" loading={fetching} />

      <Modal
        title="New/Edit Project"
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
            label="Project Name"
            rules={[{ required: true, message: 'Please enter the project name' }]}
          >
            <Input placeholder="Please enter the project name" />
          </Form.Item>
          <Form.Item
            name="code"
            label="Project code"
            rules={[{ required: true, message: 'Please enter the project code' }]}
          >
            <Input placeholder="Please enter the project code" />
          </Form.Item>
          <div className="grid grid-cols-2 gap-4">
            <Form.Item name="departmentId" label="Department">
              <Select placeholder="Select department" options={departmentOptions} />
            </Form.Item>
            <Form.Item name="managerId" label="Assignee">
              <Select placeholder="Please select manager" options={userOptions} />
            </Form.Item>
          </div>
          <Form.Item name="description" label="Description">
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
}
