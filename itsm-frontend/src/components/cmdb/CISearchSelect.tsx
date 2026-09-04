'use client';

/**
 * 服务端搜索的配置项选择器 (P1-2 React Query)
 * - search 防抖 300ms 后驱动 useCIsQuery，竞态/卸载/缓存由 RQ 自动处理
 * - excludeIds 在 queryFn filterOption 端过滤（保持前端单一过滤点）
 * - 受控 value 不在当前候选时，mergedOptions 保留已选项用于回显
 */

import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Select, Tag } from 'antd';
import { useCIsQuery } from '@/lib/hooks/useCMDB';

import { CIStatusLabels } from '@/constants/cmdb';

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
  // 搜索值（输入框立即反映） + 防抖值（驱动 query）
  const [searchInput, setSearchInput] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  useEffect(() => {
    const t = setTimeout(() => setDebouncedSearch(searchInput), SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(t);
  }, [searchInput]);

  // excludeIds 通过 ref 透传，避免父组件传新数组导致 queryKey 不稳定
  const excludeIdsRef = useRef(excludeIds);
  useEffect(() => {
    excludeIdsRef.current = excludeIds;
  }, [excludeIds]);

  // 挂载时即预加载第一页（保证未输入关键字也有候选）
  const listQuery = useCIsQuery({
    search: debouncedSearch || undefined,
    size: searchSize,
  });

  const options: CISelectOption[] = useMemo(() => {
    const items = (listQuery.data?.items ?? []) as Array<{
      id: number;
      name: string;
      type?: string;
      status?: string;
    }>;
    const excluded = excludeIdsRef.current ?? [];
    return items
      .filter(ci => !excluded.includes(ci.id))
      .map(ci => ({ id: ci.id, name: ci.name, type: ci.type, status: ci.status }));
  }, [listQuery.data]);

  const [selectedOption, setSelectedOption] = useState<CISelectOption | null>(null);

  // 受控 value 不在当前候选里时保留已选项回显
  const mergedOptions = useMemo(() => {
    if (value && selectedOption && !options.some(item => item.id === value)) {
      return [selectedOption, ...options];
    }
    return options;
  }, [options, selectedOption, value]);

  const fetching = listQuery.isFetching;

  const handleChange = (nextValue: number | undefined) => {
    const option = nextValue ? options.find(item => item.id === nextValue) ?? null : null;
    setSelectedOption(option);
    onChange?.(nextValue, option ?? undefined);
  };

  return (
    <Select<number>
      showSearch
      allowClear={allowClear}
      value={value}
      placeholder={placeholder}
      style={style}
      filterOption={false}
      onSearch={setSearchInput}
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
