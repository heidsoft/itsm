'use client';

import React, { useEffect, useState } from 'react';
import {
  App,
  Button,
  Card,
  Empty,
  Form,
  Input,
  Modal,
  Space,
  Table,
  Tag,
  Typography,
  Avatar,
  List,
  Transfer,
  Badge,
} from 'antd';
import type { TablePaginationConfig } from 'antd';
import { Edit, Plus, Search, Trash2, UserPlus, Users, User as UserIcon, X, Check } from 'lucide-react';
import BusinessStatsGrid from '@/components/common/BusinessStatsGrid';
import { GroupAPI, type Group } from '@/lib/api/group-api';
import { UserApi, type User } from '@/lib/api/user-api';
import { useI18n } from '@/lib/i18n/useI18n';

const { Title, Text } = Typography;
const { Search: AntSearch } = Input;

const GroupManagement: React.FC = () => {
  const { t } = useI18n();
  const { message, modal } = App.useApp();
  const [form] = Form.useForm();
  const [groups, setGroups] = useState<Group[]>([]);
  const [loading, setLoading] = useState(false);
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

  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0,
  });

  const loadGroups = async () => {
    setLoading(true);
    try {
      const response = await GroupAPI.getGroups({
        page: pagination.current,
        pageSize: pagination.pageSize,
        search: search || undefined,
      });
      setGroups(response.groups);
      setPagination(prev => ({
        ...prev,
        total: response.pagination.total,
      }));
    } catch (error) {
      console.error('Failed to load groups:', error);
      message.error(t('groups.messages.loadFailed'));
    } finally {
      setLoading(false);
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
    setLoading(true);
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
      setLoading(false);
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
      const response = await GroupAPI.getMembers(groupId, { page: 1, pageSize: 100 });
      setGroupMembers(response.members || []);
      setSelectedUserIds(response.members?.map(m => String(m.id)) || []);
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
      const response = await UserApi.getUsers({ page: 1, pageSize: 500 });
      setAllUsers(response.users || []);
    } catch (error) {
      console.error('Failed to load users:', error);
    }
  };

  // 打开成员管理弹窗
  const openMemberModal = async (group: Group) => {
    setSelectedGroup(group);
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
    } finally {
      setSavingMembers(false);
    }
  };

  // Transfer 的目标key（已选中的成员）
  const targetKeys = selectedUserIds.filter(id =>
    groupMembers.some(m => String(m.id) === id)
  );

  const handleTableChange = (nextPagination: TablePaginationConfig) => {
    setPagination(prev => ({
      ...prev,
      current: nextPagination.current || 1,
      pageSize: nextPagination.pageSize || prev.pageSize,
    }));
  };

  const statsItems = [
    {
      label: t('groups.stats.total'),
      value: pagination.total,
      icon: <Users size={20} />,
      tone: 'blue' as const,
    },
    {
      label: t('groups.stats.current'),
      value: groups.length,
      icon: <UserPlus size={20} />,
      tone: 'green' as const,
    },
    {
      label: t('common.search'),
      value: search ? pagination.total : '-',
      icon: <Search size={20} />,
      tone: 'cyan' as const,
    },
    {
      label: t('groups.stats.type'),
      value: t('groups.title'),
      icon: <Users size={20} />,
      tone: 'purple' as const,
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
        <Badge count={record.members?.length || 0} showZero color="blue">
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
      render: (value: string) => (value ? new Date(value).toLocaleString('zh-CN') : '-'),
      width: 160,
    },
    {
      title: t('groups.updatedAt'),
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      render: (value: string) => (value ? new Date(value).toLocaleString('zh-CN') : '-'),
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

      <BusinessStatsGrid items={statsItems} loading={loading && groups.length === 0} />

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
          loading={loading}
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
              <Button type="primary" htmlType="submit" loading={loading}>
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

          <Transfer
            dataSource={allUsers.map(u => ({
              key: String(u.id),
              title: u.name || u.username || t('groups.members.userFallback', { id: u.id }),
              description: u.email || u.username || '',
            }))}
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
              </Space>
            )}
            listStyle={{ width: 280, height: 400 }}
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
