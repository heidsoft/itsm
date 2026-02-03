"use client";

import React from 'react';
import { Col, Row, Skeleton, Space, theme } from 'antd';
import { AdminHeader } from './components/AdminHeader';
import { SystemOverview } from './components/SystemOverview';
import { SystemHealth } from './components/SystemHealth';
import { RecentActivity } from './components/RecentActivity';
import { QuickActions } from './components/QuickActions';
import { SystemInfo } from './components/SystemInfo';
import { useAdminData } from './hooks/useAdminData';
import { useI18n } from '@/lib/i18n';

const AdminDashboardSkeleton: React.FC = () => {
  return (
    <div className="p-6">
      <Skeleton.Input className="w-full h-32 mb-6" active />
      <Skeleton active paragraph={{ rows: 4 }} />
      <Row gutter={[24, 24]} className="mt-6">
        <Col xs={24} lg={8}>
          <Skeleton active paragraph={{ rows: 6 }} />
        </Col>
        <Col xs={24} lg={16}>
          <Skeleton active paragraph={{ rows: 8 }} />
        </Col>
      </Row>
    </div>
  );
};

const AdminDashboard = () => {
  const { loading, stats } = useAdminData();
  const { t } = useI18n();

  if (loading) {
    return <AdminDashboardSkeleton />;
  }

  return (
    <div className="p-6">
      <AdminHeader />
      <div className="mb-6">
        <SystemOverview stats={stats} loading={loading} />
      </div>
      <Row gutter={[24, 24]} className="mb-6">
        <Col xs={24} lg={8}>
          <SystemHealth />
        </Col>
        <Col xs={24} lg={16}>
          <RecentActivity />
        </Col>
      </Row>
      <div className="mb-6">
        <QuickActions />
      </div>
      <SystemInfo />
    </div>
  );
};

export default AdminDashboard;
