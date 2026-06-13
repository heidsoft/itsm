'use client';

import { useState, useEffect, useCallback } from 'react';
import { Modal, Input, List, Tag, Spin, Empty } from 'antd';
import {
  SearchOutlined,
  FileTextOutlined,
  AlertOutlined,
  QuestionCircleOutlined,
  SwapOutlined,
  BookOutlined,
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import { globalSearch, SearchResult } from '@/lib/api/global-search-api';
import styles from './GlobalSearch.module.css';

const iconMap = {
  ticket: <FileTextOutlined />,
  incident: <AlertOutlined />,
  problem: <QuestionCircleOutlined />,
  change: <SwapOutlined />,
  knowledge: <BookOutlined />,
};

const colorMap = {
  ticket: 'blue',
  incident: 'red',
  problem: 'orange',
  change: 'purple',
  knowledge: 'green',
};

const pathMap = {
  ticket: '/tickets',
  incident: '/incidents',
  problem: '/problems',
  change: '/changes',
  knowledge: '/knowledge',
};

interface GlobalSearchProps {
  /** Controls modal visibility */
  open: boolean;
  /** Callback to close the search modal */
  onClose: () => void;
}

export default function GlobalSearch({ open, onClose }: GlobalSearchProps) {
  const [loading, setLoading] = useState(false);
  const [results, setResults] = useState<SearchResult[]>([]);
  const [keyword, setKeyword] = useState('');
  const router = useRouter();

  const search = useCallback(async (kw: string) => {
    if (!kw.trim()) {
      setResults([]);
      return;
    }

    setLoading(true);
    try {
      const data = await globalSearch(kw);
      setResults(data.results || []);
    } catch (error) {
      console.error('Search failed:', error);
      setResults([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timer = setTimeout(() => {
      search(keyword);
    }, 300);
    return () => clearTimeout(timer);
  }, [keyword, search]);

  const handleSelect = (result: SearchResult) => {
    const basePath = pathMap[result.type];
    router.push(`${basePath}/${result.id}`);
    setKeyword('');
    onClose();
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      onClose();
    }
  };

  return (
    <Modal
      open={open}
      onCancel={onClose}
      footer={null}
      closable={false}
      width={600}
      centered
      className={styles.modal}
      destroyOnClose
    >
      <div className={styles.container} onKeyDown={handleKeyDown}>
        <Input
          prefix={<SearchOutlined className={styles.searchIcon} />}
          placeholder="搜索工单、事件、问题、变更、知识库... (按 Esc 关闭)"
          value={keyword}
          onChange={e => setKeyword(e.target.value)}
          className={styles.input}
          autoFocus
          suffix={loading ? <Spin size="small" /> : null}
        />

        <div className={styles.shortcut}>
          <span>
            <kbd>Ctrl</kbd>+<kbd>K</kbd> 打开
          </span>
          <span>
            <kbd>↑</kbd>
            <kbd>↓</kbd> 导航
          </span>
          <span>
            <kbd>Enter</kbd> 选择
          </span>
          <span>
            <kbd>Esc</kbd> 关闭
          </span>
        </div>

        {keyword && !loading && results.length === 0 && (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description="未找到相关结果"
            className={styles.empty}
          />
        )}

        <List
          dataSource={results}
          className={styles.list}
          renderItem={item => (
            <List.Item className={styles.item} onClick={() => handleSelect(item)}>
              <List.Item.Meta
                avatar={
                  <div className={styles.icon} style={{ color: colorMap[item.type] }}>
                    {iconMap[item.type]}
                  </div>
                }
                title={
                  <span>
                    {item.ticketNumber && (
                      <span className={styles.ticketNumber}>{item.ticketNumber}</span>
                    )}
                    {item.title}
                  </span>
                }
                description={
                  item.description?.slice(0, 60) +
                  (item.description && item.description.length > 60 ? '...' : '')
                }
              />
              <Tag color={colorMap[item.type]}>{item.type}</Tag>
            </List.Item>
          )}
        />
      </div>
    </Modal>
  );
}
