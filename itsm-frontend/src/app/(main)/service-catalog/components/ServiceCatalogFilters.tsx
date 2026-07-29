'use client';

import React from 'react';
import { Card, Row, Col, Input, Select, Button, Tooltip } from 'antd';
import { Search, PlusCircle } from 'lucide-react';
import { useI18n } from '@/lib/i18n';
import type { CIType, CloudService } from '@/types/biz/cmdb';

const { Search: SearchInput } = Input;

interface ServiceCatalogFiltersProps {
  onSearch: (value: string) => void;
  onCategoryFilterChange: (value: string) => void;
  onPriorityFilterChange: (value: string) => void;
  onCITypeFilterChange: (value: number) => void;
  onCloudServiceFilterChange: (value: number) => void;
  ciTypes: CIType[];
  cloudServices: CloudService[];
  optionsLoading?: boolean;
  onCreateService: () => void;
}

export const ServiceCatalogFilters: React.FC<ServiceCatalogFiltersProps> = ({
  onSearch,
  onCategoryFilterChange,
  onPriorityFilterChange,
  onCITypeFilterChange,
  onCloudServiceFilterChange,
  ciTypes,
  cloudServices,
  optionsLoading,
  onCreateService,
}) => {
  const { t } = useI18n();
  return (
    <Card style={{ marginBottom: 24 }}>
      <Row gutter={20} align="middle">
        <Col xs={24} sm={12} md={6}>
          <SearchInput
            placeholder={t('serviceCatalog.searchPlaceholder')}
            allowClear
            onSearch={onSearch}
            size="large"
            enterButton
          />
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Select
            placeholder={t('serviceCatalog.categoryFilter')}
            size="large"
            allowClear
            onChange={onCategoryFilterChange}
            style={{ width: '100%' }}
            options={[
              { value: '云资源服务', label: t('serviceCatalog.cloudResources') },
              { value: '账号与权限', label: t('serviceCatalog.accountPermissions') },
              { value: '安全服务', label: t('serviceCatalog.securityServices') },
            ]}
          />
        </Col>
        <Col xs={24} sm={12} md={3}>
          <Select
            placeholder={t('serviceCatalog.priorityFilter')}
            size="large"
            allowClear
            onChange={onPriorityFilterChange}
            style={{ width: '100%' }}
            options={[
              { value: '高', label: t('serviceCatalog.high') },
              { value: '中', label: t('serviceCatalog.medium') },
              { value: '低', label: t('serviceCatalog.low') },
            ]}
          />
        </Col>
        <Col xs={24} sm={12} md={3}>
          <Select
            placeholder="CI类型"
            size="large"
            allowClear
            loading={optionsLoading}
            onChange={onCITypeFilterChange}
            style={{ width: '100%' }}
            options={ciTypes.map(type => ({ value: type.id, label: type.name }))}
          />
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Select
            placeholder="云服务"
            size="large"
            allowClear
            loading={optionsLoading}
            showSearch
            optionFilterProp="label"
            onChange={onCloudServiceFilterChange}
            style={{ width: '100%' }}
            options={cloudServices.map(service => ({
              value: service.id,
              label: `${service.serviceName} (${service.resourceTypeName})`,
            }))}
          />
        </Col>
        <Col xs={24} sm={12} md={2}>
          <Tooltip title="刷新列表">
            <Button
              icon={<Search size={20} />}
              onClick={() => {
                // Trigger a page reload by dispatching a custom event
                window.dispatchEvent(new CustomEvent('refresh-service-catalog'));
              }}
              size="large"
              style={{ width: '100%' }}
            >
              {t('serviceCatalog.refresh')}
            </Button>
          </Tooltip>
        </Col>
        <Col xs={24} sm={12} md={2}>
          <Button
            type="primary"
            icon={<PlusCircle size={20} />}
            size="large"
            style={{ width: '100%' }}
            onClick={onCreateService}
          >
            {t('serviceCatalog.newService')}
          </Button>
        </Col>
      </Row>
    </Card>
  );
};
