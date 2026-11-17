'use client';

import React, { useEffect, useState } from 'react';
import {
  App,
  Button,
  Card,
  Col,
  Empty,
  Form,
  Input,
  Modal,
  Row,
  Select,
  Space,
  Statistic,
  Table,
  Tag,
  Typography,
  Avatar,
  List,
  Transfer,
  Badge,
  theme,
} from 'antd';
import type { TablePaginationConfig } from 'antd';
import { Edit, Plus, Search, Trash2, UserPlus, Users, User as UserIcon, X, Check } from 'lucide-react';
import { GroupAPI, type Group } from '@/lib/api/group-api';
import { UserApi, type User, PRIMARY_ROLE_OPTIONS, PRIMARY_ROLE_LABEL } from '@/lib/api/user-api';
import { useI18n } from '@/lib/i18n/useI18n';
import { BACKEND_MAX_PAGE_SIZE } from '@/lib/api/pagination';
import { formatDateTime } from '@/lib/formatters';

const { Title, Text } = Typography;
const { Search: AntSearch } = Input;

// 成员/候选用户都可能超过单页上限；只取第一页会让 >100 人的组在弹窗里
// 显示不全，保存时还会把未展示的成员误判为「已移除」。
async function fetchAllPages<T>(
  fetchPage: (page: number, pageSize: number) => Promise<{ items: T[]; total: number }>,
): Promise<T[]> {
  const collected: T[] = [];
  let page = 1;
  for (;;) {
    const response = await fetchPage(page, BACKEND_MAX_PAGE_SIZE);
    const items = response.items || [];
    collected.push(...items);
    if (items.length === 0 || collected.length >= response.total) break;
    page += 1;
  }
  return collected;
}

