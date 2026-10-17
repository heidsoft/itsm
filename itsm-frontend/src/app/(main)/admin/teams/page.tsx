'use client';

import React, { useCallback, useEffect, useRef, useState } from 'react';
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
  Spin,
} from 'antd';
import { Plus, Edit, Trash2, Users, RefreshCw, User, Search, UserPlus, X } from 'lucide-react';
import type { ColumnsType } from 'antd/es/table';
import type { Team, TeamMember, CreateTeamRequest } from '@/lib/services/team-service';
import { teamService } from '@/lib/services/team-service';
import { UserApi } from '@/lib/api/user-api';
import { useI18n } from '@/lib/i18n/useI18n';

const { Title, Text } = Typography;
const { TextArea } = Input;

const PAGE_SIZE_OPTIONS = ['10', '20', '50'];

export default function TeamManagement() {
  const { t } = useI18n();
  const { message } = App.useApp();
  const [teams, setTeams] = useState<Team[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchTerm, setSearchTerm] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [fetching, setFetching] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [selectedTeam, setSelectedTeam] = useState<Team | null>(null);
  const [form] = Form.useForm();
  const [users, setUsers] = useState<{ label: string; value: number }[]>([]);

  // 成员管理
  const [memberTeam, setMemberTeam] = useState<Team | null>(null);
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [memberLoading, setMemberLoading] = useState(false);
  const [pendingUserId, setPendingUserId] = useState<number>();

  // 分页/搜索连续变化时，慢响应不得覆盖新条件的结果。
  const requestIdRef = useRef(0);

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(searchTerm.trim()), 300);
    return () => clearTimeout(timer);
  }, [searchTerm]);

  const loadTeams = useCallback(async () => {
    const requestId = ++requestIdRef.current;
    setFetching(true);
    try {
      const data = await teamService.listTeams({
        page,
        pageSize,
        search: debouncedSearch || undefined,
      });
      if (requestId !== requestIdRef.current) return;
      setTeams(data.items);
      setTotal(data.total);
      setLoadError(null);
    } catch (error) {
      if (requestId !== requestIdRef.current) return;
      console.error('Failed to load teams:', error);
      setLoadError(error instanceof Error ? error.message : t('teamsPage.messages.loadFailed'));
    } finally {
      if (requestId === requestIdRef.current) setFetching(false);
    }
  }, [page, pageSize, debouncedSearch, t]);

  useEffect(() => {
    loadTeams();
  }, [loadTeams]);

  useEffect(() => {
    let cancelled = false;
    UserApi.getUsers({ page: 1, pageSize: 100 })
      .then(response => {
        if (cancelled) return;
        setUsers(
          response.users.map(user => ({
            label: user.name || user.username,
            value: user.id,
          }))
        );
      })
      .catch(error => {
        console.error('Failed to load users:', error);
        if (!cancelled) setUsers([]);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const openMemberModal = (team: Team) => {
    setMemberTeam(team);
    setMembers(team.members);
    setPendingUserId(undefined);
  };

  const closeMemberModal = () => {
    setMemberTeam(null);
    setMembers([]);
    setPendingUserId(undefined);
  };

  const handleAddMember = async () => {
    if (!memberTeam || pendingUserId === undefined) return;
    setMemberLoading(true);
    try {
      const freshMembers = await teamService.addMember(memberTeam.id, pendingUserId);
      setMembers(freshMembers);
      setPendingUserId(undefined);
      message.success(t('teamsPage.messages.addMemberSuccess'));
      loadTeams();
    } catch (error) {
      console.error('Failed to add team member:', error);
      message.error(error instanceof Error ? error.message : t('teamsPage.messages.addMemberFailed'));
    } finally {
      setMemberLoading(false);
    }
  };

  const handleRemoveMember = async (userId: number) => {
    if (!memberTeam) return;
    setMemberLoading(true);
    try {
      const freshMembers = await teamService.removeMember(memberTeam.id, userId);
      setMembers(freshMembers);
      message.success(t('teamsPage.messages.removeMemberSuccess'));
      loadTeams();
    } catch (error) {
      console.error('Failed to remove team member:', error);
      message.error(
        error instanceof Error ? error.message : t('teamsPage.messages.removeMemberFailed')
      );
    } finally {
      setMemberLoading(false);
    }
  };

  const handleSave = async () => {
    const values = await form.validateFields().catch(() => undefined);
    if (!values) return;
    setSaving(true);
    try {
      if (selectedTeam) {
        // 表单展示全部字段，按全量更新提交：清空负责人要显式传 0，否则后端无法与"未传"区分。
        await teamService.updateTeam(selectedTeam.id, { ...values, managerId: values.managerId ?? 0 });
        message.success(t('teamsPage.messages.updateSuccess'));
      } else {
        await teamService.createTeam(values as CreateTeamRequest);
        message.success(t('teamsPage.messages.createSuccess'));
      }
      setShowModal(false);
      form.resetFields();
      setSelectedTeam(null);
      loadTeams();
    } catch (error) {
      console.error('Failed to save team:', error);
      message.error(error instanceof Error ? error.message : t('teamsPage.messages.saveFailed'));
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await teamService.deleteTeam(id);
      message.success(t('teamsPage.messages.deleteSuccess'));
      loadTeams();
    } catch (error) {
      console.error('Failed to delete team:', error);
      message.error(error instanceof Error ? error.message : t('teamsPage.messages.deleteFailed'));
    }
  };

  const handleEdit = (record: Team) => {
    setSelectedTeam(record);
    form.setFieldsValue({
      name: record.name,
      code: record.code,
      description: record.description,
      managerId: record.managerId || undefined,
    });
    setShowModal(true);
  };

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
      dataIndex: 'managerName',
      key: 'manager',
      render: (managerName: string) => <span>{managerName || '-'}</span>,
    },
    {
      title: t('teamsPage.members'),
      key: 'members',
      render: (_: unknown, record: Team) => (
        <Space size="small">
          <Avatar.Group max={{ count: 3 }} size="small">
            {record.members.map(member => (
              <Tooltip key={member.id} title={member.name || member.username}>
                <Avatar style={{ backgroundColor: '#87d068' }} icon={<User />}>
                  {(member.name || member.username || '?')[0].toUpperCase()}
                </Avatar>
              </Tooltip>
            ))}
          </Avatar.Group>
          <Text type="secondary">{record.members.length}</Text>
          <Button type="link" size="small" onClick={() => openMemberModal(record)}>
            {t('teamsPage.manageMembers')}
          </Button>
        </Space>
      ),
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
      width: 120,
      render: (_: unknown, record: Team) => (
        <Space size="small">
          <Button
            aria-label={t('teamsPage.edit')}
            type="text"
            icon={<Edit size={16} />}
            onClick={() => handleEdit(record)}
          />
          <Popconfirm
            title={t('common.confirmDelete')}
            description={t('teamsPage.confirmDeleteContent', { name: record.name })}
            onConfirm={() => handleDelete(record.id)}
            okText={t('common.confirm')}
            cancelText={t('common.cancel')}
          >
            <Button aria-label={t('common.delete')} type="text" danger icon={<Trash2 size={16} />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const candidateUsers = users.filter(user => !members.some(member => member.id === user.value));

  return (
    <div className="space-y-6">
      <div>
        <Title level={2} className="!mb-2">
          <Users className="mr-2" />
          {t('teamsPage.title')}
        </Title>
        <Text type="secondary">{t('teamsPage.description')}</Text>
      </div>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={8}>
          <Card className="enterprise-card">
            <Statistic title={t('teamsPage.stats.total')} value={total} prefix={<Users />} />
          </Card>
        </Col>
      </Row>

      <Card>
        <Space wrap>
          <Input
            allowClear
            placeholder={t('teamsPage.searchPlaceholder')}
            prefix={<Search size={16} />}
            value={searchTerm}
            onChange={event => {
              setSearchTerm(event.target.value);
              setPage(1);
            }}
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
          <Button icon={<RefreshCw size={16} />} onClick={loadTeams} loading={fetching}>
            {t('common.refresh')}
          </Button>
        </Space>
      </Card>

      {loadError && (
        <Card>
          <Empty description={loadError}>
            <Button type="primary" onClick={loadTeams}>
              {t('common.retry')}
            </Button>
          </Empty>
        </Card>
      )}

      {!loadError && (
        <Card className="enterprise-card">
          <Table
            columns={columns}
            dataSource={teams}
            rowKey="id"
            loading={fetching}
            scroll={{ x: 900 }}
            locale={{
              emptyText: (
                <Empty
                  description={
                    debouncedSearch ? t('teamsPage.empty.searchEmpty') : t('teamsPage.empty.noData')
                  }
                >
                  <Button type="primary" onClick={() => setShowModal(true)}>
                    {t('teamsPage.create')}
                  </Button>
                </Empty>
              ),
            }}
            pagination={{
              current: page,
              pageSize,
              total,
              showSizeChanger: true,
              pageSizeOptions: PAGE_SIZE_OPTIONS,
              showTotal: value => t('common.totalLabel', { total: value }),
              onChange: (nextPage, nextPageSize) => {
                setPage(nextPage);
                setPageSize(nextPageSize);
              },
            }}
            className="enterprise-table"
          />
        </Card>
      )}

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
        confirmLoading={saving}
        okText={t('common.save')}
        cancelText={t('common.cancel')}
      >
        <Form form={form} layout="vertical" className="mt-4">
          <Form.Item
            label={t('teamsPage.teamName')}
            name="name"
            rules={[{ required: true, message: t('teamsPage.form.requiredName') }]}
          >
            <Input placeholder={t('teamsPage.form.namePlaceholder')} />
          </Form.Item>
          <Form.Item
            label={t('teamsPage.teamCode')}
            name="code"
            rules={[{ required: true, message: t('teamsPage.form.requiredCode') }]}
          >
            <Input placeholder={t('teamsPage.form.codePlaceholder')} />
          </Form.Item>
          <Form.Item label={t('teamsPage.manager')} name="managerId">
            <Select
              placeholder={t('teamsPage.form.managerPlaceholder')}
              options={users}
              allowClear
              showSearch
              optionFilterProp="label"
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Form.Item label={t('common.description')} name="description">
            <TextArea rows={3} placeholder={t('teamsPage.form.descriptionPlaceholder')} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={
          <span>
            <Users className="w-4 h-4 mr-2" />
            {t('teamsPage.manageMembers')} · {memberTeam?.name}
          </span>
        }
        open={Boolean(memberTeam)}
        onCancel={closeMemberModal}
        footer={<Button onClick={closeMemberModal}>{t('common.close')}</Button>}
        width={560}
      >
        <Text type="secondary">{t('teamsPage.memberRule')}</Text>
        <Space.Compact className="mt-3 w-full">
          <Select
            className="flex-1"
            placeholder={t('teamsPage.selectMemberPlaceholder')}
            value={pendingUserId}
            onChange={value => setPendingUserId(value)}
            options={candidateUsers}
            showSearch
            optionFilterProp="label"
            disabled={memberLoading}
          />
          <Button
            type="primary"
            icon={<UserPlus size={16} />}
            onClick={handleAddMember}
            loading={memberLoading}
            disabled={pendingUserId === undefined}
          >
            {t('teamsPage.addMember')}
          </Button>
        </Space.Compact>

        {memberLoading && members.length === 0 ? (
          <div className="py-6 text-center">
            <Spin />
          </div>
        ) : members.length === 0 ? (
          <Empty className="mt-4" description={t('teamsPage.empty.noMembers')} />
        ) : (
          <ul className="mt-3 divide-y">
            {members.map(member => (
              <li key={member.id} className="flex items-center gap-3 py-2">
                <Avatar style={{ backgroundColor: '#87d068' }} icon={<User />}>
                  {(member.name || member.username || '?')[0].toUpperCase()}
                </Avatar>
                <div className="min-w-0 flex-1">
                  <Text className="block truncate">{member.name || member.username}</Text>
                  <Text type="secondary" className="block truncate">
                    {member.email}
                  </Text>
                </div>
                <Popconfirm
                  title={t('teamsPage.confirmRemoveMember', {
                    name: member.name || member.username,
                    team: memberTeam?.name ?? '',
                  })}
                  description={t('teamsPage.removeMemberRule')}
                  onConfirm={() => handleRemoveMember(member.id)}
                  okText={t('common.confirm')}
                  cancelText={t('common.cancel')}
                >
                  <Button
                    aria-label={t('teamsPage.removeMember')}
                    type="text"
                    danger
                    icon={<X size={16} />}
                    loading={memberLoading}
                  />
                </Popconfirm>
              </li>
            ))}
          </ul>
        )}
      </Modal>
    </div>
  );
}
