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
import { useUserPickerQuery } from '@/lib/hooks/useUserListQuery';

const { Title, Text } = Typography;

// 状态颜色映射
const statusColors: Record<string, string> = {
  available: 'success',
  'in-use': 'processing',
  maintenance: 'warning',
  retired: 'default',
  disposed: 'error',
};

const statusLabels: Record<string, string> = {
  available: 'Available',
  'in-use': 'In Use',
  maintenance: 'Maintenance',
  retired: 'Retired',
  disposed: 'Disposed',
};

const typeLabels: Record<string, string> = {
  hardware: 'Hardware',
  software: 'Software',
  cloud: 'Cloud Resources',
  license: 'Licenses',
};

const AssetDetail: React.FC = () => {
  const { id } = useParams() as { id: string };
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [asset, setAsset] = useState<Asset | null>(null);
  const [assignModalVisible, setAssignModalVisible] = useState(false);
  // 选人下拉走共享缓存（5 分钟内不重复请求），不再「开一次弹窗拉一次」。
  const { users, isLoading: usersLoading } = useUserPickerQuery();
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
      message.error('Failed to load asset details');
    } finally {
      setLoading(false);
    }
  };

  const handleAssign = async () => {
    try {
      const values = await form.validateFields();
      await AssetApi.assignAsset(Number(id), values.assignedTo);
      message.success('Allocation successful');
      setAssignModalVisible(false);
      loadDetail();
    } catch (error) {
      message.error('Assignment failed');
    }
  };

  const handleRetire = async () => {
    Modal.confirm({
      title: 'Retirement confirmed',
      content: 'Are you sure you want to retire this asset?',
      onOk: async () => {
        try {
          await AssetApi.retireAsset(Number(id), 'Manual retirement');
          message.success('Retired successfully');
          loadDetail();
        } catch (error) {
          message.error('Retirement failed');
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
          subTitle="Sorry, the asset you visited does not exist"
          extra={
            <Button type="primary" onClick={() => router.push('/assets')}>

              Back to List
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

            Back to List
          </Button>
          <div
            style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}
          >
            <div>
              <Title level={3} style={{ marginBottom: 8 }}>
                {asset.name}
              </Title>
              <Text type="secondary">Asset number: {asset.assetNumber}</Text>
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
          <Descriptions.Item label="Asset Number">{asset.assetNumber}</Descriptions.Item>
          <Descriptions.Item label="Asset Name">{asset.name}</Descriptions.Item>
          <Descriptions.Item label="Type">
            <Tag>{typeLabels[asset.type]}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="Status">
            <Tag color={statusColors[asset.status]}>{statusLabels[asset.status]}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="Category">{asset.category || '-'}</Descriptions.Item>
          <Descriptions.Item label="Subcategory">{asset.subcategory || '-'}</Descriptions.Item>
          <Descriptions.Item label="Serial Number">{asset.serialNumber || '-'}</Descriptions.Item>
          <Descriptions.Item label="Model">{asset.model || '-'}</Descriptions.Item>
          <Descriptions.Item label="Manufacturer">{asset.manufacturer || '-'}</Descriptions.Item>
          <Descriptions.Item label="Supplier">{asset.vendor || '-'}</Descriptions.Item>
          <Descriptions.Item label="Assigned To">{asset.assignedToName || '-'}</Descriptions.Item>
          <Descriptions.Item label="Department">{asset.department || '-'}</Descriptions.Item>
          <Descriptions.Item label="Location">{asset.location || '-'}</Descriptions.Item>
          <Descriptions.Item label="Purchase date">{asset.purchaseDate || '-'}</Descriptions.Item>
          <Descriptions.Item label="Purchase price">
            {asset.purchasePrice ? `¥${asset.purchasePrice}` : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="Warranty expired">{asset.warrantyExpiry || '-'}</Descriptions.Item>
          <Descriptions.Item label="Support period expired">{asset.supportExpiry || '-'}</Descriptions.Item>
          <Descriptions.Item label="Created">
            {dayjs(asset.createdAt).format('YYYY-MM-DD HH:mm')}
          </Descriptions.Item>
          <Descriptions.Item label="Updated At">
            {dayjs(asset.updatedAt).format('YYYY-MM-DD HH:mm')}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {asset.description && (
        <Card title="Description">
          <Text>{asset.description}</Text>
        </Card>
      )}

      {asset.specifications && Object.keys(asset.specifications).length > 0 && (
        <Card title="Specifications">
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
        <Card title="Tags">
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

            Edit
          </Button>
          {asset.status === 'available' && (
            <Button
              icon={<UserIcon />}
              onClick={() => setAssignModalVisible(true)}
            >

              Assign assets
            </Button>
          )}
          {asset.status !== 'retired' && asset.status !== 'disposed' && (
            <Button danger onClick={handleRetire}>

              Retired assets
            </Button>
          )}
        </Space>
      </Card>

      <Modal
        title="Assign assets"
        open={assignModalVisible}
        onOk={handleAssign}
        onCancel={() => setAssignModalVisible(false)}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="assignedTo"
            label="assigned to user"
            rules={[{ required: true, message: 'Please select a user' }]}
          >
            <Select
              placeholder="Choose user"
              showSearch
              optionFilterProp="children"
              loading={usersLoading}
              notFoundContent={usersLoading ? 'Loading...' : 'No users yet'}
              options={users.map((user) => ({
                value: user.id,
                label: user.name || user.username,
              }))}
            />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
};

export default AssetDetail;
