'use client';

/**
 * 问题关联管理 Tab
 * 展示和管理问题与工单/事件/变更的关联关系
 */

import React, { useState, useEffect, useCallback } from 'react';
import { Table, Button, Modal, Space, Tag, message, Input, Empty, Card } from 'antd';
import { Search, Plus, Trash2, FileText, Link, AlertTriangle, ArrowLeftRight } from 'lucide-react';
import {
  ProblemApi,
  type AssociatedItem,
  type ProblemAssociationRequest,
  type ProblemRemoveAssociationRequest,
} from '@/lib/api/problem-api';
import { TicketApi } from '@/lib/api/ticket-api';
import { IncidentAPI } from '@/lib/api/incident-api';
import { ChangeApi } from '@/lib/api/change-api';

interface ProblemAssociationsTabProps {
  problemId: number;
}

type RelatedType = 'ticket' | 'incident' | 'change';

const TYPE_CONFIG: Record<RelatedType, { label: string; icon: React.ReactNode; color: string }> = {
  ticket: { label: '工单', icon: <FileText />, color: 'blue' },
  incident: { label: '事件', icon: <AlertTriangle />, color: 'orange' },
  change: { label: '变更', icon: <ArrowLeftRight />, color: 'purple' },
};

const STATUS_COLOR_MAP: Record<string, string> = {
  new: 'default',
  open: 'processing',
  inProgress: 'processing',
  investigating: 'processing',
  draft: 'default',
  pending: 'warning',
  approved: 'success',
  resolved: 'success',
  closed: 'default',
  completed: 'success',
  cancelled: 'error',
};

