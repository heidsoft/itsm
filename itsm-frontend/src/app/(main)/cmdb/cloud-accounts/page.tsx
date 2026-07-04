'use client';

import React, { useEffect, useState, useCallback } from 'react';
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

const { Option } = Select;

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
  const [data, setData] = useState<CloudAccount[]>([]);
  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [editingAccount, setEditingAccount] = useState<CloudAccount | null>(null);
  const [searchText, setSearchText] = useState('');
  const [filterProvider, setFilterProvider] = useState<string | undefined>(undefined);

  // 过滤后的数据
  const filteredData = useCallback(() => {
    return data.filter(item => {
      const matchSearch = !searchText ||
        (item.account_name?.toLowerCase().includes(searchText.toLowerCase())) ||
        (item.account_id?.toLowerCase().includes(searchText.toLowerCase()));
      const matchProvider = !filterProvider || item.provider === filterProvider;
      return matchSearch && matchProvider;
    });
  }, [data, searchText, filterProvider]);

  const loadData = async () => {
    const isMounted = true;
    setLoading(true);
    try {
      const list = await CMDBApi.getCloudAccounts();
      if (isMounted) {
        setData(list || []);
      }
    } catch (error) {
      if (isMounted) {
        message.error('加载云账号失败');
      }
    } finally {
      if (isMounted) {
        setLoading(false);
      }
    }
  };

  useEffect(() => {
    let isMounted = true;
    loadData();
    return () => {
      isMounted = false;
    };
  }, []);

  const handleCreate = async () => {
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
    }
  };

  const handleEdit = (record: CloudAccount) => {
    setEditingAccount(record);
    editForm.setFieldsValue({
      provider: record.provider,
      account_id: record.account_id,
      account_name: record.account_name,
      credential_ref: record.credential_ref,
      is_active: record.is_active,
    });
    setEditOpen(true);
  };

  const handleUpdate = async () => {
    if (!editingAccount) return;
    try {
      const values = await editForm.validateFields();
      // 使用 CloudAccount ID 进行更新
      await CMDBApi.updateCI(editingAccount.id, {
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
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await CMDBApi.deleteCloudAccount(String(id));
      message.success('云账号已删除');
      loadData();
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message || '删除失败');
      }
    }
  };

  const handleToggleStatus = async (record: CloudAccount) => {
    try {
      // 调用更新接口切换状态
      await CMDBApi.updateCI(record.id, {
        is_active: !record.is_active,
      });
      message.success(record.is_active ? '云账号已停用' : '云账号已启用');
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
        return <Tag color="blue">{provider?.label || value}</Tag>;
      },
    },
    {
      title: '账号ID',
      dataIndex: 'account_id',
      width: 200,
      ellipsis: true,
    },
    {
      title: '账号名称',
      dataIndex: 'account_name',
      width: 180,
      ellipsis: true,
    },
    {
      title: '凭据引用',
      dataIndex: 'credential_ref',
      width: 180,
      ellipsis: true,
      render: (value?: string) => value || '-',
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      width: 100,
      render: (value: boolean, record: CloudAccount) => (
        <Switch
          checked={value}
          checkedChildren="启用"
          unCheckedChildren="停用"
          size="small"
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
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<Pencil />}
              onClick={() => handleEdit(record)}
              size="small"
            />
          </Tooltip>
          <Popconfirm
            title="确定删除此云账号？"
            description="删除后无法恢复"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Tooltip title="删除">
              <Button type="text" danger icon={<Trash2 />} size="small" />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Card>
      <div className="mb-4">
        <h1 className="text-2xl font-bold">云账号管理</h1>
        <p className="text-gray-500 mt-1">管理云服务商的访问账号，用于资源同步和发现</p>
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
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <Input
          placeholder="搜索账号名称/ID"
          prefix={<Search className="text-gray-400" />}
          value={searchText}
          onChange={e => setSearchText(e.target.value)}
          allowClear
          style={{ width: 200 }}
        />
        <Select
          placeholder="云厂商筛选"
          value={filterProvider}
          onChange={setFilterProvider}
          allowClear
          style={{ width: 140 }}
        >
          {providerOptions.map(item => (
            <Option key={item.value} value={item.value}>
              {item.label}
            </Option>
          ))}
        </Select>
        <Space>
          <Button icon={<RotateCcw />} onClick={loadData} loading={loading}>
            刷新
          </Button>
          <Button type="primary" icon={<Plus />} onClick={() => setCreateOpen(true)}>
            新增云账号
          </Button>
        </Space>
        <span className="ml-auto text-sm text-gray-500">
          共 {filteredData().length} 个账号
        </span>
      </div>

      <Table
        rowKey="id"
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
        title="新增云账号"
        open={createOpen}
        onCancel={() => {
          setCreateOpen(false);
          createForm.resetFields();
        }}
        onOk={handleCreate}
        width={500}
      >
        <Form form={createForm} layout="vertical">
          <Form.Item
            name="provider"
            label="云厂商"
            rules={[{ required: true, message: '请选择云厂商' }]}
          >
            <Select placeholder="请选择云厂商">
              {providerOptions.map(item => (
                <Option key={item.value} value={item.value}>
                  {item.label}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item
            name="account_id"
            label="账号ID"
            rules={[{ required: true, message: '请输入账号ID' }]}
          >
            <Input placeholder="例如 1234567890123456" />
          </Form.Item>
          <Form.Item
            name="account_name"
            label="账号名称"
            rules={[{ required: true, message: '请输入账号名称' }]}
          >
            <Input placeholder="例如 生产账号" />
          </Form.Item>
          <Form.Item name="credential_ref" label="凭据引用">
            <Input placeholder="例如 aliyun-prod-credential" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑云账号模态框 */}
      <Modal
        title="编辑云账号"
        open={editOpen}
        onCancel={() => {
          setEditOpen(false);
          setEditingAccount(null);
          editForm.resetFields();
        }}
        onOk={handleUpdate}
        width={500}
      >
        <Form form={editForm} layout="vertical">
          <Form.Item
            name="provider"
            label="云厂商"
            rules={[{ required: true, message: '请选择云厂商' }]}
          >
            <Select placeholder="请选择云厂商" disabled>
              {providerOptions.map(item => (
                <Option key={item.value} value={item.value}>
                  {item.label}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item
            name="account_id"
            label="账号ID"
            rules={[{ required: true, message: '请输入账号ID' }]}
          >
            <Input placeholder="例如 1234567890123456" disabled />
          </Form.Item>
          <Form.Item
            name="account_name"
            label="账号名称"
            rules={[{ required: true, message: '请输入账号名称' }]}
          >
            <Input placeholder="例如 生产账号" />
          </Form.Item>
          <Form.Item name="credential_ref" label="凭据引用">
            <Input placeholder="例如 aliyun-prod-credential" />
          </Form.Item>
          <Form.Item name="is_active" label="状态" valuePropName="checked">
            <Switch checkedChildren="启用" unCheckedChildren="停用" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
