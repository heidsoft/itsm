'use client';

import React from 'react';
import { Card, Col, Row } from 'antd';
import styles from './BusinessStatsGrid.module.css';

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
    background: string;
    border: string;
    iconBackground: string;
    iconColor: string;
    valueColor: string;
  }
> = {
  blue: {
    background: 'linear-gradient(135deg, #eff6ff 0%, #dbeafe 100%)',
    border: '#bfdbfe',
    iconBackground: '#dbeafe',
    iconColor: '#2563eb',
    valueColor: '#1d4ed8',
  },
  orange: {
    background: 'linear-gradient(135deg, #fff7ed 0%, #ffedd5 100%)',
    border: '#fed7aa',
    iconBackground: '#ffedd5',
    iconColor: '#ea580c',
    valueColor: '#c2410c',
  },
  green: {
    background: 'linear-gradient(135deg, #f0fdf4 0%, #dcfce7 100%)',
    border: '#bbf7d0',
    iconBackground: '#dcfce7',
    iconColor: '#16a34a',
    valueColor: '#15803d',
  },
  purple: {
    background: 'linear-gradient(135deg, #faf5ff 0%, #f3e8ff 100%)',
    border: '#e9d5ff',
    iconBackground: '#f3e8ff',
    iconColor: '#9333ea',
    valueColor: '#7e22ce',
  },
  cyan: {
    background: 'linear-gradient(135deg, #ecfeff 0%, #cffafe 100%)',
    border: '#a5f3fc',
    iconBackground: '#cffafe',
    iconColor: '#0891b2',
    valueColor: '#0e7490',
  },
  red: {
    background: 'linear-gradient(135deg, #fef2f2 0%, #fee2e2 100%)',
    border: '#fecaca',
    iconBackground: '#fee2e2',
    iconColor: '#dc2626',
    valueColor: '#b91c1c',
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
          const toneStyle = {
            '--business-stat-bg': tone.background,
            '--business-stat-border': tone.border,
            '--business-stat-icon-bg': tone.iconBackground,
            '--business-stat-icon-color': tone.iconColor,
            '--business-stat-value-color': tone.valueColor,
          } as React.CSSProperties;

          return (
            <Col key={`${item.label}-${index}`} xs={24} sm={12} md={6} lg={6} className="flex">
              <Card
                className={`w-full text-center shadow-sm hover:shadow-lg transition-all duration-300 ${styles.card}`}
                style={toneStyle}
                styles={{ body: { padding: '16px' } }}
              >
                <div
                  className={`mx-auto mb-3 h-10 w-10 rounded-lg flex items-center justify-center ${styles.icon}`}
                >
                  {item.icon}
                </div>
                <div className={`${styles.value} text-2xl font-bold mb-1`}>{item.value}</div>
                <div className={`${styles.label} font-medium text-xs`}>{item.label}</div>
              </Card>
            </Col>
          );
        })}
      </Row>
    </div>
  );
};

export default BusinessStatsGrid;
