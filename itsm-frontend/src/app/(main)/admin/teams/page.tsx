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
  App,
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
import type { Team, CreateTeamRequest } from '@/lib/services/team-service';
import { teamService } from '@/lib/services/team-service';
import { UserApi } from '@/lib/api/user-api';
import { useI18n } from '@/lib/i18n/useI18n';

const { Title, Text } = Typography;
const { TextArea } = Input;

export default function TeamManagement() {
  const { t } = useI18n();
  const { message } = App.useApp();
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
      message.error(t('teamsPage.messages.loadFailed'));
    } finally {
      setFetching(false);
    }
  }, []);

  // 加载用户列表
  const loadUsers = useCallback(async () => {
    try {
      const response = await UserApi.getUsers({ page: 1, pageSize: 100 });
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
        message.success(t('teamsPage.messages.updateSuccess'));
      } else {
        // 创建
        await teamService.createTeam(values as CreateTeamRequest);
        message.success(t('teamsPage.messages.createSuccess'));
      }

      setShowModal(false);
      form.resetFields();
      setSelectedTeam(null);
      loadTeams();
    } catch (error) {
      console.error('Failed to save team:', error);
      message.error(t('teamsPage.messages.saveFailed'));
    } finally {
      setLoading(false);
    }
  };

  // 处理删除
  const handleDelete = async (id: number) => {
    try {
      await teamService.deleteTeam(id);
      message.success(t('teamsPage.messages.deleteSuccess'));
      loadTeams();
    } catch (error) {
      console.error('Failed to delete team:', error);
      message.error(t('teamsPage.messages.deleteFailed'));
    }
  };

  // 处理编辑
  const handleEdit = (record: Team) => {
    setSelectedTeam(record);
    form.setFieldsValue({
      name: record.name,
      code: record.code,
      description: record.description,
      managerId: record.managerId,
    });
    setShowModal(true);
  };

  // 表格列定义
  const columns: ColumnsType<Team> = [
    {
      title: t('teamsPage.teamName'),
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => <span className="font-medium">{text}</span>,
    },
    {
      title: t('teamsPage.teamCode'),
      dataIndex: 'code',
      key: 'code',
      render: (text: string) => <Tag color="green">{text}</Tag>,
    },
    {
      title: t('teamsPage.manager'),
      dataIndex:'managerId',
      key: 'manager',
      render: (managerId: number) => {
        const user = users.find(u => u.value === managerId);
        return <span>{user?.label || '-'}</span>;
      },
    },
    {
      title: t('teamsPage.members'),
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
      title: t('common.description'),
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: t('common.action'),
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
            title={t("common.confirmDelete")}
            description={t("teamsPage.confirmDeleteContent", { name: record.name })}
            onConfirm={() => handleDelete(record.id)}
            okText={t("common.confirm")}
            cancelText={t("common.cancel")}
          >
            <Button type="text" danger icon={<Trash2 size={16} />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <Title level={2} className="!mb-2">
          <Users className="mr-2" />
          {t('teamsPage.title')}
        </Title>
        <Text type="secondary">{t('teamsPage.description')}</Text>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={8}>
          <Card className="enterprise-card">
            <Statistic
              title={t("teamsPage.stats.total")}
              value={stats.totalTeams}
              prefix={<Users />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <Card className="enterprise-card">
            <Statistic
              title={t("teamsPage.stats.totalMembers")}
              value={stats.totalMembers}
              prefix={<Users />}
            />
          </Card>
        </Col>
      </Row>

      {/* 操作栏 */}
      <Card>
        <Space wrap>
          <Input
            allowClear
            placeholder={t("teamsPage.searchPlaceholder")}
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
            {t('teamsPage.create')}
          </Button>
          <Button
            icon={<RefreshCw size={16} />}
            onClick={() => loadTeams()}
            loading={fetching}
          >
            {t('common.refresh')}
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
              <Empty description={searchTerm ? t('teamsPage.empty.searchEmpty') : t('teamsPage.empty.noData')}>
                <Button type="primary" onClick={() => setShowModal(true)}>
                  {t('teamsPage.create')}
                </Button>
              </Empty>
            ),
          }}
          pagination={{
            total: filteredTeams.length,
            pageSize: 10,
            showSizeChanger: true,
            showTotal: total => t('common.totalLabel', { total }),
          }}
          className="enterprise-table"
        />
      </Card>

      {/* 编辑模态框 */}
      <Modal
        title={
          <span>
            <Edit className="w-4 h-4 mr-2" />
            {selectedTeam ? t('teamsPage.edit') : t('teamsPage.create')}
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
        okText={t("common.save")}
        cancelText={t("common.cancel")}
      >
        <Form form={form} layout="vertical" className="mt-4">
          <Form.Item
            label={t("teamsPage.teamName")}
            name="name"
            rules={[{ required: true, message: t('teamsPage.form.requiredName') }]}
          >
            <Input placeholder={t("teamsPage.form.namePlaceholder")} />
          </Form.Item>
          <Form.Item
            label={t("teamsPage.teamCode")}
            name="code"
            rules={[{ required: true, message: t('teamsPage.form.requiredCode') }]}
          >
            <Input placeholder={t("teamsPage.form.codePlaceholder")} />
          </Form.Item>
          <Form.Item
            label={t("teamsPage.manager")}
            name="manager_id"
          >
            <Select
              placeholder={t("teamsPage.form.managerPlaceholder")}
              options={users}
              allowClear
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Form.Item
            label={t("common.description")}
            name="description"
          >
            <TextArea rows={3} placeholder={t("teamsPage.form.descriptionPlaceholder")} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
