'use client';

/**
 * 资产详情组件
 */

import React, { useState, useEffect } from 'react';
import {
  Card,
  Descriptions,
  Tag,
  Button,
  Skeleton,
  Result,
  Space,
  Timeline,
  Modal,
  Form,
  Select,
  message,
  Typography,
} from 'antd';
import { ArrowLeft, User as UserIcon, Monitor, MapPin } from 'lucide-react';
import { useParams, useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import type { Asset } from '@/lib/api/asset-api';
import { AssetApi } from '@/lib/api/asset-api';
import type { User as UserType } from '@/lib/api/user-api';
import { UserApi } from '@/lib/api/user-api';

const { Title, Text } = Typography;
const { Option } = Select;

// 状态颜色映射
const statusColors: Record<string, string> = {
  available: 'success',
  'in-use': 'processing',
  maintenance: 'warning',
  retired: 'default',
  disposed: 'error',
};

const statusLabels: Record<string, string> = {
  available: '可用',
  'in-use': '使用中',
  maintenance: '维护中',
  retired: '已退役',
  disposed: '已处置',
};

const typeLabels: Record<string, string> = {
  hardware: '硬件',
  software: '软件',
  cloud: '云资源',
  license: '许可证',
};

const AssetDetail: React.FC = () => {
  const { id } = useParams() as { id: string };
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [asset, setAsset] = useState<Asset | null>(null);
  const [assignModalVisible, setAssignModalVisible] = useState(false);
  const [users, setUsers] = useState<UserType[]>([]);
  const [usersLoading, setUsersLoading] = useState(false);
  const [form] = Form.useForm();

  useEffect(() => {
    if (id) {
      loadDetail();
    }
  }, [id]);

  const loadDetail = async () => {
    setLoading(true);
    try {
      const data = await AssetApi.getAsset(Number(id));
      setAsset(data);
    } catch (error) {
      message.error('加载资产详情失败');
    } finally {
      setLoading(false);
    }
  };

  const loadUsers = async () => {
    setUsersLoading(true);
    try {
      const response = await UserApi.getUsers({ pageSize: 100 });
      setUsers(response.users || []);
    } catch (error) {
      console.error('Failed to load users:', error);
      message.error('加载用户列表失败');
    } finally {
      setUsersLoading(false);
    }
  };

  const handleAssign = async () => {
    try {
      const values = await form.validateFields();
      await AssetApi.assignAsset(Number(id), values.assignedTo);
      message.success('分配成功');
      setAssignModalVisible(false);
      loadDetail();
    } catch (error) {
      message.error('分配失败');
    }
  };

  const handleRetire = async () => {
    Modal.confirm({
      title: '确认退役',
      content: '确定要退役此资产吗？',
      onOk: async () => {
        try {
          await AssetApi.retireAsset(Number(id), '手动退役');
          message.success('退役成功');
          loadDetail();
        } catch (error) {
          message.error('退役失败');
        }
      },
    });
  };

  if (loading) {
    return (
      <Card>
        <Skeleton active />
      </Card>
    );
  }

  if (!asset) {
    return (
      <Card>
        <Result
          status="404"
          title="404"
          subTitle="抱歉，您访问的资产不存在"
          extra={
            <Button type="primary" onClick={() => router.push('/assets')}>
              返回列表
            </Button>
          }
        />
      </Card>
    );
  }

  return (
    <Space orientation="vertical" style={{ width: '100%' }} size="large">
      <Card>
        <div style={{ marginBottom: 24 }}>
          <Button
            icon={<ArrowLeft />}
            onClick={() => router.push('/assets')}
            style={{ marginBottom: 16 }}
          >
            返回列表
          </Button>
          <div
            style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}
          >
            <div>
              <Title level={3} style={{ marginBottom: 8 }}>
                {asset.name}
              </Title>
              <Text type="secondary">资产编号: {asset.assetNumber}</Text>
            </div>
            <Space>
              <Tag color={statusColors[asset.status]} style={{ padding: '4px 12px', fontSize: 14 }}>
                {statusLabels[asset.status]}
              </Tag>
              <Tag>{typeLabels[asset.type]}</Tag>
            </Space>
          </div>
        </div>

        <Descriptions bordered column={2}>
          <Descriptions.Item label="资产编号">{asset.assetNumber}</Descriptions.Item>
          <Descriptions.Item label="资产名称">{asset.name}</Descriptions.Item>
          <Descriptions.Item label="类型">
            <Tag>{typeLabels[asset.type]}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="状态">
            <Tag color={statusColors[asset.status]}>{statusLabels[asset.status]}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="分类">{asset.category || '-'}</Descriptions.Item>
          <Descriptions.Item label="子分类">{asset.subcategory || '-'}</Descriptions.Item>
          <Descriptions.Item label="序列号">{asset.serialNumber || '-'}</Descriptions.Item>
          <Descriptions.Item label="型号">{asset.model || '-'}</Descriptions.Item>
          <Descriptions.Item label="制造商">{asset.manufacturer || '-'}</Descriptions.Item>
          <Descriptions.Item label="供应商">{asset.vendor || '-'}</Descriptions.Item>
          <Descriptions.Item label="分配给">{asset.assignedToName || '-'}</Descriptions.Item>
          <Descriptions.Item label="所属部门">{asset.department || '-'}</Descriptions.Item>
          <Descriptions.Item label="位置">{asset.location || '-'}</Descriptions.Item>
          <Descriptions.Item label="采购日期">{asset.purchaseDate || '-'}</Descriptions.Item>
          <Descriptions.Item label="采购价格">
            {asset.purchasePrice ? `¥${asset.purchasePrice}` : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="保修期到期">{asset.warrantyExpiry || '-'}</Descriptions.Item>
          <Descriptions.Item label="支持期到期">{asset.supportExpiry || '-'}</Descriptions.Item>
          <Descriptions.Item label="创建时间">
            {dayjs(asset.createdAt).format('YYYY-MM-DD HH:mm')}
          </Descriptions.Item>
          <Descriptions.Item label="更新时间">
            {dayjs(asset.updatedAt).format('YYYY-MM-DD HH:mm')}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {asset.description && (
        <Card title="描述">
          <Text>{asset.description}</Text>
        </Card>
      )}

      {asset.specifications && Object.keys(asset.specifications).length > 0 && (
        <Card title="规格参数">
          <Descriptions column={1}>
            {Object.entries(asset.specifications).map(([key, value]) => (
              <Descriptions.Item key={key} label={key}>
                {value}
              </Descriptions.Item>
            ))}
          </Descriptions>
        </Card>
      )}

      {asset.tags && asset.tags.length > 0 && (
        <Card title="标签">
          <Space wrap>
            {asset.tags.map((tag, index) => (
              <Tag key={index}>{tag}</Tag>
            ))}
          </Space>
        </Card>
      )}

      <Card>
        <Space>
          <Button type="primary" onClick={() => router.push(`/assets/${asset.id}/edit`)}>
            编辑
          </Button>
          {asset.status === 'available' && (
            <Button
              icon={<UserIcon />}
              onClick={() => {
                loadUsers();
                setAssignModalVisible(true);
              }}
            >
              分配资产
            </Button>
          )}
          {asset.status !== 'retired' && asset.status !== 'disposed' && (
            <Button danger onClick={handleRetire}>
              退役资产
            </Button>
          )}
        </Space>
      </Card>

      <Modal
        title="分配资产"
        open={assignModalVisible}
        onOk={handleAssign}
        onCancel={() => setAssignModalVisible(false)}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="assignedTo"
            label="分配给用户"
            rules={[{ required: true, message: '请选择用户' }]}
          >
            <Select
              placeholder="选择用户"
              showSearch
              optionFilterProp="children"
              loading={usersLoading}
              notFoundContent={usersLoading ? '加载中...' : '暂无用户'}
            >
              {users.map((user) => (
                <Option key={user.id} value={user.id}>
                  {user.name || user.username}
                </Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
};

export default AssetDetail;
