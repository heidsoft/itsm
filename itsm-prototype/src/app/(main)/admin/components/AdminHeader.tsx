'use client';

import React, { useState, useEffect } from 'react';
import { Card, Col, Row, Typography, theme } from 'antd';
import { useI18n } from '@/lib/i18n';

const { Title, Text } = Typography;

export const AdminHeader: React.FC = () => {
  const { token } = theme.useToken();
  const [currentTime, setCurrentTime] = useState(new Date());
  const { t } = useI18n();

  useEffect(() => {
    const timer = setInterval(() => {
      setCurrentTime(new Date());
    }, 1000);
    return () => clearInterval(timer);
  }, []);

  return (
    <Card
      style={{
        background: `linear-gradient(135deg, ${token.colorPrimary} 0%, #722ed1 100%)`,
        marginBottom: token.marginLG,
        border: 'none',
      }}
      styles={{ body: { padding: token.paddingLG } }}
    >
      <Row justify='space-between' align='middle'>
        <Col>
          <Title
            level={1}
            style={{
              color: 'white',
              margin: 0,
              marginBottom: token.marginSM,
            }}
          >
            {t('admin.title')}
          </Title>
          <Text
            style={{
              color: 'rgba(255,255,255,0.9)',
              fontSize: token.fontSizeLG,
            }}
          >
            {t('admin.welcome')}
          </Text>
          <br />
          <Text
            style={{
              color: 'rgba(255,255,255,0.7)',
              fontSize: token.fontSizeSM,
              marginTop: token.marginXS,
            }}
          >
            {t('admin.description')}
          </Text>
        </Col>
        <Col style={{ textAlign: 'right' }}>
          <div
            style={{
              color: 'white',
              fontSize: 24,
              fontFamily: 'monospace',
              marginBottom: 4,
            }}
          >
            {currentTime.toLocaleTimeString()}
          </div>
          <Text
            style={{
              color: 'rgba(255,255,255,0.7)',
              fontSize: token.fontSizeSM,
            }}
          >
            {currentTime.toLocaleDateString('zh-CN', {
              year: 'numeric',
              month: 'long',
              day: 'numeric',
              weekday: 'long',
            })}
          </Text>
        </Col>
      </Row>
    </Card>
  );
};
