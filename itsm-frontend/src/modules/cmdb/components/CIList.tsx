'use client';

/**
 * 配置项 (CI) 列表组件
 */

import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  Table,
  Tag,
  Button,
  Card,
  Space,
  Tooltip,
  Input,
  Select,
  Form,
  message,
  Modal,
  Breadcrumb,
} from 'antd';
import {
  SearchOutlined,
  EyeOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { CMDBApi } from '../api';
import { CIStatus, CIStatusLabels } from '../constants';
import type { ConfigurationItem, CIType, CIQuery } from '../types';

const { Option } = Select;

const statusColors: Record<string, string> = {
  [CIStatus.ACTIVE]: 'green',
  [CIStatus.INACTIVE]: 'default',
  [CIStatus.MAINTENANCE]: 'orange',
  [CIStatus.DECOMMISSIONED]: 'red',
};

const getErrorMessage = (error: unknown): string => {
  if (error instanceof Error) return error.message;
  if (typeof error === 'string') return error;
  return '';
};

const CIList: React.FC = () => {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<ConfigurationItem[]>([]);
  const [total, setTotal] = useState(0);
  const [types, setTypes] = useState<CIType[]>([]);
  const [form] = Form.useForm();
  const requestIdRef = useRef(0);
  const isMountedRef = useRef(true);

  const [query, setQuery] = useState<CIQuery>({
    page: 1,
    page_size: 10,
  });

  const loadTypes = useCallback(async () => {
    try {
      const res = await CMDBApi.getTypes();
      if (!isMountedRef.current) return;
      setTypes(res || []);
    } catch (e) {
      if (!isMountedRef.current) return;
      // console.error(e);
      message.error('加载资产类型失败');
    }
  }, []);

  const loadData = useCallback(async () => {
    const requestId = ++requestIdRef.current;
    setLoading(true);
    try {
      const values = form.getFieldsValue();
      const resp = await CMDBApi.getCIs({
        ...query,
        ...values,
      });
      if (!isMountedRef.current || requestId !== requestIdRef.current) return;
      setData(resp.items || []);
      setTotal(resp.total || 0);
    } catch (error) {
      if (!isMountedRef.current || requestId !== requestIdRef.current) return;
      // console.error(error);
      const errorMessage = getErrorMessage(error);
      message.error(errorMessage ? `加载配置项列表失败：${errorMessage}` : '加载配置项列表失败');
    } finally {
      if (isMountedRef.current && requestId === requestIdRef.current) {
        setLoading(false);
      }
    }
  }, [query, form]);

  useEffect(() => {
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  useEffect(() => {
    loadTypes();
  }, [loadTypes]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleSearch = () => {
    setQuery(prev => ({ ...prev, page: 1 }));
  };

  const handleDelete = (id: number) => {
    Modal.confirm({
      title: '重申此操作',
      content: '确定要删除此配置项吗？相关关系也将受到影响。',
      onOk: async () => {
        try {
          await CMDBApi.deleteCI(id);
          message.success('删除成功');
          loadData();
        } catch (e) {
          const errorMessage = getErrorMessage(e);
          message.error(errorMessage ? `删除失败：${errorMessage}` : '删除失败');
        }
      },
    });
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 70,
    },
    {
      title: '资产名称',
      dataIndex: 'name',
      ellipsis: true,
      render: (text: string, record: ConfigurationItem) => (
        <Button
          type='link'
          onClick={() => router.push(`/cmdb/cis/${record.id}`)}
          style={{ padding: 0, height: 'auto' }}
        >
          {text}
        </Button>
      ),
    },
    {
      title: '类型',
      dataIndex: 'ci_type_id',
      width: 120,
      render: (id: number) => types.find(t => t.id === id)?.name || `类型 ${id}`,
    },
    {
      title: '云厂商',
      dataIndex: 'cloud_provider',
      width: 120,
      render: (value?: string) => value || '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (status: CIStatus) => (
        <Tag color={statusColors[status]}>{CIStatusLabels[status] || status}</Tag>
      ),
    },
    {
      title: '型号/厂商',
      key: 'model_vendor',
      width: 180,
      render: (_: any, record: ConfigurationItem) => (
        <span>
          {record.model || '-'} / {record.vendor || '-'}
        </span>
      ),
    },
    {
      title: '最后更新',
      dataIndex: 'updated_at',
      width: 160,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: any, record: ConfigurationItem) => (
        <Space size='small'>
          <Tooltip title='编辑'>
            <Button
              type='text'
              icon={<EditOutlined />}
              aria-label='编辑'
              onClick={() => router.push(`/cmdb/cis/${record.id}/edit`)}
            />
          </Tooltip>
          <Tooltip title='删除'>
            <Button
              type='text'
              danger
              icon={<DeleteOutlined />}
              aria-label='删除'
              onClick={() => handleDelete(record.id)}
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <div className='p-6'>
      <div className='flex justify-between items-center mb-6'>
        <div>
          <h1 className='text-2xl font-bold text-gray-900'>配置管理</h1>
          <p className='text-gray-500 mt-1'>管理和维护系统中的所有配置项(CI)及其关系</p>
        </div>
        <Button
          type='primary'
          icon={<PlusOutlined />}
          onClick={() => router.push('/cmdb/cis/create')}
          size='large'
        >
          录入资产
        </Button>
      </div>

      <Card className='rounded-lg shadow-sm border border-gray-200' variant="borderless">
        <Form form={form} layout='inline' className='mb-6 flex-wrap gap-y-4'>
          <Form.Item name='search' className='mb-0'>
            <Input
              placeholder='搜索名称/序列号'
              allowClear
              prefix={<SearchOutlined className='text-gray-400' />}
              className='w-64'
            />
          </Form.Item>
          <Form.Item name='ci_type_id' className='mb-0'>
            <Select placeholder='资产类型' style={{ width: 140 }} allowClear>
              {types.map(t => (
                <Option key={t.id} value={t.id}>
                  {t.name}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name='status' className='mb-0'>
            <Select placeholder='状态' style={{ width: 110 }} allowClear>
              {Object.entries(CIStatusLabels).map(([value, label]) => (
                <Option key={value} value={value}>
                  {label}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item className='mb-0'>
            <Space>
              <Button type='primary' ghost onClick={handleSearch}>
                查询
              </Button>
              <Button icon={<ReloadOutlined />} onClick={loadData} aria-label='刷新' />
              <Select
                placeholder='云资源管理'
                style={{ width: 140 }}
                onChange={value => router.push(value)}
                options={[
                  { label: '云服务目录', value: '/cmdb/cloud-services' },
                  { label: '云账号管理', value: '/cmdb/cloud-accounts' },
                  { label: '云资源列表', value: '/cmdb/cloud-resources' },
                  { label: '对账中心', value: '/cmdb/reconciliation' },
                ]}
              />
            </Space>
          </Form.Item>
        </Form>

        <Table
          rowKey='id'
          columns={columns as any}
          dataSource={data}
          loading={loading}
          locale={{ emptyText: '暂无配置项' }}
          pagination={{
            current: query.page,
            pageSize: query.page_size,
            total: total,
            showSizeChanger: true,
            showTotal: total => `共 ${total} 条记录`,
            onChange: (page, page_size) => setQuery(prev => ({ ...prev, page, page_size })),
          }}
          scroll={{ x: 1200 }}
        />
      </Card>
    </div>
  );
};

export default CIList;
