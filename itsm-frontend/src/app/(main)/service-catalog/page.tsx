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
import { usePermissions } from '@/lib/hooks/use-permissions';

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

// Static icon pool cycled for dynamic categories
const CATEGORY_ICONS = [Cloud, UserCog, ShieldCheck, Database, Lock, Server];

export default function ServiceCatalogPage() {
  const { message } = App.useApp();
  const { t } = useI18n();
  const { hasPermission } = usePermissions();
  const canManageCatalog = hasPermission('service_catalog', 'write');
  const [activeCategory, setActiveCategory] = useState('all');
  const [creating, setCreating] = useState(false);

  const {
    catalogs,
    loading,
    error,
    total,
    truncated,
    limit,
    stats,
    categoryFilter,
    ciTypeFilter,
    cloudServiceFilter,
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

  // Derive unique categories dynamically from loaded data
  const dynamicCategories = React.useMemo(() => {
    const set = new Set<string>();
    catalogs.forEach(c => {
      const cat = String(c.category || '').trim();
      if (cat) set.add(cat);
    });
    return Array.from(set).sort();
  }, [catalogs]);

  const categoryConfig = React.useMemo(() => {
    const items = [{ key: 'all', label: t('serviceCatalog.categories.all'), icon: <Server /> }];
    dynamicCategories.forEach((cat, idx) => {
      const IconComp = CATEGORY_ICONS[idx % CATEGORY_ICONS.length];
      items.push({ key: cat, label: cat, icon: <IconComp /> });
    });
    return items;
  }, [dynamicCategories, t]);

  // 根据选中分类过滤服务
  const filteredCatalogs = React.useMemo(() => {
    let result = catalogs;
    if (activeCategory !== 'all') {
      result = result.filter(catalog => String(catalog.category) === activeCategory);
    }
    if (categoryFilter) result = result.filter(catalog => String(catalog.category).includes(categoryFilter));
    if (ciTypeFilter) result = result.filter(catalog => catalog.ciTypeId === ciTypeFilter);
    if (cloudServiceFilter) result = result.filter(catalog => catalog.cloudServiceId === cloudServiceFilter);
    return result;
  }, [catalogs, activeCategory, categoryFilter, ciTypeFilter, cloudServiceFilter]);

  // 后端服务目录 DTO 目前没有请求量投影，requestCount 全为空时这张卡就是不成立的：
  // 只排真正有计数的目录，否则整屏「0 个请求」的“热门服务”比不显示更误导。
  const popularCatalogs = React.useMemo(
    () =>
      catalogs
        .filter(catalog => (catalog.requestCount ?? 0) > 0)
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
        setCloudServices(services.items);
      } catch {
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
    // categoryFilter is for text-based sub-filtering from the filters bar
    if (category === 'all') {
      setCategoryFilter('');
    }
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
        deliveryDays: values.deliveryDays,
        ...(values.status ? { status: values.status } : {}),
        tags: [],
      });

      message.success(t('serviceCatalog.createServiceSuccess'));
      setCreateModalVisible(false);
      createForm.resetFields();
      loadServiceCatalogs();
    } catch (error) {
      message.error(error instanceof Error ? error.message : t('serviceCatalog.createServiceFailed'));
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
          {t('serviceCatalog.title')}
        </Title>
        <Text type="secondary">
          {t('serviceCatalog.pageDescription')}
        </Text>
      </div>

      {/* 统计卡片 */}
      <ServiceCatalogStats stats={stats} />

      {popularCatalogs.length > 0 && (
        <Card
          className="mb-6"
          title={<span className="flex items-center gap-2"><Flame size={18} className="text-orange-500" />{t('serviceCatalog.popularServices')}</span>}
          extra={<Text type="secondary">{t('serviceCatalog.popularRecommended')}</Text>}
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
                  <Tag color={index === 0 ? 'volcano' : 'blue'}>{catalog.requestCount ?? 0} {t('serviceCatalog.requestsCount')}</Tag>
                </button>
              </Col>
            ))}
          </Row>
        </Card>
      )}

      {/* 分类标签页 */}
      <Card className="mb-6">
        <div className="mb-3 flex flex-wrap items-center gap-2">
          <Text strong>{t('serviceCatalog.quickSelect')}</Text>
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
            const count =
              cat.key === 'all'
                ? catalogs.length
                : catalogs.filter(c => String(c.category) === cat.key).length;
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
        <Alert className="mb-4" type="error" showIcon title={t('serviceCatalog.loadFailed')} description={error}
          action={<Button onClick={loadServiceCatalogs}>{t('serviceCatalog.reload')}</Button>} />
      )}
      {truncated && (
        <Alert
          className="mb-4"
          type="warning"
          showIcon
          title={`共有 ${total} 个已发布目录，本页最多载入 ${limit} 个`}
          description="分类标签计数与筛选只覆盖已载入的部分，请用搜索词收窄范围。"
        />
      )}
      <ServiceCatalogFilters
        onSearch={setSearchText}
        onCategoryFilterChange={setCategoryFilter}
        onCITypeFilterChange={setCiTypeFilter}
        onCloudServiceFilterChange={setCloudServiceFilter}
        ciTypes={ciTypes}
        cloudServices={cloudServices}
        optionsLoading={optionsLoading}
        onCreateService={canManageCatalog ? handleCreateService : undefined}
        onRefresh={loadServiceCatalogs}
      />

      {/* 服务列表 */}
      {filteredCatalogs.length === 0 ? (
        <Card className="rounded-lg">
          <Empty
            description={
              <div>
                <p className="text-gray-500 mb-4">{t('serviceCatalog.noMatchingServices')}</p>
                {canManageCatalog ? <Button type="primary" onClick={handleCreateService}>
                  {t('serviceCatalog.createFirst')}
                </Button> : <p className="text-sm text-gray-500">暂时没有可申请的服务，请联系服务目录管理员。</p>}
              </div>
            }
          />
        </Card>
      ) : (
        <Row gutter={[24, 24]}>
          {filteredCatalogs.map(catalog => (
            <Col key={catalog.id} xs={24} sm={12} md={8} lg={6}>
              <ServiceItemCard catalog={catalog} onDeleted={() => loadServiceCatalogs()} />
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
