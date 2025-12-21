'use client';

import React from 'react';
import { Card, Typography } from 'antd';
import { SLAMonitorDashboard } from '@/components/business/SLAMonitorDashboard';

const { Title } = Typography;

const SLAMonitorPage = () => {
  return (
    <div className="p-6">
      <div className="mb-6">
        <Title level={2}>SLA实时监控大屏</Title>
        <p className="text-gray-600">实时监控SLA执行情况和关键指标</p>
      </div>
      
      <Card>
        <SLAMonitorDashboard 
          autoRefresh={true} 
          refreshInterval={30}
        />
      </Card>
    </div>
  );
};

export default SLAMonitorPage;