'use client';

import React, { useEffect, useState } from 'react';
import { Row, Col, Empty, Form, App, Tabs, Card, Typography, Button, Tag, Alert } from 'antd';
import { Cloud, UserCog, ShieldCheck, Server, Database, Lock, Flame } from 'lucide-react';
import { useServiceCatalogData } from './hooks/useServiceCatalogData';
import { ServiceCatalogStats } from './components/ServiceCatalogStats';
import { ServiceCatalogFilters } from './components/ServiceCatalogFilters';
import { ServiceItemCard } from './components/ServiceItemCard';
import { CreateServiceModal } from './components/CreateServiceModal';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import { useI18n } from '@/lib/i18n';
import { CMDBApi } from '@/lib/api/cmdb-api';
import type { CIType, CloudService } from '@/types/biz/cmdb';

const { Title, Text } = Typography;

const ServiceCatalogSkeleton: React.FC = () => (
  <div>
    <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
      {[1, 2, 3, 4].map((_, index) => (
        <Col key={index} xs={24} sm={12} lg={6}>
          <Card loading className="rounded-lg" />
        </Col>
      ))}
    </Row>
    <Card loading className="mb-6" />
    <Row gutter={[24, 24]}>
      {Array.from({ length: 4 }).map((_, index) => (
        <Col key={index} xs={24} sm={12} md={8} lg={6}>
          <Card loading className="rounded-lg h-full" />
        </Col>
      ))}
    </Row>
  </div>
);

// 分类配置
const categoryConfig = [
  { key: 'all', label: '全部服务', icon: <Server /> },
  { key: 'cloud', label: '云资源服务', icon: <Cloud /> },
  { key: 'account', label: '账号与权限', icon: <UserCog /> },
  { key: 'security', label: '安全服务', icon: <ShieldCheck /> },
  { key: 'database', label: '数据库服务', icon: <Database /> },
  { key: 'network', label: '网络服务', icon: <Lock /> },
];

