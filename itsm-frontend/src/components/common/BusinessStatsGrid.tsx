'use client';

import React from 'react';
import { Card, Col, Row } from 'antd';

export type BusinessStatsTone = 'blue' | 'orange' | 'green' | 'purple' | 'cyan' | 'red';

interface BusinessStatsItem {
  label: string;
  value: React.ReactNode;
  icon: React.ReactNode;
  tone: BusinessStatsTone;
}

interface BusinessStatsGridProps {
  items: BusinessStatsItem[];
  className?: string;
  loading?: boolean;
}

const toneClasses: Record<
  BusinessStatsTone,
  {
    card: string;
    icon: string;
    value: string;
  }
> = {
  blue: {
    card: 'bg-gradient-to-br from-blue-50 to-blue-100 border-blue-200',
    icon: 'bg-blue-100 text-blue-600',
    value: 'text-blue-700',
  },
  orange: {
    card: 'bg-gradient-to-br from-orange-50 to-orange-100 border-orange-200',
    icon: 'bg-orange-100 text-orange-600',
    value: 'text-orange-700',
  },
  green: {
    card: 'bg-gradient-to-br from-green-50 to-green-100 border-green-200',
    icon: 'bg-green-100 text-green-600',
    value: 'text-green-700',
  },
  purple: {
    card: 'bg-gradient-to-br from-purple-50 to-purple-100 border-purple-200',
    icon: 'bg-purple-100 text-purple-600',
    value: 'text-purple-700',
  },
  cyan: {
    card: 'bg-gradient-to-br from-cyan-50 to-cyan-100 border-cyan-200',
    icon: 'bg-cyan-100 text-cyan-600',
    value: 'text-cyan-700',
  },
  red: {
    card: 'bg-gradient-to-br from-red-50 to-red-100 border-red-200',
    icon: 'bg-red-100 text-red-600',
    value: 'text-red-700',
  },
};

export const BusinessStatsGrid: React.FC<BusinessStatsGridProps> = ({
  items,
  className = '',
  loading = false,
}) => {
  if (loading) {
    return (
      <div className={`mb-6 ${className}`}>
        <Row gutter={[16, 16]} align="stretch">
          {Array.from({ length: 4 }).map((_, index) => (
            <Col key={index} xs={24} sm={12} md={6} lg={6} className="flex">
              <Card loading className="w-full rounded-lg shadow-sm" />
            </Col>
          ))}
        </Row>
      </div>
    );
  }

  return (
    <div className={`mb-6 ${className}`}>
      <Row gutter={[16, 16]} align="stretch">
        {items.map((item, index) => {
          const tone = toneClasses[item.tone];
          return (
            <Col key={`${item.label}-${index}`} xs={24} sm={12} md={6} lg={6} className="flex">
              <Card
                className={`w-full text-center shadow-sm hover:shadow-lg transition-all duration-300 ${tone.card}`}
                styles={{ body: { padding: '16px' } }}
              >
                <div
                  className={`mx-auto mb-3 h-10 w-10 rounded-lg flex items-center justify-center ${tone.icon}`}
                >
                  {item.icon}
                </div>
                <div className={`text-2xl font-bold mb-1 ${tone.value}`}>{item.value}</div>
                <div className="text-gray-600 font-medium text-xs">{item.label}</div>
              </Card>
            </Col>
          );
        })}
      </Row>
    </div>
  );
};

export default BusinessStatsGrid;
