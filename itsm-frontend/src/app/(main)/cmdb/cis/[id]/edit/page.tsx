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
        const res = await CMDBApi.getTypes();
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
    const ciTypeId = ci.ci_type_id ?? ci.ciTypeId ?? 0;
    const selectedType = types.find(type => type.id === ciTypeId);
    const initialTypeSchemaFields = normalizeSchemaFields(selectedType?.attribute_schema);
    const attributeRecord =
      ci.attributes && typeof ci.attributes === 'object' ? (ci.attributes as Record<string, unknown>) : undefined;
    const remainingAttributes = omitSchemaFieldValues(attributeRecord, initialTypeSchemaFields);

    const initialValues: Partial<CIFormValues> = {
      name: ci.name,
      ci_type_id: ciTypeId,
      status: ci.status,
      description: ci.description,
      serial_number: ci.serial_number ?? ci.serialNumber,
      model: ci.model,
      vendor: ci.vendor,
      location: ci.location,
      asset_tag: ci.asset_tag ?? ci.assetTag,
      assigned_to: ci.assigned_to ?? ci.assignedTo,
      owned_by: ci.owned_by ?? ci.ownedBy,
      environment: ci.environment,
      criticality: ci.criticality,
      discovery_source: ci.discovery_source ?? ci.discoverySource,
      source: ci.source,
      cloud_provider: ci.cloud_provider ?? ci.cloudProvider,
      cloud_account_id: ci.cloud_account_id ? String(ci.cloud_account_id) : undefined,
      cloud_region: ci.cloud_region ?? ci.cloudRegion,
      cloud_zone: ci.cloud_zone ?? ci.cloudZone,
      cloud_resource_id: ci.cloud_resource_id ?? ci.cloudResourceId,
      cloud_resource_type: ci.cloud_resource_type ?? ci.cloudResourceType,
      cloud_sync_status: ci.cloud_sync_status ?? ci.cloudSyncStatus,
      cloud_resource_ref_id: ci.cloud_resource_ref_id ?? ci.cloudResourceRefId,
      cloud_metadata: ci.cloud_metadata as Record<string, {} | undefined> | undefined,
      custom_attributes: attributeRecord as Record<string, {} | undefined> | undefined,
    };
    if (remainingAttributes) {
      initialValues.attributes = JSON.stringify(remainingAttributes, null, 2);
    }
    form.setFieldsValue(initialValues);
    setTypeSchemaFields(initialTypeSchemaFields);

    initializedRef.current = true;
  }, [ci, form, types, typesLoading]);

  useEffect(() => {
    const cloudResourceRefId = ci?.cloud_resource_ref_id ?? ci?.cloudResourceRefId;
    if (!cloudResourceRefId || !cloudResources.length || !cloudServices.length) return;
    const resource = cloudResources.find(item => item.id === cloudResourceRefId);
      const service = resource ? cloudServiceMap.get(resource.service_id) : undefined;
      setSchemaFields(normalizeSchemaFields(service?.attribute_schema));
  }, [ci, cloudResources, cloudServices, cloudServiceMap]);

  const handleCloudResourceChange = (value?: number) => {
    if (!value) {
      setSchemaFields([]);
      return;
    }
    const resource = cloudResources.find(item => item.id === value);
    const service = resource ? cloudServiceMap.get(resource.service_id) : undefined;
    setSchemaFields(normalizeSchemaFields(service?.attribute_schema));
    if (!resource) return;
    form.setFieldsValue({
      cloud_resource_id: resource.resource_id,
      cloud_region: resource.region,
      cloud_zone: resource.zone,
      cloud_account_id: String(resource.cloud_account_id),
      cloud_provider: service?.provider,
      cloud_resource_type: service?.resource_type_code,
    });
  };

  const handleCITypeChange = (value?: number) => {
    const selectedType = types.find(type => type.id === value);
    setTypeSchemaFields(normalizeSchemaFields(selectedType?.attribute_schema));
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
      const customAttributes = compactRecord(values.custom_attributes as Record<string, unknown> | undefined);
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
        ci_type_id: values.ci_type_id,
        status: values.status,
        description: values.description,
        attributes,
        serial_number: values.serial_number,
        model: values.model,
        vendor: values.vendor,
        location: values.location,
        asset_tag: values.asset_tag,
        assigned_to: values.assigned_to,
        owned_by: values.owned_by,
        environment: values.environment,
        criticality: values.criticality,
        discovery_source: values.discovery_source,
        source: values.source,
        cloud_provider: values.cloud_provider,
        cloud_account_id: values.cloud_account_id,
        cloud_region: values.cloud_region,
        cloud_zone: values.cloud_zone,
        cloud_resource_id: values.cloud_resource_id,
        cloud_resource_type: values.cloud_resource_type,
        cloud_sync_status: values.cloud_sync_status,
        cloud_resource_ref_id: values.cloud_resource_ref_id,
        cloud_metadata: values.cloud_metadata,
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
