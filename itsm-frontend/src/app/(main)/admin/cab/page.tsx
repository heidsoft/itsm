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
import { CabApi } from '@/lib/api/';
import { UserApi, type User } from '@/lib/api/user-api';
import type {
  CabMember,
  CabBoardType,
  CabMemberRole,
  AddCabMemberRequest,
} from '@/types/cab';

const { Title, Text } = Typography;

const BOARD_OPTIONS: { label: string; value: CabBoardType }[] = [
  { label: 'CAB（变更咨询委员会）', value: 'CAB' },
  { label: 'ECAB（紧急 CAB）', value: 'ECAB' },
];

const ROLE_OPTIONS: { label: string; value: CabMemberRole }[] = [
  { label: '评审成员：参与变更评审和投票', value: 'member' },
  { label: '主持人：组织评审并确认结论', value: 'chair' },
  { label: '记录人：维护会议与决议记录', value: 'secretary' },
];

const CabManagementPage: React.FC = () => {
  const { message } = App.useApp();
  const [board, setBoard] = useState<CabBoardType>('CAB');
  const [members, setMembers] = useState<CabMember[]>([]);
  const [loading, setLoading] = useState(false);

  // 新增弹窗
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [users, setUsers] = useState<User[]>([]);
  const [userSearch, setUserSearch] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm<AddCabMemberRequest & { role?: CabMemberRole }>();

  const loadMembers = useCallback(async () => {
    setLoading(true);
    try {
      const data = await CabApi.getMembers(board);
      setMembers(data);
    } catch (err) {
      message.error('加载 CAB 成员失败');
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
      setUsers(res.users ?? []);
    } catch {
      setUsers([]);
    }
    setIsModalOpen(true);
  };

  const handleAdd = async () => {
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      await CabApi.addMember({
        userId: values.userId,
        type: board,
        role: values.role,
      });
      message.success('已添加 CAB 成员');
      setIsModalOpen(false);
      loadMembers();
    } catch (err) {
      message.error('添加失败：' + (err instanceof Error ? err.message : '未知错误'));
    } finally {
      setSubmitting(false);
    }
  };

  const handleToggleActive = async (m: CabMember, next: boolean) => {
    try {
      await CabApi.updateMember(m.id, { role: m.role, isActive: next });
      message.success(next ? '已启用' : '已停用');
      loadMembers();
    } catch (err) {
      message.error('更新失败：' + (err instanceof Error ? err.message : '未知错误'));
    }
  };

  const handleRemove = async (m: CabMember) => {
    try {
      await CabApi.removeMember(m.id);
      message.success('已移除');
      loadMembers();
    } catch (err) {
      message.error('移除失败：' + (err instanceof Error ? err.message : '未知错误'));
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
      title: '用户',
      dataIndex: 'userName',
      key: 'userName',
      render: (_: string, m: CabMember) => (
        <Space orientation="vertical" size={0}>
          <Text strong>{m.userName}</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>
            {m.email}
          </Text>
        </Space>
      ),
    },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      render: (role: string) => <Tag>{ROLE_OPTIONS.find(r => r.value === role)?.label ?? role}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'isActive',
      key: 'isActive',
      render: (active: boolean, m: CabMember) => (
        <Switch
          checked={active}
          checkedChildren="启用"
          unCheckedChildren="停用"
          onChange={next => handleToggleActive(m, next)}
        />
      ),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: unknown, m: CabMember) => (
        <Popconfirm title="确认移除该成员？" onConfirm={() => handleRemove(m)}>
          <Button danger type="link" icon={<Trash2 size={14} />}>
            移除
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
          CAB 成员管理
        </Title>
        <Space>
          <Segmented
            options={BOARD_OPTIONS}
            value={board}
            onChange={v => setBoard(v as CabBoardType)}
          />
          <Button type="primary" icon={<Plus size={14} />} onClick={openModal}>
            新增成员
          </Button>
        </Space>
      </Space>

      <UsageGuideCard
        style={{ marginBottom: 16 }}
        intro="CAB（变更咨询委员会）负责评审高风险变更；这里维护的是委员会名单，不是审批流程本身。"
        steps={[
          '用右上角分段控件在 CAB 与 ECAB（紧急变更委员会）两个名单之间切换。',
          '点击“新增成员”选择用户并指定角色：主持人组织评审并确认结论，评审成员参与投票，记录人维护会议与决议记录。',
          '变更工单进入评审阶段时，审批候选人来自这份名单；常规变更配 CAB，紧急变更配 ECAB。',
          '不需要某位成员时直接在列表中移除，不影响历史评审记录。',
        ]}
      />

      <Card>
        <Alert
          className="mb-4"
          type="info"
          showIcon
          message="CAB 用于评审常规或高风险变更；ECAB 用于紧急变更"
          description="先选择委员会，再添加具备变更决策职责的用户并赋予其在会议中的职责。成员启用后，流程中的 CAB 审批步骤才会将其纳入候选审批人；停用不会删除历史评审记录。"
        />
        <Table
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={members}
          pagination={false}
          locale={{ emptyText: '暂无成员' }}
        />
      </Card>

      <Modal
        title={`新增 ${board} 成员`}
        open={isModalOpen}
        onOk={handleAdd}
        onCancel={() => setIsModalOpen(false)}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form form={form} layout="vertical" initialValues={{ role: 'member' }}>
          <Form.Item
            name="userId"
            label="用户"
            rules={[{ required: true, message: '请选择用户' }]}
          >
            <Select
              showSearch
              placeholder="搜索并选择用户"
              filterOption={false}
              onSearch={setUserSearch}
              options={filteredUsers.map(u => ({
                label: `${u.name}（${u.email}）`,
                value: u.id,
              }))}
            />
          </Form.Item>
          <Form.Item name="role" label="委员会角色" rules={[{ required: true }]}>
            <Select options={ROLE_OPTIONS} />
          </Form.Item>
          <Text type="secondary" style={{ fontSize: 12 }}>
            新增成员默认启用；停用后审批链引擎（cab:{board} 步骤）将不再纳入该成员。
          </Text>
        </Form>
      </Modal>
    </div>
  );
};

export default CabManagementPage;