export default function ServiceCatalogPage() {
  const { message } = App.useApp();
  const { t } = useI18n();
  const [activeCategory, setActiveCategory] = useState('all');
  const [creating, setCreating] = useState(false);

  const {
    catalogs,
    loading,
    error,
    categoryFilter,
    ciTypeFilter,
    cloudServiceFilter,
    stats,
    setSearchText,
    setCategoryFilter,
    setCiTypeFilter,
    setCloudServiceFilter,
    loadServiceCatalogs,
  } = useServiceCatalogData();

  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [createForm] = Form.useForm();
  const [ciTypes, setCiTypes] = useState<CIType[]>([]);
  const [cloudServices, setCloudServices] = useState<CloudService[]>([]);
  const [optionsLoading, setOptionsLoading] = useState(false);

  // 根据选中分类过滤服务
  const filteredCatalogs = React.useMemo(() => {
    let result = catalogs;

    const categoryMap: Record<string, string[]> = {
      cloud: ['云资源服务', 'Cloud Service'],
      account: ['账号与权限', 'Account Service'],
      security: ['安全服务', 'Security Service'],
      database: ['数据库服务', 'Database Service'],
      network: ['网络服务', 'Network Service'],
    };

    const targetCategories = categoryMap[activeCategory] || [];
    if (activeCategory !== 'all') {
      result = result.filter(catalog => targetCategories.some(cat => String(catalog.category).includes(cat)));
    }
    if (categoryFilter) result = result.filter(catalog => String(catalog.category).includes(categoryFilter));
    if (ciTypeFilter) result = result.filter(catalog => catalog.ciTypeId === ciTypeFilter);
    if (cloudServiceFilter) result = result.filter(catalog => catalog.cloudServiceId === cloudServiceFilter);
    return result;
  }, [catalogs, activeCategory, categoryFilter, ciTypeFilter, cloudServiceFilter]);

  const popularCatalogs = React.useMemo(
    () => [...catalogs]
      .filter(catalog => catalog.status === 'published')
      .sort((a, b) => (b.requestCount ?? 0) - (a.requestCount ?? 0))
      .slice(0, 3),
    [catalogs]
  );

  // 加载选项数据
  useEffect(() => {
    const loadOptions = async () => {
      try {
        setOptionsLoading(true);
        const [types, services] = await Promise.all([
		  CMDBApi.getCITypes(),
          CMDBApi.getCloudServices(),
        ]);
        setCiTypes(types || []);
        setCloudServices(services || []);
      } catch (error) {
        message.error(t('common.getFailed'));
      } finally {
        setOptionsLoading(false);
      }
    };
    loadOptions();
  }, [message, t]);

  // 处理分类标签切换
  const handleCategoryChange = (category: string) => {
    setActiveCategory(category);

    // 更新筛选条件
    const categoryMap: Record<string, string> = {
      all: '',
      cloud: '云资源服务',
      account: '账号与权限',
      security: '安全服务',
      database: '数据库服务',
      network: '网络服务',
    };
    setCategoryFilter(categoryMap[category] || '');
  };

  const handleCreateService = () => {
    setCreateModalVisible(true);
  };

  const handleCreateServiceConfirm = async () => {
    if (creating) return;
    try {
      setCreating(true);
      const values = await createForm.validateFields();
      await ServiceCatalogApi.createService({
        name: values.name,
        category: values.category,
        shortDescription: values.description,
        availability: {
          responseTime: values.deliveryTime ? Number(values.deliveryTime) : undefined,
        },
        tags: [],
      });

      message.success(t('serviceCatalog.createServiceSuccess'));
      setCreateModalVisible(false);
      createForm.resetFields();
      loadServiceCatalogs();
    } catch (error) {
      console.error(t('serviceCatalog.createServiceFailed'), error);
      message.error(t('serviceCatalog.createServiceFailed'));
    } finally {
      setCreating(false);
    }
  };

  if (loading && catalogs.length === 0) {
    return <ServiceCatalogSkeleton />;
  }

  return (
    <div className="p-6 min-h-screen" style={{ backgroundColor: 'var(--color-bg-secondary, #f9fafb)' }}>
      {/* 页面头部 */}
      <div className="mb-6">
        <Title level={2} style={{ marginBottom: 4 }}>
          服务目录
        </Title>
        <Text type="secondary">
          浏览和申请IT服务，支持云资源、账号权限、安全服务等多种服务类型
        </Text>
      </div>

      {/* 统计卡片 */}
      <ServiceCatalogStats stats={stats} />

      {popularCatalogs.length > 0 && (
        <Card
          className="mb-6"
          title={<span className="flex items-center gap-2"><Flame size={18} className="text-orange-500" />热门服务</span>}
          extra={<Text type="secondary">按申请热度推荐</Text>}
        >
          <Row gutter={[16, 16]}>
            {popularCatalogs.map((catalog, index) => (
              <Col key={catalog.id} xs={24} md={8}>
                <button
                  type="button"
                  className="flex w-full items-center justify-between rounded-lg border border-gray-200 bg-white p-4 text-left transition-colors hover:border-blue-400 hover:bg-blue-50"
                  onClick={() => window.location.assign(`/service-catalog/request/${catalog.id}`)}
                >
                  <span><span className="block font-medium">{catalog.name}</span><span className="text-xs text-gray-500">{catalog.category}</span></span>
                  <Tag color={index === 0 ? 'volcano' : 'blue'}>{catalog.requestCount ?? 0} 次申请</Tag>
                </button>
              </Col>
            ))}
          </Row>
        </Card>
      )}

      {/* 分类标签页 */}
      <Card className="mb-6">
        <div className="mb-3 flex flex-wrap items-center gap-2">
          <Text strong>快速选择：</Text>
          {categoryConfig.slice(1).map(category => (
            <Button key={category.key} size="small" type={activeCategory === category.key ? 'primary' : 'default'} onClick={() => handleCategoryChange(category.key)}>
              {category.label}
            </Button>
          ))}
        </div>
        <Tabs
          activeKey={activeCategory}
          onChange={handleCategoryChange}
          type="card"
          size="large"
          items={categoryConfig.map(cat => {
            const categoryMap: Record<string, string[]> = {
              cloud: ['云资源服务', 'Cloud Service'],
              account: ['账号与权限', 'Account Service'],
              security: ['安全服务', 'Security Service'],
              database: ['数据库服务', 'Database Service'],
              network: ['网络服务', 'Network Service'],
            };
            const keywords = categoryMap[cat.key] || [];
            const count =
              cat.key === 'all'
                ? catalogs.length
                : catalogs.filter(c =>
                    keywords.some(k => String(c.category).includes(k))
                  ).length;
            return {
              key: cat.key,
              label: (
                <span className="flex items-center gap-2">
                  {cat.icon}
                  {cat.label}
                  <span className="text-xs text-gray-400 ml-1">({count})</span>
                </span>
              ),
            };
          })}
        />
      </Card>

      {/* 筛选和搜索 */}
      {error && (
        <Alert className="mb-4" type="error" showIcon message="服务目录加载失败" description={error}
          action={<Button onClick={loadServiceCatalogs}>重新加载</Button>} />
      )}
      <ServiceCatalogFilters
        onSearch={setSearchText}
        onCategoryFilterChange={setCategoryFilter}
        onCITypeFilterChange={setCiTypeFilter}
        onCloudServiceFilterChange={setCloudServiceFilter}
        ciTypes={ciTypes}
        cloudServices={cloudServices}
        optionsLoading={optionsLoading}
        onCreateService={handleCreateService}
        onRefresh={loadServiceCatalogs}
      />

      {/* 服务列表 */}
      {filteredCatalogs.length === 0 ? (
        <Card className="rounded-lg">
          <Empty
            description={
              <div>
                <p className="text-gray-500 mb-4">暂无符合条件的服务</p>
                <Button type="primary" onClick={handleCreateService}>
                  创建第一个服务
                </Button>
              </div>
            }
          />
        </Card>
      ) : (
        <Row gutter={[24, 24]}>
          {filteredCatalogs.map(catalog => (
            <Col key={catalog.id} xs={24} sm={12} md={8} lg={6}>
              <ServiceItemCard catalog={catalog} />
            </Col>
          ))}
        </Row>
      )}

      {/* 创建服务模态框 */}
      <CreateServiceModal
        visible={createModalVisible}
        onCancel={() => {
          setCreateModalVisible(false);
          createForm.resetFields();
        }}
        onConfirm={handleCreateServiceConfirm}
        form={createForm}
        loading={creating}
      />
    </div>
  );
}
