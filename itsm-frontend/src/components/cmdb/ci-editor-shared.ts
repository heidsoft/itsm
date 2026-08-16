import { CIStatus, CIStatusLabels } from '@/constants/cmdb';
import type { CIType, CloudResource, CloudService } from '@/types/biz/cmdb';

import type { useI18n } from '@/lib/i18n/useI18n';

export const statusOptions = [CIStatus.ACTIVE, CIStatus.INACTIVE, CIStatus.MAINTENANCE];

export type TranslationFn = ReturnType<typeof useI18n>['t'];

export const buildEnvironmentOptions = (t: TranslationFn) => [
  { label: t('ciEditor.environmentOptions.production'), value: 'production' },
  { label: t('ciEditor.environmentOptions.staging'), value: 'staging' },
  { label: t('ciEditor.environmentOptions.development'), value: 'development' },
];

export const buildCriticalityOptions = (t: TranslationFn) => [
  { label: t('ciEditor.criticalityOptions.low'), value: 'low' },
  { label: t('ciEditor.criticalityOptions.medium'), value: 'medium' },
  { label: t('ciEditor.criticalityOptions.high'), value: 'high' },
  { label: t('ciEditor.criticalityOptions.critical'), value: 'critical' },
];

export const buildSourceOptions = (t: TranslationFn) => [
  { label: t('ciEditor.sourceOptions.manual'), value: 'manual' },
  { label: t('ciEditor.sourceOptions.discovery'), value: 'discovery' },
  { label: t('ciEditor.sourceOptions.import'), value: 'import' },
];

export const buildCloudProviderOptions = (t: TranslationFn) => [
  { label: t('ciEditor.cloudProviderOptions.aliyun'), value: 'aliyun' },
  { label: t('ciEditor.cloudProviderOptions.huawei'), value: 'huawei' },
  { label: t('ciEditor.cloudProviderOptions.tencent'), value: 'tencent' },
  { label: t('ciEditor.cloudProviderOptions.azure'), value: 'azure' },
  { label: t('ciEditor.cloudProviderOptions.aws'), value: 'aws' },
  { label: t('ciEditor.cloudProviderOptions.onprem'), value: 'onprem' },
];

export const buildCloudSyncStatusOptions = (t: TranslationFn) => [
  { label: t('ciEditor.cloudSyncStatusOptions.success'), value: 'success' },
  { label: t('ciEditor.cloudSyncStatusOptions.failed'), value: 'failed' },
  { label: t('ciEditor.cloudSyncStatusOptions.unknown'), value: 'unknown' },
];

export type SchemaField = {
  key: string;
  label: string;
  type?: string;
  required?: boolean;
  options?: string[];
  placeholder?: string;
  validation?: {
    minValue?: number;
    maxValue?: number;
    precision?: number;
    pattern?: string;
  };
};

export interface CIFormValues {
  name: string;
  ciTypeId: number;
  status: CIStatus;
  description?: string;
  attributes?: string;
  serialNumber?: string;
  model?: string;
  vendor?: string;
  location?: string;
  assetTag?: string;
  assignedTo?: string;
  ownedBy?: string;
  environment?: string;
  criticality?: string;
  discoverySource?: string;
  source?: string;
  cloudProvider?: string;
  cloudAccountId?: string;
  cloudRegion?: string;
  cloudZone?: string;
  cloudResourceId?: string;
  cloudResourceType?: string;
  cloudSyncStatus?: string;
  cloudResourceRefId?: number;
  cloudMetadata?: Record<string, {} | undefined>;
  customAttributes?: Record<string, {} | undefined>;
}

export const normalizeSchemaFields = (schema: unknown): SchemaField[] => {
  if (!schema) return [];
  if (typeof schema === 'string') {
    if (!schema.trim()) return [];
    try {
      return normalizeSchemaFields(JSON.parse(schema));
    } catch {
      return [];
    }
  }
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
            ? record.options
                .map(option =>
                  typeof option === 'string' || typeof option === 'number' ? String(option) : ''
                )
                .filter(Boolean)
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
            ? entry.options
                .map(option =>
                  typeof option === 'string' || typeof option === 'number' ? String(option) : ''
                )
                .filter(Boolean)
            : undefined,
          placeholder: typeof entry.placeholder === 'string' ? entry.placeholder : undefined,
        };
      }
      return { key, label: key };
    });
  }

  return [];
};

export const resolveEffectiveTypeSchemaFields = (
  types: CIType[],
  selectedTypeId?: number
): SchemaField[] => {
  if (!selectedTypeId) return [];
  const byID = new Map(types.map(type => [type.id, type]));
  const visited = new Set<number>();
  const chain: CIType[] = [];
  let selectedType = byID.get(selectedTypeId);
  while (selectedType && !visited.has(selectedType.id)) {
    visited.add(selectedType.id);
    chain.unshift(selectedType);
    selectedType = selectedType.parentTypeId ? byID.get(selectedType.parentTypeId) : undefined;
  }
  const merged = new Map<string, SchemaField>();
  for (const type of chain) {
    for (const field of normalizeSchemaFields(type.attributeSchema)) {
      merged.set(field.key, field);
    }
  }
  return Array.from(merged.values());
};

export const buildCloudResourceOptions = (
  cloudResources: CloudResource[],
  cloudServiceMap: Map<number, CloudService>,
  t: TranslationFn
) =>
  cloudResources.map(resource => {
    const service = cloudServiceMap.get(resource.serviceId);
    const unknownType = t('ciEditor.unknownType');
    const label = `${resource.resourceName || resource.resourceId}（${service?.resourceTypeName || unknownType}）`;
    return {
      label,
      value: resource.id,
    };
  });

export const getStatusSelectOptions = (t?: TranslationFn) =>
  statusOptions.map(status => ({
    label: t ? t(`ciEditor.ciStatus.${status}`) : CIStatusLabels[status],
    value: status,
  }));

export const compactRecord = (record?: Record<string, unknown>) => {
  if (!record) return undefined;
  const entries = Object.entries(record).filter(
    ([, value]) => value !== undefined && value !== null && value !== ''
  );
  return entries.length > 0 ? Object.fromEntries(entries) : undefined;
};

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
