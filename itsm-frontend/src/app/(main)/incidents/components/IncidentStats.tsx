'use client';

import React from 'react';
import { Card, Col, Row } from 'antd';
import { AlertTriangle, Clock, CheckCircle, AlertCircle } from 'lucide-react';
import { useI18n } from '@/lib/i18n';

interface IncidentStatsProps {
  metrics: {
    total_incidents: number;
    critical_incidents: number;
    major_incidents: number;
    avg_resolution_time: number;
  };
}

export const IncidentStats: React.FC<IncidentStatsProps> = ({ metrics }) => {
  const { t } = useI18n();
  
  if (!metrics) {
    return null;
  }

  return (
    <div className='mb-6'>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} md={6} lg={6}>
          <Card
            className='text-center rounded-lg shadow-sm border-0 bg-gradient-to-br from-blue-500 to-blue-600 text-white overflow-hidden relative'
            bordered={false}
            styles={{ body: { padding: '16px' } }}
          >
            <div className='w-10 h-10 bg-white/20 rounded-lg flex items-center justify-center mx-auto mb-3'>
              <AlertTriangle className='w-5 h-5 text-white' />
            </div>
            <div className='text-2xl font-bold mb-1'>{metrics.total_incidents}</div>
            <div className='text-blue-100 font-medium text-xs'>{t('incidents.total')}</div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6} lg={6}>
          <Card
            className='text-center rounded-lg shadow-sm border-0 bg-gradient-to-br from-orange-500 to-orange-600 text-white overflow-hidden relative'
            bordered={false}
            styles={{ body: { padding: '16px' } }}
          >
            <div className='w-10 h-10 bg-white/20 rounded-lg flex items-center justify-center mx-auto mb-3'>
              <Clock className='w-5 h-5 text-white' />
            </div>
            <div className='text-2xl font-bold mb-1'>{metrics.critical_incidents}</div>
            <div className='text-orange-100 font-medium text-xs'>{t('incidents.critical')}</div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6} lg={6}>
          <Card
            className='text-center rounded-lg shadow-sm border-0 bg-gradient-to-br from-green-500 to-green-600 text-white overflow-hidden relative'
            bordered={false}
            styles={{ body: { padding: '16px' } }}
          >
            <div className='w-10 h-10 bg-white/20 rounded-lg flex items-center justify-center mx-auto mb-3'>
              <CheckCircle className='w-5 h-5 text-white' />
            </div>
            <div className='text-2xl font-bold mb-1'>{metrics.major_incidents}</div>
            <div className='text-green-100 font-medium text-xs'>{t('incidents.major')}</div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6} lg={6}>
          <Card
            className='text-center rounded-lg shadow-sm border-0 bg-gradient-to-br from-purple-500 to-purple-600 text-white overflow-hidden relative'
            bordered={false}
            styles={{ body: { padding: '16px' } }}
          >
            <div className='w-10 h-10 bg-white/20 rounded-lg flex items-center justify-center mx-auto mb-3'>
              <AlertCircle className='w-5 h-5 text-white' />
            </div>
            <div className='text-2xl font-bold mb-1'>{metrics.avg_resolution_time}</div>
            <div className='text-purple-100 font-medium text-xs'>{t('incidents.avgResolutionTime')}</div>
          </Card>
        </Col>
      </Row>
    </div>
  );
};