const ProblemAssociationsTab: React.FC<ProblemAssociationsTabProps> = ({ problemId }) => {
  const [tickets, setTickets] = useState<AssociatedItem[]>([]);
  const [incidents, setIncidents] = useState<AssociatedItem[]>([]);
  const [changes, setChanges] = useState<AssociatedItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [addModalVisible, setAddModalVisible] = useState(false);
  const [addType, setAddType] = useState<RelatedType>('ticket');
  const [searchResults, setSearchResults] = useState<AssociatedItem[]>([]);
  const [searchLoading, setSearchLoading] = useState(false);
  const [selectedIds, setSelectedIds] = useState<number[]>([]);

  const loadData = useCallback(async () => {
    if (!problemId) return;
    setLoading(true);
    try {
      const data = await ProblemApi.getAssociations(problemId);
      setTickets(data.tickets || []);
      setIncidents(data.incidents || []);
      setChanges(data.changes || []);
    } catch {
      message.error('加载关联数据失败');
    } finally {
      setLoading(false);
    }
  }, [problemId]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleRemove = async (type: RelatedType, id: number) => {
    try {
      const req: ProblemRemoveAssociationRequest = { relatedType: type, relatedId: id };
      await ProblemApi.removeAssociation(problemId, req);
      message.success('已移除关联');
      loadData();
    } catch {
      message.error('移除关联失败');
    }
  };

  const handleAdd = async () => {
    if (selectedIds.length === 0) {
      message.warning('请选择要关联的项');
      return;
    }
    try {
      const req: ProblemAssociationRequest = { relatedType: addType, relatedIds: selectedIds };
      await ProblemApi.addAssociation(problemId, req);
      message.success('关联添加成功');
      setAddModalVisible(false);
      setSelectedIds([]);
      setSearchResults([]);
      loadData();
    } catch {
      message.error('添加关联失败');
    }
  };

  const openAddModal = (type: RelatedType) => {
    setAddType(type);
    setSelectedIds([]);
    setSearchResults([]);
    setAddModalVisible(true);
  };

  const handleSearch = async (value: string) => {
    if (!value.trim()) {
      setSearchResults([]);
      return;
    }
    setSearchLoading(true);
    try {
      let results: AssociatedItem[] = [];
      switch (addType) {
        case 'ticket': {
          const data = await TicketApi.getTickets({ keyword: value, page: 1, pageSize: 20 });
          results = (data.tickets || []).map((t: any) => ({
            id: t.id,
            title: t.title,
            status: t.status,
            number: t.ticketNumber,
            type: 'ticket' as const,
          }));
          break;
        }
        case 'incident': {
          const data = await IncidentAPI.listIncidents({ keyword: value, page: 1, pageSize: 20 });
          results = (data.incidents || data.data || []).map((i: any) => ({
            id: i.id,
            title: i.title,
            status: i.status,
            number: i.incidentNumber,
            type: 'incident' as const,
          }));
          break;
        }
        case 'change': {
          const data = await ChangeApi.getChanges({ search: value, page: 1, pageSize: 20 });
          results = (data.changes || []).map((c: any) => ({
            id: c.id,
            title: c.title,
            status: c.status,
            number: c.changeNumber,
            type: 'change' as const,
          }));
          break;
        }
      }
      setSearchResults(results);
    } catch {
      message.error('搜索失败');
    } finally {
      setSearchLoading(false);
    }
  };

  const renderItemTable = (type: RelatedType, items: AssociatedItem[]) => {
    const config = TYPE_CONFIG[type];
    const columns = [
      {
        title: '编号',
        dataIndex: 'number',
        key: 'number',
        width: 140,
        render: (text: string) => text || '-',
      },
      {
        title: '标题',
        dataIndex: 'title',
        key: 'title',
        ellipsis: true,
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 100,
        render: (status: string) => (
          <Tag color={STATUS_COLOR_MAP[status] || 'default'}>{status}</Tag>
        ),
      },
      {
        title: '操作',
        key: 'action',
        width: 80,
        render: (_: unknown, record: AssociatedItem) => (
          <Button
            type="link"
            danger
            size="small"
            icon={<Trash2 />}
            onClick={() => handleRemove(type, record.id)}
          >
            移除
          </Button>
        ),
      },
    ];

    return (
      <Card
        title={
          <Space>
            {config.icon}
            <span>{config.label}</span>
            <Tag color={config.color}>{items.length}</Tag>
          </Space>
        }
        size="small"
        style={{ marginBottom: 16 }}
        extra={
          <Button
            type="dashed"
            size="small"
            icon={<Plus />}
            onClick={() => openAddModal(type)}
          >
            添加{config.label}
          </Button>
        }
      >
        {items.length === 0 ? (
          <Empty description={`暂无关联${config.label}`} image={Empty.PRESENTED_IMAGE_SIMPLE} />
        ) : (
          <Table
            columns={columns}
            dataSource={items}
            rowKey="id"
            size="small"
            pagination={false}
          />
        )}
      </Card>
    );
  };

  const existingIds = new Set([
    ...tickets.map((t) => t.id),
    ...incidents.map((i) => i.id),
    ...changes.map((c) => c.id),
  ]);

  const filteredSearchResults = searchResults.filter((r) => !existingIds.has(r.id));

  return (
    <div>
      {renderItemTable('ticket', tickets)}
      {renderItemTable('incident', incidents)}
      {renderItemTable('change', changes)}

      <Modal
        title={
          <Space>
            <Link />
            <span>添加{TYPE_CONFIG[addType].label}关联</span>
          </Space>
        }
        open={addModalVisible}
        onOk={handleAdd}
        onCancel={() => {
          setAddModalVisible(false);
          setSelectedIds([]);
          setSearchResults([]);
        }}
        okText="确认关联"
        cancelText="取消"
        width={600}
      >
        <Input.Search
          placeholder={`搜索${TYPE_CONFIG[addType].label}标题或编号`}
          enterButton={<Search />}
          onSearch={handleSearch}
          loading={searchLoading}
          style={{ marginBottom: 16 }}
        />
        <Table
          columns={[
            {
              title: '编号',
              dataIndex: 'number',
              width: 120,
              render: (t: string) => t || '-',
            },
            { title: '标题', dataIndex: 'title', ellipsis: true },
            {
              title: '状态',
              dataIndex: 'status',
              width: 80,
              render: (s: string) => <Tag color={STATUS_COLOR_MAP[s] || 'default'}>{s}</Tag>,
            },
          ]}
          dataSource={filteredSearchResults}
          rowKey="id"
          size="small"
          pagination={false}
          rowSelection={{
            selectedRowKeys: selectedIds,
            onChange: (keys) => setSelectedIds(keys as number[]),
          }}
        />
      </Modal>
    </div>
  );
};

export default ProblemAssociationsTab;
