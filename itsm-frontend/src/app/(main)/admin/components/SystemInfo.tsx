'use client';

import React from 'react';
import Link from 'next/link';
import { Card, Col, List, Row, Space, Tag, Typography, theme } from 'antd';
import { Settings, FileText, ArrowUpRight } from 'lucide-react';
import { useI18n } from '@/lib/i18n';

const { Text } = Typography;

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
              <Settings className='w-5 h-5' />
              {t('admin.systemInfo')}
            </Space>
          }
          style={{ height: '100%' }}
        >
          <Space orientation='vertical' size='middle' style={{ width: '100%' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <Text type='secondary'>{t('admin.systemVersion')}</Text>
              <Text strong>ITSM Pro v2.0.1</Text>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <Text type='secondary'>{t('admin.databaseVersion')}</Text>
              <Text strong>PostgreSQL 14.2</Text>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <Text type='secondary'>{t('admin.licenseStatus')}</Text>
              <Tag color='success'>{t('admin.licenseActivated')}</Tag>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <Text type='secondary'>{t('admin.licenseExpiry')}</Text>
              <Text strong>2024-12-31</Text>
            </div>
          </Space>
        </Card>
      </Col>

      {/* 帮助和支持 */}
      <Col xs={24} lg={12}>
        <Card
          title={
            <Space>
              <FileText className='w-5 h-5' />
              {t('admin.helpSupport')}
            </Space>
          }
          style={{ height: '100%' }}
        >
          <List
            dataSource={[
              { title: t('admin.configGuide'), href: '#' },
              { title: t('admin.apiDocs'), href: '#' },
              { title: t('admin.techSupport'), href: '#' },
              { title: t('admin.updateLog'), href: '#' },
            ]}
            renderItem={item => (
              <List.Item>
                <Link
                  href={item.href}
                  style={{ textDecoration: 'none', width: '100%' }}
                >
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
                    <ArrowUpRight
                      className='w-4 h-4'
                      style={{ color: token.colorTextSecondary }}
                    />
                  </div>
                </Link>
              </List.Item>
            )}
          />
        </Card>
      </Col>
    </Row>
  );
};