'use client';

/**
 * 许可证详情组件
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
  Progress,
  Modal,
  Select,
  App,
  Typography,
} from 'antd';
import { ArrowLeft, User as UserIcon, Key } from 'lucide-react';
import { useParams, useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import type { License } from '@/lib/api/asset-api';
import { AssetApi } from '@/lib/api/asset-api';
import { serverErrorMessage } from '@/lib/api/http-client';
import { useUserPickerQuery } from '@/lib/hooks/useUserListQuery';

const { Title, Text } = Typography;

// 状态颜色映射
const statusColors: Record<string, string> = {
  active: 'success',
  expired: 'error',
  'expiring-soon': 'warning',
  depleted: 'default',
};

const statusLabels: Record<string, string> = {
  active: 'valid',
  expired: 'Expired',
  'expiring-soon': 'Expiring soon',
  depleted: 'Exhausted',
};

const typeLabels: Record<string, string> = {
  perpetual: 'Permanent',
  subscription: 'Subscribe',
  'per-user': 'By user',
  'per-seat': 'By seat',
  site: 'site',
};

const LicenseDetail: React.FC = () => {
  const { id } = useParams() as { id: string };
  const router = useRouter();
  const { message, modal } = App.useApp();
  const [loading, setLoading] = useState(true);
  const [license, setLicense] = useState<License | null>(null);
  const [assignModalVisible, setAssignModalVisible] = useState(false);
  const [selectedUsers, setSelectedUsers] = useState<number[]>([]);
  const [clearingUsers, setClearingUsers] = useState(false);
  // 选人下拉走共享缓存（5 分钟内不重复请求），不再「开一次弹窗拉一次」。
  const { users, isLoading: usersLoading } = useUserPickerQuery();

  useEffect(() => {
    if (id) {
      loadDetail();
    }
  }, [id]);

  const loadDetail = async () => {
    setLoading(true);
    try {
      const data = await AssetApi.getLicense(Number(id));
      setLicense(data);
    } catch (error) {
      console.error('Failed to load license:', error);
      message.error('Failed to load license details');
    } finally {
      setLoading(false);
    }
  };

  const handleAssign = async () => {
    if (selectedUsers.length === 0) {
      message.warning('Please select a user');
      return;
    }
    try {
      await AssetApi.assignLicenseUsers(Number(id), selectedUsers);
      message.success('Allocation successful');
      setAssignModalVisible(false);
      loadDetail();
    } catch (error) {
      message.error('Assignment failed');
    }
  };

  const handleClearUsers = () => {
    if (!license || clearingUsers) {
      return;
    }
    modal.confirm({
      title: 'Clear license authorization?',
      content: `Will remove "${license.name}" ${license.users?.length ?? 0} authorized users and released ${license.usedQuantity} seats.`,
      okText: 'Clear authorization',
      cancelText: 'Cancel',
      okButtonProps: { danger: true },
      onOk: async () => {
        setClearingUsers(true);
        try {
          const updated = await AssetApi.updateLicense(Number(id), { users: [] });
          setLicense(updated);
          message.success('The authorization has been cleared and the seat has been released');
        } catch (error) {
          message.error(serverErrorMessage(error, 'Failed to clear authorization, please try again later.'));
        } finally {
          setClearingUsers(false);
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

  if (!license) {
    return (
      <Card>
        <Result
          status='404'
          title='404'
          subTitle="Sorry, the license you accessed does not exist"
          extra={
            <Button type='primary' onClick={() => router.push('/licenses')}>

              Back to List
            </Button>
          }
        />
      </Card>
    );
  }

  const usagePercent =
    license.totalQuantity > 0 ? (license.usedQuantity / license.totalQuantity) * 100 : 0;

  return (
    <Space orientation='vertical' style={{ width: '100%' }} size='large'>
      <Card>
        <div style={{ marginBottom: 24 }}>
          <Button
            icon={<ArrowLeft />}
            onClick={() => router.push('/licenses')}
            style={{ marginBottom: 16 }}
          >

            Back to List
          </Button>
          <div
            style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}
          >
            <div>
              <Title level={3} style={{ marginBottom: 8 }}>
                {license.name}
              </Title>
              {license.licenseKey && <Text type='secondary'>License key: {license.licenseKey}</Text>}
            </div>
            <Space>
              <Tag
                color={statusColors[license.status]}
                style={{ padding: '4px 12px', fontSize: 14 }}
              >
                {statusLabels[license.status]}
              </Tag>
              <Tag>{typeLabels[license.licenseType]}</Tag>
            </Space>
          </div>
        </div>

        <Card type='inner' style={{ marginBottom: 16 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <div style={{ flex: 1 }}>
              <Text>Usage</Text>
              <Progress
                percent={Math.round(usagePercent)}
                status={
                  usagePercent >= 100 ? 'exception' : usagePercent >= 80 ? 'normal' : 'success'
                }
                format={() => `${license.usedQuantity} / ${license.totalQuantity}`}
              />
            </div>
            <div style={{ textAlign: 'center', minWidth: 100 }}>
              <div style={{ fontSize: 24, fontWeight: 'bold' }}>{license.availableQuantity}</div>
              <div>Available quantity</div>
            </div>
          </div>
        </Card>

        <Descriptions bordered column={2}>
          <Descriptions.Item label="LicensesName">{license.name}</Descriptions.Item>
          <Descriptions.Item label="License type">
            <Tag>{typeLabels[license.licenseType]}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="Status">
            <Tag color={statusColors[license.status]}>{statusLabels[license.status]}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="Supplier">{license.vendor || '-'}</Descriptions.Item>
          <Descriptions.Item label="Total quantity">{license.totalQuantity}</Descriptions.Item>
          <Descriptions.Item label="Already used">{license.usedQuantity}</Descriptions.Item>
          <Descriptions.Item label="Purchase date">{license.purchaseDate || '-'}</Descriptions.Item>
          <Descriptions.Item label="Purchase price">
            {license.purchasePrice ? `¥${license.purchasePrice}` : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="Expiration date">
            <Text
              style={{
                color:
                  license.status === 'expired'
                    ? '#ff4d4f'
                    : license.status === 'expiring-soon'
                      ? '#faad14'
                      : undefined,
              }}
            >
              {license.expiryDate || 'Permanent'}
            </Text>
          </Descriptions.Item>
          <Descriptions.Item label="Renewal cost">{license.renewalCost || '-'}</Descriptions.Item>
          <Descriptions.Item label="Support Supplier">{license.supportVendor || '-'}</Descriptions.Item>
          <Descriptions.Item label="Support Contact">
            {license.supportContact || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="Created">
            {dayjs(license.createdAt).format('YYYY-MM-DD HH:mm')}
          </Descriptions.Item>
          <Descriptions.Item label="Updated At">
            {dayjs(license.updatedAt).format('YYYY-MM-DD HH:mm')}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {license.description && (
        <Card title="Description">
          <Text>{license.description}</Text>
        </Card>
      )}

      {license.notes && (
        <Card title="Notes">
          <Text>{license.notes}</Text>
        </Card>
      )}

      {license.users && license.users.length > 0 && (
        <Card
          title="Authorized user"
          extra={
            <Button danger size='small' loading={clearingUsers} onClick={handleClearUsers}>

              Clear authorization
            </Button>
          }
        >
          <Space wrap>
            {license.users.map((userId, index) => (
              <Tag key={userId} icon={<UserIcon />}>
                {license.userNames?.[index] || `Users ${userId}`}
              </Tag>
            ))}
          </Space>
        </Card>
      )}

      {license.tags && license.tags.length > 0 && (
        <Card title="Tags">
          <Space wrap>
            {license.tags.map((tag, index) => (
              <Tag key={index}>{tag}</Tag>
            ))}
          </Space>
        </Card>
      )}

      <Card>
        <Space>
          <Button type='primary' onClick={() => router.push(`/licenses/${license.id}/edit`)}>

            Edit
          </Button>
          {license.status === 'active' && license.availableQuantity > 0 && (
            <Button icon={<UserIcon />} onClick={() => setAssignModalVisible(true)}>

              assigned to user
            </Button>
          )}
        </Space>
      </Card>

      <Modal
        title="Assign license to user"
        open={assignModalVisible}
        onOk={handleAssign}
        onCancel={() => setAssignModalVisible(false)}
      >
        <div style={{ marginBottom: 16 }}>
          <Text>Available quantity: {license.availableQuantity}</Text>
        </div>
        <Select
          mode='multiple'
          style={{ width: '100%' }}
          placeholder="Choose user"
          value={selectedUsers}
          onChange={setSelectedUsers}
          maxCount={license.availableQuantity}
          optionFilterProp='children'
          loading={usersLoading}
          notFoundContent={usersLoading ? 'Loading...' : 'No users yet'}
          options={users.map(user => ({
            value: user.id,
            label: user.name || user.username,
          }))}
        />
      </Modal>
    </Space>
  );
};

export default LicenseDetail;
