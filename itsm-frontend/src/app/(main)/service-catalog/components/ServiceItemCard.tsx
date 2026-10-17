'use client';

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  Card,
  Tag,
  Button,
  Typography,
  Rate,
  Space,
  Dropdown,
  Popconfirm,
  App,
} from 'antd';
import type { MenuProps } from 'antd';
import {
  HardDrive,
  UserCog,
  ShieldCheck,
  Clock,
  ArrowRight,
  MoreHorizontal,
  Edit,
  Eye,
} from 'lucide-react';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import type { ServiceItem} from '@/types/service-catalog';
import { ServiceCategory } from '@/types/service-catalog';
import { useI18n } from '@/lib/i18n';
import { usePermissions } from '@/lib/hooks/use-permissions';

const { Title, Text } = Typography;

const categoryIcons: Record<string, typeof HardDrive> = {
  云计算: HardDrive,
  支持: UserCog,
  安全: ShieldCheck,
  [ServiceCategory.IT_SERVICE]: HardDrive,
  [ServiceCategory.BUSINESS_SERVICE]: UserCog,
  [ServiceCategory.SUPPORT_SERVICE]: ShieldCheck,
};

interface ServiceItemCardProps {
  catalog: ServiceItem & { priority?: string };
  /** 删除成功后的局部刷新回调；没有它整页 reload 会丢掉筛选和滚动位置。 */
  onDeleted?: (id: string) => void;
}

export const ServiceItemCard: React.FC<ServiceItemCardProps> = ({ catalog, onDeleted }) => {
  const { t } = useI18n();
  const router = useRouter();
  const { message, modal } = App.useApp();
  const { hasPermission } = usePermissions();
  const canManageCatalog = hasPermission('service_catalog', 'write');
  const [deleting, setDeleting] = useState(false);
  const categoryKey = String(catalog.category);
  const IconComponent = categoryIcons[categoryKey] || HardDrive;

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case '高':
        return 'red';
      case '中':
        return 'orange';
      case '低':
        return 'green';
      default:
        return 'default';
    }
  };

  const handleCardClick = (e: React.MouseEvent) => {
    // Don't navigate if clicking on buttons/dropdown
    const target = e.target as HTMLElement;
    if (target.closest('button') || target.closest('.ant-dropdown')) {
      return;
    }
    router.push(`/service-catalog/detail/${catalog.id}`);
  };

  // 删除是后端软归档：目录转为「已停用」，门户不再展示，历史申请记录保留。
  const handleDelete = async () => {
    setDeleting(true);
    try {
      await ServiceCatalogApi.deleteService(String(catalog.id));
      message.success(t('common.deleteSuccess'));
      onDeleted?.(String(catalog.id));
    } catch (error) {
      message.error(error instanceof Error ? error.message : t('common.deleteFailed'));
    } finally {
      setDeleting(false);
    }
  };

  const confirmDelete = () => {
    modal.confirm({
      title: `删除「${catalog.name}」？`,
      content: '删除后该目录转为「已停用」，门户不再展示，已有申请记录保留。',
      okText: t('common.delete'),
      okButtonProps: { danger: true, loading: deleting },
      cancelText: t('common.cancel'),
      onOk: handleDelete,
    });
  };

  const actionItems: MenuProps['items'] = [
    {
      key: 'detail',
      icon: <Eye size={14} />,
      label: t('common.view'),
      onClick: () => router.push(`/service-catalog/detail/${catalog.id}`),
    },
    {
      key: 'edit',
      icon: <Edit size={14} />,
      label: t('common.edit'),
      onClick: () => router.push(`/service-catalog/edit/${catalog.id}`),
    },
    {
      type: 'divider',
    },
    {
      key: 'delete',
      icon: <span style={{ color: '#ff4d4f' }}>×</span>,
      label: <span style={{ color: '#ff4d4f' }}>{t('common.delete')}</span>,
      danger: true,
      onClick: confirmDelete,
    },
  ];

  return (
    <Card
      className="h-full rounded-lg shadow-sm border border-gray-200 cursor-pointer transition-shadow hover:shadow-md"
      onClick={handleCardClick}
      styles={{
        body: {
          padding: 24,
        },
      }}
    >
      <div className="flex items-start mb-4">
        <div className="w-12 h-12 rounded-lg bg-blue-50 flex items-center justify-center mr-4">
          <IconComponent size={24} className="text-blue-500" />
        </div>
        <div className="flex-1">
          <div className="flex justify-between items-start">
            <Title level={4} className="!m-0 !text-base">
              {catalog.name}
            </Title>
            <Tag color={getPriorityColor(catalog.priority || '')} className="!m-0">
              {catalog.priority || '—'}
            </Tag>
          </div>
          <Text type="secondary" className="!text-xs mt-1 block">
            {catalog.category}
          </Text>
        </div>
      </div>

      <Text className="mb-4 block min-h-[40px]">
        {catalog.shortDescription || catalog.fullDescription || ''}
      </Text>

      <div className="flex justify-between items-center mt-4">
        <div className="flex items-center">
          <Clock size={14} className="mr-1 text-gray-500" />
          <Text type="secondary" className="!text-xs">
            交付周期：{catalog.deliveryDays != null ? `${catalog.deliveryDays} 天` : '待确认'}
          </Text>
        </div>
        <div>
          <Rate disabled value={catalog.rating ?? 0} count={5} className="!text-xs" />
        </div>
      </div>

      <div className="mt-4 pt-4 border-t border-gray-100">
        <div className="flex items-center justify-between gap-2">
          <Button
            type="primary"
            block
            icon={<ArrowRight size={16} />}
            onClick={e => {
              e.stopPropagation();
              router.push(`/service-catalog/request/${catalog.id}`);
            }}
          >
            {t('serviceCatalog.applyService')}
          </Button>
          {canManageCatalog && <Dropdown menu={{ items: actionItems }} trigger={['click']} placement="bottomRight">
            <Button
              aria-label="更多操作"
              icon={<MoreHorizontal size={16} />}
              onClick={e => e.stopPropagation()}
              loading={deleting}
            />
          </Dropdown>}
        </div>
      </div>
    </Card>
  );
};
