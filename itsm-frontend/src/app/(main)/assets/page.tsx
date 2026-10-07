'use client';

import React, { useState, useEffect } from 'react';
import { App } from 'antd';
import { Package, CheckCircle, Clock, Plus } from 'lucide-react';
import { useRouter } from 'next/navigation';
import AssetList from '@/components/asset/AssetList';
import { AssetApi } from '@/lib/api/asset-api';
import BusinessPageTemplate from '@/components/layout/BusinessPageTemplate';
import { useI18n } from '@/lib/i18n/useI18n';

export default function AssetsPage() {
  const router = useRouter();
  const { message } = App.useApp();
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [searchValue, setSearchValue] = useState('');
  const [stats, setStats] = useState({
    totalAssets: 0,
    inUse: 0,
    available: 0,
    maintenance: 0,
  });

  const fetchStats = async () => {
    try {
      setLoading(true);
      const assetStats = await AssetApi.getAssetStats();
      setStats({
        totalAssets: assetStats.total || 0,
        inUse: assetStats.inUse || 0,
        available: assetStats.available || 0,
        maintenance: assetStats.maintenance || 0,
      });
    } catch (error) {
      console.error('Failed to fetch asset stats:', error);
      message.error(t('assets.loadStatsFailed'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStats();
  }, []);

  const statsData = [
    {
      label: t('assets.totalAssets'),
      value: stats.totalAssets,
      color: '#1890ff',
      icon: <Package className="h-5 w-5" />,
    },
    {
      label: t('assets.inUse'),
      value: stats.inUse,
      color: '#52c41a',
      icon: <CheckCircle className="h-5 w-5" />,
    },
    {
      label: t('assets.available'),
      value: stats.available,
      color: '#1890ff',
      icon: <Package className="h-5 w-5" />,
    },
    {
      label: t('assets.maintenance'),
      value: stats.maintenance,
      color: '#fa8c16',
      icon: <Clock className="h-5 w-5" />,
    },
  ];

  return (
    <BusinessPageTemplate
      title={t('assets.title')}
      description={t('assets.description')}
      stats={statsData}
      statsLoading={loading}
      searchPlaceholder={t('assets.searchPlaceholder')}
      searchValue={searchValue}
      onSearch={setSearchValue}
      primaryAction={{
        label: t('assets.addAsset'),
        icon: <Plus className="h-4 w-4" />,
        onClick: () => router.push('/assets/new'),
      }}
      showViewSwitch={false}
    >
      <AssetList showActions={false} />
    </BusinessPageTemplate>
  );
}
