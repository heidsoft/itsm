'use client';

import React from 'react';
import { BranchesOutlined } from '@ant-design/icons';
import { useI18n } from '@/lib/i18n';

interface TopologyViewProps {
  cis: any[];
  relations: any[];
}

export const TopologyView: React.FC<TopologyViewProps> = ({ cis, relations }) => {
    const { t } = useI18n();
  return (
    <div style={{ padding: 24 }}>
      <div
        style={{
          height: 600,
          border: '1px solid #f0f0f0',
          borderRadius: 8,
          backgroundColor: '#fafafa',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <div style={{ textAlign: 'center' }}>
          <div style={{ fontSize: 48, color: '#1890ff', marginBottom: 16 }}>
            <BranchesOutlined />
          </div>
          <h3>{t('cmdb.topologyViewTitle')}</h3>
          <p style={{ color: '#666', maxWidth: 400, margin: '0 auto' }}>
            {t('cmdb.topologyViewDescription')}
          </p>
          <div style={{ marginTop: 20, fontSize: 14, color: '#666' }}>
            {t('cmdb.ciCount', { cisCount: cis.length, relationsCount: relations.length })}
          </div>
        </div>
      </div>
    </div>
  );
};
