/**
 * 租户上下文 - 独立的租户状态管理
 * 替代 httpClient 内部状态，解决 auth-store 与 httpClient 的耦合
 *
 * 原则：单一数据源，httpClient 和 auth-store 都只读/订阅此状态
 */

export interface TenantContextState {
  tenantId: number | null;
  tenantCode: string | null;
}

// 租户状态
let state: TenantContextState = {
  tenantId: null,
  tenantCode: null,
};

// 订阅者列表
type Subscriber = (state: TenantContextState) => void;
const subscribers = new Set<Subscriber>();

/**
 * 设置租户 ID
 */
export function setTenantId(tenantId: number | null): void {
  state = { ...state, tenantId };
  notify();
}

/**
 * 设置租户 Code
 */
export function setTenantCode(tenantCode: string | null): void {
  state = { ...state, tenantCode };
  notify();
}

/**
 * 批量设置租户
 */
export function setTenant(tenantId: number | null, tenantCode: string | null): void {
  state = { tenantId, tenantCode };
  notify();
}

/**
 * 获取当前租户 ID
 */
export function getTenantId(): number | null {
  return state.tenantId;
}

/**
 * 获取当前租户 Code
 */
export function getTenantCode(): string | null {
  return state.tenantCode;
}

/**
 * 清空租户状态
 */
export function clearTenant(): void {
  state = { tenantId: null, tenantCode: null };
  notify();
}

/**
 * 订阅状态变化
 * @returns 取消订阅的函数
 */
export function subscribe(subscriber: Subscriber): () => void {
  subscribers.add(subscriber);
  return () => {
    subscribers.delete(subscriber);
  };
}

/**
 * 获取当前完整状态
 */
export function getState(): TenantContextState {
  return state;
}

function notify(): void {
  subscribers.forEach(fn => fn(state));
}
