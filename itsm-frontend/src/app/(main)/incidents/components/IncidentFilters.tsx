'use client';

import React from 'react';
import { Card, Row, Col, Input, Select, Button } from 'antd';
import { SearchOutlined } from '@ant-design/icons';
import { useI18n } from '@/lib/i18n';

const { Search } = Input;
const { Option } = Select;

interface IncidentFiltersProps {
  loading: boolean;
  onSearch: (value: string) => void;
  onFilterChange: (key: string, value: string) => void;
  onRefresh: () => void;
}

export const IncidentFilters: React.FC<IncidentFiltersProps> = ({
  loading,
  onSearch,
  onFilterChange,
  onRefresh,
}) => {
  const { t } = useI18n();
  
  return (
    <Card className='mb-4 shadow-sm border-0' styles={{ body: { padding: '16px' } }}>
      <div className='mb-3'>
        <h3 className='text-sm font-semibold text-gray-800 mb-3'>{t('incidents.filterAndSearch')}</h3>
        <Row gutter={[12, 12]} align='middle'>
          <Col xs={24} sm={12} md={8}>
            <div className='space-y-2'>
              <label className='text-sm font-medium text-gray-700'>{t('incidents.searchIncidents')}</label>
              <Search
                placeholder={t('incidents.searchPlaceholder')}
                allowClear
                onSearch={onSearch}
                size='large'
                enterButton
                className='rounded-lg shadow-sm hover:shadow-md transition-shadow duration-200'
              />
            </div>
          </Col>
          <Col xs={24} sm={12} md={4}>
            <div className='space-y-2'>
              <label className='text-sm font-medium text-gray-700'>{t('incidents.statusFilter')}</label>
              <Select
                placeholder={t('incidents.selectStatus')}
                size='large'
                allowClear
                onChange={value => onFilterChange('status', value)}
                className='w-full rounded-lg shadow-sm hover:shadow-md transition-shadow duration-200'
              >
                <Option value='open'>
                  <div className='flex items-center space-x-2'>
                    <div className='w-2 h-2 bg-orange-500 rounded-full'></div>
                    <span>{t('incidents.statusOpen')}</span>
                  </div>
                </Option>
                <Option value='in-progress'>
                  <div className='flex items-center space-x-2'>
                    <div className='w-2 h-2 bg-blue-500 rounded-full'></div>
                    <span>{t('incidents.statusInProgress')}</span>
                  </div>
                </Option>
                <Option value='resolved'>
                  <div className='flex items-center space-x-2'>
                    <div className='w-2 h-2 bg-green-500 rounded-full'></div>
                    <span>{t('incidents.statusResolved')}</span>
                  </div>
                </Option>
                <Option value='closed'>
                  <div className='flex items-center space-x-2'>
                    <div className='w-2 h-2 bg-gray-500 rounded-full'></div>
                    <span>{t('incidents.statusClosed')}</span>
                  </div>
                </Option>
              </Select>
            </div>
          </Col>
          <Col xs={24} sm={12} md={4}>
            <div className='space-y-2'>
              <label className='text-sm font-medium text-gray-700'>{t('incidents.priority')}</label>
              <Select
                placeholder={t('incidents.selectPriority')}
                size='large'
                allowClear
                onChange={value => onFilterChange('priority', value)}
                className='w-full rounded-lg shadow-sm hover:shadow-md transition-shadow duration-200'
              >
                <Option value='low'>
                  <div className='flex items-center space-x-2'>
                    <div className='w-2 h-2 bg-green-500 rounded-full'></div>
                    <span>{t('incidents.priorityLow')}</span>
                  </div>
                </Option>
                <Option value='medium'>
                  <div className='flex items-center space-x-2'>
                    <div className='w-2 h-2 bg-blue-500 rounded-full'></div>
                    <span>{t('incidents.priorityMedium')}</span>
                  </div>
                </Option>
                <Option value='high'>
                  <div className='flex items-center space-x-2'>
                    <div className='w-2 h-2 bg-orange-500 rounded-full'></div>
                    <span>{t('incidents.priorityHigh')}</span>
                  </div>
                </Option>
                <Option value='critical'>
                  <div className='flex items-center space-x-2'>
                    <div className='w-2 h-2 bg-red-500 rounded-full animate-pulse'></div>
                    <span>{t('incidents.priorityCritical')}</span>
                  </div>
                </Option>
              </Select>
            </div>
          </Col>
          <Col xs={24} sm={12} md={4}>
            <div className='space-y-2'>
              <label className='text-sm font-medium text-gray-700'>{t('incidents.source')}</label>
              <Select
                placeholder={t('incidents.selectSource')}
                size='large'
                allowClear
                onChange={value => onFilterChange('source', value)}
                className='w-full rounded-lg shadow-sm hover:shadow-md transition-shadow duration-200'
              >
                <Option value='email'>📧 {t('incidents.sourceEmail')}</Option>
                <Option value='phone'>📞 {t('incidents.sourcePhone')}</Option>
                <Option value='web'>🌐 {t('incidents.sourceWeb')}</Option>
                <Option value='system'>⚙️ {t('incidents.sourceSystem')}</Option>
              </Select>
            </div>
          </Col>
          <Col xs={24} sm={12} md={4}>
            <div className='space-y-2'>
              <label className='text-sm font-medium text-gray-700'>{t('incidents.actions')}</label>
              <Button
                icon={<SearchOutlined />}
                onClick={onRefresh}
                loading={loading}
                size='large'
                className='w-full bg-gradient-to-r from-blue-500 to-indigo-600 border-0 text-white hover:from-blue-600 hover:to-indigo-700 shadow-lg hover:shadow-xl transition-all duration-200 rounded-lg font-medium'
              >
                {t('incidents.refresh')}
              </Button>
            </div>
          </Col>
        </Row>
      </div>
    </Card>
  );
};
