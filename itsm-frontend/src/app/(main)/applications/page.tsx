'use client';

import React, { useState, useEffect } from 'react';
import { App, Table, Button, Tag, Space, Modal, Form, Input, Select, Tabs } from 'antd';
import { Plus, Pencil, Trash2, LayoutGrid, RefreshCw, Plug } from 'lucide-react';
import { PageContainer } from '@/app/components/PageContainer';
import type { Application, Microservice } from '@/lib/services/application-service';
import { applicationService } from '@/lib/services/application-service';
import { projectService } from '@/lib/services/project-service';
import { useI18n } from '@/lib/i18n';

// 应用类型中文映射，与下方新建/编辑表单的 Select options 保持一致
const appTypeLabels: Record<string, string> = {
  web: 'Web Application',
  mobile: 'Mobile App',
  backend: 'Backend service',
};

export default function ApplicationsPage() {
  const { t } = useI18n();
  const { message, modal } = App.useApp();
  const [activeTab, setActiveTab] = useState('applications');
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [modalType, setModalType] = useState<'application' | 'microservice'>('application');
  const [editingRecord, setEditingRecord] = useState<ApplicationRecord | MicroserviceRecord | null>(null);
  const [form] = Form.useForm();
  // 保存动作态（列表加载由 fetching 负责）
  const [actionLoading, setActionLoading] = useState(false);
  const [applications, setApplications] = useState<Application[]>([]);
  const [microservices, setMicroservices] = useState<Microservice[]>([]);
  const [fetching, setFetching] = useState(false);
  const [projectOptions, setProjectOptions] = useState<{ label: string; value: number }[]>([]);

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
      console.error('Failed to fetch applications:', error);
      message.error(t('common.getFailed'));
    } finally {
      setFetching(false);
    }
  };

  useEffect(() => {
    fetchData();
    projectService
      .listProjects()
      .then(data => setProjectOptions(data.map(p => ({ label: p.name, value: p.id }))))
      .catch(error => console.error('Failed to fetch projects:', error));
  }, []);

  const appColumns = [
    {
      title: 'Application Name',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => (
        <Space>
          <LayoutGrid />
          <span className="font-medium">{text}</span>
        </Space>
      ),
    },
    { title: 'Application code', dataIndex: 'code', key: 'code' },
    {
      title: 'Project',
      dataIndex:'projectId',
      key: 'project',
      render: (id: number) => <span>ID: {id}</span>,
    },
    // { title: "负责人", dataIndex: "owner", key: "owner" }, // Not in schema currently
    {
      title: 'Type',
      dataIndex: 'type',
      key: 'type',
      render: (type: string) => <Tag color="blue">{appTypeLabels[type] || type || '-'}</Tag>,
    },
    {
      title: 'Number of microservices',
      key: 'microservices',
      render: (_: unknown, record: Application) => (
        <Tag color="purple">{record.edges?.microservices?.length || 0}</Tag>
      ),
    },
    {
      title: 'Actions',
      key: 'action',
      render: (_: unknown, record: Application & { id: number }) => (
        <Space size="middle">
          <Button
            type="text"
            aria-label="Edit application"
            icon={<Pencil />}
            onClick={() => handleEdit(record as ApplicationRecord, 'application')}
          />
          <Button
            type="text"
            danger
            aria-label="Delete app"
            icon={<Trash2 />}
            onClick={() => handleDelete(record as ApplicationRecord)}
          />
        </Space>
      ),
    },
  ];

  const msColumns = [
    {
      title: 'Microservice name',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => (
        <Space>
          <Plug />
          <span className="font-medium">{text}</span>
        </Space>
      ),
    },
    { title: 'Service code', dataIndex: 'code', key: 'code' },
    {
      title: 'Belonging application',
      dataIndex:'applicationId',
      key: 'application',
      render: (id: number) => <span>App ID: {id}</span>,
    },
    {
      title: 'Technology stack',
      key: 'tech',
      render: (_: unknown, record: Microservice) => (
        <Space>
          <Tag>{record.language || '-'}</Tag>
          <Tag>{record.framework || '-'}</Tag>
        </Space>
      ),
    },
    {
      title: 'Actions',
      key: 'action',
      render: (_: unknown, record: Microservice & { id: number }) => (
        <Space size="middle">
          <Button
            type="text"
            aria-label="Edit microservices"
            icon={<Pencil />}
            onClick={() => handleEdit(record as MicroserviceRecord, 'microservice')}
          />
          <Button
            type="text"
            danger
            aria-label="Delete microservice"
            icon={<Trash2 />}
            onClick={() => handleDelete(record as MicroserviceRecord)}
          />
        </Space>
      ),
    },
  ];

  const handleCreate = () => {
    setModalType(activeTab === 'applications' ? 'application' : 'microservice');
    setEditingRecord(null);
    form.resetFields();
    setIsModalVisible(true);
  };

