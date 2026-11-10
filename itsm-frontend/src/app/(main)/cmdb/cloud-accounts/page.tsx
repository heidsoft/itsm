'use client';

import React, { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  App,
  Breadcrumb,
  Button,
  Card,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Popconfirm,
  Switch,
} from 'antd';
import { Search, Plus, Pencil, Trash2, RotateCcw } from 'lucide-react';

import { CMDBApi } from '@/lib/api/cmdb-api';
import { CMDB_KEYS, useCloudAccountsQuery } from '@/lib/hooks/useCMDB';
import { serverErrorMessage } from '@/lib/api/http-client';
import type { CloudAccount } from '@/types/biz/cmdb';

// 与后端 cloud.KnownProviders 同源（后端 dto 枚举由守卫测试强制对齐）。
// 火山引擎与 k8s 的发现适配器已接入，缺这两项时前端根本选不到、能力等于零。
const providerOptions = [
  { value: 'aliyun', label: 'Aliyun' },
  { value: 'huawei', label: 'Huawei Cloud' },
  { value: 'tencent', label: 'Tencent Cloud' },
  { value: 'azure', label: 'Azure' },
  { value: 'aws', label: 'AWS' },
  { value: 'volcengine', label: 'Volcengine' },
  { value: 'k8s', label: 'Kubernetes' },
  { value: 'onprem', label: 'Private Cloud' },
];

