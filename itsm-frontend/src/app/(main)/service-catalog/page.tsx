'use client';

import React, { useEffect, useState } from 'react';
import { Row, Col, Empty, Form, Skeleton, App } from 'antd';
import { useServiceCatalogData } from './hooks/useServiceCatalogData';
import { ServiceCatalogStats } from './components/ServiceCatalogStats';
import { ServiceCatalogFilters } from './components/ServiceCatalogFilters';
import { ServiceItemCard } from './components/ServiceItemCard';
import { CreateServiceModal } from './components/CreateServiceModal';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import { useI18n } from '@/lib/i18n';
import { CMDBApi } from '@/modules/cmdb/api';
import type { CIType, CloudService } from '@/modules/cmdb/types';

const ServiceCatalogSkeleton: React.FC = () => (
  <div>
    <Skeleton active paragraph={{ rows: 2 }} />
    <Skeleton active paragraph={{ rows: 2 }} style={{ marginTop: 24 }} />
    <Row gutter={[24, 24]} style={{ marginTop: 24 }}>
      {Array.from({ length: 4 }).map((_, index) => (
        <Col key={index} xs={24} sm={12} md={8} lg={6}>
          <Skeleton active paragraph={{ rows: 4 }} />
        </Col>
      ))}
    </Row>
  </div>
);

export default function ServiceCatalogPage() {
  const { message } = App.useApp();
  const { t } = useI18n();
  const {
    catalogs,
    loading,
    stats,
    setSearchText,
    setCategoryFilter,
    setPriorityFilter,
    setCiTypeFilter,
    setCloudServiceFilter,
    loadServiceCatalogs,
  } = useServiceCatalogData();
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [createForm] = Form.useForm();
  const [ciTypes, setCiTypes] = useState<CIType[]>([]);
  const [cloudServices, setCloudServices] = useState<CloudService[]>([]);
  const [optionsLoading, setOptionsLoading] = useState(false);

  useEffect(() => {
    const loadOptions = async () => {
      try {
        setOptionsLoading(true);
        const [types, services] = await Promise.all([
          CMDBApi.getTypes(),
          CMDBApi.getCloudServices(),
        ]);
        setCiTypes(types || []);
        setCloudServices(services || []);
      } catch (error) {
        message.error('加载CMDB选项失败');
      } finally {
        setOptionsLoading(false);
      }
    };
    loadOptions();
  }, [message]);

  const handleCreateService = () => {
    setCreateModalVisible(true);
  };

  const handleCreateServiceConfirm = async () => {
    try {
      const values = await createForm.validateFields();
      await ServiceCatalogApi.createService({
        name: values.name,
        category: values.category,
        shortDescription: values.description,
        availability: {
          responseTime: values.deliveryTime ? Number(values.deliveryTime) : undefined,
        },
        tags: [],
      });

      message.success(t('serviceCatalog.createServiceSuccess'));
      setCreateModalVisible(false);
      createForm.resetFields();
      loadServiceCatalogs();
    } catch (error) {
      console.error(t('serviceCatalog.createServiceFailed'), error);
      message.error(t('serviceCatalog.createServiceFailed'));
    }
  };

  if (loading) {
    return <ServiceCatalogSkeleton />;
  }

  return (
    <div>
      <ServiceCatalogStats stats={stats} />
      <ServiceCatalogFilters
        onSearch={setSearchText}
        onCategoryFilterChange={setCategoryFilter}
        onPriorityFilterChange={setPriorityFilter}
        onCITypeFilterChange={setCiTypeFilter}
        onCloudServiceFilterChange={setCloudServiceFilter}
        ciTypes={ciTypes}
        cloudServices={cloudServices}
        optionsLoading={optionsLoading}
        onCreateService={handleCreateService}
      />

      {catalogs.length === 0 ? (
        <Empty description={t('serviceCatalog.noMatchingServices')} />
      ) : (
        <Row gutter={[24, 24]}>
          {catalogs.map(catalog => (
            <Col key={catalog.id} xs={24} sm={12} md={8} lg={6}>
              <ServiceItemCard catalog={catalog} />
            </Col>
          ))}
        </Row>
      )}

      <CreateServiceModal
        visible={createModalVisible}
        onCancel={() => {
          setCreateModalVisible(false);
          createForm.resetFields();
        }}
        onConfirm={handleCreateServiceConfirm}
        form={createForm}
      />
    </div>
  );
}
