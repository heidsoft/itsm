'use client';

/**
 * 新建服务请求的第一步：选服务。
 *
 * 服务请求必须带 catalogId，不能凭空建单，所以创建入口是「选一个已发布服务」，
 * 选中后再进入已有的申请表单 `/service-catalog/request/[id]`。
 */

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Alert, Button, Card, Col, Empty, Input, Row, Spin, Tag, Typography } from 'antd';
import { ArrowLeft, ArrowRight, Clock, Search } from 'lucide-react';
import { useRouter } from 'next/navigation';

import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import { ServiceStatus, type ServiceItem } from '@/types/service-catalog';
import { usePermissions } from '@/lib/hooks/use-permissions';

const { Title, Text } = Typography;

const FETCH_LIMIT = 500;

export default function NewServiceRequestPage() {
  const router = useRouter();
  const { hasPermission } = usePermissions();
  const canCreate =
    hasPermission('service_request', 'write') || hasPermission('service_request', 'create');

  const [catalogs, setCatalogs] = useState<ServiceItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [keyword, setKeyword] = useState('');

  const loadCatalogs = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await ServiceCatalogApi.getAllServices(
        { status: ServiceStatus.PUBLISHED },
        FETCH_LIMIT
      );
      setCatalogs(result.services);
    } catch (err) {
      setCatalogs([]);
      setError(err instanceof Error ? err.message : '服务目录加载失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!canCreate) {
      setLoading(false);
      return;
    }
    void loadCatalogs();
  }, [canCreate, loadCatalogs]);

  const filteredCatalogs = useMemo(() => {
    const query = keyword.trim().toLowerCase();
    if (!query) return catalogs;
    return catalogs.filter(catalog =>
      [catalog.name, String(catalog.category), catalog.shortDescription]
        .filter(Boolean)
        .some(value => String(value).toLowerCase().includes(query))
    );
  }, [catalogs, keyword]);

  if (!canCreate) {
    return (
      <div className="p-6 min-h-screen bg-gray-50">
        <Alert
          type="warning"
          showIcon
          title="没有创建服务请求的权限"
          description="请联系管理员确认你的角色是否包含 service_request:write。"
          action={
            <Button onClick={() => router.push('/service-requests')}>返回服务请求</Button>
          }
        />
      </div>
    );
  }

  return (
    <div className="p-6 min-h-screen bg-gray-50">
      <div className="mb-6 flex items-start justify-between gap-4">
        <div>
          <Button
            type="text"
            icon={<ArrowLeft size={16} />}
            onClick={() => router.push('/service-requests')}
            className="!px-0 mb-2"
          >
            返回服务请求
          </Button>
          <Title level={2} style={{ marginBottom: 4 }}>
            新建服务请求
          </Title>
          <Text type="secondary">先选择一个已发布的服务，再填写申请信息。</Text>
        </div>
      </div>

      <Card className="mb-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <Input
            allowClear
            prefix={<Search size={16} className="text-gray-400" />}
            placeholder="搜索服务名称或分类"
            value={keyword}
            onChange={e => setKeyword(e.target.value)}
            style={{ maxWidth: 320 }}
          />
          <Text type="secondary">
            共 {filteredCatalogs.length} 个可申请服务
          </Text>
        </div>
      </Card>

      {error && (
        <Alert
          className="mb-4"
          type="error"
          showIcon
          title="服务目录加载失败"
          description={error}
          action={<Button onClick={loadCatalogs}>重新加载</Button>}
        />
      )}

      {loading ? (
        <div className="flex justify-center py-16">
          <Spin size="large" />
        </div>
      ) : filteredCatalogs.length === 0 ? (
        <Card>
          <Empty
            description={
              error ? '服务目录暂时不可用' : '没有可申请的服务，请联系服务目录管理员'
            }
          >
            <Button onClick={() => router.push('/service-catalog')}>查看服务目录</Button>
          </Empty>
        </Card>
      ) : (
        <Row gutter={[16, 16]}>
          {filteredCatalogs.map(catalog => (
            <Col key={catalog.id} xs={24} sm={12} lg={8} xxl={6}>
              <Card className="h-full rounded-lg shadow-sm border border-gray-200">
                <div className="flex items-start justify-between gap-2">
                  <Title level={4} className="!m-0 !text-base">
                    {catalog.name}
                  </Title>
                  <Tag className="!m-0">{catalog.category}</Tag>
                </div>
                <Text className="mt-2 block min-h-[40px] text-gray-600">
                  {catalog.shortDescription || catalog.fullDescription || '暂无服务说明'}
                </Text>
                <div className="mt-3 flex items-center text-gray-500">
                  <Clock size={14} className="mr-1" />
                  <Text type="secondary" className="!text-xs">
                    交付周期：{catalog.deliveryDays != null ? `${catalog.deliveryDays} 天` : '待确认'}
                  </Text>
                </div>
                <Button
                  className="mt-4"
                  type="primary"
                  block
                  icon={<ArrowRight size={16} />}
                  aria-label={`申请 ${catalog.name}`}
                  onClick={() => router.push(`/service-catalog/request/${catalog.id}`)}
                >
                  申请
                </Button>
              </Card>
            </Col>
          ))}
        </Row>
      )}
    </div>
  );
}
