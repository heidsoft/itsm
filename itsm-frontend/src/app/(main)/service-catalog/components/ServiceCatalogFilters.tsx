'use client';

import React from 'react';
import { Card, Row, Col, Input, Select, Button } from 'antd';
import { Search, PlusCircle } from 'lucide-react';
import { useI18n } from '@/lib/i18n';
import type { CIType, CloudService } from '@/modules/cmdb/types';

const { Search: SearchInput } = Input;
const { Option } = Select;

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
      <Row gutter={20} align='middle'>
        <Col xs={24} sm={12} md={6}>
          <SearchInput
            placeholder={t('serviceCatalog.searchPlaceholder')}
            allowClear
            onSearch={onSearch}
            size='large'
            enterButton
          />
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Select
            placeholder={t('serviceCatalog.categoryFilter')}
            size='large'
            allowClear
            onChange={onCategoryFilterChange}
            style={{ width: '100%' }}
          >
            <Option value='云资源服务'>{t('serviceCatalog.cloudResources')}</Option>
            <Option value='账号与权限'>{t('serviceCatalog.accountPermissions')}</Option>
            <Option value='安全服务'>{t('serviceCatalog.securityServices')}</Option>
          </Select>
        </Col>
        <Col xs={24} sm={12} md={3}>
          <Select
            placeholder={t('serviceCatalog.priorityFilter')}
            size='large'
            allowClear
            onChange={onPriorityFilterChange}
            style={{ width: '100%' }}
          >
            <Option value='高'>{t('serviceCatalog.high')}</Option>
            <Option value='中'>{t('serviceCatalog.medium')}</Option>
            <Option value='低'>{t('serviceCatalog.low')}</Option>
          </Select>
        </Col>
        <Col xs={24} sm={12} md={3}>
          <Select
            placeholder='CI类型'
            size='large'
            allowClear
            loading={optionsLoading}
            onChange={onCITypeFilterChange}
            style={{ width: '100%' }}
          >
            {ciTypes.map(type => (
              <Option key={type.id} value={type.id}>
                {type.name}
              </Option>
            ))}
          </Select>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Select
            placeholder='云服务'
            size='large'
            allowClear
            loading={optionsLoading}
            showSearch
            optionFilterProp='label'
            onChange={onCloudServiceFilterChange}
            style={{ width: '100%' }}
          >
            {cloudServices.map(service => (
              <Option
                key={service.id}
                value={service.id}
                label={`${service.service_name} (${service.resource_type_name})`}
              >
                {service.service_name} ({service.resource_type_name})
              </Option>
            ))}
          </Select>
        </Col>
        <Col xs={24} sm={12} md={2}>
          <Button icon={<Search size={20} />} onClick={() => {}} size='large' style={{ width: '100%' }}>
            {t('serviceCatalog.refresh')}
          </Button>
        </Col>
        <Col xs={24} sm={12} md={2}>
          <Button
            type='primary'
            icon={<PlusCircle size={20} />}
            size='large'
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
