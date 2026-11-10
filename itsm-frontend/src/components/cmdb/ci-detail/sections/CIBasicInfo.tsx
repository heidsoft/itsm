/**
 * CI 基本信息标签页组件
 */

import React from 'react';
import { Descriptions } from 'antd';
import dayjs from 'dayjs';
import type { ConfigurationItem, CIType } from '@/types/biz/cmdb';
import { normalizeSchemaFields } from '../../ci-editor-shared';
import { STATUS_COLORS } from '../constants';

interface CIBasicInfoProps {
  ci: ConfigurationItem;
  typeInfo?: CIType;
}

export const CIBasicInfo: React.FC<CIBasicInfoProps> = ({ ci, typeInfo }) => {
  const schemaLabels = new Map(
    normalizeSchemaFields(typeInfo?.attributeSchema).map(field => [field.key, field.label])
  );
  const dynamicAttributes =
    ci.attributes && typeof ci.attributes === 'object' ? Object.entries(ci.attributes) : [];
  const formatAttributeValue = (value: unknown) => {
    if (value === null || value === undefined || value === '') return '-';
    if (typeof value === 'boolean') return value ? 'Yes' : 'No';
    if (typeof value === 'object') return JSON.stringify(value);
    return String(value);
  };

  return (
    <Descriptions bordered column={2}>
      <Descriptions.Item label="Asset Classification">
        {typeInfo?.name || `Type ${ci.ciTypeId}`}
      </Descriptions.Item>
      <Descriptions.Item label="Asset Tag">{ci.assetTag || '-'}</Descriptions.Item>
      <Descriptions.Item label="Serial Number">{ci.serialNumber || '-'}</Descriptions.Item>
      <Descriptions.Item label="Model">{ci.model || '-'}</Descriptions.Item>
      <Descriptions.Item label="Vendor">{ci.vendor || '-'}</Descriptions.Item>
      <Descriptions.Item label="Location">{ci.location || '-'}</Descriptions.Item>
      <Descriptions.Item label="Environment">{ci.environment || '-'}</Descriptions.Item>
      <Descriptions.Item label="Criticality">{ci.criticality || '-'}</Descriptions.Item>
      <Descriptions.Item label="Assigned To">{ci.assignedTo || '-'}</Descriptions.Item>
      <Descriptions.Item label="Owned By">{ci.ownedBy || '-'}</Descriptions.Item>
      <Descriptions.Item label="Discovery Source">{ci.discoverySource || '-'}</Descriptions.Item>
      <Descriptions.Item label="Data Source">{ci.source || '-'}</Descriptions.Item>
      <Descriptions.Item label="Cloud Provider">{ci.cloudProvider || '-'}</Descriptions.Item>
      <Descriptions.Item label="Cloud account">{ci.cloudAccountId || '-'}</Descriptions.Item>
      <Descriptions.Item label='Region'>{ci.cloudRegion || '-'}</Descriptions.Item>
      <Descriptions.Item label='Zone'>{ci.cloudZone || '-'}</Descriptions.Item>
      <Descriptions.Item label="Cloud resource ID">{ci.cloudResourceId || '-'}</Descriptions.Item>
      <Descriptions.Item label="Cloud Resource Type">{ci.cloudResourceType || '-'}</Descriptions.Item>
      <Descriptions.Item label="Cloud resource reference ID">{ci.cloudResourceRefId || '-'}</Descriptions.Item>
      <Descriptions.Item label="Sync Status">{ci.cloudSyncStatus || '-'}</Descriptions.Item>
      <Descriptions.Item label="Sync time">
        {ci.cloudSyncTime ? dayjs(ci.cloudSyncTime).format('YYYY-MM-DD HH:mm:ss') : '-'}
      </Descriptions.Item>
      <Descriptions.Item label="Tenant to which it belongs">{ci.tenantId}</Descriptions.Item>
      <Descriptions.Item label="Created">
        {dayjs(ci.createdAt).format('YYYY-MM-DD HH:mm:ss')}
      </Descriptions.Item>
      <Descriptions.Item label="Last Update">
        {dayjs(ci.updatedAt).format('YYYY-MM-DD HH:mm:ss')}
      </Descriptions.Item>
      <Descriptions.Item label="Description" span={2}>
        {ci.description || 'None'}
      </Descriptions.Item>
      {dynamicAttributes.map(([key, value]) => (
        <Descriptions.Item key={key} label={schemaLabels.get(key) || key}>
          {formatAttributeValue(value)}
        </Descriptions.Item>
      ))}
    </Descriptions>
  );
};

CIBasicInfo.displayName = 'CIBasicInfo';
