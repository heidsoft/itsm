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
  Avatar,
  Tooltip,
  message,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  UserOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { PageContainer } from '@/app/components/PageContainer';
import { teamService, Team } from '@/lib/services/team-service';

import { UserApi } from '@/lib/api/user-api';
import { useI18n } from '@/lib/i18n';

export default function TeamsPage() {
  const { t } = useI18n();
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [teams, setTeams] = useState<Team[]>([]);
  const [fetching, setFetching] = useState(false);
  const [users, setUsers] = useState<{ label: string; value: number }[]>([]);

  const fetchUsers = async () => {
    try {
      const response = await UserApi.getUsers({ page: 1, page_size: 100 });
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

  const fetchTeams = async () => {
    setFetching(true);
    try {
      const data = await teamService.listTeams();
      setTeams(data);
    } catch (error) {
      console.error('Failed to fetch teams:', error);
      message.error(t('common.getFailed'));
    } finally {
      setFetching(false);
    }
  };

  useEffect(() => {
    fetchTeams();
    fetchUsers();
  }, []);

  const columns = [
    {
      title: t('enterprise.teams.teamName'),
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => <span className="font-medium">{text}</span>,
    },
    {
      title: t('enterprise.teams.teamCode'),
      dataIndex: 'code',
      key: 'code',
      render: (text: string) => <Tag>{text}</Tag>,
    },
    {
      title: t('enterprise.teams.manager'),
      dataIndex: 'manager_id',
      key: 'manager',
      render: (text: number) => <span>{text ? `用户ID: ${text}` : '-'}</span>,
    },
    {
      title: t('enterprise.teams.members'),
      key: 'members',
      render: (_: unknown, record: Team) => {
        const members = record.edges?.users || [];
        return (
          <Avatar.Group maxCount={3}>
            {members.map(member => (
              <Tooltip key={member.id} title={member.name || member.username}>
                <Avatar style={{ backgroundColor: '#87d068' }} icon={<UserOutlined />}>
                  {(member.name || member.username || '?')[0].toUpperCase()}
                </Avatar>
              </Tooltip>
            ))}
          </Avatar.Group>
        );
      },
    },
    {
      title: t('common.description'),
      dataIndex: 'description',
      key: 'description',
    },
    {
      title: t('common.action'),
      key: 'action',
      render: (_: unknown, record: Team) => (
        <Space size="middle">
          <Button type="text" icon={<EditOutlined />} onClick={() => handleEdit(record)} />
          <Button
            type="text"
            danger
            icon={<DeleteOutlined />}
            onClick={() => handleDelete(record)}
          />
        </Space>
      ),
    },
  ];

  const handleEdit = (record: Team) => {
    form.setFieldsValue(record);
    setIsModalVisible(true);
  };

  const handleDelete = (record: Team) => {
    Modal.confirm({
      title: t('common.confirmDelete'),
      content: t('enterprise.teams.deleteConfirm', { name: record.name }),
      onOk: async () => {
        try {
          await teamService.deleteTeam(record.id);
          message.success(t('common.deleteSuccess'));
          fetchTeams();
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

      await teamService.createTeam(values);

      message.success(t('common.saveSuccess'));
      setIsModalVisible(false);
      form.resetFields();
      fetchTeams();
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
        title: t('enterprise.teams.title'),
        breadcrumb: {
          items: [
            { title: t('common.back') },
            { title: '企业管理' },
            { title: t('enterprise.teams.title') },
          ],
        },
      }}
      extra={[
        <Button key="refresh" icon={<SyncOutlined />} onClick={fetchTeams} loading={fetching}>
          {t('common.refresh')}
        </Button>,
        <Button
          key="create"
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => {
            form.resetFields();
            setIsModalVisible(true);
          }}
        >
          {t('enterprise.teams.create')}
        </Button>,
      ]}
    >
      <Table columns={columns} dataSource={teams} rowKey="id" loading={fetching} />

      <Modal
        title={t('enterprise.teams.createOrEdit')}
        open={isModalVisible}
        onOk={handleOk}
        onCancel={() => setIsModalVisible(false)}
        confirmLoading={loading}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label={t('enterprise.teams.teamName')}
            rules={[
              {
                required: true,
                message: t('common.inputPlaceholder') + t('enterprise.teams.teamName'),
              },
            ]}
          >
            <Input placeholder={t('common.inputPlaceholder') + t('enterprise.teams.teamName')} />
          </Form.Item>
          <Form.Item
            name="code"
            label={t('enterprise.teams.teamCode')}
            rules={[
              {
                required: true,
                message: t('common.inputPlaceholder') + t('enterprise.teams.teamCode'),
              },
            ]}
          >
            <Input placeholder={t('common.inputPlaceholder') + t('enterprise.teams.teamCode')} />
          </Form.Item>
          <Form.Item name="manager_id" label={t('enterprise.teams.manager')}>
            <Select
              placeholder={t('common.selectPlaceholder') + t('enterprise.teams.manager')}
              options={users}
              showSearch
              filterOption={(input, option) =>
                (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
              }
            />
          </Form.Item>
          <Form.Item name="members" label={t('enterprise.teams.members')}>
            <Select
              mode="multiple"
              placeholder={t('common.selectPlaceholder') + t('enterprise.teams.members')}
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
