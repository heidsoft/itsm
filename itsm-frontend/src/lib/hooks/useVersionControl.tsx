/**
 * 乐观锁版本控制 Hook
 * 用于检测并发编辑冲突，防止数据覆盖
 */

import { useState, useCallback, useRef, useEffect } from 'react';
import { Modal, message } from 'antd';
import { ExclamationCircleOutlined, SyncOutlined } from '@ant-design/icons';

/**
 * 版本冲突错误
 */
export class VersionConflictError extends Error {
  constructor(
    public currentVersion: number,
    public serverVersion: number,
    public serverData?: unknown
  ) {
    super(`版本冲突：当前版本 ${currentVersion}，服务器版本 ${serverVersion}`);
    this.name = 'VersionConflictError';
  }
}

/**
 * 版本控制配置
 */
interface VersionControlConfig<T> {
  /** 初始数据 */
  initialData: T | null;
  /** 获取服务器最新数据的函数 */
  fetchLatest?: () => Promise<T>;
  /** 更新数据的函数（返回新版本） */
  updateData: (data: Partial<T> & { version: number }) => Promise<T>;
  /** 数据名称（用于提示信息） */
  dataName?: string;
  /** 是否自动检测冲突 */
  autoDetectConflict?: boolean;
  /** 冲突检测间隔（毫秒），0 表示不自动检测 */
  conflictCheckInterval?: number;
}

/**
 * 版本控制返回值
 */
interface VersionControlReturn<T> {
  /** 当前数据 */
  data: T | null;
  /** 当前版本 */
  version: number;
  /** 是否正在更新 */
  isUpdating: boolean;
  /** 是否检测到冲突 */
  hasConflict: boolean;
  /** 冲突详情 */
  conflictInfo: {
    localVersion: number;
    serverVersion: number;
    serverData?: T;
  } | null;
  /** 更新数据（带版本检查） */
  update: (updates: Partial<T>) => Promise<T>;
  /** 刷新数据 */
  refresh: () => Promise<T>;
  /** 强制覆盖（忽略版本冲突） */
  forceOverwrite: (updates: Partial<T>) => Promise<T>;
  /** 放弃本地更改，使用服务器版本 */
  discardLocalChanges: () => Promise<void>;
  /** 重置冲突状态 */
  clearConflict: () => void;
  /** 初始数据加载状态 */
  isLoading: boolean;
}

/**
 * 检测 API 响应是否为版本冲突
 */
export function isVersionConflictError(error: unknown): boolean {
  if (error instanceof VersionConflictError) {
    return true;
  }
  // 检查 HTTP 409 Conflict 响应
  if (error && typeof error === 'object') {
    const err = error as { status?: number; code?: number; statusCode?: number };
    return err.status === 409 || err.code === 409 || err.statusCode === 409;
  }
  return false;
}

/**
 * 解析版本冲突错误
 */
export function parseVersionConflictError(error: unknown): VersionConflictError | null {
  if (error instanceof VersionConflictError) {
    return error;
  }

  if (error && typeof error === 'object') {
    const err = error as {
      status?: number;
      code?: number;
      statusCode?: number;
      data?: { version?: number; current_version?: number; server_data?: unknown };
    };

    if (err.status === 409 || err.code === 409 || err.statusCode === 409) {
      const serverVersion = err.data?.version ?? err.data?.current_version ?? 0;
      return new VersionConflictError(0, serverVersion, err.data?.server_data);
    }
  }

  return null;
}

/**
 * 乐观锁版本控制 Hook
 *
 * @example
 * ```tsx
 * const {
 *   data: ticket,
 *   version,
 *   update,
 *   hasConflict,
 *   conflictInfo,
 * } = useVersionControl({
 *   initialData: ticketData,
 *   updateData: async (updates) => {
 *     return TicketApi.updateTicket(ticketId, updates);
 *   },
 *   dataName: '工单',
 * });
 *
 * // 更新数据
 * await update({ title: 'New Title' });
 *
 * // 处理冲突
 * if (hasConflict) {
 *   // 显示冲突解决 UI
 * }
 * ```
 */
