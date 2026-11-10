'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Button,
  Modal,
  Form,
  Select,
  Switch,
  Space,
  Tag,
  App,
  Alert,
  Input,
  Segmented,
  Popconfirm,
  Typography,
} from 'antd';
import { Plus, Trash2, Users } from 'lucide-react';
import { UsageGuideCard } from '@/components/common/UsageGuideCard';
import { ChangeReviewApi } from '@/lib/api/';
import { UserApi, type User } from '@/lib/api/user-api';
import type {
  ReviewMember,
  ReviewBoardType,
  ReviewMemberRole,
  AddReviewMemberRequest,
} from '@/types/change-review';

const { Title, Text } = Typography;

const BOARD_OPTIONS: { label: string; value: ReviewBoardType }[] = [
  { label: 'Regular review group', value: 'REVIEW' },
  { label: 'Emergency Review Team', value: 'EREVIEW' },
];

const ROLE_OPTIONS: { label: string; value: ReviewMemberRole }[] = [
  { label: 'Review member: Participate in change review and voting', value: 'member' },
  { label: 'Moderator: organize the review and confirm the conclusion', value: 'chair' },
  { label: 'Recorder: Maintain meeting and resolution records', value: 'secretary' },
];

const ChangeReviewManagementPage: React.FC = () => {
  const { message } = App.useApp();
  const [board, setBoard] = useState<ReviewBoardType>('REVIEW');
  const [members, setMembers] = useState<ReviewMember[]>([]);
  const [loading, setLoading] = useState(false);

  // 新增弹窗
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [users, setUsers] = useState<User[]>([]);
  const [userSearch, setUserSearch] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm<AddReviewMemberRequest & { role?: ReviewMemberRole }>();

  const loadMembers = useCallback(async () => {
    setLoading(true);
    try {
      const data = await ChangeReviewApi.getMembers(board);
      setMembers(data);
    } catch (err) {
      message.error('Failed to load review team members');
    } finally {
      setLoading(false);
    }
  }, [board, message]);

  useEffect(() => {
    loadMembers();
  }, [loadMembers]);

  const openModal = async () => {
    form.resetFields();
    setUserSearch('');
    try {
      const res = await UserApi.getUsers({ page: 1, pageSize: 50, search: '' });
      setUsers(res.items ?? []);
    } catch {
      setUsers([]);
    }
    setIsModalOpen(true);
  };

  const handleAdd = async () => {
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      await ChangeReviewApi.addMember({
        userId: values.userId,
        type: board,
        role: values.role,
      });
      message.success('Review team members have been added');
      setIsModalOpen(false);
      loadMembers();
    } catch (err) {
      message.error('Add failed:' + (err instanceof Error ? err.message : 'Unknown error'));
    } finally {
      setSubmitting(false);
    }
  };

  const handleToggleActive = async (m: ReviewMember, next: boolean) => {
    try {
      await ChangeReviewApi.updateMember(m.id, { role: m.role, isActive: next });
      message.success(next ? 'Enabled' : 'Disabled');
      loadMembers();
    } catch (err) {
      message.error('Update failed:' + (err instanceof Error ? err.message : 'Unknown error'));
    }
  };

  const handleRemove = async (m: ReviewMember) => {
    try {
      await ChangeReviewApi.removeMember(m.id);
      message.success('Removed');
      loadMembers();
    } catch (err) {
      message.error('Removal failed:' + (err instanceof Error ? err.message : 'Unknown error'));
    }
  };

  const filteredUsers = users.filter(u =>
    !userSearch ||
    u.name?.includes(userSearch) ||
    u.email?.includes(userSearch) ||
    u.username?.includes(userSearch)
  );

  const columns = [
    {
      title: 'Users',
      dataIndex: 'userName',
      key: 'userName',
      render: (_: string, m: ReviewMember) => (
        <Space orientation="vertical" size={0}>
          <Text strong>{m.userName}</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>
            {m.email}
          </Text>
        </Space>
      ),
    },
    {
      title: 'Role',
      dataIndex: 'role',
      key: 'role',
      render: (role: string) => <Tag>{ROLE_OPTIONS.find(r => r.value === role)?.label ?? role}</Tag>,
    },
    {
      title: 'Status',
      dataIndex: 'isActive',
      key: 'isActive',
      render: (active: boolean, m: ReviewMember) => (
        <Switch
          aria-label={`switch ${m.userName} enabled status`}
          checked={active}
          checkedChildren="Enabled"
          unCheckedChildren="Disable"
          onChange={next => handleToggleActive(m, next)}
        />
      ),
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_: unknown, m: ReviewMember) => (
        <Popconfirm title="Confirm to remove this member?" onConfirm={() => handleRemove(m)}>
          <Button danger type="link" icon={<Trash2 size={14} />}>

            Remove
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={3} style={{ margin: 0 }}>
          <Users style={{ marginRight: 8 }} />

          Review team management
        </Title>
        <Space>
          <Segmented
            options={BOARD_OPTIONS}
            value={board}
            onChange={v => setBoard(v as ReviewBoardType)}
          />
          <Button type="primary" icon={<Plus size={14} />} onClick={openModal}>

            Add new member
          </Button>
        </Space>
      </Space>

      <UsageGuideCard
        style={{ marginBottom: 16 }}
        intro="The review team is responsible for reviewing high-risk changes and emergency changes; what is maintained here is the list of review teams, not the approval process itself."
        steps={[
          'Use the segmented control in the upper right corner to switch between the regular review group and the emergency review group.',
          'Click "Add Member" to select a user and assign a role: the moderator organizes the review and confirms the conclusion, the review members participate in voting, and the recorder maintains meeting and resolution records.',
          'When a change work order enters the review stage, approval candidates come from this list; regular changes are assigned to the regular review team, and emergency changes are assigned to the emergency review team.',
          'When a member is no longer needed, remove it directly from the list without affecting the historical review record.',
        ]}
      />

      <Card>
        <Alert
          className="mb-4"
          type="info"
          showIcon
          title="After adding review members and enabling them, they will be included in the candidate approvers of the corresponding review group."
        />
        <Table
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={members}
          pagination={false}
          locale={{ emptyText: 'No members yet' }}
        />
      </Card>

      <Modal
        title={`New ${board === 'REVIEW' ? 'General' : 'Urgent'} Review Team Member`}
        open={isModalOpen}
        onOk={handleAdd}
        onCancel={() => setIsModalOpen(false)}
        confirmLoading={submitting}
        destroyOnHidden
      >
        <Form form={form} layout="vertical" initialValues={{ role: 'member' }}>
          <Form.Item
            name="userId"
            label="Users"
            rules={[{ required: true, message: 'Please select a user' }]}
          >
            <Select
              showSearch
              placeholder="Search and select user"
              filterOption={false}
              onSearch={setUserSearch}
              options={filteredUsers.map(u => ({
                label: `${u.name}（${u.email}）`,
                value: u.id,
              }))}
            />
          </Form.Item>
          <Form.Item name="role" label="Review team role" rules={[{ required: true }]}>
            <Select options={ROLE_OPTIONS} />
          </Form.Item>
          <Text type="secondary" style={{ fontSize: 12 }}>

            New members are enabled by default; the post-approval chain engine is disabled (review:{board}  step) will no longer be included in this member.
          </Text>
        </Form>
      </Modal>
    </div>
  );
};

export default ChangeReviewManagementPage;
