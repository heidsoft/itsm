'use client';

import React, { useState, useEffect, useMemo } from 'react';
import { Select, Avatar, Spin, Typography } from 'antd';
import type { User } from '@/lib/api/user-api';
import { useUserPickerQuery } from '@/lib/hooks/useUserListQuery';
import { truncationHint } from '@/lib/api/pagination';
import { User as UserIcon } from 'lucide-react';

const { Text } = Typography;

interface UserSelectProps {
  value?: number[];
  onChange?: (userIds: number[]) => void;
  mode?: 'multiple' | 'tags';
  placeholder?: string;
  style?: React.CSSProperties;
  disabled?: boolean;
  allowClear?: boolean;
}

/** 搜索防抖间隔：避免每敲一个字符就打一次用户列表接口。 */
const SEARCH_DEBOUNCE_MS = 300;

export const UserSelect: React.FC<UserSelectProps> = ({
  value,
  onChange,
  mode = 'multiple',
  placeholder = '选择用户',
  style,
  disabled = false,
  allowClear = true,
}) => {
  const [searchInput, setSearchInput] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(searchInput.trim()), SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(timer);
  }, [searchInput]);

  // 走 React Query：同一份参数在 5 分钟内共享缓存，页面上多个选人下拉不会
  // 各自重复请求一次用户列表。
  const { users, truncated, total, isFetching, isError } = useUserPickerQuery({
    search: debouncedSearch || undefined,
  });

  const options = useMemo(
    () =>
      users.map(user => ({
        value: user.id,
        label: (
          <div className="flex items-center space-x-2">
            <Avatar size="small" icon={<UserIcon size={14} />}>
              {user.name?.[0] || user.username?.[0]}
            </Avatar>
            <div className="flex flex-col">
              <span className="text-sm font-medium">{user.name || user.username}</span>
              {user.email && <span className="text-xs text-gray-500">{user.email}</span>}
            </div>
          </div>
        ),
      })),
    [users]
  );

  const hint = truncationHint(total, users.length);

  return (
    <Select
      mode={mode}
      value={value}
      onChange={onChange}
      placeholder={placeholder}
      style={style}
      disabled={disabled}
      allowClear={allowClear}
      showSearch
      filterOption={false}
      onSearch={setSearchInput}
      notFoundContent={isFetching ? <Spin size="small" /> : isError ? '用户列表加载失败' : '未找到用户'}
      optionLabelProp="label"
      options={options}
      dropdownRender={menu => (
        <>
          {menu}
          {hint && (
            <div className="border-t border-gray-200 px-3 py-2">
              <Text type="warning" className="text-xs">
                {hint}
              </Text>
            </div>
          )}
        </>
      )}
    />
  );
};
