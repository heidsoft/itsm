'use client';

import React from 'react';
import { Card, Col, Row } from 'antd';
import { AlertTriangle, RefreshCw } from 'lucide-react';

interface ProblemStatsProps {
  metrics: {
    total: number;
    open: number;
    inProgress: number;
    resolved: number;
  };
}

export const ProblemStats: React.FC<ProblemStatsProps> = ({ metrics }) => {
  return (
    <div className='mb-4'>
      <Row gutter={[12, 12]}>
        <Col xs={24} sm={12} md={6} lg={6}>
          <Card
            className='text-center hover:shadow-lg transition-all duration-300 border-0 bg-gradient-to-br from-blue-500 to-blue-600 text-white'
            styles={{ body: { padding: '16px' } }}
          >
            <div className='w-10 h-10 bg-white/20 rounded-lg flex items-center justify-center mx-auto mb-3'>
              <AlertTriangle className='w-5 h-5 text-white' />
            </div>
            <div className='text-2xl font-bold mb-1'>{metrics.total}</div>
            <div className='text-blue-100 font-medium text-xs'>总问题数</div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6} lg={6}>
          <Card
            className='text-center hover:shadow-lg transition-all duration-300 border-0 bg-gradient-to-br from-orange-500 to-orange-600 text-white'
            styles={{ body: { padding: '16px' } }}
          >
            <div className='w-10 h-10 bg-white/20 rounded-lg flex items-center justify-center mx-auto mb-3'>
              <AlertTriangle className='w-5 h-5 text-white' />
            </div>
            <div className='text-2xl font-bold mb-1'>{metrics.open}</div>
            <div className='text-orange-100 font-medium text-xs'>待处理</div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6} lg={6}>
          <Card
            className='text-center hover:shadow-lg transition-all duration-300 border-0 bg-gradient-to-br from-cyan-500 to-cyan-600 text-white'
            styles={{ body: { padding: '16px' } }}
          >
            <div className='w-10 h-10 bg-white/20 rounded-lg flex items-center justify-center mx-auto mb-3'>
              <RefreshCw className='w-5 h-5 text-white' />
            </div>
            <div className='text-2xl font-bold mb-1'>{metrics.inProgress}</div>
            <div className='text-cyan-100 font-medium text-xs'>处理中</div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6} lg={6}>
          <Card
            className='text-center hover:shadow-lg transition-all duration-300 border-0 bg-gradient-to-br from-green-500 to-green-600 text-white'
            styles={{ body: { padding: '16px' } }}
          >
            <div className='w-10 h-10 bg-white/20 rounded-lg flex items-center justify-center mx-auto mb-3'>
              <AlertTriangle className='w-5 h-5 text-white' />
            </div>
            <div className='text-2xl font-bold mb-1'>{metrics.resolved}</div>
            <div className='text-green-100 font-medium text-xs'>已解决</div>
          </Card>
        </Col>
      </Row>
    </div>
  );
};
