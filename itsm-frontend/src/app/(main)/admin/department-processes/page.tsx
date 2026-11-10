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
      message.error('Failed to load department');
    }
  };

  const loadDeptProcesses = async (deptId: number) => {
    setLoading(true);
    try {
      const data = await ProcessBindingApi.listDepartmentProcesses(deptId);
      setDeptProcesses(data);
    } catch (error) {
      console.error('Failed to load department processes:', error);
      message.error('Failed to load department process');
    } finally {
      setLoading(false);
    }
  };

  const handleInitDefaults = async () => {
    if (!selectedDeptId) return;

    try {
      await ProcessBindingApi.initDepartmentProcesses(selectedDeptId, departmentType);
      message.success('Department process initialization successful');
      loadDeptProcesses(selectedDeptId);
      setShowInitModal(false);
    } catch (error) {
      console.error('Failed to initialize department processes:', error);
      message.error('Failed to initialize department process');
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
      title: 'Scene',
      dataIndex: 'scenario',
      key: 'scenario',
      render: (scenario: string) => (
        <Tag color={getScenarioColor(scenario)}>
          {scenario || 'Default'}
        </Tag>
      ),
    },
    {
      title: 'Business Type',
      dataIndex:'businessType',
      key:'businessType',
      render: (type: string) => <Tag>{type}</Tag>,
    },
    {
      title: 'Process Definitions',
      dataIndex:'processDefinitionKey',
      key:'processDefinitionKey',
      render: (key: string) => <Tag color="cyan">{key}</Tag>,
    },
    {
      title: 'Priority',
      dataIndex: 'priority',
      key: 'priority',
    },
    {
      title: 'Status',
      dataIndex: 'isActive',
      key: 'isActive',
      render: (active: boolean) => (
        <Tag icon={active ? <CheckCircle /> : <AlertTriangle />} color={active ? 'success' : 'error'}>
          {active ? 'Enabled' : 'Disable'}
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
        title="Department process configuration is used to view the processes bound to each department"
      />
      <Card title="Department process configuration">
        <Row gutter={24}>
          {/* Department Tree */}
          <Col span={8}>
            <Card title="Department list" size="small">
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
                    <Descriptions.Item label="Department Name">
                      {selectedDept?.name}
                    </Descriptions.Item>
                    <Descriptions.Item label="Encoding">
                      <Tag>{selectedDept?.code}</Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="Description">
                      {selectedDept?.description || '-'}
                    </Descriptions.Item>
                  </Descriptions>
                </Card>

                {/* Statistics */}
                <Row gutter={16} style={{ marginBottom: 16 }}>
                  <Col span={8}>
                    <Card size="small">
                      <Statistic title="Total number of processes" value={stats.total} />
                    </Card>
                  </Col>
                  <Col span={8}>
                    <Card size="small">
                      <Statistic title="Enabled" value={stats.active} styles={{ content: { color: '#3f8600' } }} />
                    </Card>
                  </Col>
                  <Col span={8}>
                    <Card size="small">
                      <Statistic title="Number of scenes" value={stats.scenarios} />
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

                      Initialize default process template
                    </Button>
                    <Button
                      icon={<RefreshCw />}
                      onClick={() => loadDeptProcesses(selectedDeptId)}
                    >

                      Refresh
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
                  <p>Please select a department from the left to view the process configuration</p>
                </div>
              </Card>
            )}
          </Col>
        </Row>
      </Card>

      {/* Initialize Modal */}
      <Modal
        title="Initialize department process"
        open={showInitModal}
        onCancel={() => setShowInitModal(false)}
        onOk={handleInitDefaults}
      >
        <Alert
          title="A default process template will be created for the selected department (will not affect existing configuration)"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />
        <Form layout="vertical">
          <Form.Item label="Department type">
            <Select
              value={departmentType}
              onChange={setDepartmentType}
              options={[{ value: 'operations', label: 'Operation and maintenance (alarm processing, change release)' }, { value: 'rd', label: 'Research and development (code release, requirement changes)' }, { value: 'finance', label: 'Finance (expense approval, budget management)' }, { value: 'hr', label: 'Manpower (leave, recruitment)' }]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
