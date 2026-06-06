import { CIStatus, CIStatusLabels } from '@/constants/cmdb';
import type { CloudResource, CloudService } from '@/types/biz/cmdb';

export const statusOptions = [CIStatus.ACTIVE, CIStatus.INACTIVE, CIStatus.MAINTENANCE];

export const environmentOptions = [
  { label: '生产', value: 'production' },
  { label: '预发布', value: 'staging' },
  { label: '开发', value: 'development' },
];

export const criticalityOptions = [
  { label: '低', value: 'low' },
  { label: '中', value: 'medium' },
  { label: '高', value: 'high' },
  { label: '关键', value: 'critical' },
];

export const sourceOptions = [
  { label: '手工录入', value: 'manual' },
  { label: '自动发现', value: 'discovery' },
  { label: '批量导入', value: 'import' },
];

export const cloudProviderOptions = [
  { label: '阿里云', value: 'aliyun' },
  { label: '华为云', value: 'huawei' },
  { label: '腾讯云', value: 'tencent' },
  { label: 'Azure', value: 'azure' },
  { label: '私有云', value: 'onprem' },
];

export const cloudSyncStatusOptions = [
  { label: '成功', value: 'success' },
  { label: '失败', value: 'failed' },
  { label: '未知', value: 'unknown' },
];

export type SchemaField = {
  key: string;
  label: string;
  type?: string;
  required?: boolean;
  options?: string[];
  placeholder?: string;
};

export interface CIFormValues {
  name: string;
  ci_type_id: number;
  status: CIStatus;
  description?: string;
  attributes?: string;
  serial_number?: string;
  model?: string;
  vendor?: string;
  location?: string;
  asset_tag?: string;
  assigned_to?: string;
  owned_by?: string;
  environment?: string;
  criticality?: string;
  discovery_source?: string;
  source?: string;
  cloud_provider?: string;
  cloud_account_id?: string;
  cloud_region?: string;
  cloud_zone?: string;
  cloud_resource_id?: string;
  cloud_resource_type?: string;
  cloud_sync_status?: string;
  cloud_resource_ref_id?: number;
  cloud_metadata?: Record<string, unknown>;
}

export const normalizeSchemaFields = (schema: unknown): SchemaField[] => {
  if (!schema) return [];
  if (Array.isArray(schema)) {
    return schema
      .map((item): SchemaField | null => {
        if (typeof item !== 'object' || item === null) return null;
        const record = item as Record<string, unknown>;
        const key = record.key || record.name;
        if (typeof key !== 'string' || !key) return null;

        return {
          key,
          label: typeof record.label === 'string' ? record.label : key,
          type: typeof record.type === 'string' ? record.type : undefined,
          required: Boolean(record.required),
          options: Array.isArray(record.options)
            ? record.options.filter((option): option is string => typeof option === 'string')
            : undefined,
          placeholder: typeof record.placeholder === 'string' ? record.placeholder : undefined,
        };
      })
      .filter((item): item is SchemaField => item !== null);
  }

  if (typeof schema === 'object') {
    const record = schema as Record<string, unknown>;
    if (Array.isArray(record.fields)) {
      return normalizeSchemaFields(record.fields);
    }

    return Object.entries(record).map(([key, value]): SchemaField => {
      if (typeof value === 'string') {
        return { key, label: value };
      }
      if (typeof value === 'object' && value !== null) {
        const entry = value as Record<string, unknown>;
        return {
          key,
          label: typeof entry.label === 'string' ? entry.label : key,
          type: typeof entry.type === 'string' ? entry.type : undefined,
          required: Boolean(entry.required),
          options: Array.isArray(entry.options)
            ? entry.options.filter((option): option is string => typeof option === 'string')
            : undefined,
          placeholder: typeof entry.placeholder === 'string' ? entry.placeholder : undefined,
        };
      }
      return { key, label: key };
    });
  }

  return [];
};

export const buildCloudResourceOptions = (
  cloudResources: CloudResource[],
  cloudServiceMap: Map<number, CloudService>
) =>
  cloudResources.map(resource => {
    const service = cloudServiceMap.get(resource.service_id);
    const label = `${resource.resource_name || resource.resource_id}（${service?.resource_type_name || '未知类型'}）`;
    return {
      label,
      value: resource.id,
    };
  });

export const getStatusSelectOptions = () =>
  statusOptions.map(status => ({
    label: CIStatusLabels[status],
    value: status,
  }));

export const extractCloudDataList = <T>(response: unknown): T[] => {
  if (Array.isArray(response)) {
    return response as T[];
  }
  if (response && typeof response === 'object') {
    const record = response as Record<string, unknown>;
    if (Array.isArray(record.items)) return record.items as T[];
    if (Array.isArray(record.data)) return record.data as T[];
  }
  return [];
};
