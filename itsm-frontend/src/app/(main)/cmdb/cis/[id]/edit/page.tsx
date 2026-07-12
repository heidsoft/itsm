'use client';

import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { App, Card } from 'antd';
import { Form } from 'antd';

import { CIEditorForm } from '@/components/cmdb/CIEditorForm';
import type {
  CIFormValues,
  SchemaField} from '@/components/cmdb/ci-editor-shared';
import {
  compactRecord,
  extractCloudDataList,
  normalizeSchemaFields
} from '@/components/cmdb/ci-editor-shared';
import { ManagementNotice, ManagementPageHeader } from '@/components/ui/ManagementPageHeader';
import { CMDBApi } from '@/lib/api/cmdb-api';
import type { CIType, CloudResource, CloudService, ConfigurationItem } from '@/types/biz/cmdb';

const EditCIPage: React.FC = () => {
  const router = useRouter();
  const { id } = useParams() as { id: string };
  const { message } = App.useApp();
  const [form] = Form.useForm<CIFormValues>();
  const [types, setTypes] = useState<CIType[]>([]);
  const [typesLoading, setTypesLoading] = useState(true);
  const [cloudResources, setCloudResources] = useState<CloudResource[]>([]);
  const [cloudServices, setCloudServices] = useState<CloudService[]>([]);
  const [cloudLoading, setCloudLoading] = useState(true);
  const [schemaFields, setSchemaFields] = useState<SchemaField[]>([]);
  const [typeSchemaFields, setTypeSchemaFields] = useState<SchemaField[]>([]);
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);
  const [ci, setCi] = useState<ConfigurationItem | null>(null);
  const initializedRef = useRef(false);

  const cloudServiceMap = useMemo(
    () => new Map(cloudServices.map(service => [service.id, service])),
    [cloudServices]
  );

  useEffect(() => {
    const loadTypes = async () => {
      try {
		const res = await CMDBApi.getCITypes();
        setTypes(res || []);
      } catch {
        message.error('加载资产类型失败');
      } finally {
        setTypesLoading(false);
      }
    };
    loadTypes();
  }, [message]);

  useEffect(() => {
    const loadCloudData = async () => {
      setCloudLoading(true);
      try {
        const [resources, services] = await Promise.all([
          CMDBApi.getCloudResources(),
          CMDBApi.getCloudServices(),
        ]);
        setCloudResources(extractCloudDataList<CloudResource>(resources));
        setCloudServices(extractCloudDataList<CloudService>(services));
      } catch {
        message.error('加载云资源数据失败');
      } finally {
        setCloudLoading(false);
      }
    };
    loadCloudData();
  }, [message]);

  useEffect(() => {
    const loadCI = async () => {
      setLoading(true);
      try {
        const res = await CMDBApi.getCI(id);
        setCi(res as unknown as ConfigurationItem);
      } catch {
        message.error('加载配置项失败');
      } finally {
        setLoading(false);
      }
    };
    if (id) {
      loadCI();
    }
  }, [id, message]);

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
    const ciTypeId = ci.ciTypeId ?? ci.ciTypeId ?? 0;
    const selectedType = types.find(type => type.id === ciTypeId);
    const initialTypeSchemaFields = normalizeSchemaFields(selectedType?.attributeSchema);
    const attributeRecord =
      ci.attributes && typeof ci.attributes === 'object' ? (ci.attributes as Record<string, unknown>) : undefined;
    const remainingAttributes = omitSchemaFieldValues(attributeRecord, initialTypeSchemaFields);

    const initialValues: Partial<CIFormValues> = {
      name: ci.name,
      ciTypeId: ciTypeId,
      status: ci.status,
      description: ci.description,
      serialNumber: ci.serialNumber ?? ci.serialNumber,
      model: ci.model,
      vendor: ci.vendor,
      location: ci.location,
      assetTag: ci.assetTag ?? ci.assetTag,
      assignedTo: ci.assignedTo ?? ci.assignedTo,
      ownedBy: ci.ownedBy ?? ci.ownedBy,
      environment: ci.environment,
      criticality: ci.criticality,
      discoverySource: ci.discoverySource ?? ci.discoverySource,
      source: ci.source,
      cloudProvider: ci.cloudProvider ?? ci.cloudProvider,
      cloudAccountId: ci.cloudAccountId ? String(ci.cloudAccountId) : undefined,
      cloudRegion: ci.cloudRegion ?? ci.cloudRegion,
      cloudZone: ci.cloudZone ?? ci.cloudZone,
      cloudResourceId: ci.cloudResourceId ?? ci.cloudResourceId,
      cloudResourceType: ci.cloudResourceType ?? ci.cloudResourceType,
      cloudSyncStatus: ci.cloudSyncStatus ?? ci.cloudSyncStatus,
      cloudResourceRefId: ci.cloudResourceRefId ?? ci.cloudResourceRefId,
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
    const selectedType = types.find(type => type.id === value);
    setTypeSchemaFields(normalizeSchemaFields(selectedType?.attributeSchema));
    form.setFieldValue('custom_attributes', undefined);
  };

  const handleSubmit = async (values: CIFormValues) => {
    try {
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
      const customAttributes = compactRecord(values.customAttributes as Record<string, unknown> | undefined);
      attributes = {
        ...(attributes || {}),
        ...(customAttributes || {}),
      };
      if (Object.keys(attributes).length === 0) {
        attributes = undefined;
      }
      setSaving(true);
      await CMDBApi.updateCI(id, {
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
      });
      message.success('配置项更新成功');
      router.push('/cmdb');
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message || '更新配置项失败');
      } else {
        message.error('更新配置项失败');
      }
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <ManagementPageHeader
        title="编辑配置项"
        description="统一维护资产主数据、云资源映射和扩展属性，避免创建页与编辑页字段表现不一致。"
        notice={
          <ManagementNotice
            message="编辑时会保留原有云资源映射"
            description="变更云资源引用后，系统会同步刷新动态属性字段，请确认扩展属性是否仍然适配。"
          />
        }
      />

      <Card className="rounded-xl shadow-sm" loading={loading}>
        <CIEditorForm
          form={form}
          types={types}
          typesLoading={typesLoading}
          cloudResources={cloudResources}
          cloudServices={cloudServices}
          cloudLoading={cloudLoading}
          schemaFields={schemaFields}
          typeSchemaFields={typeSchemaFields}
          saving={saving}
          submitText="保存修改"
          onSubmit={handleSubmit}
          onCancel={() => router.back()}
          onCITypeChange={handleCITypeChange}
          onCloudResourceChange={handleCloudResourceChange}
        />
      </Card>
    </div>
  );
};

export default EditCIPage;
