'use client';

import React from 'react';
import { Card, Row, Col, Input, Select, Button } from 'antd';
import { Search, PlusCircle } from 'lucide-react';
import { useI18n } from '@/lib/i18n';

const { Search: SearchInput } = Input;
const { Option } = Select;

interface KnowledgeBaseFiltersProps {
  loading: boolean;
  onSearch: (value: string) => void;
  onCategoryFilterChange: (value: string) => void;
  onCreateArticle: () => void;
}

export const KnowledgeBaseFilters: React.FC<KnowledgeBaseFiltersProps> = ({
  loading,
  onSearch,
  onCategoryFilterChange,
  onCreateArticle,
}) => {
    const { t } = useI18n();
  return (
    <Card style={{ marginBottom: 24 }}>
      <Row gutter={20} align='middle'>
        <Col xs={24} sm={12} md={8}>
          <SearchInput
            placeholder={t('knowledgeBase.searchPlaceholder')}
            allowClear
            onSearch={onSearch}
            size='large'
            enterButton
          />
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Select
            placeholder={t('knowledgeBase.categoryFilter')}
            size='large'
            allowClear
            onChange={onCategoryFilterChange}
            style={{ width: '100%' }}
          >
            <Option value='账号管理'>{t('knowledgeBase.accountManagement')}</Option>
            <Option value='故障排除'>{t('knowledgeBase.troubleshooting')}</Option>
            <Option value='网络连接'>{t('knowledgeBase.network')}</Option>
            <Option value='流程指南'>{t('knowledgeBase.processGuide')}</Option>
            <Option value='系统配置'>{t('knowledgeBase.systemConfig')}</Option>
          </Select>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Button icon={<Search size={20} />} onClick={() => {}} loading={loading} size='large' style={{ width: '100%' }}>
            {t('knowledgeBase.refresh')}
          </Button>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Button
            type='primary'
            icon={<PlusCircle size={20} />}
            size='large'
            style={{ width: '100%' }}
            onClick={onCreateArticle}
          >
            {t('knowledgeBase.newArticle')}
          </Button>
        </Col>
      </Row>
    </Card>
  );
};