interface ApplicationRecord extends Application {
  key: string;
}

interface MicroserviceRecord extends Microservice {
  key: string;
}

type RecordData = ApplicationRecord | MicroserviceRecord;

const handleEdit = (record: RecordData, type: 'application' | 'microservice') => {
  setModalType(type);
  setEditingRecord(record);
  form.setFieldsValue(record);
  setIsModalVisible(true);
};

const handleDelete = (record: RecordData) => {
  modal.confirm({
    title: 'Confirm Delete',
    content: `Are you sure you want to delete "${record.name}"?`,
    onOk: async () => {
      try {
        if (modalType === 'application' || activeTab === 'applications') {
          await applicationService.deleteApplication(record.id);
        } else {
          await applicationService.deleteMicroservice(record.id);
        }
        message.success(t('common.deleteSuccess'));
        fetchData();
      } catch (error) {
        message.error(t('common.deleteFailed'));
      }
    },
  });
  };

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      setActionLoading(true);

      // 根据是否存在 editingRecord 决定走新增还是更新；避免“修改却走 create”导致数据污染
      if (modalType === 'application') {
        if (editingRecord) {
          await applicationService.updateApplication(editingRecord.id, values);
        } else {
          await applicationService.createApplication(values);
        }
      } else {
        if (editingRecord) {
          await applicationService.updateMicroservice(editingRecord.id, values);
        } else {
          await applicationService.createMicroservice(values);
        }
      }

      message.success(t('common.saveSuccess'));
      setIsModalVisible(false);
      setEditingRecord(null);
      form.resetFields();
      fetchData();
    } catch (error) {
      console.error('Operation Failed:', error);
      message.error(t('common.operationFailed'));
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <PageContainer
      header={{
        title: 'Application and Service Management',
        breadcrumb: { items: [{ title: 'Home page' }, { title: 'Application Management' }] },
      }}
      extra={[
        <Button key="refresh" icon={<RefreshCw />} onClick={fetchData} loading={fetching}>

          Refresh
        </Button>,
        <Button key="create" type="primary" icon={<Plus />} onClick={handleCreate}>

          Create{activeTab === 'applications' ? 'Application' : 'Microservices'}
        </Button>,
      ]}
    >
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        type="card"
        items={[
          {
            key: 'applications',
            label: 'Application Systems',
            children: <Table columns={appColumns} dataSource={applications} rowKey="id" loading={fetching} />,
          },
          {
            key: 'microservices',
            label: 'Microservices',
            children: <Table columns={msColumns} dataSource={microservices} rowKey="id" loading={fetching} />,
          },
        ]}
      />

      <Modal
        title={modalType === 'application' ? (editingRecord ? 'Edit application' : 'New application') : (editingRecord ? 'Edit microservices' : 'Create a new microservice')}
        open={isModalVisible}
        onOk={handleOk}
        onCancel={() => {
          setIsModalVisible(false);
          setEditingRecord(null);
        }}
        confirmLoading={actionLoading}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true, message: 'Please enter a name' }]}>
            <Input placeholder="Please enter a name" />
          </Form.Item>
          <Form.Item name="code" label="Code" rules={[{ required: true, message: 'Please enter code' }]}>
            <Input placeholder="Please enter code" />
          </Form.Item>

          {modalType === 'application' ? (
            <>
              <Form.Item name="projectId" label="Project">
                <Select placeholder="Please select an item" options={projectOptions} />
              </Form.Item>
              <Form.Item name="type" label="Application type">
                <Select
                  placeholder="Please select type"
                  options={[
                    { label: 'Web Application', value: 'web' },
                    { label: 'Mobile App', value: 'mobile' },
                    { label: 'Backend service', value: 'backend' },
                  ]}
                />
              </Form.Item>
            </>
          ) : (
            <>
              <Form.Item
                name="applicationId"
                label="Belonging application"
                rules={[{ required: true, message: 'Please select the application' }]}
              >
                <Select
                  placeholder="Please select an application"
                  options={applications.map(app => ({
                    label: app.name,
                    value: app.id,
                  }))}
                />
              </Form.Item>
              <div className="grid grid-cols-2 gap-4">
                <Form.Item name="language" label="Development Language">
                  <Select
                    placeholder="Please select language"
                    options={[
                      { label: 'Go', value: 'go' },
                      { label: 'Java', value: 'java' },
                      { label: 'Python', value: 'python' },
                      { label: 'Node.js', value: 'nodejs' },
                    ]}
                  />
                </Form.Item>
                <Form.Item name="framework" label="Frame">
                  <Input placeholder="Such as: Gin, Spring Boot" />
                </Form.Item>
              </div>
            </>
          )}
        </Form>
      </Modal>
    </PageContainer>
  );
}
