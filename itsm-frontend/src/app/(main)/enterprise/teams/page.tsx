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
import { PageContainer } from '@ant-design/pro-components';
import { teamService, Team } from '@/lib/services/team-service';

export default function TeamsPage() {
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [teams, setTeams] = useState<Team[]>([]);
  const [fetching, setFetching] = useState(false);

  const fetchTeams = async () => {
    setFetching(true);
    try {
      const data = await teamService.listTeams();
      setTeams(data);
    } catch (error) {
      console.error('Failed to fetch teams:', error);
      message.error('获取团队列表失败');
    } finally {
      setFetching(false);
    }
  };

  useEffect(() => {
    fetchTeams();
  }, []);

  const columns = [
    {
      title: '团队名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => <span className='font-medium'>{text}</span>,
    },
    {
      title: '团队代码',
      dataIndex: 'code',
      key: 'code',
      render: (text: string) => <Tag>{text}</Tag>,
    },
    {
      title: '负责人',
      dataIndex: 'manager_id',
      key: 'manager',
      render: (text: number) => <span>{text ? `用户ID: ${text}` : '-'}</span>,
    },
    {
      title: '成员',
      key: 'members',
      render: (_: any, record: Team) => {
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
      title: '描述',
      dataIndex: 'description',
      key: 'description',
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: Team) => (
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

  const handleEdit = (record: Team) => {
    form.setFieldsValue(record);
    setIsModalVisible(true);
  };

  const handleDelete = (record: Team) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除团队 "${record.name}" 吗？`,
      onOk: async () => {
        try {
          await teamService.deleteTeam(record.id);
          message.success('删除成功');
          fetchTeams();
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

      await teamService.createTeam(values);

      message.success('保存成功');
      setIsModalVisible(false);
      form.resetFields();
      fetchTeams();
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
        title: '团队管理',
        breadcrumb: { items: [{ title: '首页' }, { title: '企业管理' }, { title: '团队管理' }] },
      }}
      extra={[
        <Button key='refresh' icon={<SyncOutlined />} onClick={fetchTeams} loading={fetching}>
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
          新建团队
        </Button>,
      ]}
    >
      <Table columns={columns} dataSource={teams} rowKey='id' loading={fetching} />

      <Modal
        title='新建/编辑团队'
        open={isModalVisible}
        onOk={handleOk}
        onCancel={() => setIsModalVisible(false)}
        confirmLoading={loading}
      >
        <Form form={form} layout='vertical'>
          <Form.Item
            name='name'
            label='团队名称'
            rules={[{ required: true, message: '请输入团队名称' }]}
          >
            <Input placeholder='请输入团队名称' />
          </Form.Item>
          <Form.Item
            name='code'
            label='团队代码'
            rules={[{ required: true, message: '请输入团队代码' }]}
          >
            <Input placeholder='请输入团队代码' />
          </Form.Item>
          <Form.Item name='manager_id' label='负责人'>
            {/* TODO: Load users */}
            <Select placeholder='请选择负责人'>
              <Select.Option value={1}>管理员</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name='members' label='成员'>
            {/* TODO: Load users */}
            <Select mode='multiple' placeholder='请选择成员'>
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
