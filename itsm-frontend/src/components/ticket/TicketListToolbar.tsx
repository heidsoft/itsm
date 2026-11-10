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
              placeholder="Search for ticket title, description or ticket number"
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

              filter
            </Button>
            <Button icon={<RotateCcw />} onClick={onRefresh} loading={loading}>

              Refresh
            </Button>
          </Space>
        </Col>
        <Col>
          <Space>
            {canBatchDelete && selectedCount > 0 && (
              <Button danger icon={<Trash2 />} onClick={onBatchDelete}>

                Batch delete ({selectedCount})
              </Button>
            )}
            {canExport && (
              <Button icon={<Download />} onClick={onExport}>

                Export
              </Button>
            )}
            {canCreate && (
              <Button type='primary' icon={<Plus />} onClick={onCreate}>

                Create a work order
              </Button>
            )}
          </Space>
        </Col>
      </Row>
    </Card>
  );
}
