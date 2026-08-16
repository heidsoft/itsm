'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Tree,
  Table,
  Button,
  Select,
  App,
  Descriptions,
  Tag,
  Space,
  Modal,
  Form,
  Input,
  Row,
  Col,
  Statistic,
  Alert,
  Tooltip,
  Popconfirm,
} from 'antd';
import Link from 'next/link';
import {
  Building2,
  Plus,
  RefreshCw,
  CheckCircle,
  AlertTriangle,
} from 'lucide-react';
import type { ProcessBinding} from '@/lib/api/process-binding-api';
import { ProcessBindingApi } from '@/lib/api/process-binding-api';
import type { Department } from '@/lib/services/department-service';
import { departmentService } from '@/lib/services/department-service';



interface DepartmentTreeNode {
  title: string;
  key: number;
  icon: React.ReactNode;
  children?: DepartmentTreeNode[];
}

export default function DepartmentProcessPage() {
  const { message } = App.useApp();
  const [departments, setDepartments] = useState<Department[]>([]);
  const [selectedDeptId, setSelectedDeptId] = useState<number | null>(null);
  const [selectedDept, setSelectedDept] = useState<Department | null>(null);
  const [deptProcesses, setDeptProcesses] = useState<ProcessBinding[]>([]);
  const [loading, setLoading] = useState(false);
  const [showInitModal, setShowInitModal] = useState(false);
  const [departmentType, setDepartmentType] = useState('operations');

  // Load departments
  useEffect(() => {
    loadDepartments();
  }, []);

  // Load department processes when selected
  useEffect(() => {
    if (selectedDeptId) {
      loadDeptProcesses(selectedDeptId);
      const dept = departments.find(d => d.id === selectedDeptId);
      setSelectedDept(dept || null);
    }
  }, [selectedDeptId, departments]);

  const loadDepartments = async () => {
    try {
      const data = await departmentService.getDepartmentTree();
      setDepartments(flattenDepartments(data));
    } catch (error) {
      console.error('Failed to load departments:', error);
      message.error('加载部门失败');
    }
  };

  const loadDeptProcesses = async (deptId: number) => {
    setLoading(true);
    try {
      const data = await ProcessBindingApi.listDepartmentProcesses(deptId);
      setDeptProcesses(data);
    } catch (error) {
      console.error('Failed to load department processes:', error);
      message.error('加载部门流程失败');
    } finally {
      setLoading(false);
    }
  };

  const handleInitDefaults = async () => {
    if (!selectedDeptId) return;

    try {
      await ProcessBindingApi.initDepartmentProcesses(selectedDeptId, departmentType);
      message.success('部门流程初始化成功');
      loadDeptProcesses(selectedDeptId);
      setShowInitModal(false);
    } catch (error) {
      console.error('Failed to initialize department processes:', error);
      message.error('初始化部门流程失败');
    }
  };

  const flattenDepartments = (items: Department[]): Department[] => {
    return items.flatMap(item => [item, ...flattenDepartments(item.children || [])]);
  };

  // Build department tree
  const buildDeptTree = (depts: Department[], parentId: number = 0): DepartmentTreeNode[] => {
    return depts
      .filter(d => (d as any).parentId === parentId)
      .map(d => ({
        title: `${d.name} (${d.code})`,
        key: d.id,
        icon: <Building2 />,
        children: buildDeptTree(depts, d.id),
      }));
  };

  // Scenario color mapping
  const getScenarioColor = (scenario?: string) => {
    if (!scenario) return 'default';
    if (scenario.includes('alert')) return 'red';
    if (scenario.includes('change')) return 'blue';
    if (scenario.includes('release')) return 'green';
    if (scenario.includes('expense') || scenario.includes('budget')) return 'gold';
    if (scenario.includes('leave') || scenario.includes('recruitment')) return 'purple';
    return 'default';
  };

  // Table columns
  const columns = [
    {
      title: '场景',
      dataIndex: 'scenario',
      key: 'scenario',
      render: (scenario: string) => (
        <Tag color={getScenarioColor(scenario)}>
          {scenario || '默认'}
        </Tag>
      ),
    },
    {
      title: '业务类型',
      dataIndex:'businessType',
      key:'businessType',
      render: (type: string) => <Tag>{type}</Tag>,
    },
    {
      title: '流程定义',
      dataIndex:'processDefinitionKey',
      key:'processDefinitionKey',
      render: (key: string) => <Tag color="cyan">{key}</Tag>,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
    },
    {
      title: '状态',
      dataIndex: 'isActive',
      key: 'isActive',
      render: (active: boolean) => (
        <Tag icon={active ? <CheckCircle /> : <AlertTriangle />} color={active ? 'success' : 'error'}>
          {active ? '启用' : '停用'}
        </Tag>
      ),
    },
  ];

  // Statistics
  const stats = {
    total: deptProcesses.length,
    active: deptProcesses.filter(p => (p as any).isActive).length,
    scenarios: new Set(deptProcesses.map(p => p.scenario).filter(Boolean)).size,
  };

  return (
    <div className="space-y-6">
      {/* 关联页面入口 */}
      <Alert
        type="info"
        showIcon
        message="部门流程配置用于查看各部门已绑定的流程"
        description={
          <Space wrap>
            <span>相关配置：</span>
            <Link href="/admin/process-routing">流程路由规则</Link>
            <Link href="/admin/workflows">工作流管理</Link>
          </Space>
        }
      />
      <Card title="部门流程配置">
        <Row gutter={24}>
          {/* Department Tree */}
          <Col span={8}>
            <Card title="部门列表" size="small">
              <Tree
                showIcon
                treeData={buildDeptTree(departments)}
                onSelect={(keys) => {
                  if (keys.length > 0) {
                    setSelectedDeptId(keys[0] as number);
                  }
                }}
                style={{ maxHeight: 600, overflow: 'auto' }}
              />
            </Card>
          </Col>

          {/* Process Configuration */}
          <Col span={16}>
            {selectedDeptId ? (
              <>
                {/* Department Info */}
                <Card size="small" style={{ marginBottom: 16 }}>
                  <Descriptions size="small">
                    <Descriptions.Item label="部门名称">
                      {selectedDept?.name}
                    </Descriptions.Item>
                    <Descriptions.Item label="编码">
                      <Tag>{selectedDept?.code}</Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="描述">
                      {selectedDept?.description || '-'}
                    </Descriptions.Item>
                  </Descriptions>
                </Card>

                {/* Statistics */}
                <Row gutter={16} style={{ marginBottom: 16 }}>
                  <Col span={8}>
                    <Card size="small">
                      <Statistic title="流程总数" value={stats.total} />
                    </Card>
                  </Col>
                  <Col span={8}>
                    <Card size="small">
                      <Statistic title="已启用" value={stats.active} valueStyle={{ color: '#3f8600' }} />
                    </Card>
                  </Col>
                  <Col span={8}>
                    <Card size="small">
                      <Statistic title="场景数" value={stats.scenarios} />
                    </Card>
                  </Col>
                </Row>

                {/* Actions */}
                <Card size="small" style={{ marginBottom: 16 }}>
                  <Space>
                    <Button
                      type="primary"
                      icon={<Plus />}
                      onClick={() => setShowInitModal(true)}
                    >
                      初始化默认流程模板
                    </Button>
                    <Button
                      icon={<RefreshCw />}
                      onClick={() => loadDeptProcesses(selectedDeptId)}
                    >
                      刷新
                    </Button>
                  </Space>
                </Card>

                {/* Process Table */}
                <Card size="small">
                  <Table
                    columns={columns}
                    dataSource={deptProcesses}
                    loading={loading}
                    rowKey="id"
                    pagination={false}
                  />
                </Card>
              </>
            ) : (
              <Card>
                <div style={{ textAlign: 'center', padding: 48, color: '#999' }}>
                  <Building2 style={{ width: 48, height: 48, marginBottom: 16 }} />
                  <p>请从左侧选择一个部门查看流程配置</p>
                </div>
              </Card>
            )}
          </Col>
        </Row>
      </Card>

      {/* Initialize Modal */}
      <Modal
        title="初始化部门流程"
        open={showInitModal}
        onCancel={() => setShowInitModal(false)}
        onOk={handleInitDefaults}
      >
        <Alert
          message="将为所选部门创建默认流程模板（不会影响已有配置）"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />
        <Form layout="vertical">
          <Form.Item label="部门类型">
            <Select
              value={departmentType}
              onChange={setDepartmentType}
              options={[{ value: 'operations', label: '运维（告警处理、变更发布）' }, { value: 'rd', label: '研发（代码发布、需求变更）' }, { value: 'finance', label: '财务（费用审批、预算管理）' }, { value: 'hr', label: '人力（请假、招聘）' }]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
