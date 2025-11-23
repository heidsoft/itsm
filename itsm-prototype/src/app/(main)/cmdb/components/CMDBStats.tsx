'use client';

import React from 'react';
import { Card, Col, Row } from 'antd';
import {
  DatabaseOutlined,
  CloudServerOutlined,
  DesktopOutlined,
  HddOutlined,
} from '@ant-design/icons';
import { useI18n } from '@/lib/i18n';

interface CMDBStatsProps {
  cis: any[];
}

export const CMDBStats: React.FC<CMDBStatsProps> = ({ cis }) => {
  const { t } = useI18n();
  return (
    <div style={{ marginBottom: 16 }}>
      <Row gutter={[12, 12]}>
        <Col xs={24} sm={12} md={6} lg={6}>
          <Card
            style={{
              background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
              border: 'none',
              borderRadius: 12,
            }}
            styles={{ body: { padding: '16px' } }}
          >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div>
                <div style={{ color: 'rgba(255, 255, 255, 0.8)', fontSize: 14, marginBottom: 8 }}>
                  {t('cmdb.totalCIs')}
                </div>
                <div style={{ color: '#fff', fontSize: 32, fontWeight: 'bold', lineHeight: 1 }}>
                  {cis.length}
                </div>
              </div>
              <div
                style={{
                  width: 56,
                  height: 56,
                  backgroundColor: 'rgba(255, 255, 255, 0.2)',
                  borderRadius: 16,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: '#fff',
                  fontSize: 24,
                }}
              >
                <DatabaseOutlined />
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6} lg={6}>
          <Card
            style={{
              background: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)',
              border: 'none',
              borderRadius: 12,
            }}
            styles={{ body: { padding: '16px' } }}
          >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div>
                <div style={{ color: 'rgba(255, 255, 255, 0.8)', fontSize: 14, marginBottom: 8 }}>
                  {t('cmdb.cloudServers')}
                </div>
                <div style={{ color: '#fff', fontSize: 32, fontWeight: 'bold', lineHeight: 1 }}>
                  {cis.filter(ci => ci.type === 'Cloud Server').length}
                </div>
              </div>
              <div
                style={{
                  width: 56,
                  height: 56,
                  backgroundColor: 'rgba(255, 255, 255, 0.2)',
                  borderRadius: 16,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: '#fff',
                  fontSize: 24,
                }}
              >
                <CloudServerOutlined />
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6} lg={6}>
          <Card
            style={{
              background: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
              border: 'none',
              borderRadius: 12,
            }}
            styles={{ body: { padding: '16px' } }}
          >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div>
                <div style={{ color: 'rgba(255, 255, 255, 0.8)', fontSize: 14, marginBottom: 8 }}>
                  {t('cmdb.running')}
                </div>
                <div style={{ color: '#fff', fontSize: 32, fontWeight: 'bold', lineHeight: 1 }}>
                  {cis.filter(ci => ci.status === 'Running').length}
                </div>
              </div>
              <div
                style={{
                  width: 56,
                  height: 56,
                  backgroundColor: 'rgba(255, 255, 255, 0.2)',
                  borderRadius: 16,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: '#fff',
                  fontSize: 24,
                }}
              >
                <DesktopOutlined />
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6} lg={6}>
          <Card
            style={{
              background: 'linear-gradient(135deg, #fa709a 0%, #fee140 100%)',
              border: 'none',
              borderRadius: 12,
            }}
            styles={{ body: { padding: '16px' } }}
          >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div>
                <div style={{ color: 'rgba(255, 255, 255, 0.8)', fontSize: 14, marginBottom: 8 }}>
                  {t('cmdb.maintenance')}
                </div>
                <div style={{ color: '#fff', fontSize: 32, fontWeight: 'bold', lineHeight: 1 }}>
                  {cis.filter(ci => ci.status === 'Maintenance').length}
                </div>
              </div>
              <div
                style={{
                  width: 56,
                  height: 56,
                  backgroundColor: 'rgba(255, 255, 255, 0.2)',
                  borderRadius: 16,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: '#fff',
                  fontSize: 24,
                }}
              >
                <HddOutlined />
              </div>
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  );
};
