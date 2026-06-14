'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Typography,
  Modal,
  Form,
  Input,
  Select,
  Avatar,
  Tooltip,
  message,
  Popconfirm,
  Tag,
  Row,
  Col,
  Statistic,
  Empty,
} from 'antd';
import {
  Plus,
  Edit,
  Trash2,
  Users,
  RefreshCw,
  User,
  Search,
} from 'lucide-react';
import type { ColumnsType } from 'antd/es/table';
import { teamService, Team, CreateTeamRequest } from '@/lib/services/team-service';
import { UserApi } from '@/lib/api/user-api';

const { Title, Text } = Typography;
const { TextArea } = Input;

export default function TeamManagement() {
  const [teams, setTeams] = useState<Team[]>([]);
  const [loading, setLoading] = useState(false);
  const [fetching, setFetching] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [selectedTeam, setSelectedTeam] = useState<Team | null>(null);
  const [form] = Form.useForm();
  const [users, setUsers] = useState<{ label: string; value: number }[]>([]);
  const [searchTerm, setSearchTerm] = useState('');

  // 加载团队数据
  const loadTeams = useCallback(async () => {
    setFetching(true);
    try {
      const data = await teamService.listTeams();
      setTeams(data);
    } catch (error) {
      console.error('Failed to load teams:', error);
      message.error('加载团队数据失败');
    } finally {
      setFetching(false);
    }
  }, []);

  // 加载用户列表
  const loadUsers = useCallback(async () => {
    try {
      const response = await UserApi.getUsers({ page: 1, page_size: 100 });
      setUsers(
        response.users.map(user => ({
          label: user.name || user.username,
          value: user.id,
        }))
      );
    } catch (error) {
      console.error('Failed to load users:', error);
    }
  }, []);

  // 初始化加载
  useEffect(() => {
    loadTeams();
    loadUsers();
  }, [loadTeams, loadUsers]);

  // 统计信息
  const stats = {
    totalTeams: teams.length,
    totalMembers: teams.reduce((sum, team) => sum + (team.edges?.users?.length || 0), 0),
  };

  const filteredTeams = teams.filter(team => {
    const keyword = searchTerm.trim().toLowerCase();
    if (!keyword) return true;
    return (
      team.name.toLowerCase().includes(keyword) ||
      team.code.toLowerCase().includes(keyword) ||
      (team.description || '').toLowerCase().includes(keyword)
    );
  });

  // 处理保存
  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      if (selectedTeam) {
        // 更新
        await teamService.updateTeam(selectedTeam.id, values);
        message.success('团队更新成功');
      } else {
        // 创建
        await teamService.createTeam(values as CreateTeamRequest);
        message.success('团队创建成功');
      }

      setShowModal(false);
      form.resetFields();
      setSelectedTeam(null);
      loadTeams();
    } catch (error) {
      console.error('Failed to save team:', error);
      message.error('保存团队失败');
    } finally {
      setLoading(false);
    }
  };

  // 处理删除
  const handleDelete = async (id: number) => {
    try {
      await teamService.deleteTeam(id);
      message.success('团队删除成功');
      loadTeams();
    } catch (error) {
      console.error('Failed to delete team:', error);
      message.error('删除团队失败');
    }
  };

  // 处理编辑
  const handleEdit = (record: Team) => {
    setSelectedTeam(record);
    form.setFieldsValue({
      name: record.name,
      code: record.code,
      description: record.description,
      manager_id: record.manager_id,
    });
    setShowModal(true);
  };

  // 表格列定义
  const columns: ColumnsType<Team> = [
    {
      title: '团队名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => <span className="font-medium">{text}</span>,
    },
    {
      title: '团队编码',
      dataIndex: 'code',
      key: 'code',
      render: (text: string) => <Tag color="green">{text}</Tag>,
    },
    {
      title: '团队经理',
      dataIndex: 'manager_id',
      key: 'manager',
      render: (managerId: number) => {
        const user = users.find(u => u.value === managerId);
        return <span>{user?.label || '-'}</span>;
      },
    },
    {
      title: '成员',
      key: 'members',
      render: (_: unknown, record: Team) => {
        const members = record.edges?.users || [];
        return (
          <Avatar.Group max={{ count: 3 }} size="small">
            {members.map(member => (
              <Tooltip key={member.id} title={member.name || member.username}>
                <Avatar style={{ backgroundColor: '#87d068' }} icon={<User />}>
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
      ellipsis: true,
    },
    {
      title: '操作',
      key: 'actions',
      width: 150,
      render: (_: unknown, record: Team) => (
        <Space size="small">
          <Button
            type="text"
            icon={<Edit size={16} />}
            onClick={() => handleEdit(record)}
          />
          <Popconfirm
            title="确认删除"
            description={`确定要删除团队"${record.name}"吗？`}
            onConfirm={() => handleDelete(record.id)}
            okText="确认"
            cancelText="取消"
          >
            <Button type="text" danger icon={<Trash2 size={16} />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <div className="mb-6">
        <Title level={2} className="!mb-2">
          <Users className="mr-2" />
          团队管理
        </Title>
        <Text type="secondary">管理团队和团队成员</Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={8}>
          <Card className="enterprise-card">
            <Statistic
              title="团队总数"
              value={stats.totalTeams}
              prefix={<Users />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <Card className="enterprise-card">
            <Statistic
              title="团队成员总数"
              value={stats.totalMembers}
              prefix={<Users />}
            />
          </Card>
        </Col>
      </Row>

      {/* 操作栏 */}
      <Card className="mb-6">
        <Space wrap>
          <Input
            allowClear
            placeholder="搜索团队名称、编码或描述"
            prefix={<Search size={16} />}
            value={searchTerm}
            onChange={event => setSearchTerm(event.target.value)}
            style={{ width: 280 }}
          />
          <Button
            type="primary"
            icon={<Plus size={16} />}
            onClick={() => {
              setSelectedTeam(null);
              form.resetFields();
              setShowModal(true);
            }}
          >
            新建团队
          </Button>
          <Button
            icon={<RefreshCw size={16} />}
            onClick={() => loadTeams()}
            loading={fetching}
          >
            刷新
          </Button>
        </Space>
      </Card>

      {/* 团队列表 */}
      <Card className="enterprise-card">
        <Table
          columns={columns}
          dataSource={filteredTeams}
          rowKey="id"
          loading={fetching}
          scroll={{ x: 820 }}
          locale={{
            emptyText: (
              <Empty description={searchTerm ? '没有匹配的团队' : '暂无团队'}>
                <Button type="primary" onClick={() => setShowModal(true)}>
                  新建团队
                </Button>
              </Empty>
            ),
          }}
          pagination={{
            total: filteredTeams.length,
            pageSize: 10,
            showSizeChanger: true,
            showTotal: total => `共 ${total} 条记录`,
          }}
          className="enterprise-table"
        />
      </Card>

      {/* 编辑模态框 */}
      <Modal
        title={
          <span>
            <Edit className="w-4 h-4 mr-2" />
            {selectedTeam ? '编辑团队' : '新建团队'}
          </span>
        }
        open={showModal}
        onOk={handleSave}
        onCancel={() => {
          setShowModal(false);
          setSelectedTeam(null);
          form.resetFields();
        }}
        width={600}
        confirmLoading={loading}
        okText="保存"
        cancelText="取消"
      >
        <Form form={form} layout="vertical" className="mt-4">
          <Form.Item
            label="团队名称"
            name="name"
            rules={[{ required: true, message: '请输入团队名称' }]}
          >
            <Input placeholder="请输入团队名称" />
          </Form.Item>
          <Form.Item
            label="团队编码"
            name="code"
            rules={[{ required: true, message: '请输入团队编码' }]}
          >
            <Input placeholder="请输入团队编码（如：TEAM001）" />
          </Form.Item>
          <Form.Item
            label="团队经理"
            name="manager_id"
          >
            <Select
              placeholder="选择团队经理"
              options={users}
              allowClear
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Form.Item
            label="描述"
            name="description"
          >
            <TextArea rows={3} placeholder="请输入团队描述" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