const GroupManagement: React.FC = () => {
  const { token } = theme.useToken();
  const { t } = useI18n();
  const { message, modal } = App.useApp();
  const [form] = Form.useForm();
  const [groups, setGroups] = useState<Group[]>([]);
  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染
  // 「用户组总数 0」的假统计卡与空表格，再被真实数据覆盖。保存动作另用 actionLoading。
  const [listLoading, setListLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [search, setSearch] = useState('');
  const [modalOpen, setModalOpen] = useState(false);
  const [selectedGroup, setSelectedGroup] = useState<Group | null>(null);

  // 成员管理状态
  const [memberModalOpen, setMemberModalOpen] = useState(false);
  const [groupMembers, setGroupMembers] = useState<User[]>([]);
  const [allUsers, setAllUsers] = useState<User[]>([]);
  const [loadingMembers, setLoadingMembers] = useState(false);
  const [savingMembers, setSavingMembers] = useState(false);
  const [selectedUserIds, setSelectedUserIds] = useState<string[]>([]);
  // 成员选择器角色筛选：审批组=从角色拉人组成的窄集合，按角色过滤是核心操作路径
  const [memberRoleFilter, setMemberRoleFilter] = useState<string | undefined>(undefined);

  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0,
  });

  const loadGroups = async () => {
    setListLoading(true);
    try {
      const response = await GroupAPI.getGroups({
        page: pagination.current,
        pageSize: pagination.pageSize,
        search: search || undefined,
      });
      setGroups(response.items);
      setPagination(prev => ({
        ...prev,
        total: response.total,
      }));
    } catch (error) {
      console.error('Failed to load groups:', error);
      message.error(t('groups.messages.loadFailed'));
    } finally {
      setListLoading(false);
    }
  };

  useEffect(() => {
    loadGroups();
  }, [pagination.current, pagination.pageSize, search]);

  const openCreateModal = () => {
    setSelectedGroup(null);
    form.resetFields();
    setModalOpen(true);
  };

  const openEditModal = (group: Group) => {
    setSelectedGroup(group);
    form.setFieldsValue({
      name: group.name,
      description: group.description,
    });
    setModalOpen(true);
  };

  const handleSave = async (values: { name: string; description?: string }) => {
    setActionLoading(true);
    try {
      if (selectedGroup) {
        await GroupAPI.updateGroup(selectedGroup.id, values);
        message.success(t('groups.updateSuccess'));
      } else {
        await GroupAPI.createGroup(values);
        message.success(t('groups.createSuccess'));
      }
      setModalOpen(false);
      setSelectedGroup(null);
      form.resetFields();
      await loadGroups();
    } catch (error) {
      console.error('Failed to save group:', error);
      message.error(t('groups.messages.saveFailed'));
    } finally {
      setActionLoading(false);
    }
  };

  const handleDelete = (group: Group) => {
    modal.confirm({
      title: t('groups.confirmDelete'),
      content: t('groups.confirmDeleteContent', { name: group.name }),
      okText: t('common.delete'),
      okButtonProps: { danger: true },
      cancelText: t('common.cancel'),
      onOk: async () => {
        try {
          await GroupAPI.deleteGroup(group.id);
          message.success(t('groups.deleteSuccess'));
          await loadGroups();
        } catch (error) {
          console.error('Failed to delete group:', error);
          message.error(t('groups.messages.deleteFailed'));
        }
      },
    });
  };

  // 加载组成员
  const loadGroupMembers = async (groupId: number) => {
    setLoadingMembers(true);
    try {
      const members = await fetchAllPages<User>((page, pageSize) =>
        GroupAPI.getMembers(groupId, { page, pageSize }),
      );
      setGroupMembers(members);
      setSelectedUserIds(members.map(m => String(m.id)));
    } catch (error) {
      console.error('Failed to load group members:', error);
      message.error(t('groups.messages.loadMembersFailed'));
    } finally {
      setLoadingMembers(false);
    }
  };

  // 加载所有用户（用于添加成员）
  const loadAllUsers = async () => {
    try {
      const users = await fetchAllPages<User>((page, pageSize) =>
        UserApi.getUsers({ page, pageSize }),
      );
      setAllUsers(users);
    } catch (error) {
      console.error('Failed to load users:', error);
      message.error(t('groups.messages.loadFailed'));
    }
  };

  // 打开成员管理弹窗
  const openMemberModal = async (group: Group) => {
    setSelectedGroup(group);
    setMemberRoleFilter(undefined);
    setMemberModalOpen(true);
    await loadGroupMembers(group.id);
    await loadAllUsers();
  };

  // 保存成员变更
  const handleSaveMembers = async () => {
    if (!selectedGroup) return;

    setSavingMembers(true);
    try {
      const currentMemberIds = groupMembers.map(m => String(m.id));
      const newMemberIds = selectedUserIds;

      // 找出需要添加和移除的成员
      const toAdd = newMemberIds.filter(id => !currentMemberIds.includes(id)).map(Number);
      const toRemove = currentMemberIds.filter(id => !newMemberIds.includes(id)).map(Number);

      // 执行添加
      for (const userId of toAdd) {
        await GroupAPI.addMember(selectedGroup.id, userId);
      }

      // 执行移除
      for (const userId of toRemove) {
        await GroupAPI.removeMember(selectedGroup.id, userId);
      }

      message.success(t('groups.messages.membersUpdated'));
      setMemberModalOpen(false);
      await loadGroups();
    } catch (error) {
      console.error('Failed to save members:', error);
      message.error(t('groups.messages.saveMembersFailed'));
      // 串行增删可能已经部分成功，重拉真实成员列表，避免弹窗停留在脏状态。
      await loadGroupMembers(selectedGroup.id);
    } finally {
      setSavingMembers(false);
    }
  };

  // Transfer 的 targetKeys 就是当前选中集合本身。历史实现与 groupMembers
  // 取交集，导致刚从左侧添加的新成员不在右侧渲染，视觉上「加不进去」。
  const targetKeys = selectedUserIds;

  const selectedIdSet = new Set(selectedUserIds);
  const toUserOption = (u: User) => ({
    key: String(u.id),
    title: u.name || u.username || t('groups.members.userFallback', { id: u.id }),
    description: u.email || u.username || '',
    role: u.role,
  });

  // 角色筛选只作用于左侧「可添加」候选；已选中的成员必须始终留在 dataSource，
  // 否则 Antd Transfer 右侧会渲染空项，保存时也无法正确计算增删差集。
  const filteredUserOptions = [
    ...allUsers
      .filter(u => selectedIdSet.has(String(u.id)) || !memberRoleFilter || u.role === memberRoleFilter)
      .map(toUserOption),
    // 成员可能不在候选用户页里（停用/被过滤），补齐以免右侧丢项。
    ...groupMembers
      .filter(m => !allUsers.some(u => String(u.id) === String(m.id)))
      .map(toUserOption),
  ];

  // 角色标签：显示用户主角色（词表单一源=PRIMARY_ROLE_LABEL）
  const memberRoleTag = (role?: string) => {
    if (!role) return null;
    const label = PRIMARY_ROLE_LABEL[role];
    if (!label) return null;
    return <Tag>{label}</Tag>;
  };

  const handleTableChange = (nextPagination: TablePaginationConfig) => {
    setPagination(prev => ({
      ...prev,
      current: nextPagination.current || 1,
      pageSize: nextPagination.pageSize || prev.pageSize,
    }));
  };

  // 与其他 admin 页面（users/roles 等）保持一致的 Card + Statistic 风格，只展示真实可得的指标
  const statsCards = [
    {
      title: t('groups.stats.total'),
      value: pagination.total,
      icon: <Users size={18} style={{ color: token.colorPrimary }} />,
    },
    {
      title: t('groups.stats.current'),
      value: groups.length,
      icon: <UserPlus size={18} style={{ color: '#52c41a' }} />,
    },
  ];

  const columns = [
    {
      title: t('groups.groupName'),
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: Group) => (
        <Space orientation="vertical" size={0}>
          <Text strong>{text}</Text>
          <Text type="secondary">{record.description || t('common.noData')}</Text>
        </Space>
      ),
    },
    {
      title: t('groups.memberCount'),
      key: 'members',
      width: 120,
      render: (_: unknown, record: Group) => (
        <Badge count={record.memberCount ?? record.members?.length ?? 0} showZero color="blue">
          <Button
            type="link"
            icon={<Users size={16} />}
            onClick={() => openMemberModal(record)}
          >
            {t('groups.actions.members')}
          </Button>
        </Badge>
      ),
    },
    {
      title: t('groups.createdAt'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (value: string) => formatDateTime(value) || '-',
      width: 160,
    },
    {
      title: t('groups.updatedAt'),
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      render: (value: string) => formatDateTime(value) || '-',
      width: 160,
    },
    {
      title: t('common.action'),
      key: 'actions',
      width: 180,
      render: (_: unknown, record: Group) => (
        <Space size="small">
          <Button type="link" size="small" icon={<Users size={14} />} onClick={() => openMemberModal(record)}>
{t('groups.actions.members')}
          </Button>
          <Button type="link" size="small" icon={<Edit size={14} />} onClick={() => openEditModal(record)}>
{t('common.edit')}
          </Button>
          <Button
            type="link"
            size="small"
            danger
            icon={<Trash2 size={14} />}
            onClick={() => handleDelete(record)}
          >
{t('common.delete')}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div>
          <Title level={2} style={{ margin: 0 }}>
{t('groups.title')}
          </Title>
          <Text type="secondary">{t('groups.description')}</Text>
        </div>
        <Button type="primary" icon={<Plus size={16} />} onClick={openCreateModal}>
          {t('groups.create')}
        </Button>
      </div>

      {/* 统计卡必须门控：还没拉到数据时渲染成「用户组总数 0」会被误读成「没有用户组」 */}
      {!listLoading && (
        <Row gutter={16}>
          {statsCards.map(item => (
            <Col span={12} md={8} lg={6} key={item.title}>
              <Card>
                <Statistic title={item.title} value={item.value} prefix={item.icon} />
              </Card>
            </Col>
          ))}
        </Row>
      )}

      <Card>
        <Space wrap className="w-full justify-between">
          <AntSearch
            allowClear
            enterButton
            placeholder={t("groups.searchPlaceholder")}
            style={{ width: 320 }}
            onSearch={value => {
              setSearch(value.trim());
              setPagination(prev => ({ ...prev, current: 1 }));
            }}
          />
          <Button onClick={loadGroups}>{t('common.refresh')}</Button>
        </Space>
      </Card>

      <Card>
        <Table
          columns={columns}
          dataSource={groups}
          rowKey="id"
          loading={listLoading}
          onChange={handleTableChange}
          scroll={{ x: 860 }}
          locale={{
            emptyText: (
              <Empty description={search ? t('groups.empty.searchEmpty') : t('groups.empty.noData')}>
                <Button type="primary" onClick={openCreateModal}>
                  {t('groups.create')}
                </Button>
              </Empty>
            ),
          }}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: total => t('common.totalLabel', { total }),
          }}
        />
      </Card>

      <Modal
        title={selectedGroup ? t('groups.edit') : t('groups.create')}
        open={modalOpen}
        onCancel={() => {
          setModalOpen(false);
          setSelectedGroup(null);
          form.resetFields();
        }}
        footer={null}
        width={600}
      >
        <Form form={form} layout="vertical" onFinish={handleSave}>
          <Form.Item
            name="name"
            label={t("groups.groupName")}
            rules={[
              { required: true, message: t('groups.requiredName') },
              { max: 100, message: t('groups.nameMaxLength') },
            ]}
          >
            <Input placeholder={t('groups.form.namePlaceholder')} />
          </Form.Item>
          <Form.Item
            name="description"
            label={t('groups.groupDescription')}
            rules={[{ max: 500, message: t('groups.form.descriptionMaxLength') }]}
          >
            <Input.TextArea rows={4} placeholder={t('groups.form.descriptionPlaceholder')} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={actionLoading}>
                {selectedGroup ? t('common.save') : t('common.create')}
              </Button>
              <Button
                onClick={() => {
                  setModalOpen(false);
                  setSelectedGroup(null);
                  form.resetFields();
                }}
              >
                {t('common.cancel')}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 成员管理模态框 */}
      <Modal
        title={
          <Space>
            <Users size={18} />
            <span>{t('groups.members.title')} - {selectedGroup?.name}</span>
          </Space>
        }
        open={memberModalOpen}
        onCancel={() => {
          setMemberModalOpen(false);
          setSelectedGroup(null);
          setGroupMembers([]);
          setSelectedUserIds([]);
        }}
        width={700}
        footer={[
          <Button
            key="cancel"
            onClick={() => {
              setMemberModalOpen(false);
              setSelectedGroup(null);
              setGroupMembers([]);
              setSelectedUserIds([]);
            }}
          >
            {t('common.cancel')}
          </Button>,
          <Button
            key="save"
            type="primary"
            loading={savingMembers}
            onClick={handleSaveMembers}
          >
            {t('common.save')}
          </Button>,
        ]}
      >
        <div className="py-4">
          <Text type="secondary" className="mb-4 block">
            {t('groups.members.description')}
          </Text>

          <div className="mb-3 flex items-center gap-2">
            <Text type="secondary">{t('groups.members.filterByRole')}</Text>
            <Select
              allowClear
              showSearch
              optionFilterProp="label"
              placeholder={t('groups.members.filterPlaceholder')}
              style={{ width: 220 }}
              value={memberRoleFilter}
              onChange={(value?: string) => setMemberRoleFilter(value)}
              options={PRIMARY_ROLE_OPTIONS}
            />
          </div>

          <Transfer
            dataSource={filteredUserOptions}
            titles={[t('groups.members.available'), t('groups.members.current')]}
            targetKeys={targetKeys}
            onChange={(keys) => setSelectedUserIds(keys.map(k => String(k)))}
            render={item => (
              <Space>
                <Avatar size="small" icon={<UserIcon size={14} />} />
                <span>{item.title}</span>
                <Text type="secondary" className="text-xs">
                  {item.description}
                </Text>
                {memberRoleTag((item as { role?: string }).role)}
              </Space>
            )}
            listStyle={{ width: 300, height: 400 }}
            showSearch
            filterOption={(input, item) =>
              item.title.toLowerCase().includes(input.toLowerCase()) ||
              item.description.toLowerCase().includes(input.toLowerCase())
            }
            locale={{
              itemsUnit: t('groups.members.itemsUnit'),
              itemUnit: t('groups.members.itemUnit'),
            }}
          />

          {groupMembers.length > 0 && (
            <div className="mt-4">
              <Text strong>{t('groups.members.preview')}</Text>
              <List
                size="small"
                className="mt-2"
                bordered
                dataSource={groupMembers}
                renderItem={item => (
                  <List.Item>
                    <Space>
                      <Avatar size="small" src={('avatar' in item ? item.avatar : undefined) as string | undefined}>
                        <UserIcon size={14} />
                      </Avatar>
                      <Text>{item.name || item.username || t('groups.members.userFallback', { id: item.id })}</Text>
                      {memberRoleTag(item.role)}
                      {item.email && (
                        <Text type="secondary" className="text-xs">
                          {item.email}
                        </Text>
                      )}
                    </Space>
                  </List.Item>
                )}
              />
            </div>
          )}
        </div>
      </Modal>
    </div>
  );
};

export default GroupManagement;
