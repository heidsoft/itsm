'use client';

import React, { useEffect, useState, useCallback, useRef } from 'react';
import { useRouter } from 'next/navigation';
import {
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
  message,
  Switch,
} from 'antd';
import { Search, Plus, Pencil, Trash2, RotateCcw } from 'lucide-react';

import { CMDBApi } from '@/lib/api/cmdb-api';
import type { CloudAccount } from '@/types/biz/cmdb';



const providerOptions = [
  { value: 'aliyun', label: '阿里云' },
  { value: 'huawei', label: '华为云' },
  { value: 'tencent', label: '腾讯云' },
  { value: 'azure', label: 'Azure' },
  { value: 'aws', label: 'AWS' },
  { value: 'onprem', label: '私有云' },
];

export default function CloudAccountPage() {
  const router = useRouter();
  const [form] = Form.useForm();
  const [createForm] = Form.useForm();
  const [editForm] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [createSubmitting, setCreateSubmitting] = useState(false);
  const [updateSubmitting, setUpdateSubmitting] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [data, setData] = useState<CloudAccount[]>([]);
  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [editingAccount, setEditingAccount] = useState<CloudAccount | null>(null);
  const [searchText, setSearchText] = useState('');
  const [filterProvider, setFilterProvider] = useState<string | undefined>(undefined);
  const isMountedRef = useRef(true);

  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  // 过滤后的数据
  const filteredData = useCallback(() => {
    return data.filter(item => {
      const matchSearch =
        !searchText ||
        item.accountName?.toLowerCase().includes(searchText.toLowerCase()) ||
        item.accountId?.toLowerCase().includes(searchText.toLowerCase());
      const matchProvider = !filterProvider || item.provider === filterProvider;
      return matchSearch && matchProvider;
    });
  }, [data, searchText, filterProvider]);

  const loadData = async () => {
    setLoading(true);
    try {
      const list = await CMDBApi.getCloudAccounts();
      if (isMountedRef.current) {
        setData(list || []);
      }
    } catch (error) {
      if (isMountedRef.current) {
        message.error('加载云账号失败');
      }
    } finally {
      if (isMountedRef.current) {
        setLoading(false);
      }
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleCreate = async () => {
    if (createSubmitting) return;
    setCreateSubmitting(true);
    try {
      const values = await createForm.validateFields();
      await CMDBApi.createCloudAccount(values);
      message.success('云账号已创建');
      setCreateOpen(false);
      createForm.resetFields();
      loadData();
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message || '创建失败');
      }
    } finally {
      setCreateSubmitting(false);
    }
  };

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
    if (updateSubmitting) return;
    setUpdateSubmitting(true);
    try {
      const values = await editForm.validateFields();
      // 使用 CloudAccount ID 进行更新
      await CMDBApi.updateCloudAccount(editingAccount.id, {
        ...values,
      });
      message.success('云账号已更新');
      setEditOpen(false);
      setEditingAccount(null);
      editForm.resetFields();
      loadData();
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message || '更新失败');
      }
    } finally {
      setUpdateSubmitting(false);
    }
  };

  const handleDelete = async (id: number) => {
    if (deletingId !== null) return;
    setDeletingId(id);
    try {
      await CMDBApi.deleteCloudAccount(String(id));
      message.success('云账号已删除');
      loadData();
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message || '删除失败');
      }
    } finally {
      setDeletingId(null);
    }
  };

  const handleToggleStatus = async (record: CloudAccount) => {
    try {
      // 调用更新接口切换状态
      await CMDBApi.updateCloudAccount(record.id, {
        isActive: !record.isActive,
      });
      message.success(record.isActive ? '云账号已停用' : '云账号已启用');
      loadData();
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message || '操作失败');
      }
    }
  };

  const columns = [
    {
      title: '厂商',
      dataIndex: 'provider',
      width: 110,
      render: (value: string) => {
        const provider = providerOptions.find(p => p.value === value);
        return <Tag color='blue'>{provider?.label || value}</Tag>;
      },
    },
    {
      title: '账号ID',
      dataIndex: 'accountId',
      width: 200,
      ellipsis: true,
    },
    {
      title: '账号名称',
      dataIndex: 'accountName',
      width: 180,
      ellipsis: true,
    },
    {
      title: '凭据状态',
      dataIndex: 'hasCredential',
      width: 180,
      render: (value?: boolean) => (
        <Tag color={value ? 'green' : 'orange'}>{value ? '已配置' : '未配置'}</Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'isActive',
      width: 100,
      render: (value: boolean, record: CloudAccount) => (
        <Switch
          checked={value}
          checkedChildren='启用'
          unCheckedChildren='停用'
          size='small'
          onChange={() => handleToggleStatus(record)}
        />
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: unknown, record: CloudAccount) => (
        <Space>
          <Tooltip title='编辑'>
            <Button
              type='text'
              icon={<Pencil />}
              onClick={() => handleEdit(record)}
              size='small'
              aria-label={`编辑云账号 ${record.accountName}`}
            />
          </Tooltip>
          <Popconfirm
            title='确定删除此云账号？'
            description='删除后无法恢复'
            onConfirm={() => handleDelete(record.id)}
            okText='确定'
            cancelText='取消'
          >
            <Tooltip title='删除'>
              <Button
                type='text'
                danger
                icon={<Trash2 />}
                size='small'
                loading={deletingId === record.id}
                aria-label={`删除云账号 ${record.accountName}`}
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
        <h1 className='text-2xl font-bold'>云账号管理</h1>
        <p className='text-gray-500 mt-1'>管理云服务商的访问账号，用于资源同步和发现</p>
      </div>

      <Breadcrumb
        style={{ marginBottom: 16 }}
        items={[
          { title: '首页' },
          { title: '配置管理' },
          { title: <a onClick={() => router.push('/cmdb')}>CMDB</a> },
          { title: '云账号管理' },
        ]}
      />

      {/* 搜索筛选工具栏 */}
      <div className='mb-4 flex flex-wrap items-center gap-3'>
        <Input
          placeholder='搜索账号名称/ID'
          prefix={<Search className='text-gray-400' />}
          value={searchText}
          onChange={e => setSearchText(e.target.value)}
          allowClear
          style={{ width: 200 }}
        />
        <Select
          placeholder='云厂商筛选'
          value={filterProvider}
          onChange={setFilterProvider}
          allowClear
          style={{ width: 140 }}
          options={providerOptions}
        />
        <Space>
          <Button icon={<RotateCcw />} onClick={loadData} loading={loading}>
            刷新
          </Button>
          <Button type='primary' icon={<Plus />} onClick={() => setCreateOpen(true)}>
            新增云账号
          </Button>
        </Space>
        <span className='ml-auto text-sm text-gray-500'>共 {filteredData().length} 个账号</span>
      </div>

      <Table
        rowKey='id'
        loading={loading}
        dataSource={filteredData()}
        columns={columns as any}
        pagination={{
          pageSize: 10,
          showSizeChanger: true,
          showTotal: total => `共 ${total} 条记录`,
          pageSizeOptions: ['10', '20', '50', '100'],
        }}
        scroll={{ x: 800 }}
      />

      {/* 新增云账号模态框 */}
      <Modal
        title='新增云账号'
        open={createOpen}
        onCancel={() => {
          setCreateOpen(false);
          createForm.resetFields();
        }}
        onOk={handleCreate}
        confirmLoading={createSubmitting}
        width={500}
      >
        <Form form={createForm} layout='vertical'>
          <Form.Item
            name='provider'
            label='云厂商'
            rules={[{ required: true, message: '请选择云厂商' }]}
          >
            <Select placeholder='请选择云厂商' options={providerOptions} />
          </Form.Item>
          <Form.Item
            name='accountId'
            label='账号ID'
            rules={[{ required: true, message: '请输入账号ID' }]}
          >
            <Input placeholder='例如 1234567890123456' />
          </Form.Item>
          <Form.Item
            name='accountName'
            label='账号名称'
            rules={[{ required: true, message: '请输入账号名称' }]}
          >
            <Input placeholder='例如 生产账号' />
          </Form.Item>
          <Form.Item
            name='credentialRef'
            label='更新凭据引用'
            extra='留空将保留现有凭据；服务端不会返回已保存的凭据内容。'
          >
            <Input.Password placeholder='例如 secret://tenant-42/aliyun-production' />
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑云账号模态框 */}
      <Modal
        title='编辑云账号'
        open={editOpen}
        onCancel={() => {
          setEditOpen(false);
          setEditingAccount(null);
          editForm.resetFields();
        }}
        onOk={handleUpdate}
        confirmLoading={updateSubmitting}
        width={500}
      >
        <Form form={editForm} layout='vertical'>
          <Form.Item
            name='provider'
            label='云厂商'
            rules={[{ required: true, message: '请选择云厂商' }]}
          >
            <Select placeholder='请选择云厂商' disabled options={providerOptions} />
          </Form.Item>
          <Form.Item
            name='accountId'
            label='账号ID'
            rules={[{ required: true, message: '请输入账号ID' }]}
          >
            <Input placeholder='例如 1234567890123456' disabled />
          </Form.Item>
          <Form.Item
            name='accountName'
            label='账号名称'
            rules={[{ required: true, message: '请输入账号名称' }]}
          >
            <Input placeholder='例如 生产账号' />
          </Form.Item>
          <Form.Item name='credentialRef' label='凭据引用'>
            <Input placeholder='例如 aliyun-prod-credential' />
          </Form.Item>
          <Form.Item name='isActive' label='状态' valuePropName='checked'>
            <Switch checkedChildren='启用' unCheckedChildren='停用' />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
