'use client';

import React from 'react';
import { Card, Button, Table } from 'antd';
import { ClusterOutlined, CloseOutlined } from '@ant-design/icons';
import { useI18n } from '@/lib/i18n';

interface RelationGraphProps {
  selectedCI: any;
  relations: any[];
  cis: any[];
  onClose: () => void;
}

export const RelationGraph: React.FC<RelationGraphProps> = ({
  selectedCI,
  relations,
  cis,
  onClose,
}) => {
    const { t } = useI18n();
  if (!selectedCI) {
    return (
      <div style={{ textAlign: 'center', padding: '100px 0', color: '#666' }}>
        <div style={{ margin: '0 auto 16px', color: '#1890ff', fontSize: 48 }}>
          <ClusterOutlined />
        </div>
        <p>{t('cmdb.selectCIForRelations')}</p>
      </div>
    );
  }

  const relatedRelations = relations.filter(
    rel => rel.source === selectedCI.id || rel.target === selectedCI.id
  );

  return (
    <div style={{ padding: 24 }}>
      <div
        style={{
          marginBottom: 24,
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <h3 style={{ margin: 0 }}>{selectedCI.name} {t('cmdb.ciRelations')}</h3>
        <Button icon={<CloseOutlined />} onClick={onClose}>
          {t('cmdb.close')}
        </Button>
      </div>

      <div
        style={{
          height: 500,
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
            <ClusterOutlined />
          </div>
          <div style={{ fontSize: 16, color: '#666' }}>{t('cmdb.relationsCount', { count: relatedRelations.length })}</div>
        </div>
      </div>

      <div style={{ marginTop: 24 }}>
        <h4>{t('cmdb.relationDetails')}</h4>
        <Table
          columns={[
            {
              title: t('cmdb.sourceCI'),
              dataIndex: 'source',
              key: 'source',
              render: (source: string) => {
                const ci = cis.find(c => c.id === source);
                return ci ? ci.name : source;
              },
            },
            {
              title: t('cmdb.targetCI'),
              dataIndex: 'target',
              key: 'target',
              render: (target: string) => {
                const ci = cis.find(c => c.id === target);
                return ci ? ci.name : target;
              },
            },
            {
              title: t('cmdb.relationType'),
              dataIndex: 'type',
              key: 'type',
            },
            {
              title: t('cmdb.relationDescription'),
              dataIndex: 'description',
              key: 'description',
            },
          ]}
          dataSource={relatedRelations}
          pagination={false}
          rowKey='id'
        />
      </div>
    </div>
  );
};
