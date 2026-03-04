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
        {typeInfo?.name || `类型 ${ci.ci_type_id}`}
      </Descriptions.Item>
      <Descriptions.Item label="资产标签">{ci.asset_tag || '-'}</Descriptions.Item>
      <Descriptions.Item label="序列号">{ci.serial_number || '-'}</Descriptions.Item>
      <Descriptions.Item label="型号">{ci.model || '-'}</Descriptions.Item>
      <Descriptions.Item label="厂商">{ci.vendor || '-'}</Descriptions.Item>
      <Descriptions.Item label="所在位置">{ci.location || '-'}</Descriptions.Item>
      <Descriptions.Item label="环境">{ci.environment || '-'}</Descriptions.Item>
      <Descriptions.Item label="重要性">{ci.criticality || '-'}</Descriptions.Item>
      <Descriptions.Item label="分配给">{ci.assigned_to || '-'}</Descriptions.Item>
      <Descriptions.Item label="拥有者">{ci.owned_by || '-'}</Descriptions.Item>
      <Descriptions.Item label="发现源">{ci.discovery_source || '-'}</Descriptions.Item>
      <Descriptions.Item label="数据来源">{ci.source || '-'}</Descriptions.Item>
      <Descriptions.Item label="云厂商">{ci.cloud_provider || '-'}</Descriptions.Item>
      <Descriptions.Item label="云账号">{ci.cloud_account_id || '-'}</Descriptions.Item>
      <Descriptions.Item label="Region">{ci.cloud_region || '-'}</Descriptions.Item>
      <Descriptions.Item label="Zone">{ci.cloud_zone || '-'}</Descriptions.Item>
      <Descriptions.Item label="云资源ID">{ci.cloud_resource_id || '-'}</Descriptions.Item>
      <Descriptions.Item label="云资源类型">
        {ci.cloud_resource_type || '-'}
      </Descriptions.Item>
      <Descriptions.Item label="云资源引用ID">
        {ci.cloud_resource_ref_id || '-'}
      </Descriptions.Item>
      <Descriptions.Item label="同步状态">{ci.cloud_sync_status || '-'}</Descriptions.Item>
      <Descriptions.Item label="同步时间">
        {ci.cloud_sync_time ? dayjs(ci.cloud_sync_time).format('YYYY-MM-DD HH:mm:ss') : '-'}
      </Descriptions.Item>
      <Descriptions.Item label="所属租户">{ci.tenant_id}</Descriptions.Item>
      <Descriptions.Item label="创建时间">
        {dayjs(ci.created_at).format('YYYY-MM-DD HH:mm:ss')}
      </Descriptions.Item>
      <Descriptions.Item label="最后更新">
        {dayjs(ci.updated_at).format('YYYY-MM-DD HH:mm:ss')}
      </Descriptions.Item>
      <Descriptions.Item label="描述" span={2}>
        {ci.description || '无'}
      </Descriptions.Item>
    </Descriptions>
  );
};

CIBasicInfo.displayName = 'CIBasicInfo';
