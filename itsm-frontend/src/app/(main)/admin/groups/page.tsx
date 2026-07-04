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
import { Edit, Plus, Search, Trash2, UserPlus, Users, User, X, Check } from 'lucide-react';
import BusinessStatsGrid from '@/components/common/BusinessStatsGrid';
import { GroupAPI, type Group } from '@/lib/api/group-api';
import { UserApi, type User } from '@/lib/api/user-api';

const { Title, Text } = Typography;
const { Search: AntSearch } = Input;

const GroupManagement: React.FC = () => {
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
        page_size: pagination.pageSize,
        search: search || undefined,
      });
      setGroups(response.groups);
      setPagination(prev => ({
        ...prev,
        total: response.pagination.total,
      }));
    } catch (error) {
      console.error('Failed to load groups:', error);
      message.error('加载用户组失败');
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
        message.success('用户组更新成功');
      } else {
        await GroupAPI.createGroup(values);
        message.success('用户组创建成功');
      }
      setModalOpen(false);
      setSelectedGroup(null);
      form.resetFields();
      await loadGroups();
    } catch (error) {
      console.error('Failed to save group:', error);
      message.error('保存用户组失败');
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = (group: Group) => {
    modal.confirm({
      title: '确认删除用户组',
      content: `删除「${group.name}」后，关联成员关系也会被移除。确定继续吗？`,
      okText: '删除',
      okButtonProps: { danger: true },
      cancelText: '取消',
      onOk: async () => {
        try {
          await GroupAPI.deleteGroup(group.id);
          message.success('用户组删除成功');
          await loadGroups();
        } catch (error) {
          console.error('Failed to delete group:', error);
          message.error('删除用户组失败');
        }
      },
    });
  };

  // 加载组成员
  const loadGroupMembers = async (groupId: number) => {
    setLoadingMembers(true);
    try {
      const response = await GroupAPI.getMembers(groupId, { page: 1, page_size: 100 });
      setGroupMembers(response.members || []);
      setSelectedUserIds(response.members?.map(m => String(m.id)) || []);
    } catch (error) {
      console.error('Failed to load group members:', error);
      message.error('加载组成员失败');
    } finally {
      setLoadingMembers(false);
    }
  };

  // 加载所有用户（用于添加成员）
  const loadAllUsers = async () => {
    try {
      const response = await UserApi.getUsers({ page: 1, page_size: 500 });
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

      message.success('组成员更新成功');
      setMemberModalOpen(false);
      await loadGroups();
    } catch (error) {
      console.error('Failed to save members:', error);
      message.error('保存组成员失败');
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
      label: '总用户组数',
      value: pagination.total,
      icon: <Users size={20} />,
      tone: 'blue' as const,
    },
    {
      label: '当前页用户组',
      value: groups.length,
      icon: <UserPlus size={20} />,
      tone: 'green' as const,
    },
    {
      label: '搜索结果',
      value: search ? pagination.total : '-',
      icon: <Search size={20} />,
      tone: 'cyan' as const,
    },
    {
      label: '业务类型',
      value: '用户组',
      icon: <Users size={20} />,
      tone: 'purple' as const,
    },
  ];

  const columns = [
    {
      title: '用户组名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: Group) => (
        <Space orientation="vertical" size={0}>
          <Text strong>{text}</Text>
          <Text type="secondary">{record.description || '暂无描述'}</Text>
        </Space>
      ),
    },
    {
      title: '成员',
      key: 'members',
      width: 120,
      render: (_: unknown, record: Group) => (
        <Badge count={record.members?.length || 0} showZero color="blue">
          <Button
            type="link"
            icon={<Users size={16} />}
            onClick={() => openMemberModal(record)}
          >
            管理
          </Button>
        </Badge>
      ),
    },
    {
      title: '租户ID',
      dataIndex: 'tenantId',
      key: 'tenantId',
      width: 100,
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (value: string) => (value ? new Date(value).toLocaleString('zh-CN') : '-'),
      width: 160,
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      render: (value: string) => (value ? new Date(value).toLocaleString('zh-CN') : '-'),
      width: 160,
    },
    {
      title: '操作',
      key: 'actions',
      width: 180,
      render: (_: unknown, record: Group) => (
        <Space size="small">
          <Button type="link" size="small" icon={<Users size={14} />} onClick={() => openMemberModal(record)}>
            成员
          </Button>
          <Button type="link" size="small" icon={<Edit size={14} />} onClick={() => openEditModal(record)}>
            编辑
          </Button>
          <Button
            type="link"
            size="small"
            danger
            icon={<Trash2 size={14} />}
            onClick={() => handleDelete(record)}
          >
            删除
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
            用户组管理
          </Title>
          <Text type="secondary">管理用户组基础信息，为后续成员关系和审批候选组提供组织基础。</Text>
        </div>
        <Button type="primary" icon={<Plus size={16} />} onClick={openCreateModal}>
          新建用户组
        </Button>
      </div>

      <BusinessStatsGrid items={statsItems} loading={loading && groups.length === 0} />

      <Card>
        <Space wrap className="w-full justify-between">
          <AntSearch
            allowClear
            enterButton
            placeholder="搜索用户组名称或描述"
            style={{ width: 320 }}
            onSearch={value => {
              setSearch(value.trim());
              setPagination(prev => ({ ...prev, current: 1 }));
            }}
          />
          <Button onClick={loadGroups}>刷新</Button>
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
              <Empty description={search ? '没有匹配的用户组' : '暂无用户组'}>
                <Button type="primary" onClick={openCreateModal}>
                  创建用户组
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
            showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条/共 ${total} 条`,
          }}
        />
      </Card>

      <Modal
        title={selectedGroup ? '编辑用户组' : '新建用户组'}
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
            label="用户组名称"
            rules={[
              { required: true, message: '请输入用户组名称' },
              { max: 100, message: '用户组名称不能超过100个字符' },
            ]}
          >
            <Input placeholder="例如：一线支持组、变更审批组" />
          </Form.Item>
          <Form.Item
            name="description"
            label="描述"
            rules={[{ max: 500, message: '描述不能超过500个字符' }]}
          >
            <Input.TextArea rows={4} placeholder="说明这个用户组的职责范围" />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={loading}>
                {selectedGroup ? '保存' : '创建'}
              </Button>
              <Button
                onClick={() => {
                  setModalOpen(false);
                  setSelectedGroup(null);
                  form.resetFields();
                }}
              >
                取消
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
            <span>管理组成员 - {selectedGroup?.name}</span>
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
            取消
          </Button>,
          <Button
            key="save"
            type="primary"
            loading={savingMembers}
            onClick={handleSaveMembers}
          >
            保存更改
          </Button>,
        ]}
      >
        <div className="py-4">
          <Text type="secondary" className="mb-4 block">
            从左侧选择用户添加到本组，或移除已有成员。已在本组的成员显示在右侧。
          </Text>

          <Transfer
            dataSource={allUsers.map(u => ({
              key: String(u.id),
              title: u.name || u.username || `用户#${u.id}`,
              description: u.email || u.username || '',
            }))}
            titles={['可添加的用户', '当前成员']}
            targetKeys={targetKeys}
            onChange={setSelectedUserIds}
            render={item => (
              <Space>
                <Avatar size="small" icon={<User size={14} />} />
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
              itemsUnit: '用户',
              itemUnit: '用户',
            }}
          />

          {groupMembers.length > 0 && (
            <div className="mt-4">
              <Text strong>当前成员预览：</Text>
              <List
                size="small"
                className="mt-2"
                bordered
                dataSource={groupMembers}
                renderItem={item => (
                  <List.Item>
                    <Space>
                      <Avatar size="small" src={item.avatar}>
                        <User size={14} />
                      </Avatar>
                      <Text>{item.name || item.username || `用户#${item.id}`}</Text>
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
