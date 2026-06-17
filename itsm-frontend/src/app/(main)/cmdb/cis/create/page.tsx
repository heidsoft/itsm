'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { App, Card, Form, Spin } from 'antd';

import { CIEditorForm } from '@/components/cmdb/CIEditorForm';
import {
  CIFormValues,
  compactRecord,
  extractCloudDataList,
  normalizeSchemaFields,
  SchemaField,
} from '@/components/cmdb/ci-editor-shared';
import { ManagementNotice, ManagementPageHeader } from '@/components/ui/ManagementPageHeader';
import { CMDBApi } from '@/lib/api/cmdb-api';
import type { CIType, CloudResource, CloudService } from '@/types/biz/cmdb';
import { useI18n } from '@/lib/i18n';

const CreateCIPage: React.FC = () => {
  const { t } = useI18n();
  const router = useRouter();
  const searchParams = useSearchParams();
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
        message.error(t('cmdb.loadCITypesFailed'));
      } finally {
        setTypesLoading(false);
      }
    };
    loadTypes();
  }, [message, t]);

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
        message.error(t('cmdb.loadCloudResourcesFailed'));
      } finally {
        setCloudLoading(false);
      }
    };
    loadCloudData();
  }, [message, t]);

  useEffect(() => {
    if (!cloudResources.length || !cloudServices.length) return;
    const resourceRefId = searchParams.get('cloud_resource_ref_id');
    if (!resourceRefId) return;
    const parsed = Number(resourceRefId);
    if (Number.isNaN(parsed)) return;
    form.setFieldsValue({ cloud_resource_ref_id: parsed });
    const resource = cloudResources.find(item => item.id === parsed);
    if (!resource) return;
    const service = cloudServiceMap.get(resource.service_id);
    setSchemaFields(normalizeSchemaFields(service?.attribute_schema));
    form.setFieldsValue({
      cloud_resource_id: resource.resource_id,
      cloud_region: resource.region,
      cloud_zone: resource.zone,
      cloud_account_id: String(resource.cloud_account_id),
      cloud_provider: service?.provider,
      cloud_resource_type: service?.resource_type_code,
    });
  }, [cloudResources, cloudServices, cloudServiceMap, form, searchParams]);

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
          message.error(t('cmdb.invalidJSON'));
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
      await CMDBApi.createCI({
        name: values.name,
        ciTypeId: Number(values.ci_type_id),
        status: values.status,
        description: values.description,
        attributes,
        serialNumber: values.serial_number,
        model: values.model,
        vendor: values.vendor,
        assetTag: values.asset_tag,
        location: values.location,
        assignedTo: values.assigned_to,
        ownedBy: values.owned_by,
        environment: values.environment || '',
        criticality: values.criticality || '',
        discoverySource: values.discovery_source,
        source: values.source,
        cloudProvider: values.cloud_provider,
        cloudAccountId: values.cloud_account_id,
        cloudRegion: values.cloud_region,
        cloudZone: values.cloud_zone,
        cloudResourceId: values.cloud_resource_id,
        cloudResourceType: values.cloud_resource_type,
        cloudSyncStatus: values.cloud_sync_status,
        cloudResourceRefId: values.cloud_resource_ref_id,
        cloudMetadata: values.cloud_metadata,
      });
      message.success(t('cmdb.createCISuccess'));
      router.push('/cmdb');
    } catch (error) {
      let errorMessage = t('cmdb.createCIFailed');
      if (error && typeof error === 'object') {
        const errObj = error as Record<string, unknown>;
        if (typeof errObj.message === 'string') {
          errorMessage = errObj.message;
        }
        if (errObj.response && typeof errObj.response === 'object') {
          const response = errObj.response as Record<string, unknown>;
          if (response.data && typeof response.data === 'object') {
            const data = response.data as Record<string, unknown>;
            if (data.message) {
              errorMessage = String(data.message);
            }
          }
          if (response.status) {
            errorMessage = `HTTP ${response.status}: ${errorMessage}`;
          }
        }
      }
      message.error(errorMessage);
    } finally {
      setSaving(false);
    }
  };

  const formReady = !typesLoading && !cloudLoading;

  return (
    <div className="space-y-6">
      <ManagementPageHeader
        title="录入配置项"
        description="统一录入基础资产信息、云资源关联和扩展属性，减少后续补录和字段不一致。"
        notice={
          <ManagementNotice
            message="优先选择云资源引用"
            description="如果配置项来自云发现，先绑定云资源，系统会自动带出 Region、Zone、资源类型和动态属性。"
          />
        }
      />

      <Card className="rounded-xl shadow-sm">
        {formReady ? (
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
            submitText="保存配置项"
            onSubmit={handleSubmit}
            onCancel={() => router.back()}
            onCITypeChange={handleCITypeChange}
            onCloudResourceChange={handleCloudResourceChange}
          />
        ) : (
          <div className="flex min-h-[240px] items-center justify-center">
            <Spin size="large" />
          </div>
        )}
      </Card>
    </div>
  );
};

export default CreateCIPage;
