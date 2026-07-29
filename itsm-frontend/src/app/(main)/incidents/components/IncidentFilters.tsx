'use client';

import React from 'react';
import { Card, Row, Col, Input, Select, Button } from 'antd';
import { Search as SearchIcon } from 'lucide-react';
import { useI18n } from '@/lib/i18n';

const { Search } = Input;


interface IncidentFiltersProps {
  loading?: boolean;
  onSearch?: (value: string) => void;
  status?: string;
  priority?: string;
  source?: string;
  onFilterChange?: (status?: string, priority?: string, source?: string) => void;
  onRefresh?: () => void;
}

export const IncidentFilters: React.FC<IncidentFiltersProps> = ({
  loading,
  onSearch,
  status,
  priority,
  source,
  onFilterChange,
  onRefresh,
}) => {
  const { t } = useI18n();

  return (
    <Card className="mb-4 shadow-sm border-0" styles={{ body: { padding: '16px' } }}>
      <div className="mb-3">
        <h3 className="text-sm font-semibold text-gray-800 mb-3">
          {t('incidents.filterAndSearch')}
        </h3>
        <Row gutter={[12, 12]} align="middle">
          <Col xs={24} sm={12} md={8}>
            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-700">
                {t('incidents.searchIncidents')}
              </label>
              <Search
                placeholder={t('incidents.searchPlaceholder')}
                allowClear
                onSearch={onSearch || (() => {})}
                size="large"
                enterButton
                className="rounded-lg shadow-sm hover:shadow-md transition-shadow duration-200"
              />
            </div>
          </Col>
          <Col xs={24} sm={12} md={4}>
            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-700">
                {t('incidents.statusFilter')}
              </label>
              <Select
                placeholder={t('incidents.selectStatus')}
                size="large"
                allowClear
                value={status}
                onChange={value => onFilterChange?.(value, priority, source)}
                className="w-full rounded-lg shadow-sm hover:shadow-md transition-shadow duration-200"
                options={[
                  { value: "new", label: <div className="flex items-center space-x-2"><div className="w-2 h-2 bg-blue-500 rounded-full"></div><span>{t('incidents.statusNew')}</span></div> },
                  { value: "acknowledged", label: <div className="flex items-center space-x-2"><div className="w-2 h-2 bg-purple-500 rounded-full"></div><span>{t('incidents.statusAcknowledged')}</span></div> },
                  { value: "assigned", label: <div className="flex items-center space-x-2"><div className="w-2 h-2 bg-teal-500 rounded-full"></div><span>{t('incidents.statusAssigned')}</span></div> },
                  { value: "in_progress", label: <div className="flex items-center space-x-2"><div className="w-2 h-2 bg-blue-500 rounded-full"></div><span>{t('incidents.statusInProgress')}</span></div> },
                  { value: "escalated", label: <div className="flex items-center space-x-2"><div className="w-2 h-2 bg-red-500 rounded-full"></div><span>{t('incidents.statusEscalated')}</span></div> },
                  { value: "resolved", label: <div className="flex items-center space-x-2"><div className="w-2 h-2 bg-green-500 rounded-full"></div><span>{t('incidents.statusResolved')}</span></div> },
                  { value: "closed", label: <div className="flex items-center space-x-2"><div className="w-2 h-2 bg-gray-500 rounded-full"></div><span>{t('incidents.statusClosed')}</span></div> },
                ]}
              />
            </div>
          </Col>
          <Col xs={24} sm={12} md={4}>
            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-700">{t('incidents.priority')}</label>
              <Select
                placeholder={t('incidents.selectPriority')}
                size="large"
                allowClear
                value={priority}
                onChange={value => onFilterChange?.(status, value, source)}
                className="w-full rounded-lg shadow-sm hover:shadow-md transition-shadow duration-200"
                options={[
                  { value: "low", label: <div className="flex items-center space-x-2"><div className="w-2 h-2 bg-green-500 rounded-full"></div><span>{t('incidents.priorityLow')}</span></div> },
                  { value: "medium", label: <div className="flex items-center space-x-2"><div className="w-2 h-2 bg-blue-500 rounded-full"></div><span>{t('incidents.priorityMedium')}</span></div> },
                  { value: "high", label: <div className="flex items-center space-x-2"><div className="w-2 h-2 bg-orange-500 rounded-full"></div><span>{t('incidents.priorityHigh')}</span></div> },
                  { value: "critical", label: <div className="flex items-center space-x-2"><div className="w-2 h-2 bg-red-500 rounded-full animate-pulse"></div><span>{t('incidents.priorityCritical')}</span></div> },
                ]}
              />
            </div>
          </Col>
          <Col xs={24} sm={12} md={4}>
            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-700">{t('incidents.source')}</label>
              <Select
                placeholder={t('incidents.selectSource')}
                size="large"
                allowClear
                value={source}
                onChange={value => onFilterChange?.(status, priority, value)}
                className="w-full rounded-lg shadow-sm hover:shadow-md transition-shadow duration-200"
                options={[
                  { value: "email", label: "📧 " + t('incidents.sourceEmail') },
                  { value: "phone", label: "📞 " + t('incidents.sourcePhone') },
                  { value: "web", label: "🌐 " + t('incidents.sourceWeb') },
                  { value: "system", label: "⚙️ " + t('incidents.sourceSystem') },
                ]}
              />
            </div>
          </Col>
          <Col xs={24} sm={12} md={4}>
            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-700">{t('incidents.actions')}</label>
              <Button
                icon={<SearchIcon />}
                onClick={onRefresh || (() => {})}
                loading={loading}
                size="large"
                className="w-full bg-gradient-to-r from-blue-500 to-indigo-600 border-0 text-white hover:from-blue-600 hover:to-indigo-700 shadow-lg hover:shadow-xl transition-all duration-200 rounded-lg font-medium"
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
