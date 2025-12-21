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
  const { token } = theme.useToken();
  return (
    <div style={{ padding: token.paddingLG }}>
      <Skeleton.Input style={{ width: '100%', height: '120px', marginBottom: token.marginLG }} active />
      <Skeleton active paragraph={{ rows: 4 }} />
      <Row gutter={[24, 24]} style={{ marginTop: token.marginLG }}>
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
  const { token } = theme.useToken();
  const { loading } = useAdminData();
  const { t } = useI18n();

  if (loading) {
    return <AdminDashboardSkeleton />;
  }

  return (
    <div style={{ padding: token.paddingLG }}>
      <AdminHeader />
      <div style={{ marginBottom: token.marginLG }}>
        <SystemOverview />
      </div>
      <Row gutter={[24, 24]} style={{ marginBottom: token.marginLG }}>
        <Col xs={24} lg={8}>
          <SystemHealth />
        </Col>
        <Col xs={24} lg={16}>
          <RecentActivity />
        </Col>
      </Row>
      <div style={{ marginBottom: token.marginLG }}>
        <QuickActions />
      </div>
      <SystemInfo />
    </div>
  );
};

export default AdminDashboard;
