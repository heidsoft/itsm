'use client';

import React, { useState, useEffect } from 'react';
import { Card, Col, List, Row, Space, Tag, Typography, theme } from 'antd';
import { Settings, FileText, ArrowUpRight } from 'lucide-react';
import { useI18n } from '@/lib/i18n';
import { httpClient } from '@/lib/api/http-client';

const { Text } = Typography;

const REPO_URL = 'https://github.com/heidsoft/itsm';

export const SystemInfo: React.FC = () => {
  const { token } = theme.useToken();
  const { t } = useI18n();
  const [version, setVersion] = useState<string | null>(null);

  useEffect(() => {
    // 版本号来自后端 /api/v1/version（APP_VERSION 注入），取不到时显示占位符而不是虚构值。
    let cancelled = false;
    httpClient
      .get<{ version?: string }>('/api/v1/version')
      .then(res => {
        if (!cancelled && res?.version) setVersion(res.version);
      })
      .catch(() => {
        /* 接口不可用时保持 '—' */
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <Row gutter={[24, 24]}>
      {/* 系统信息 */}
      <Col xs={24} lg={12}>
        <Card
          title={
            <Space>
              <Settings className="w-5 h-5" />
              {t('admin.systemInfo')}
            </Space>
          }
          style={{ height: '100%' }}
        >
          <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <Text type="secondary">{t('admin.systemVersion')}</Text>
              <Text strong>{version ?? '—'}</Text>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <Text type="secondary">{t('admin.licenseType')}</Text>
              <a href={`${REPO_URL}/blob/main/LICENSE`} target="_blank" rel="noreferrer">
                <Text strong>Apache-2.0</Text>
              </a>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <Text type="secondary">{t('admin.codeRepository')}</Text>
              <a href={REPO_URL} target="_blank" rel="noreferrer">
                <Text strong>heidsoft/itsm</Text>
              </a>
            </div>
          </Space>
        </Card>
      </Col>

      {/* 帮助和支持：全部指向真实可用的资源 */}
      <Col xs={24} lg={12}>
        <Card
          title={
            <Space>
              <FileText className="w-5 h-5" />
              {t('admin.helpSupport')}
            </Space>
          }
          style={{ height: '100%' }}
        >
          <List
            dataSource={[
              { title: t('admin.productReadme'), href: REPO_URL },
              { title: t('admin.apiDocs'), href: `${REPO_URL}/blob/main/docs/api/API_REFERENCE.md` },
              { title: t('admin.updateLog'), href: `${REPO_URL}/blob/main/CHANGELOG.md` },
              { title: t('admin.techSupport'), href: `${REPO_URL}/issues` },
            ]}
            renderItem={item => (
              <List.Item>
                <a
                  href={item.href}
                  target="_blank"
                  rel="noreferrer"
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
                    <Tag color="blue">在线</Tag>
                    <ArrowUpRight className="w-4 h-4" style={{ color: token.colorTextSecondary }} />
                  </Space>
                </a>
              </List.Item>
            )}
          />
        </Card>
      </Col>
    </Row>
  );
};
