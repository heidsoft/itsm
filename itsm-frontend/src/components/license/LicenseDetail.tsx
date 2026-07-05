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
  message,
  Typography,
} from 'antd';
import { ArrowLeft, User, Key } from 'lucide-react';
import { useParams, useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import type { License } from '@/lib/api/asset-api';
import { AssetApi } from '@/lib/api/asset-api';

const { Title, Text } = Typography;
const { Option } = Select;

// 状态颜色映射
const statusColors: Record<string, string> = {
  active: 'success',
  expired: 'error',
  'expiring-soon': 'warning',
  depleted: 'default',
};

const statusLabels: Record<string, string> = {
  active: '有效',
  expired: '已过期',
  'expiring-soon': '即将过期',
  depleted: '已耗尽',
};

const typeLabels: Record<string, string> = {
  perpetual: '永久',
  subscription: '订阅',
  'per-user': '按用户',
  'per-seat': '按席位',
  site: '站点',
};

const LicenseDetail: React.FC = () => {
  const { id } = useParams() as { id: string };
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [license, setLicense] = useState<License | null>(null);
  const [assignModalVisible, setAssignModalVisible] = useState(false);
  const [selectedUsers, setSelectedUsers] = useState<number[]>([]);

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
    } finally {
      setLoading(false);
    }
  };

  const handleAssign = async () => {
    if (selectedUsers.length === 0) {
      message.warning('请选择用户');
      return;
    }
    try {
      await AssetApi.assignLicenseUsers(Number(id), selectedUsers);
      message.success('分配成功');
      setAssignModalVisible(false);
      loadDetail();
    } catch (error) {
      message.error('分配失败');
    }
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
          status="404"
          title="404"
          subTitle="抱歉，您访问的许可证不存在"
          extra={
            <Button type="primary" onClick={() => router.push('/licenses')}>
              返回列表
            </Button>
          }
        />
      </Card>
    );
  }

  const usagePercent =
    license.totalQuantity > 0 ? (license.usedQuantity / license.totalQuantity) * 100 : 0;

  return (
    <Space orientation="vertical" style={{ width: '100%' }} size="large">
      <Card>
        <div style={{ marginBottom: 24 }}>
          <Button
            icon={<ArrowLeft />}
            onClick={() => router.push('/licenses')}
            style={{ marginBottom: 16 }}
          >
            返回列表
          </Button>
          <div
            style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}
          >
            <div>
              <Title level={3} style={{ marginBottom: 8 }}>
                {license.name}
              </Title>
              {license.licenseKey && (
                <Text type="secondary">许可证密钥: {license.licenseKey}</Text>
              )}
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

        <Card type="inner" style={{ marginBottom: 16 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <div style={{ flex: 1 }}>
              <Text>使用情况</Text>
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
              <div>可用数量</div>
            </div>
          </div>
        </Card>

        <Descriptions bordered column={2}>
          <Descriptions.Item label="许可证名称">{license.name}</Descriptions.Item>
          <Descriptions.Item label="许可证类型">
            <Tag>{typeLabels[license.licenseType]}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="状态">
            <Tag color={statusColors[license.status]}>{statusLabels[license.status]}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="供应商">{license.vendor || '-'}</Descriptions.Item>
          <Descriptions.Item label="总数量">{license.totalQuantity}</Descriptions.Item>
          <Descriptions.Item label="已使用">{license.usedQuantity}</Descriptions.Item>
          <Descriptions.Item label="采购日期">{license.purchaseDate || '-'}</Descriptions.Item>
          <Descriptions.Item label="采购价格">
            {license.purchasePrice ? `¥${license.purchasePrice}` : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="到期日期">
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
              {license.expiryDate || '永久'}
            </Text>
          </Descriptions.Item>
          <Descriptions.Item label="续费成本">{license.renewalCost || '-'}</Descriptions.Item>
          <Descriptions.Item label="支持供应商">{license.supportVendor || '-'}</Descriptions.Item>
          <Descriptions.Item label="支持联系方式">
            {license.supportContact || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="创建时间">
            {dayjs(license.createdAt).format('YYYY-MM-DD HH:mm')}
          </Descriptions.Item>
          <Descriptions.Item label="更新时间">
            {dayjs(license.updatedAt).format('YYYY-MM-DD HH:mm')}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {license.description && (
        <Card title="描述">
          <Text>{license.description}</Text>
        </Card>
      )}

      {license.notes && (
        <Card title="备注">
          <Text>{license.notes}</Text>
        </Card>
      )}

      {license.users && license.users.length > 0 && (
        <Card title="授权用户">
          <Space wrap>
            {license.users.map((userId, index) => (
              <Tag key={index} icon={<User />}>
                {license.userNames?.[index] || `用户 ${userId}`}
              </Tag>
            ))}
          </Space>
        </Card>
      )}

      {license.tags && license.tags.length > 0 && (
        <Card title="标签">
          <Space wrap>
            {license.tags.map((tag, index) => (
              <Tag key={index}>{tag}</Tag>
            ))}
          </Space>
        </Card>
      )}

      <Card>
        <Space>
          <Button type="primary" onClick={() => router.push(`/licenses/${license.id}`)}>
            编辑
          </Button>
          {license.status === 'active' && license.availableQuantity > 0 && (
            <Button icon={<User />} onClick={() => setAssignModalVisible(true)}>
              分配给用户
            </Button>
          )}
        </Space>
      </Card>

      <Modal
        title="分配许可证给用户"
        open={assignModalVisible}
        onOk={handleAssign}
        onCancel={() => setAssignModalVisible(false)}
      >
        <div style={{ marginBottom: 16 }}>
          <Text>可用数量: {license.availableQuantity}</Text>
        </div>
        <Select
          mode="multiple"
          style={{ width: '100%' }}
          placeholder="选择用户"
          value={selectedUsers}
          onChange={setSelectedUsers}
          maxCount={license.availableQuantity}
        >
          {/* 这里需要添加用户列表 */}
          <Option value={1}>用户1</Option>
          <Option value={2}>用户2</Option>
        </Select>
      </Modal>
    </Space>
  );
};

export default LicenseDetail;
