'use client';

import React from 'react';
import { Card, Col, List, Row, Space, Tag, Typography, theme } from 'antd';
import { Settings, FileText, ArrowUpRight } from 'lucide-react';
import { useI18n } from '@/lib/i18n';

const { Text } = Typography;

const getCurrentYear = () => new Date().getFullYear();

export const SystemInfo: React.FC = () => {
  const { token } = theme.useToken();
  const { t } = useI18n();

  return (
    <Row gutter={[24, 24]}>
      {/* 系统信息 */}
      <Col xs={24} lg={12}>
        <Card
          title={
            <Space>
              <Settings className="w-5 h-5" />
              {t('admin.systemInfo')}
              <Tag color="gold">静态展示</Tag>
            </Space>
          }
          style={{ height: '100%' }}
        >
          <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <Text type="secondary">{t('admin.systemVersion')}</Text>
              <Text strong>ITSM Pro v2.5.0</Text>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <Text type="secondary">{t('admin.databaseVersion')}</Text>
              <Text strong>PostgreSQL 15.0</Text>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <Text type="secondary">{t('admin.licenseStatus')}</Text>
              <Tag color="success">{t('admin.licenseActivated')}</Tag>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <Text type="secondary">{t('admin.licenseExpiry')}</Text>
              <Text strong>{getCurrentYear() + 1}-12-31</Text>
            </div>
          </Space>
        </Card>
      </Col>

      {/* 帮助和支持 */}
      <Col xs={24} lg={12}>
        <Card
          title={
            <Space>
              <FileText className="w-5 h-5" />
              {t('admin.helpSupport')}
              <Tag color="blue">待接入</Tag>
            </Space>
          }
          style={{ height: '100%' }}
        >
          <List
            dataSource={[
              { title: t('admin.configGuide'), status: '规划中' },
              { title: t('admin.apiDocs'), status: '规划中' },
              { title: t('admin.techSupport'), status: '规划中' },
              { title: t('admin.updateLog'), status: '规划中' },
            ]}
            renderItem={item => (
              <List.Item>
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    padding: `${token.paddingSM}px 0`,
                    width: '100%',
                  }}
                >
                  <Text>{item.title}</Text>
                  <Space size={8}>
                    <Tag>{item.status}</Tag>
                    <ArrowUpRight className="w-4 h-4" style={{ color: token.colorTextSecondary }} />
                  </Space>
                </div>
              </List.Item>
            )}
          />
        </Card>
      </Col>
    </Row>
  );
};
