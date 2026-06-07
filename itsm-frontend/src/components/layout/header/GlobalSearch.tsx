'use client';

import React, { useState, useEffect } from 'react';
import { Input, Modal, List, Typography } from 'antd';
import { Search, Ticket } from 'lucide-react';
import { useRouter } from 'next/navigation';
import {
  globalSearch,
  type GlobalSearchResponse,
  type SearchResult,
} from '@/lib/api/global-search-api';
import { DESIGN } from '@/design-system/tokens';

const { Text } = Typography;

interface GlobalSearchProps {
  open: boolean;
  onClose: () => void;
  initialResults?: GlobalSearchResponse | null;
}

export const GlobalSearch: React.FC<GlobalSearchProps> = ({
  open,
  onClose,
  initialResults = null,
}) => {
  const router = useRouter();
  const [searchValue, setSearchValue] = useState('');
  const [searchResults, setSearchResults] = useState<GlobalSearchResponse | null>(initialResults);
  const [isSearching, setIsSearching] = useState(false);

  useEffect(() => {
    if (!open) {
      setSearchValue('');
    }
    setSearchResults(initialResults);
  }, [initialResults, open]);

  // Ctrl+K 快捷键
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  // 搜索处理
  const handleSearch = async (value: string) => {
    if (value.trim()) {
      setIsSearching(true);
      try {
        const results = await globalSearch(value);
        setSearchResults(results);
      } catch (error) {
        console.error('Search failed:', error);
      } finally {
        setIsSearching(false);
      }
    }
  };

  const handleItemClick = (item: SearchResult) => {
    const pathMap: Record<string, string> = {
      ticket: 'tickets',
      incident: 'incidents',
      problem: 'problems',
      change: 'changes',
      knowledge: 'knowledge',
    };
    const basePath = pathMap[item.type] || item.type;
    router.push(`/${basePath}/${item.id}`);
    onClose();
  };

  return (
    <Modal
      title={
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <Search size={18} style={{ color: DESIGN.colors.accent }} />
          <span>全局搜索</span>
        </div>
      }
      open={open}
      onCancel={onClose}
      footer={null}
      width={560}
      centered
      destroyOnHidden
    >
      <div style={{ padding: '0 20px' }}>
        <Input
          placeholder="搜索工单、事件..."
          prefix={<Search size={16} style={{ color: DESIGN.colors.textMuted }} />}
          value={searchValue}
          onChange={e => setSearchValue(e.target.value)}
          onPressEnter={() => handleSearch(searchValue)}
          style={{
            width: '100%',
            height: 40,
            borderRadius: DESIGN.radius.md,
            marginBottom: 16,
          }}
          autoFocus
        />
      </div>

      {searchResults && searchResults.results.length > 0 ? (
        <div style={{ padding: '0 20px' }}>
          <List
            header={<strong>搜索结果</strong>}
            dataSource={searchResults.results}
            renderItem={(item: SearchResult) => (
              <List.Item>
                <List.Item.Meta
                  avatar={<Ticket size={16} style={{ color: DESIGN.colors.accent }} />}
                  title={<a onClick={() => handleItemClick(item)}>{item.title}</a>}
                  description={item.status}
                />
              </List.Item>
            )}
          />
        </div>
      ) : searchResults ? (
        <div style={{ padding: 40, textAlign: 'center', color: DESIGN.colors.textMuted }}>
          未找到结果
        </div>
      ) : (
        <div style={{ padding: 40, textAlign: 'center', color: DESIGN.colors.textMuted }}>
          输入关键词搜索...
        </div>
      )}
    </Modal>
  );
};

// 独立的搜索框组件（用于 Header 右侧）
interface SearchInputProps {
  value: string;
  onChange: (value: string) => void;
  onSearch: (value: string) => void;
  onFocus?: () => void;
}

export const SearchInput: React.FC<SearchInputProps> = ({ value, onChange, onSearch, onFocus }) => {
  return (
    <Input
      placeholder="搜索..."
      prefix={<Search size={14} style={{ color: DESIGN.colors.textMuted }} />}
      suffix={
        <kbd
          style={{
            fontSize: 11,
            padding: '1px 5px',
            borderRadius: 3,
            background: DESIGN.colors.bgSubtle,
            border: `1px solid ${DESIGN.colors.border}`,
            color: DESIGN.colors.textMuted,
            lineHeight: 1.4,
          }}
        >
          Ctrl K
        </kbd>
      }
      value={value}
      onChange={e => onChange(e.target.value)}
      onPressEnter={() => onSearch(value)}
      onFocus={onFocus}
      style={{
        width: 180,
        height: 32,
        borderRadius: DESIGN.radius.md,
        border: `1px solid ${DESIGN.colors.border}`,
        background: DESIGN.colors.bgSubtle,
        fontSize: 13,
      }}
    />
  );
};
