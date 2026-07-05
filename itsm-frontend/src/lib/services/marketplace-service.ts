import { httpClient } from '@/lib/api/http-client';

export type MarketplaceItemType = 'connector' | 'skill' | 'plugin';
export type InstallationStatus = 'installing' | 'active' | 'disabled' | 'failed' | 'uninstalled';

export interface MarketplaceItem {
  id: number;
  name: string;
  title: string;
  type: MarketplaceItemType;
  provider: string;
  description?: string;
  longDescription?: string;
  iconUrl?: string;
  screenshots?: string[];
  rating?: number;
  installCount?: number;
  category?: string;
  tags?: string[];
  isOfficial?: boolean;
  isFree?: boolean;
  price?: number;
  latestVersion?: string;
  minSystemVersion?: string;
  authorId?: string;
  authorName?: string;
  homepage?: string;
  repository?: string;
  license?: string;
  capabilities?: string[];
  requiredPermissions?: string[];
  configSchema?: Record<string, unknown>;
  createdAt?: string;
  updatedAt?: string;
  edges?: {
    versions?: ItemVersion[];
  };
}

export interface ItemVersion {
  version: string;
  changelog?: string;
  releasedAt?: string;
}

export interface TenantInstallation {
  id: number;
  tenantId: number;
  itemId: number;
  installedVersion: string;
  status: InstallationStatus;
  config?: Record<string, unknown>;
  autoUpgrade?: boolean;
  installedBy?: string;
  installedAt: string;
  lastUpdatedAt?: string;
  lastUsedAt?: string;
  errorMessage?: string;
  createdAt?: string;
  updatedAt?: string;
  edges?: {
    item?: MarketplaceItem;
  };
}

class MarketplaceService {
  private readonly baseUrl = '/api/v1/marketplace';

  async listItems(params: {
    type?: MarketplaceItemType;
    category?: string;
    search?: string;
    isOfficial?: boolean;
    page?: number;
    pageSize?: number;
  } = {}): Promise<{ items: MarketplaceItem[]; total: number; page: number; pageSize: number }> {
    return httpClient.get(`${this.baseUrl}/items`, {
      type: params.type,
      category: params.category,
      search: params.search,
      isOfficial: params.isOfficial,
      page: params.page ?? 1,
      pageSize: params.pageSize ?? 100,
    });
  }

  async getItem(id: number): Promise<MarketplaceItem> {
    return httpClient.get(`${this.baseUrl}/items/${id}`);
  }

  async installItem(id: number): Promise<TenantInstallation> {
    return httpClient.post(`${this.baseUrl}/items/${id}/install`);
  }

  async uninstallItem(id: number): Promise<void> {
    await httpClient.post(`${this.baseUrl}/items/${id}/uninstall`);
  }

  async listInstallations(status?: string): Promise<TenantInstallation[]> {
    return httpClient.get(`${this.baseUrl}/installations`, status ? { status } : undefined);
  }

  async getInstallation(itemId: number): Promise<TenantInstallation | null> {
    try {
      return await httpClient.get(`${this.baseUrl}/installations/${itemId}`);
    } catch {
      return null;
    }
  }

  async updateInstallationConfig(itemId: number, config: Record<string, unknown>): Promise<TenantInstallation> {
    return httpClient.put(`${this.baseUrl}/installations/${itemId}/config`, config);
  }
}

export const marketplaceService = new MarketplaceService();
export default marketplaceService;
