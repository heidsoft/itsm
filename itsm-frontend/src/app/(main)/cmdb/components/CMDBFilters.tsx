'use client';

import React from 'react';
import { Card, Row, Col, Input, Select, Button, Space } from 'antd';
import {
  ReloadOutlined,
  PlusCircleOutlined,
} from '@ant-design/icons';
import { useI18n } from '@/lib/i18n';

const { Search: SearchInput } = Input;
const { Option } = Select;

interface CMDBFiltersProps {
  loading: boolean;
  onSearch: (value: string) => void;
  onFilterTypeChange: (value: string) => void;
  onFilterStatusChange: (value: string) => void;
  onRefresh: () => void;
  onCreateCI: () => void;
}

export const CMDBFilters: React.FC<CMDBFiltersProps> = ({
  loading,
  onSearch,
  onFilterTypeChange,
  onFilterStatusChange,
  onRefresh,
  onCreateCI,
}) => {
    const { t } = useI18n();
  return (
    <Card
      style={{
        marginBottom: 16,
        borderRadius: 12,
      }}
      styles={{ body: { padding: '16px' } }}
    >
      <Row gutter={[20, 16]} align='middle'>
        <Col xs={24} sm={12} md={8}>
          <SearchInput
            placeholder={t('cmdb.searchPlaceholder')}
            allowClear
            onSearch={onSearch}
            size='large'
          />
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Select
            placeholder={t('cmdb.typeFilter')}
            size='large'
            allowClear
            onChange={onFilterTypeChange}
            style={{ width: '100%' }}
          >
            <Option value='云服务器'>☁️ {t('cmdb.cloudServer')}</Option>
            <Option value='物理服务器'>🖥️ {t('cmdb.physicalServer')}</Option>
            <Option value='关系型数据库'>🗄️ {t('cmdb.relationalDatabase')}</Option>
            <Option value='存储设备'>💾 {t('cmdb.storageDevice')}</Option>
            <Option value='网络设备'>🌐 {t('cmdb.networkDevice')}</Option>
          </Select>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Select
            placeholder={t('cmdb.statusFilter')}
            size='large'
            allowClear
            onChange={onFilterStatusChange}
            style={{ width: '100%' }}
          >
            <Option value='运行中'>🟢 {t('cmdb.running')}</Option>
            <Option value='维护中'>🟡 {t('cmdb.maintenance')}</Option>
            <Option value='已停用'>⚫ {t('cmdb.disabled')}</Option>
          </Select>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Space size={12} style={{ width: '100%', justifyContent: 'flex-end' }}>
            <Button icon={<ReloadOutlined />} onClick={onRefresh} loading={loading} size='large'>
              {t('cmdb.refresh')}
            </Button>
            <Button
              type='primary'
              icon={<PlusCircleOutlined />} 
              onClick={onCreateCI}
              size='large'
            >
              {t('cmdb.newCI')}
            </Button>
          </Space>
        </Col>
      </Row>
    </Card>
  );
};
