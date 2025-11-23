'use client';

import React from 'react';
import { Card, Col, Row } from 'antd';
import { GitBranch, AlertTriangle } from 'lucide-react';
import { ChangeStats } from '@/lib/services/change-service';

interface ChangeStatsProps {
  stats: ChangeStats;
}

export const ChangeStats: React.FC<ChangeStatsProps> = ({ stats }) => {
  return (
    <Row gutter={[16, 16]} className='mb-6'>
      <Col xs={24} sm={12} md={6}>
        <Card className='bg-gradient-to-br from-blue-500 to-blue-600 border-0 shadow-lg hover:shadow-xl transition-all duration-300'>
          <div className='text-center'>
            <GitBranch className='mx-auto mb-3 text-white' size={32} />
            <div className='text-3xl font-bold text-white'>{stats.total}</div>
            <div className='text-blue-100'>总变更数</div>
          </div>
        </Card>
      </Col>
      <Col xs={24} sm={12} md={6}>
        <Card className='bg-gradient-to-br from-orange-500 to-orange-600 border-0 shadow-lg hover:shadow-xl transition-all duration-300'>
          <div className='text-center'>
            <AlertTriangle className='mx-auto mb-3 text-white' size={32} />
            <div className='text-3xl font-bold text-white'>{stats.pending}</div>
            <div className='text-orange-100'>待审批</div>
          </div>
        </Card>
      </Col>
      <Col xs={24} sm={12} md={6}>
        <Card className='bg-gradient-to-br from-green-500 to-green-600 border-0 shadow-lg hover:shadow-xl transition-all duration-300'>
          <div className='text-center'>
            <GitBranch className='mx-auto mb-3 text-white' size={32} />
            <div className='text-3xl font-bold text-white'>{stats.approved}</div>
            <div className='text-green-100'>已批准</div>
          </div>
        </Card>
      </Col>
      <Col xs={24} sm={12} md={6}>
        <Card className='bg-gradient-to-br from-purple-500 to-purple-600 border-0 shadow-lg hover:shadow-xl transition-all duration-300'>
          <div className='text-center'>
            <GitBranch className='mx-auto mb-3 text-white' size={32} />
            <div className='text-3xl font-bold text-white'>{stats.completed}</div>
            <div className='text-purple-100'>已完成</div>
          </div>
        </Card>
      </Col>
    </Row>
  );
};
