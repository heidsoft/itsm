'use client';

import React, { useState, useEffect, useMemo } from 'react';
import { Select, Avatar, Spin } from 'antd';
import { UserApi } from '@/lib/api/user-api';
import type { User } from '@/lib/api/user-api';
import { User as UserIcon } from 'lucide-react';

const { Option } = Select;

interface UserSelectProps {
  value?: number[];
  onChange?: (userIds: number[]) => void;
  mode?: 'multiple' | 'tags';
  placeholder?: string;
  style?: React.CSSProperties;
  disabled?: boolean;
  allowClear?: boolean;
}

export const UserSelect: React.FC<UserSelectProps> = ({
  value,
  onChange,
  mode = 'multiple',
  placeholder = '选择用户',
  style,
  disabled = false,
  allowClear = true,
}) => {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchValue, setSearchValue] = useState('');

  // 获取用户列表
  const fetchUsers = async (search?: string) => {
    try {
      setLoading(true);
      const response = await UserApi.getUsers({
        page: 1,
        page_size: 100,
        search: search || '',
      });
      setUsers(response.users || []);
    } catch (error) {
      console.error('Failed to fetch users:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchUsers();
  }, []);

  // 搜索用户
  const handleSearch = (value: string) => {
    setSearchValue(value);
    if (value) {
      fetchUsers(value);
    } else {
      fetchUsers();
    }
  };

  // 过滤用户
  const filteredUsers = useMemo(() => {
    if (!searchValue) return users;
    const lowerSearch = searchValue.toLowerCase();
    return users.filter(
      user =>
        user.name?.toLowerCase().includes(lowerSearch) ||
        user.username?.toLowerCase().includes(lowerSearch) ||
        user.email?.toLowerCase().includes(lowerSearch)
    );
  }, [users, searchValue]);

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
      onSearch={handleSearch}
      notFoundContent={loading ? <Spin size="small" /> : '未找到用户'}
      optionLabelProp="label"
    >
      {filteredUsers.map(user => (
        <Option key={user.id} value={user.id} label={user.name || user.username}>
          <div className="flex items-center space-x-2">
            <Avatar size="small" icon={<UserIcon size={14} />}>
              {user.name?.[0] || user.username?.[0]}
            </Avatar>
            <div className="flex flex-col">
              <span className="text-sm font-medium">{user.name || user.username}</span>
              {user.email && (
                <span className="text-xs text-gray-500">{user.email}</span>
              )}
            </div>
          </div>
        </Option>
      ))}
    </Select>
  );
};

