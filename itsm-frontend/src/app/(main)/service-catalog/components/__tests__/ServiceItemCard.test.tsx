import { render, screen } from '@testing-library/react';
import { App } from 'antd';
import type { ServiceItem } from '@/types/service-catalog';
import { ServiceItemCard } from '../ServiceItemCard';

jest.mock('next/navigation', () => ({ useRouter: () => ({ push: jest.fn() }) }));
jest.mock('@/lib/hooks/use-permissions', () => ({
  usePermissions: () => ({ hasPermission: () => false }),
}));

describe('ServiceItemCard display labels', () => {
  it('translates stored category and priority values while preserving customer content', () => {
    const catalog = {
      id: '7',
      name: '客户自定义服务',
      category: '云计算',
      priority: '高',
      shortDescription: '客户提供的服务描述',
    } as unknown as ServiceItem & { priority: string };

    render(<App><ServiceItemCard catalog={catalog} /></App>);

    expect(screen.getByText('Cloud Resource Services')).toBeInTheDocument();
    expect(screen.getByText('High')).toBeInTheDocument();
    expect(screen.queryByText('云计算')).not.toBeInTheDocument();
    expect(screen.queryByText('高')).not.toBeInTheDocument();
    expect(screen.getByText('客户自定义服务')).toBeInTheDocument();
    expect(screen.getByText('客户提供的服务描述')).toBeInTheDocument();
    expect(catalog.category).toBe('云计算');
    expect(catalog.priority).toBe('高');
  });
});
