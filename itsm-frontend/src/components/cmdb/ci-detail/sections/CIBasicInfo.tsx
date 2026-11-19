/**
 * CI 基本信息标签页组件
 */

import React from 'react';
import { Descriptions } from 'antd';
import dayjs from 'dayjs';
import type { ConfigurationItem, CIType } from '@/types/biz/cmdb';
import {
  buildCloudProviderOptions,
  buildCloudSyncStatusOptions,
  buildCriticalityOptions,
  buildEnumLabelResolver,
  buildEnvironmentOptions,
  buildSourceOptions,
  normalizeSchemaFields,
} from '../../ci-editor-shared';
import { useI18n } from '@/lib/i18n';

interface CIBasicInfoProps {
  ci: ConfigurationItem;
  typeInfo?: CIType;
}

export const CIBasicInfo: React.FC<CIBasicInfoProps> = ({ ci, typeInfo }) => {
  const { t } = useI18n();
  // 枚举展示一律复用 create/edit 表单的同一份选项列表（单一词表），
  // 避免同一值在表单显示「生产/中」、在详情页显示 production/medium。
  const environmentLabel = buildEnumLabelResolver(buildEnvironmentOptions(t));
  const criticalityLabel = buildEnumLabelResolver(buildCriticalityOptions(t));
  const sourceLabel = buildEnumLabelResolver(buildSourceOptions(t));
  const cloudProviderLabel = buildEnumLabelResolver(buildCloudProviderOptions(t));
  const cloudSyncStatusLabel = buildEnumLabelResolver(buildCloudSyncStatusOptions(t));

  // ci_types.name 是租户内唯一代码（server/database/...），中文名在 description。
  // 展示层与列表页 getCITypeLabel 保持同一口径：优先中文描述，再回退代码/ID。
  const typeLabel =
    typeInfo?.description?.trim() || typeInfo?.name || `类型 ${ci.ciTypeId}`;

  const schemaLabels = new Map(
    normalizeSchemaFields(typeInfo?.attributeSchema).map(field => [field.key, field.label])
  );
  const dynamicAttributes =
    ci.attributes && typeof ci.attributes === 'object' ? Object.entries(ci.attributes) : [];
  const formatAttributeValue = (value: unknown) => {
    if (value === null || value === undefined || value === '') return '-';
    if (typeof value === 'boolean') return value ? '是' : '否';
    if (typeof value === 'object') return JSON.stringify(value);
    return String(value);
  };

  return (
    <Descriptions bordered column={2}>
      <Descriptions.Item label='资产分类'>{typeLabel}</Descriptions.Item>
      <Descriptions.Item label='资产标签'>{ci.assetTag || '-'}</Descriptions.Item>
      <Descriptions.Item label='序列号'>{ci.serialNumber || '-'}</Descriptions.Item>
      <Descriptions.Item label='型号'>{ci.model || '-'}</Descriptions.Item>
      <Descriptions.Item label='厂商'>{ci.vendor || '-'}</Descriptions.Item>
      <Descriptions.Item label='所在位置'>{ci.location || '-'}</Descriptions.Item>
      <Descriptions.Item label='环境'>{environmentLabel(ci.environment)}</Descriptions.Item>
      <Descriptions.Item label='重要性'>{criticalityLabel(ci.criticality)}</Descriptions.Item>
      <Descriptions.Item label='分配给'>{ci.assignedTo || '-'}</Descriptions.Item>
      <Descriptions.Item label='拥有者'>{ci.ownedBy || '-'}</Descriptions.Item>
      <Descriptions.Item label='发现源'>{ci.discoverySource || '-'}</Descriptions.Item>
      <Descriptions.Item label='数据来源'>{sourceLabel(ci.source)}</Descriptions.Item>
      <Descriptions.Item label='云厂商'>
        {cloudProviderLabel(ci.cloudProvider)}
      </Descriptions.Item>
      <Descriptions.Item label='云账号'>{ci.cloudAccountId || '-'}</Descriptions.Item>
      <Descriptions.Item label='Region'>{ci.cloudRegion || '-'}</Descriptions.Item>
      <Descriptions.Item label='Zone'>{ci.cloudZone || '-'}</Descriptions.Item>
      <Descriptions.Item label='云资源ID'>{ci.cloudResourceId || '-'}</Descriptions.Item>
      <Descriptions.Item label='云资源类型'>{ci.cloudResourceType || '-'}</Descriptions.Item>
      <Descriptions.Item label='云资源引用ID'>{ci.cloudResourceRefId || '-'}</Descriptions.Item>
      <Descriptions.Item label='同步状态'>
        {cloudSyncStatusLabel(ci.cloudSyncStatus)}
      </Descriptions.Item>
      <Descriptions.Item label='同步时间'>
        {ci.cloudSyncTime ? dayjs(ci.cloudSyncTime).format('YYYY-MM-DD HH:mm:ss') : '-'}
      </Descriptions.Item>
      <Descriptions.Item label='所属租户'>{ci.tenantId}</Descriptions.Item>
      <Descriptions.Item label='创建时间'>
        {dayjs(ci.createdAt).format('YYYY-MM-DD HH:mm:ss')}
      </Descriptions.Item>
      <Descriptions.Item label='最后更新'>
        {dayjs(ci.updatedAt).format('YYYY-MM-DD HH:mm:ss')}
      </Descriptions.Item>
      <Descriptions.Item label='描述' span={2}>
        {ci.description || '无'}
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
