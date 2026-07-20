/**
 * CI 基本信息标签页组件
 */

import React from 'react';
import { Descriptions } from 'antd';
import dayjs from 'dayjs';
import type { ConfigurationItem, CIType } from '@/types/biz/cmdb';
import { STATUS_COLORS } from '../constants';

interface CIBasicInfoProps {
  ci: ConfigurationItem;
  typeInfo?: CIType;
}

export const CIBasicInfo: React.FC<CIBasicInfoProps> = ({ ci, typeInfo }) => {
  return (
    <Descriptions bordered column={2}>
      <Descriptions.Item label="资产分类">
        {typeInfo?.name || `类型 ${ci.ciTypeId}`}
      </Descriptions.Item>
      <Descriptions.Item label="资产标签">{ci.assetTag || '-'}</Descriptions.Item>
      <Descriptions.Item label="序列号">{ci.serialNumber || '-'}</Descriptions.Item>
      <Descriptions.Item label="型号">{ci.model || '-'}</Descriptions.Item>
      <Descriptions.Item label="厂商">{ci.vendor || '-'}</Descriptions.Item>
      <Descriptions.Item label="所在位置">{ci.location || '-'}</Descriptions.Item>
      <Descriptions.Item label="环境">{ci.environment || '-'}</Descriptions.Item>
      <Descriptions.Item label="重要性">{ci.criticality || '-'}</Descriptions.Item>
      <Descriptions.Item label="分配给">{ci.assignedTo || '-'}</Descriptions.Item>
      <Descriptions.Item label="拥有者">{ci.ownedBy || '-'}</Descriptions.Item>
      <Descriptions.Item label="发现源">{ci.discoverySource || '-'}</Descriptions.Item>
      <Descriptions.Item label="数据来源">{ci.source || '-'}</Descriptions.Item>
      <Descriptions.Item label="云厂商">{ci.cloudProvider || '-'}</Descriptions.Item>
      <Descriptions.Item label="云账号">{ci.cloudAccountId || '-'}</Descriptions.Item>
      <Descriptions.Item label="Region">{ci.cloudRegion || '-'}</Descriptions.Item>
      <Descriptions.Item label="Zone">{ci.cloudZone || '-'}</Descriptions.Item>
      <Descriptions.Item label="云资源ID">{ci.cloudResourceId || '-'}</Descriptions.Item>
      <Descriptions.Item label="云资源类型">
        {ci.cloudResourceType || '-'}
      </Descriptions.Item>
      <Descriptions.Item label="云资源引用ID">
        {ci.cloudResourceRefId || '-'}
      </Descriptions.Item>
      <Descriptions.Item label="同步状态">{ci.cloudSyncStatus || '-'}</Descriptions.Item>
      <Descriptions.Item label="同步时间">
        {ci.cloudSyncTime ? dayjs(ci.cloudSyncTime).format('YYYY-MM-DD HH:mm:ss') : '-'}
      </Descriptions.Item>
      <Descriptions.Item label="所属租户">{ci.tenantId}</Descriptions.Item>
      <Descriptions.Item label="创建时间">
        {dayjs(ci.createdAt).format('YYYY-MM-DD HH:mm:ss')}
      </Descriptions.Item>
      <Descriptions.Item label="最后更新">
        {dayjs(ci.updatedAt).format('YYYY-MM-DD HH:mm:ss')}
      </Descriptions.Item>
      <Descriptions.Item label="描述" span={2}>
        {ci.description || '无'}
      </Descriptions.Item>
    </Descriptions>
  );
};

CIBasicInfo.displayName = 'CIBasicInfo';
