import { httpClient } from './http-client';

/**
 * 商品目录（平台级）API。
 *
 * 与 `marketplace-api.ts` 的区别：那个文件是**租户**视角（浏览已上架商品、
 * 安装/卸载）。这里是**平台运营**视角，能看见草稿与已下架条目，并能上架/下架、
 * 维护版本。两条路径挂不同权限码（marketplace_catalog:* vs marketplace:*），
 * 因为 marketplace_items 是全局表，租户管理员写入即跨租户污染。
 */

export type CatalogItemType = 'connector' | 'skill' | 'plugin';
export type CatalogItemStatus = 'draft' | 'reviewing' | 'published' | 'rejected' | 'deprecated';
export type CatalogVersionStatus = 'draft' | 'published' | 'deprecated' | 'withdrawn';

export interface CatalogItem {
  id: number;
  name: string;
  type: CatalogItemType;
  title: string;
  provider: string;
  description?: string;
  longDescription?: string;
  iconUrl?: string;
  screenshots?: string[];
  tags?: string[];
  rating: number;
  installCount: number;
  latestVersion: string;
  minSystemVersion?: string;
  status: CatalogItemStatus;
  isOfficial: boolean;
  isFree: boolean;
  price: number;
  category?: string;
  capabilities?: string[];
  requiredPermissions?: string[];
  authorName?: string;
  homepage?: string;
  repository?: string;
  license?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CatalogItemVersion {
  version: string;
  changelog?: string;
  downloadUrl?: string;
  manifestPath?: string;
  minSystemVersion?: string;
  dependencies?: Record<string, string>;
  status: CatalogVersionStatus;
  downloadCount: number;
  releasedAt: string;
  createdAt: string;
  updatedAt: string;
}

export interface CatalogItemDetail {
  item: CatalogItem;
  versions: CatalogItemVersion[];
}

export interface CatalogListResponse {
  items: CatalogItem[];
  total: number;
  page: number;
  pageSize: number;
}

export interface CatalogListParams {
  status?: CatalogItemStatus;
  type?: CatalogItemType;
  search?: string;
  page?: number;
  pageSize?: number;
}

/**
 * 创建/更新入参。
 *
 * 全部可选：更新是 PATCH 语义，未提交的字段保持原值。注意 name/type 在更新时
 * 会被后端拒绝（它们是身份：name 决定连接器 provider）。
 */
export interface CatalogItemInput {
  name?: string;
  type?: CatalogItemType;
  title?: string;
  provider?: string;
  description?: string;
  longDescription?: string;
  iconUrl?: string;
  tags?: string[];
  category?: string;
  capabilities?: string[];
  requiredPermissions?: string[];
  configSchema?: Record<string, unknown>;
  isOfficial?: boolean;
  isFree?: boolean;
  price?: number;
  authorName?: string;
  homepage?: string;
  repository?: string;
  license?: string;
  minSystemVersion?: string;
  /** 仅创建时生效：初始版本号。连接器商品可省略（取运行时 manifest 版本）。 */
  version?: string;
}

export interface ItemVersionInput {
  version: string;
  changelog?: string;
  downloadUrl?: string;
  manifestPath?: string;
  minSystemVersion?: string;
  dependencies?: Record<string, string>;
  publish?: boolean;
}

const BASE = '/api/v1/marketplace/catalog/items';

export const listCatalogItems = (params?: CatalogListParams): Promise<CatalogListResponse> =>
  httpClient.get<CatalogListResponse>(BASE, params);

export const getCatalogItem = (id: number): Promise<CatalogItemDetail> =>
  httpClient.get<CatalogItemDetail>(`${BASE}/${id}`);

export const createCatalogItem = (input: CatalogItemInput): Promise<CatalogItemDetail> =>
  httpClient.post<CatalogItemDetail>(BASE, input);

export const updateCatalogItem = (id: number, input: CatalogItemInput): Promise<CatalogItemDetail> =>
  httpClient.patch<CatalogItemDetail>(`${BASE}/${id}`, input);

export const deleteCatalogItem = (id: number): Promise<void> =>
  httpClient.delete<void>(`${BASE}/${id}`);

export const publishCatalogItem = (id: number): Promise<CatalogItemDetail> =>
  httpClient.post<CatalogItemDetail>(`${BASE}/${id}/publish`);

export const unpublishCatalogItem = (id: number): Promise<CatalogItemDetail> =>
  httpClient.post<CatalogItemDetail>(`${BASE}/${id}/unpublish`);

export const listItemVersions = (id: number): Promise<CatalogItemVersion[]> =>
  httpClient.get<CatalogItemVersion[]>(`${BASE}/${id}/versions`);

export const createItemVersion = (
  id: number,
  input: ItemVersionInput
): Promise<CatalogItemVersion> => httpClient.post<CatalogItemVersion>(`${BASE}/${id}/versions`, input);

/**
 * 发布指定版本（推进 latest_version）。
 * 已发布的版本再发布一次即回滚——后端允许，因为「把线上版本拉回旧版」是运维刚需。
 */
export const publishItemVersion = (id: number, version: string): Promise<CatalogItemDetail> =>
  httpClient.post<CatalogItemDetail>(`${BASE}/${id}/versions/${encodeURIComponent(version)}/publish`);

export const CATALOG_STATUS_META: Record<
  CatalogItemStatus,
  { label: string; color: string; hint: string }
> = {
  draft: { label: '草稿', color: 'default', hint: '尚未上架，租户看不到' },
  reviewing: { label: '待审', color: 'processing', hint: '审核中，尚未上架' },
  published: { label: '已上架', color: 'green', hint: '租户可浏览并安装' },
  rejected: { label: '已驳回', color: 'red', hint: '审核未通过，需修改后重新上架' },
  deprecated: { label: '已下架', color: 'orange', hint: '停止新安装；已安装租户不受影响' },
};
