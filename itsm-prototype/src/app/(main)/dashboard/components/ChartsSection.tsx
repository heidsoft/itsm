'use client';

import React from 'react';
import { Card, Row, Col, Spin, Empty } from 'antd';
import {
  TicketTrendData,
  IncidentDistributionData,
  SLAData,
  SatisfactionData,
  TicketTypeDistribution,
  ResponseTimeDistribution,
  TeamWorkload,
  PriorityDistribution,
  PeakHourData,
} from '../types/dashboard.types';

interface ChartsSectionProps {
  loading?: boolean;
  children?: React.ReactNode;
}

// 图表区域主组件
export const ChartsSection: React.FC<ChartsSectionProps> = React.memo(
  ({ loading = false, children }) => {
    if (loading) {
      return (
        <div className='mb-6'>
          <Row gutter={[16, 16]}>
            {Array.from({ length: 4 }).map((_, index) => (
              <Col key={index} xs={24} lg={12}>
                <Card
                  className='border border-gray-200'
                  style={{
                    borderRadius: '8px',
                    boxShadow: '0 1px 2px rgba(0, 0, 0, 0.05)',
                    minHeight: '420px',
                  }}
                >
                  <div className='flex flex-col items-center justify-center h-full min-h-[400px]'>
                    <Spin size='large' />
                    <p className='text-sm text-gray-400 mt-4'>加载图表数据...</p>
                  </div>
                </Card>
              </Col>
            ))}
          </Row>
        </div>
      );
    }

    if (!children) {
      return (
        <div className='mb-6'>
          <Card
            className='text-center py-16 border border-gray-200'
            style={{
              borderRadius: '8px',
              background: '#ffffff',
              boxShadow: '0 1px 2px rgba(0, 0, 0, 0.05)',
            }}
          >
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={
                <div>
                  <p className='text-base font-semibold text-gray-700 mb-1'>暂无图表数据</p>
                  <p className='text-sm text-gray-500'>系统正在收集和分析数据，请稍后查看</p>
                </div>
              }
            />
          </Card>
        </div>
      );
    }

    return (
      <div className='mb-6'>
        <Row gutter={[16, 16]}>{children}</Row>
      </div>
    );
  }
);

ChartsSection.displayName = 'ChartsSection';
