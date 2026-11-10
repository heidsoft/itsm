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

import { serviceCategoryLabel } from '@/constants/service-catalog';
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
      setError(err instanceof Error ? err.message : 'Service directory loading failed, please try again later.');
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
      [catalog.name, serviceCategoryLabel(catalog.category), String(catalog.category), catalog.shortDescription]
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
          title="No permission to create service request"
          description="Please contact your administrator to confirm whether your role includes service_request:write."
          action={
            <Button onClick={() => router.push('/service-requests')}>Return service request</Button>
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

            Return service request
          </Button>
          <Title level={2} style={{ marginBottom: 4 }}>

            New service request
          </Title>
          <Text type="secondary">First select a published service and then fill in the application information.</Text>
        </div>
      </div>

      <Card className="mb-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <Input
            allowClear
            prefix={<Search size={16} className="text-gray-400" />}
            placeholder="Search service name or category"
            value={keyword}
            onChange={e => setKeyword(e.target.value)}
            style={{ maxWidth: 320 }}
          />
          <Text type="secondary">

            Total {filteredCatalogs.length}  Available services
          </Text>
        </div>
      </Card>

      {error && (
        <Alert
          className="mb-4"
          type="error"
          showIcon
          title="Failed to load service catalog"
          description={error}
          action={<Button onClick={loadCatalogs}>Reload</Button>}
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
              error ? 'Service catalog is temporarily unavailable' : 'No services available for application, please contact the service catalog administrator'
            }
          >
            <Button onClick={() => router.push('/service-catalog')}>View service catalog</Button>
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
                  <Tag className="!m-0">{serviceCategoryLabel(catalog.category)}</Tag>
                </div>
                <Text className="mt-2 block min-h-[40px] text-gray-600">
                  {catalog.shortDescription || catalog.fullDescription || 'No service description yet'}
                </Text>
                <div className="mt-3 flex items-center text-gray-500">
                  <Clock size={14} className="mr-1" />
                  <Text type="secondary" className="!text-xs">

                    Delivery time:{catalog.deliveryDays != null ? `${catalog.deliveryDays} days` : 'Pending'}
                  </Text>
                </div>
                <Button
                  className="mt-4"
                  type="primary"
                  block
                  icon={<ArrowRight size={16} />}
                  aria-label={`Apply ${catalog.name}`}
                  onClick={() => router.push(`/service-catalog/request/${catalog.id}`)}
                >

                  Apply
                </Button>
              </Card>
            </Col>
          ))}
        </Row>
      )}
    </div>
  );
}