export function useVersionControl<T extends { version?: number; id?: number }>(
  config: VersionControlConfig<T>
): VersionControlReturn<T> {
  const {
    initialData,
    fetchLatest,
    updateData,
    dataName = '数据',
    autoDetectConflict = true,
    conflictCheckInterval = 0,
  } = config;

  const [data, setData] = useState<T | null>(initialData);
  const [version, setVersion] = useState<number>(initialData?.version ?? 0);
  const [isUpdating, setIsUpdating] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [hasConflict, setHasConflict] = useState(false);
  const [conflictInfo, setConflictInfo] = useState<VersionControlReturn<T>['conflictInfo']>(null);

  const lastCheckRef = useRef<number>(Date.now());
  const intervalRef = useRef<NodeJS.Timeout | null>(null);

  // 同步版本和数据
  useEffect(() => {
    if (initialData) {
      setData(initialData);
      setVersion(initialData.version ?? 0);
    }
  }, [initialData]);

  // 显示冲突对话框
  const showConflictDialog = useCallback(
    async (serverData?: T, serverVersion?: number) => {
      return new Promise<boolean>(resolve => {
        Modal.confirm({
          title: (
            <span>
              <ExclamationCircleOutlined style={{ color: '#faad14', marginRight: 8 }} />
              检测到并发修改
            </span>
          ),
          content: (
            <div>
              <p>
                该{dataName}已被其他用户修改（服务器版本: {serverVersion ?? '?'}， 您的版本:{' '}
                {version}）。
              </p>
              <p>请选择操作：</p>
              <ul>
                <li>点击&quot;刷新&quot;放弃您的更改，加载最新版本</li>
                <li>点击&quot;覆盖&quot;使用您的更改覆盖服务器版本（可能导致数据丢失）</li>
              </ul>
            </div>
          ),
          okText: '刷新',
          cancelText: '覆盖',
          onOk: () => resolve(false),
          onCancel: () => resolve(true),
        });
      });
    },
    [dataName, version]
  );

  // 更新数据（带版本检查）
  const update = useCallback(
    async (updates: Partial<T>): Promise<T> => {
      if (!data) {
        throw new Error('No data to update');
      }

      setIsUpdating(true);
      setHasConflict(false);
      setConflictInfo(null);

      try {
        // 发送更新请求，包含当前版本
        const updatePayload = {
          ...updates,
          version, // 发送当前版本用于服务器验证
        };

        const result = await updateData(updatePayload);

        // 更新成功，更新本地状态
        const newVersion = (result as { version?: number }).version ?? version + 1;
        setData(result);
        setVersion(newVersion);

        return result;
      } catch (error) {
        // 检查是否为版本冲突
        if (isVersionConflictError(error)) {
          const conflictErr = parseVersionConflictError(error);
          if (conflictErr) {
            setHasConflict(true);
            setConflictInfo({
              localVersion: version,
              serverVersion: conflictErr.serverVersion,
              serverData: conflictErr.serverData as T,
            });

            // 显示冲突对话框
            const shouldOverwrite = await showConflictDialog(
              conflictErr.serverData as T,
              conflictErr.serverVersion
            );

            if (shouldOverwrite) {
              // 用户选择覆盖
              return forceOverwrite(updates);
            } else {
              // 用户选择刷新
              await discardLocalChanges();
              throw new VersionConflictError(
                version,
                conflictErr.serverVersion,
                conflictErr.serverData
              );
            }
          }
        }

        throw error;
      } finally {
        setIsUpdating(false);
      }
    },
    [data, version, updateData, showConflictDialog]
  );

  // 强制覆盖（忽略版本冲突）
  const forceOverwrite = useCallback(
    async (updates: Partial<T>): Promise<T> => {
      if (!data) {
        throw new Error('No data to update');
      }

      setIsUpdating(true);

      try {
        // 强制更新，不发送版本号或发送特殊标记
        const updatePayload = {
          ...updates,
          version: -1, // -1 表示强制覆盖
          force: true,
        };

        const result = await updateData(updatePayload);

        const newVersion = (result as { version?: number }).version ?? version + 1;
        setData(result);
        setVersion(newVersion);
        setHasConflict(false);
        setConflictInfo(null);

        message.success('已强制覆盖，请注意检查数据完整性');
        return result;
      } catch (error) {
        message.error('强制更新失败');
        throw error;
      } finally {
        setIsUpdating(false);
      }
    },
    [data, version, updateData]
  );

  // 刷新数据
  const refresh = useCallback(async (): Promise<T> => {
    if (!fetchLatest) {
      throw new Error('fetchLatest not provided');
    }

    setIsLoading(true);
    try {
      const result = await fetchLatest();
      const newVersion = (result as { version?: number }).version ?? 0;
      setData(result);
      setVersion(newVersion);
      setHasConflict(false);
      setConflictInfo(null);
      lastCheckRef.current = Date.now();
      return result;
    } catch (error) {
      message.error(`刷新${dataName}失败`);
      throw error;
    } finally {
      setIsLoading(false);
    }
  }, [fetchLatest, dataName]);

  // 放弃本地更改，使用服务器版本
  const discardLocalChanges = useCallback(async (): Promise<void> => {
    try {
      await refresh();
      message.info('已加载最新版本');
    } catch (error) {
      // Error already handled in refresh
    }
  }, [refresh]);

  // 清除冲突状态
  const clearConflict = useCallback(() => {
    setHasConflict(false);
    setConflictInfo(null);
  }, []);

  // 自动冲突检测
  useEffect(() => {
    if (!autoDetectConflict || conflictCheckInterval <= 0 || !fetchLatest) {
      return;
    }

    const checkConflict = async () => {
      // 如果正在更新，跳过检测
      if (isUpdating) return;

      try {
        const serverData = await fetchLatest();
        const serverVersion = (serverData as { version?: number }).version ?? 0;

        if (serverVersion > version) {
          // 检测到版本差异
          setHasConflict(true);
          setConflictInfo({
            localVersion: version,
            serverVersion,
            serverData,
          });
        }

        lastCheckRef.current = Date.now();
      } catch (error) {
        console.error('Conflict check failed:', error);
      }
    };

    intervalRef.current = setInterval(checkConflict, conflictCheckInterval);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [autoDetectConflict, conflictCheckInterval, fetchLatest, version, isUpdating]);

  return {
    data,
    version,
    isUpdating,
    hasConflict,
    conflictInfo,
    update,
    refresh,
    forceOverwrite,
    discardLocalChanges,
    clearConflict,
    isLoading,
  };
}

/**
 * 简化版的版本检查 Hook
 * 仅提供版本追踪和冲突检测，不管理数据
 */
export function useVersionTracker(initialVersion: number = 0) {
  const [version, setVersion] = useState(initialVersion);
  const [baseVersion, setBaseVersion] = useState(initialVersion);
  const [hasChanges, setHasChanges] = useState(false);

  // 更新版本
  const updateVersion = useCallback(
    (newVersion: number) => {
      setVersion(newVersion);
      setHasChanges(newVersion !== baseVersion);
    },
    [baseVersion]
  );

  // 标记为已保存
  const markSaved = useCallback(
    (newVersion?: number) => {
      const v = newVersion ?? version;
      setBaseVersion(v);
      setVersion(v);
      setHasChanges(false);
    },
    [version]
  );

  // 重置
  const reset = useCallback((newVersion: number = 0) => {
    setVersion(newVersion);
    setBaseVersion(newVersion);
    setHasChanges(false);
  }, []);

  // 检查是否过期
  const checkStale = useCallback(
    (serverVersion: number): boolean => {
      return serverVersion > baseVersion;
    },
    [baseVersion]
  );

  return {
    version,
    baseVersion,
    hasChanges,
    updateVersion,
    markSaved,
    reset,
    checkStale,
  };
}

export default useVersionControl;
