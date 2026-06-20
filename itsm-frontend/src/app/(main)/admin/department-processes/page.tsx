'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Tree,
  Table,
  Button,
  Select,
  message,
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

const { Option } = Select;

interface Department {
  id: number;
  name: string;
  code: string;
  parent_id: number;
  manager_id: number;
  description?: string;
}

interface ProcessBinding {
  id: number;
  business_type: string;
  business_sub_type?: string;
  scenario?: string;
  process_definition_key: string;
  priority: number;
  is_active: boolean;
  conditions?: Record<string, any>;
}

interface DepartmentTreeNode {
  title: string;
  key: number;
  icon: React.ReactNode;
  children?: DepartmentTreeNode[];
}

export default function DepartmentProcessPage() {
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
      const response = await fetch('/api/v1/departments');
      if (response.ok) {
        const data = await response.json();
        setDepartments(data.data || []);
      }
    } catch (error) {
      console.error('Failed to load departments:', error);
    }
  };

  const loadDeptProcesses = async (deptId: number) => {
    setLoading(true);
    try {
      const response = await fetch(`/api/v1/departments/${deptId}/processes`);
      if (response.ok) {
        const data = await response.json();
        setDeptProcesses(data.data || []);
      }
    } catch (error) {
      console.error('Failed to load department processes:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleInitDefaults = async () => {
    if (!selectedDeptId) return;

    try {
      const response = await fetch(`/api/v1/departments/${selectedDeptId}/init-processes`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ department_type: departmentType }),
      });

      if (response.ok) {
        message.success('Department processes initialized successfully');
        loadDeptProcesses(selectedDeptId);
        setShowInitModal(false);
      } else {
        message.error('Failed to initialize department processes');
      }
    } catch (error) {
      message.error('Failed to initialize department processes');
    }
  };

  // Build department tree
  const buildDeptTree = (depts: Department[], parentId: number = 0): DepartmentTreeNode[] => {
    return depts
      .filter(d => d.parent_id === parentId)
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
      title: 'Scenario',
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
      dataIndex: 'business_type',
      key: 'business_type',
      render: (type: string) => <Tag>{type}</Tag>,
    },
    {
      title: 'Process',
      dataIndex: 'process_definition_key',
      key: 'process_definition_key',
      render: (key: string) => <Tag color="cyan">{key}</Tag>,
    },
    {
      title: 'Priority',
      dataIndex: 'priority',
      key: 'priority',
    },
    {
      title: 'Status',
      dataIndex: 'is_active',
      key: 'is_active',
      render: (active: boolean) => (
        <Tag icon={active ? <CheckCircle /> : <AlertTriangle />} color={active ? 'success' : 'error'}>
          {active ? 'Active' : 'Inactive'}
        </Tag>
      ),
    },
  ];

  // Statistics
  const stats = {
    total: deptProcesses.length,
    active: deptProcesses.filter(p => p.is_active).length,
    scenarios: new Set(deptProcesses.map(p => p.scenario).filter(Boolean)).size,
  };

  return (
    <div style={{ padding: 24 }}>
      <Card title="Department Process Configuration">
        <Row gutter={24}>
          {/* Department Tree */}
          <Col span={8}>
            <Card title="Departments" size="small">
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
                    <Descriptions.Item label="Department">
                      {selectedDept?.name}
                    </Descriptions.Item>
                    <Descriptions.Item label="Code">
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
                      <Statistic title="Total Processes" value={stats.total} />
                    </Card>
                  </Col>
                  <Col span={8}>
                    <Card size="small">
                      <Statistic title="Active" value={stats.active} valueStyle={{ color: '#3f8600' }} />
                    </Card>
                  </Col>
                  <Col span={8}>
                    <Card size="small">
                      <Statistic title="Scenarios" value={stats.scenarios} />
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
                      Initialize Default Templates
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
                  <p>Select a department from the tree to configure processes</p>
                </div>
              </Card>
            )}
          </Col>
        </Row>
      </Card>

      {/* Initialize Modal */}
      <Modal
        title="Initialize Department Processes"
        open={showInitModal}
        onCancel={() => setShowInitModal(false)}
        onOk={handleInitDefaults}
      >
        <Alert
          message="This will create default process templates for the selected department"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />
        <Form layout="vertical">
          <Form.Item label="Department Type">
            <Select
              value={departmentType}
              onChange={setDepartmentType}
            >
              <Option value="operations">Operations (Alert Handling, Change Release)</Option>
              <Option value="rd">R&D (Code Release, Requirement Change)</Option>
              <Option value="finance">Finance (Expense Approval, Budget)</Option>
              <Option value="hr">HR (Leave, Recruitment)</Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
