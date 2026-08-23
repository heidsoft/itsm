'use client';

/**
 * 服务端搜索的配置项选择器
 *
 * 替代"手输 CI ID"与"一次性拉全量列表"两类交互：
 * - 输入关键字后防抖 300ms 调用后端 search 接口
 * - 选项展示名称 + 类型，支持清空
 * - 受控 value 不在当前候选中时，保留已选项用于回显
 */

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Select, Tag } from 'antd';

import { CMDBApi } from '@/lib/api/cmdb-api';
import { CIStatusLabels } from '@/constants/cmdb';
import type { ConfigurationItem } from '@/types/biz/cmdb';

export interface CISelectOption {
  id: number;
  name: string;
  type?: string;
  status?: string;
}

export interface CISearchSelectProps {
  value?: number;
  onChange?: (value: number | undefined, option?: CISelectOption) => void;
  placeholder?: string;
  style?: React.CSSProperties;
  allowClear?: boolean;
  /** 需要从候选中排除的 CI ID（避免自关联等场景） */
  excludeIds?: number[];
  /** 每次搜索拉取的候选数量 */
  searchSize?: number;
}

const SEARCH_DEBOUNCE_MS = 300;

const statusLabels: Record<string, string> = CIStatusLabels as Record<string, string>;

const CISearchSelect: React.FC<CISearchSelectProps> = ({
  value,
  onChange,
  placeholder = '输入名称搜索配置项',
  style,
  allowClear = true,
  excludeIds,
  searchSize = 20,
}) => {
  const [options, setOptions] = useState<CISelectOption[]>([]);
  const [fetching, setFetching] = useState(false);
  const [selectedOption, setSelectedOption] = useState<CISelectOption | null>(null);
  const searchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const fetchSeqRef = useRef(0);
  const isMountedRef = useRef(true);
  // excludeIds 走 ref，避免父组件每次渲染传新数组导致重复请求
  const excludeIdsRef = useRef(excludeIds);

  useEffect(() => {
    excludeIdsRef.current = excludeIds;
  }, [excludeIds]);

  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
      if (searchTimerRef.current) {
        clearTimeout(searchTimerRef.current);
      }
    };
  }, []);

  const searchCIs = useCallback(
    async (search?: string) => {
      const seq = ++fetchSeqRef.current;
      setFetching(true);
      try {
        const resp = await CMDBApi.getCIs({ search: search || undefined, size: searchSize });
        if (!isMountedRef.current || seq !== fetchSeqRef.current) return;
        const items = resp.items ?? [];
        setOptions(
          items
            .filter(ci => !excludeIdsRef.current?.includes(ci.id))
            .map(ci => ({
              id: ci.id,
              name: ci.name,
              type: ci.type,
              status: ci.status,
            }))
        );
      } catch {
        if (isMountedRef.current && seq === fetchSeqRef.current) {
          setOptions([]);
        }
      } finally {
        if (isMountedRef.current && seq === fetchSeqRef.current) {
          setFetching(false);
        }
      }
    },
    [searchSize]
  );

  // 挂载时预加载第一页，保证未输入关键字也有候选
  useEffect(() => {
    searchCIs();
  }, [searchCIs]);

  const handleSearch = (search: string) => {
    if (searchTimerRef.current) {
      clearTimeout(searchTimerRef.current);
    }
    searchTimerRef.current = setTimeout(() => {
      searchCIs(search);
    }, SEARCH_DEBOUNCE_MS);
  };

  const handleChange = (nextValue: number | undefined) => {
    const option = nextValue
      ? options.find(item => item.id === nextValue) ?? null
      : null;
    setSelectedOption(option);
    onChange?.(nextValue, option ?? undefined);
  };

  // 受控值不在当前候选里时，把已选项补进候选，保证回显不退化成裸 ID
  const mergedOptions = useMemo(() => {
    if (value && selectedOption && !options.some(item => item.id === value)) {
      return [selectedOption, ...options];
    }
    return options;
  }, [options, selectedOption, value]);

  return (
    <Select<number>
      showSearch
      allowClear={allowClear}
      value={value}
      placeholder={placeholder}
      style={style}
      filterOption={false}
      onSearch={handleSearch}
      onChange={handleChange}
      loading={fetching}
      notFoundContent={fetching ? '搜索中...' : '无匹配配置项'}
      options={mergedOptions.map(item => ({
        value: item.id,
        label: (
          <span>
            <span>{item.name}</span>
            {item.type && <Tag style={{ marginLeft: 8 }}>{item.type}</Tag>}
            {item.status && statusLabels[item.status] && (
              <Tag style={{ marginLeft: 0 }}>{statusLabels[item.status]}</Tag>
            )}
          </span>
        ),
      }))}
    />
  );
};

export default CISearchSelect;
