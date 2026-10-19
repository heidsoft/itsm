'use client';

import { Button, Card, Col, Input, Row, Space } from 'antd';
import { Download, Filter, Plus, RotateCcw, Trash2 } from 'lucide-react';

const { Search } = Input;

interface TicketListToolbarProps {
  readonly searchValue: string;
  readonly showFilters: boolean;
  readonly selectedCount: number;
  readonly loading: boolean;
  /** 与路由 RequirePermission 对齐：无 ticket:delete / ticket:export / ticket:create 时不提供入口。 */
  readonly canBatchDelete: boolean;
  readonly canExport: boolean;
  readonly canCreate: boolean;
  readonly onSearchChange: (value: string) => void;
  readonly onSearchSubmit: (value: string) => void;
  readonly onToggleFilters: () => void;
  readonly onRefresh: () => void;
  readonly onBatchDelete: () => void;
  readonly onExport: () => void;
  readonly onCreate: () => void;
}

/**
 * Top action bar for the ticket list: search, filter toggle, refresh, batch
 * delete (conditional), export, create. Controlled — owns no state.
 */
export function TicketListToolbar({
  searchValue,
  showFilters,
  selectedCount,
  loading,
  canBatchDelete,
  canExport,
  canCreate,
  onSearchChange,
  onSearchSubmit,
  onToggleFilters,
  onRefresh,
  onBatchDelete,
  onExport,
  onCreate,
}: TicketListToolbarProps) {
  return (
    <Card className='rounded-lg shadow-sm'>
      <Row gutter={[16, 16]} align='middle'>
        <Col flex='auto'>
          <Space size='middle'>
            <Search
              placeholder='搜索工单标题、描述或工单号'
              value={searchValue}
              onChange={e => onSearchChange(e.target.value)}
              onSearch={onSearchSubmit}
              style={{ width: 300 }}
              allowClear
            />
            <Button
              icon={<Filter />}
              onClick={onToggleFilters}
              type={showFilters ? 'primary' : 'default'}
            >
              过滤器
            </Button>
            <Button icon={<RotateCcw />} onClick={onRefresh} loading={loading}>
              刷新
            </Button>
          </Space>
        </Col>
        <Col>
          <Space>
            {canBatchDelete && selectedCount > 0 && (
              <Button danger icon={<Trash2 />} onClick={onBatchDelete}>
                批量删除 ({selectedCount})
              </Button>
            )}
            {canExport && (
              <Button icon={<Download />} onClick={onExport}>
                导出
              </Button>
            )}
            {canCreate && (
              <Button type='primary' icon={<Plus />} onClick={onCreate}>
                创建工单
              </Button>
            )}
          </Space>
        </Col>
      </Row>
    </Card>
  );
}
