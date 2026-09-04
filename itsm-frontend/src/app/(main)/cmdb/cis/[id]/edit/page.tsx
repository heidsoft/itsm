'use client';

import React, { useEffect, useMemo, useRef } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { App, Card, Form } from 'antd';

import { CIEditorForm } from '@/components/cmdb/CIEditorForm';
import { useUnsavedChangesGuard } from '@/components/cmdb/useUnsavedChangesGuard';
import type { CIFormValues, SchemaField } from '@/components/cmdb/ci-editor-shared';
import {
  compactRecord,
  extractCloudDataList,
  normalizeSchemaFields,
  resolveEffectiveTypeSchemaFields,
} from '@/components/cmdb/ci-editor-shared';
import { ManagementNotice, ManagementPageHeader } from '@/components/ui/ManagementPageHeader';
import {
  useCIQuery,
  useCITypesQuery,
  useCloudResourcesQuery,
  useCloudServicesQuery,
  useUpdateCIMutation,
} from '@/lib/hooks/useCMDB';
import type { CIType, CloudResource, CloudService, ConfigurationItem } from '@/types/biz/cmdb';

const EditCIPage: React.FC = () => {
  const router = useRouter();
  const { id } = useParams() as { id: string };
  const { message } = App.useApp();
  const [form] = Form.useForm<CIFormValues>();

  // React Query：CI 类型 / 云资源 / 云服务 / 当前 CI（缓存 + 自动重试）
  const typesQuery = useCITypesQuery();
  const cloudResourcesQuery = useCloudResourcesQuery();
  const cloudServicesQuery = useCloudServicesQuery();
  const ciQuery = useCIQuery(id);

  const types: CIType[] = (typesQuery.data as unknown as CIType[]) ?? [];
  const cloudResources: CloudResource[] =
    extractCloudDataList<CloudResource>(cloudResourcesQuery.data) ?? [];
  const cloudServices: CloudService[] =
    extractCloudDataList<CloudService>(cloudServicesQuery.data) ?? [];
  const ci = (ciQuery.data as unknown as ConfigurationItem) ?? null;

  const typesLoading = typesQuery.isLoading;
  const cloudLoading = cloudResourcesQuery.isLoading || cloudServicesQuery.isLoading;
  const loading = ciQuery.isLoading;

  // 错误提示
  useEffect(() => {
    if (typesQuery.isError) message.error('加载资产类型失败');
  }, [typesQuery.isError, message]);
  useEffect(() => {
    if (cloudResourcesQuery.isError && cloudServicesQuery.isError)
      message.error('加载云资源数据失败');
  }, [cloudResourcesQuery.isError, cloudServicesQuery.isError, message]);
  useEffect(() => {
    if (ciQuery.isError) message.error('加载配置项失败');
  }, [ciQuery.isError, message]);

  const schemaFieldsStateRef = useRef<SchemaField[]>([]);
  const typeSchemaFieldsStateRef = useRef<SchemaField[]>([]);
  const [schemaFields, setSchemaFields] = React.useState<SchemaField[]>([]);
  const [typeSchemaFields, setTypeSchemaFields] = React.useState<SchemaField[]>([]);
  // 维持原 ref 行为以便引用一致
  schemaFieldsStateRef.current = schemaFields;
  typeSchemaFieldsStateRef.current = typeSchemaFields;

  const { markDirty, clearDirty, handleCancel } = useUnsavedChangesGuard(router);

  const cloudServiceMap = useMemo(
    () => new Map(cloudServices.map(service => [service.id, service])),
    [cloudServices]
  );

  const initializedRef = useRef(false);

  const omitSchemaFieldValues = (
    attributes: Record<string, unknown> | undefined,
    fields: SchemaField[]
  ) => {
    if (!attributes) return undefined;
    const schemaKeys = new Set(fields.map(field => field.key));
    const entries = Object.entries(attributes).filter(([key]) => !schemaKeys.has(key));
    return entries.length > 0 ? Object.fromEntries(entries) : undefined;
  };

  useEffect(() => {
    if (!ci || typesLoading || initializedRef.current) return;
    const ciTypeId = ci.ciTypeId ?? 0;
    const initialTypeSchemaFields = resolveEffectiveTypeSchemaFields(types, ciTypeId);
    const attributeRecord =
      ci.attributes && typeof ci.attributes === 'object'
        ? (ci.attributes as Record<string, unknown>)
        : undefined;
    const remainingAttributes = omitSchemaFieldValues(attributeRecord, initialTypeSchemaFields);

    const initialValues: Partial<CIFormValues> = {
      name: ci.name,
      ciTypeId: ciTypeId,
      status: ci.status,
      description: ci.description,
      serialNumber: ci.serialNumber,
      model: ci.model,
      vendor: ci.vendor,
      location: ci.location,
      assetTag: ci.assetTag,
      assignedTo: ci.assignedTo,
      ownedBy: ci.ownedBy,
      environment: ci.environment,
      criticality: ci.criticality,
      discoverySource: ci.discoverySource,
      source: ci.source,
      cloudProvider: ci.cloudProvider,
      cloudAccountId: ci.cloudAccountId ? String(ci.cloudAccountId) : undefined,
      cloudRegion: ci.cloudRegion,
      cloudZone: ci.cloudZone,
      cloudResourceId: ci.cloudResourceId,
      cloudResourceType: ci.cloudResourceType,
      cloudSyncStatus: ci.cloudSyncStatus,
      cloudResourceRefId: ci.cloudResourceRefId,
      cloudMetadata: ci.cloudMetadata as Record<string, {} | undefined> | undefined,
      customAttributes: attributeRecord as Record<string, {} | undefined> | undefined,
    };
    if (remainingAttributes) {
      initialValues.attributes = JSON.stringify(remainingAttributes, null, 2);
    }
    form.setFieldsValue(initialValues);
    setTypeSchemaFields(initialTypeSchemaFields);

    initializedRef.current = true;
  }, [ci, form, types, typesLoading]);

  useEffect(() => {
    const cloudResourceRefId = ci?.cloudResourceRefId ?? ci?.cloudResourceRefId;
    if (!cloudResourceRefId || !cloudResources.length || !cloudServices.length) return;
    const resource = cloudResources.find(item => item.id === cloudResourceRefId);
    const service = resource ? cloudServiceMap.get(resource.serviceId) : undefined;
    setSchemaFields(normalizeSchemaFields(service?.attributeSchema));
  }, [ci, cloudResources, cloudServices, cloudServiceMap]);

  const handleCloudResourceChange = (value?: number) => {
    if (!value) {
      setSchemaFields([]);
      return;
    }
    const resource = cloudResources.find(item => item.id === value);
    const service = resource ? cloudServiceMap.get(resource.serviceId) : undefined;
    setSchemaFields(normalizeSchemaFields(service?.attributeSchema));
    if (!resource) return;
    form.setFieldsValue({
      cloudResourceId: resource.resourceId,
      cloudRegion: resource.region,
      cloudZone: resource.zone,
      cloudAccountId: String(resource.cloudAccountId),
      cloudProvider: service?.provider,
      cloudResourceType: service?.resourceTypeCode,
    });
  };

  const handleCITypeChange = (value?: number) => {
    setTypeSchemaFields(resolveEffectiveTypeSchemaFields(types, value));
    form.setFieldValue('customAttributes', undefined);
  };

  const updateMutation = useUpdateCIMutation();

  const handleSubmit = async (values: CIFormValues) => {
    let attributes: Record<string, unknown> | undefined;
    if (values.attributes) {
      try {
        attributes =
          typeof values.attributes === 'string'
            ? JSON.parse(values.attributes)
            : values.attributes;
      } catch {
        message.error('扩展属性需要是有效的 JSON');
        return;
      }
    }
    const customAttributes = compactRecord(
      values.customAttributes as Record<string, unknown> | undefined
    );
    attributes = {
      ...(attributes || {}),
      ...(customAttributes || {}),
    };
    if (Object.keys(attributes).length === 0) {
      attributes = undefined;
    }
    try {
      await updateMutation.mutateAsync({
        id,
        data: {
          name: values.name,
          ciTypeId: values.ciTypeId,
          status: values.status,
          description: values.description,
          attributes,
          serialNumber: values.serialNumber,
          model: values.model,
          vendor: values.vendor,
          location: values.location,
          assetTag: values.assetTag,
          assignedTo: values.assignedTo,
          ownedBy: values.ownedBy,
          environment: values.environment,
          criticality: values.criticality,
          discoverySource: values.discoverySource,
          source: values.source,
          cloudProvider: values.cloudProvider,
          cloudAccountId: values.cloudAccountId,
          cloudRegion: values.cloudRegion,
          cloudZone: values.cloudZone,
          cloudResourceId: values.cloudResourceId,
          cloudResourceType: values.cloudResourceType,
          cloudSyncStatus: values.cloudSyncStatus,
          cloudResourceRefId: values.cloudResourceRefId,
          cloudMetadata: values.cloudMetadata,
        },
      });
      // mutation onSuccess 已展示 '配置项已更新'
      clearDirty();
      router.push(`/cmdb/cis/${id}`);
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message || '更新配置项失败');
      } else {
        message.error('更新配置项失败');
      }
    }
  };

  return (
    <div className='space-y-6'>
      <ManagementPageHeader
        title='编辑配置项'
        description='修改这个配置项的基础信息、云资源关联和扩展属性。'
        notice={
          <ManagementNotice
            message='编辑时会保留原有云资源映射'
            description='如果更换云资源，动态属性字段会重新加载，请确认扩展属性是否仍然适用。'
          />
        }
      />

      <Card className='rounded-xl shadow-sm' loading={loading}>
        <CIEditorForm
          form={form}
          types={types}
          typesLoading={typesLoading}
          cloudResources={cloudResources}
          cloudServices={cloudServices}
          cloudLoading={cloudLoading}
          schemaFields={schemaFields}
          typeSchemaFields={typeSchemaFields}
          saving={updateMutation.isPending}
          submitText='保存修改'
          onSubmit={handleSubmit}
          onCancel={handleCancel}
          onCITypeChange={handleCITypeChange}
          onCloudResourceChange={handleCloudResourceChange}
          onValuesChange={() => {
            if (initializedRef.current) markDirty();
          }}
        />
      </Card>
    </div>
  );
};

export default EditCIPage;
