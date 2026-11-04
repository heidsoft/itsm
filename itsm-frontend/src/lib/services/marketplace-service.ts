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
  /**
   * 平台是否真正交付该商品（连接器/技能）。安装入口只由它决定，
   * 界面不得再写一份 type 白名单——目录放开新类型时前端会静默拦。
   */
  deliverable: boolean;
  /**
   * 已发布版本历史，按发布日期倒序。仅详情接口返回；列表接口缺该字段，
   * 缺失表示「未加载」，不得渲染成「该商品没有历史版本」。
   */
  versions?: ItemVersion[];
  createdAt?: string;
  updatedAt?: string;
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
  // 以下来自安装记录的商品边，由服务端一次查询带出；缺失表示边未加载，
  // 不得为了补齐名称/描述再拉一次目录，形成第二份真相。
  itemName?: string;
  itemTitle?: string;
  itemDescription?: string;
  itemIconUrl?: string;
  itemType?: MarketplaceItemType;
  deliverable: boolean;
  latestVersion?: string;
  /** 目录 latest 与实际安装版本不同且类型可交付。为 false 时不代表「已最新」也可能只是边未加载。 */
  upgradeAvailable: boolean;
}

/** 单个已安装组件的依赖对齐结论；declaration_missing 不能并入健康。 */
export type DependencyStatus = 'ok' | 'unsatisfied' | 'declaration_missing';

export interface MarketplaceDependencyFinding {
  itemId: number;
  itemName: string;
  itemTitle: string;
  itemType: MarketplaceItemType;
  status: DependencyStatus;
  installedVersion: string;
  installationStatus: InstallationStatus;
  dependencies?: Record<string, string>;
  problems?: string[];
}

export interface MarketplaceDependencyReport {
  tenantId: number;
  checkedAt: string;
  total: number;
  satisfied: number;
  unsatisfied: number;
  declarationMissing: number;
  findings: MarketplaceDependencyFinding[];
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
  } = {}): Promise<{
    items: MarketplaceItem[];
    total: number;
    page: number;
    pageSize: number;
    totalPages: number;
  }> {
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

  /**
   * 安装列表走「不分页的列表」契约：后端整表取回本租户的安装记录，
   * data 只有 {items,total}，没有 page/pageSize/totalPages。
   */
  async listInstallations(status?: string): Promise<TenantInstallation[]> {
    const { items } = await httpClient.get<{ items: TenantInstallation[]; total: number }>(
      `${this.baseUrl}/installations`,
      status ? { status } : undefined
    );
    return items;
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

  /**
   * 升级或回滚到某个已发布版本。省略 targetVersion 即升到目录当前 latest；
   * 传空字符串会被后端判为参数错误，因为「升到一个空版本号」不是「升最新」。
   */
  async upgradeItem(itemId: number, targetVersion?: string): Promise<TenantInstallation> {
    return httpClient.post(`${this.baseUrl}/items/${itemId}/upgrade`, targetVersion ? { targetVersion } : undefined);
  }

  /** 自动升级意愿。必须显式传布尔：缺字段会被后端拒绝，避免误发一次就永久关掉。 */
  async setAutoUpgrade(itemId: number, autoUpgrade: boolean): Promise<TenantInstallation> {
    return httpClient.patch(`${this.baseUrl}/items/${itemId}/auto-upgrade`, { autoUpgrade });
  }

  /** 存量安装的依赖对齐诊断（只读，不改数据）。 */
  async dependencyStatus(): Promise<MarketplaceDependencyReport> {
    return httpClient.get(`${this.baseUrl}/dependency-status`);
  }
}

export const marketplaceService = new MarketplaceService();
export default marketplaceService;
