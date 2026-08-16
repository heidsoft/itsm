'use client';

import React, { useState, useEffect } from 'react';
import { App } from 'antd';
import { Package, CheckCircle, Clock, Plus } from 'lucide-react';
import { useRouter } from 'next/navigation';
import AssetList from '@/components/asset/AssetList';
import { AssetApi } from '@/lib/api/asset-api';
import BusinessPageTemplate from '@/components/layout/BusinessPageTemplate';

export default function AssetsPage() {
  const router = useRouter();
  const { message } = App.useApp();
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
      message.error('获取资产统计数据失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStats();
  }, []);

  const statsData = [
    {
      label: '资产总数',
      value: stats.totalAssets,
      color: '#1890ff',
      icon: <Package className="h-5 w-5" />,
    },
    {
      label: '使用中',
      value: stats.inUse,
      color: '#52c41a',
      icon: <CheckCircle className="h-5 w-5" />,
    },
    {
      label: '可用',
      value: stats.available,
      color: '#1890ff',
      icon: <Package className="h-5 w-5" />,
    },
    {
      label: '维护中',
      value: stats.maintenance,
      color: '#fa8c16',
      icon: <Clock className="h-5 w-5" />,
    },
  ];

  return (
    <BusinessPageTemplate
      title="资产管理"
      description="管理企业IT资产，包括硬件、软件、云资源和许可证"
      stats={statsData}
      statsLoading={loading}
      searchPlaceholder="搜索资产名称、编号、型号..."
      searchValue={searchValue}
      onSearch={setSearchValue}
      primaryAction={{
        label: '新增资产',
        icon: <Plus className="h-4 w-4" />,
        onClick: () => router.push('/assets/new'),
      }}
      showViewSwitch={false}
    >
      <AssetList showActions={false} />
    </BusinessPageTemplate>
  );
}