export default function CloudAccountPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const [createForm] = Form.useForm();
  const [editForm] = Form.useForm();

  // React Query：云账号列表（替代手写 loadData + isMountedRef）
  const accountsQuery = useCloudAccountsQuery();
  const data = accountsQuery.data?.items ?? [];
  // 用 isPending 而非 isLoading：SSR/首帧时 query 尚未发起（isFetching=false），
  // isLoading 为 false 会把「暂无数据 / 共 0 个账号」的空态在数据到达前就渲染出来（D55）。
  // isPending 在服务端渲染与首次拉取期间均为 true，首帧即呈现加载态。
  const loading = accountsQuery.isPending;
  // 失败必须显式呈现：只消费 data/isLoading 时，接口 500/403 会退化成
  // 「共 0 个账号」的空表，用户无法区分「真的没数据」和「加载失败」。
  const loadError = accountsQuery.isError;
  const loadErrorText = serverErrorMessage(accountsQuery.error, 'Failed to load cloud account, please try again');

  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [editingAccount, setEditingAccount] = useState<CloudAccount | null>(null);
  const [searchText, setSearchText] = useState('');
  const [filterProvider, setFilterProvider] = useState<string | undefined>(undefined);

  // 客户端过滤（搜索/厂商筛选）
  const filteredData = useMemo(() => {
    return data.filter(item => {
      const matchSearch =
        !searchText ||
        item.accountName?.toLowerCase().includes(searchText.toLowerCase()) ||
        item.accountId?.toLowerCase().includes(searchText.toLowerCase());
      const matchProvider = !filterProvider || item.provider === filterProvider;
      return matchSearch && matchProvider;
    });
  }, [data, searchText, filterProvider]);

  const invalidateAccounts = () =>
    queryClient.invalidateQueries({ queryKey: [...CMDB_KEYS.all, 'cloud-accounts'] });

  // 新增云账号
  const createMutation = useMutation({
    mutationFn: (values: Record<string, unknown>) => CMDBApi.createCloudAccount(values),
    onSuccess: () => {
      message.success('Cloud account has been created');
      setCreateOpen(false);
      createForm.resetFields();
      invalidateAccounts();
    },
    onError: error => {
      if (error instanceof Error) {
        message.error(error.message || 'Creation failed');
      }
    },
  });

  const handleCreate = async () => {
    try {
      const values = await createForm.validateFields();
      createMutation.mutate(values);
    } catch {
      // 表单校验失败，由 antd Form 自行提示
    }
  };

  // 编辑云账号
  const updateMutation = useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, unknown> }) =>
      CMDBApi.updateCloudAccount(id, values),
    onSuccess: () => {
      message.success('Cloud account has been updated');
      setEditOpen(false);
      setEditingAccount(null);
      editForm.resetFields();
      invalidateAccounts();
    },
    onError: error => {
      if (error instanceof Error) {
        message.error(error.message || 'Update failed');
      }
    },
  });

  const handleEdit = (record: CloudAccount) => {
    setEditingAccount(record);
    editForm.setFieldsValue({
      provider: record.provider,
      accountId: record.accountId,
      accountName: record.accountName,
      credentialRef: undefined,
      isActive: record.isActive,
    });
    setEditOpen(true);
  };

  const handleUpdate = async () => {
    if (!editingAccount) return;
    try {
      const values = await editForm.validateFields();
      updateMutation.mutate({ id: editingAccount.id, values });
    } catch {
      // 表单校验失败，由 antd Form 自行提示
    }
  };

  // 删除云账号（行级 loading：isPending + variables 比对）
  const deleteMutation = useMutation({
    mutationFn: (id: number) => CMDBApi.deleteCloudAccount(String(id)),
    onSuccess: () => {
      message.success('Cloud account has been deleted');
      invalidateAccounts();
    },
    onError: error => {
      if (error instanceof Error) {
        message.error(error.message || 'Delete failed');
      }
    },
  });

  const handleDelete = (id: number) => {
    deleteMutation.mutate(id);
  };

  // 启用/停用切换（复用更新接口）
  const toggleMutation = useMutation({
    mutationFn: (record: CloudAccount) =>
      CMDBApi.updateCloudAccount(record.id, { isActive: !record.isActive }),
    onSuccess: (_result, record) => {
      message.success(record.isActive ? 'Cloud account has been deactivated' : 'Cloud account has been enabled');
      invalidateAccounts();
    },
    onError: error => {
      if (error instanceof Error) {
        message.error(error.message || 'Operation failed');
      }
    },
  });

  const handleToggleStatus = (record: CloudAccount) => {
    toggleMutation.mutate(record);
  };

  const columns = [
    {
      title: 'Vendor',
      dataIndex: 'provider',
      width: 110,
      render: (value: string) => {
        const provider = providerOptions.find(p => p.value === value);
        return <Tag color='blue'>{provider?.label || value}</Tag>;
      },
    },
    {
      title: 'Account ID',
      dataIndex: 'accountId',
      width: 200,
      ellipsis: true,
    },
    {
      title: 'Account name',
      dataIndex: 'accountName',
      width: 180,
      ellipsis: true,
    },
    {
      title: 'Credential status',
      dataIndex: 'hasCredential',
      width: 180,
      render: (value?: boolean) => (
        <Tag color={value ? 'green' : 'orange'}>{value ? 'Configured' : 'Not configured'}</Tag>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'isActive',
      width: 100,
      render: (value: boolean, record: CloudAccount) => (
        <Switch
          aria-label="Switch the enabled status of this cloud account"
          checked={value}
          checkedChildren="Enabled"
          unCheckedChildren="Disable"
          size='small'
          loading={toggleMutation.isPending && toggleMutation.variables?.id === record.id}
          onChange={() => handleToggleStatus(record)}
        />
      ),
    },
    {
      title: 'Actions',
      key: 'action',
      width: 120,
      render: (_: unknown, record: CloudAccount) => (
        <Space>
          <Tooltip title="Edit">
            <Button
              type='text'
              icon={<Pencil />}
              onClick={() => handleEdit(record)}
              size='small'
              aria-label={`Edit cloud account ${record.accountName}`}
            />
          </Tooltip>
          <Popconfirm
            title="Are you sure you want to delete this cloud account?"
            description="Unable to recover after deletion"
            onConfirm={() => handleDelete(record.id)}
            okText="OK"
            cancelText="Cancel"
          >
            <Tooltip title="Delete">
              <Button
                type='text'
                danger
                icon={<Trash2 />}
                size='small'
                loading={deleteMutation.isPending && deleteMutation.variables === record.id}
                aria-label={`Delete cloud account ${record.accountName}`}
              />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Card>
      <div className='mb-4'>
        <h1 className='text-2xl font-bold'>Cloud account management</h1>
        <p className='text-gray-500 mt-1'>Manage access accounts of cloud service providers for resource synchronization and discovery</p>
      </div>

      <Breadcrumb
        style={{ marginBottom: 16 }}
        items={[
          { title: 'Home page' },
          { title: 'Configuration Management' },
          { title: <a onClick={() => router.push('/cmdb')}>CMDB</a> },
          { title: 'Cloud account management' },
        ]}
      />

      {/* 搜索筛选工具栏 */}
      <div className='mb-4 flex flex-wrap items-center gap-3'>
        <Input
          placeholder="Search account name/ID"
          prefix={<Search className='text-gray-400' />}
          value={searchText}
          onChange={e => setSearchText(e.target.value)}
          allowClear
          style={{ width: 200 }}
        />
        <Select
          placeholder="Cloud vendor filtering"
          value={filterProvider}
          onChange={setFilterProvider}
          allowClear
          style={{ width: 140 }}
          options={providerOptions}
        />
        <Space>
          <Button
            icon={<RotateCcw />}
            onClick={() => accountsQuery.refetch()}
            loading={accountsQuery.isFetching}
          >

            Refresh
          </Button>
          <Button type='primary' icon={<Plus />} onClick={() => setCreateOpen(true)}>

            Add a new cloud account
          </Button>
        </Space>
        {loading ? (
          <span className='ml-auto text-sm text-gray-400'>Loading…</span>
        ) : loadError ? (
          <span className='ml-auto text-sm text-red-500'>Data loading failed</span>
        ) : (
          <span className='ml-auto text-sm text-gray-500'>Total {filteredData.length}  accounts</span>
        )}
      </div>

      {loadError && (
        <Alert
          type='error'
          showIcon
          className='mb-4'
          title={loadErrorText}
          action={
            <Button size='small' onClick={() => accountsQuery.refetch()}>

              Retry
            </Button>
          }
        />
      )}

      <Table
        rowKey='id'
        loading={loading}
        dataSource={loadError ? [] : filteredData}
        columns={columns}
        locale={
          loadError
            ? { emptyText: 'Data loading failed, please click "Retry" above' }
            : undefined
        }
        pagination={{
          pageSize: 10,
          showSizeChanger: true,
          showTotal: total => `Total ${total} records`,
          pageSizeOptions: ['10', '20', '50', '100'],
        }}
        scroll={{ x: 800 }}
      />

      {/* 新增云账号模态框 */}
      <Modal
        title="Add a new cloud account"
        open={createOpen}
        onCancel={() => {
          setCreateOpen(false);
          createForm.resetFields();
        }}
        onOk={handleCreate}
        confirmLoading={createMutation.isPending}
        width={500}
      >
        <Form form={createForm} layout='vertical'>
          <Form.Item
            name='provider'
            label="Cloud Provider"
            rules={[{ required: true, message: 'Please select a cloud vendor' }]}
          >
            <Select placeholder="Please select a cloud vendor" options={providerOptions} />
          </Form.Item>
          <Form.Item
            name='accountId'
            label="Account ID"
            rules={[{ required: true, message: 'Please enter the account ID' }]}
          >
            <Input placeholder="For example 1234567890123456" />
          </Form.Item>
          <Form.Item
            name='accountName'
            label="Account name"
            rules={[{ required: true, message: 'Please enter the account name' }]}
          >
            <Input placeholder="For example, production account" />
          </Form.Item>
          <Form.Item
            name='credentialRef'
            label="Credential reference"
            extra="Enter a secret://<tenant>/<name> reference; leave blank to configure credentials later."
          >
            <Input.Password placeholder="For example secret://tenant-42/aliyun-production" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑云账号模态框 */}
      <Modal
        title="Edit cloud account"
        open={editOpen}
        onCancel={() => {
          setEditOpen(false);
          setEditingAccount(null);
          editForm.resetFields();
        }}
        onOk={handleUpdate}
        confirmLoading={updateMutation.isPending}
        width={500}
      >
        <Form form={editForm} layout='vertical'>
          <Form.Item
            name='provider'
            label="Cloud Provider"
            rules={[{ required: true, message: 'Please select a cloud vendor' }]}
          >
            <Select placeholder="Please select a cloud vendor" disabled options={providerOptions} />
          </Form.Item>
          <Form.Item
            name='accountId'
            label="Account ID"
            rules={[{ required: true, message: 'Please enter the account ID' }]}
          >
            <Input placeholder="For example 1234567890123456" disabled />
          </Form.Item>
          <Form.Item
            name='accountName'
            label="Account name"
            rules={[{ required: true, message: 'Please enter the account name' }]}
          >
            <Input placeholder="For example, production account" />
          </Form.Item>
          <Form.Item
            name='credentialRef'
            label="Update credential reference"
            extra="Leave it blank to retain existing credentials; the server will not return the contents of the saved credentials."
          >
            <Input placeholder="e.g. aliyun-prod-credential" />
          </Form.Item>
          <Form.Item name='isActive' label="Status" valuePropName='checked'>
            <Switch checkedChildren="Enabled" unCheckedChildren="Disable" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
