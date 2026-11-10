// 工作流属性面板组件
// Workflow Properties Component - 版本管理/流程配置 Tab

'use client';

import React from 'react';
import {
  Card,
  Row,
  Col,
  Select,
  Input,
  Checkbox,
  Button,
  Timeline,
  Badge,
  Space,
  Typography,
  Tag,
  Empty,
  Tooltip,
} from 'antd';

const { Text } = Typography;
import { Info } from 'lucide-react';
import { Eye, GitBranch, Users } from 'lucide-react';
import { formatDateTime } from '@/lib/formatters';
import type {
  WorkflowDefinition,
  WorkflowVersion,
  ApprovalConfig,
  UserInfo,
  RoleInfo,
  GroupInfo,
  SLAConfig,
} from './WorkflowTypes';

interface WorkflowPropertiesProps {
  workflow: WorkflowDefinition | null;
  approvalConfig: ApprovalConfig;
  setApprovalConfig: (config: ApprovalConfig) => void;
  workflowVersions: WorkflowVersion[];
  userList: UserInfo[];
  roleList: RoleInfo[];
  groupList?: GroupInfo[];
  loadingUsers: boolean;
  loadingRoles: boolean;
  loadingGroups?: boolean;
  activeTab?: string;
  /** 显示模式：versions=版本管理, config=流程配置 */
  mode?: 'versions' | 'config';
  onSwitchVersion?: (versionId: string) => void;
  onShowVersionModal?: () => void;
  onUpdateSLA?: (config: Partial<SLAConfig>) => void;
}

export default function WorkflowProperties({
  workflow,
  approvalConfig,
  setApprovalConfig,
  workflowVersions,
  userList,
  roleList,
  groupList = [],
  loadingUsers,
  loadingRoles,
  loadingGroups = false,
  mode = 'config',
  onSwitchVersion,
  onShowVersionModal,
  onUpdateSLA,
}: WorkflowPropertiesProps) {
  // 审批类型变更
  const handleApprovalTypeChange = (value: string) => {
    setApprovalConfig({
      ...approvalConfig,
      approvalType: value as 'single' | 'parallel' | 'sequential' | 'conditional',
    });
  };

  // 审批人变更（流程级兜底，不覆盖节点级）
  const handleApproversChange = (value: string[]) => {
    setApprovalConfig({
      ...approvalConfig,
      approvers: value,
    });
  };

  // 自动审批角色变更（控件已禁用；保留以备后续接后端后启用）
  // const handleAutoApproveRolesChange = (value: string[]) => {
  //   setApprovalConfig({
  //     ...approvalConfig,
  //     autoApproveRoles: value,
  //   });
  // };

  // 响应时间变更
  const handleResponseTimeChange = (value: string) => {
    onUpdateSLA?.({
      responseTimeHours: parseInt(value) || 24,
    });
  };

  // 解决时间变更
  const handleResolutionTimeChange = (value: string) => {
    onUpdateSLA?.({
      resolutionTimeHours: parseInt(value) || 72,
    });
  };

  // 仅工作时间变更
  const handleBusinessHoursChange = (checked: boolean) => {
    onUpdateSLA?.({
      businessHoursOnly: checked,
    });
  };

  // 排除周末变更
  const handleExcludeWeekendsChange = (checked: boolean) => {
    onUpdateSLA?.({
      excludeWeekends: checked,
    });
  };

  // 排除节假日变更
  const handleExcludeHolidaysChange = (checked: boolean) => {
    onUpdateSLA?.({
      excludeHolidays: checked,
    });
  };

  // 版本管理模式：只显示版本历史
  if (mode === 'versions') {
    return (
      <div className="h-full">
        <Card className="rounded-lg shadow-sm border border-gray-200">
          <div className="flex justify-between items-center mb-6">
            <h3 className="text-base font-semibold mb-0">Version History</h3>
            <Button
              type="primary"
              icon={<GitBranch className="w-4 h-4" />}
              onClick={onShowVersionModal}
            >

              Create new version
            </Button>
          </div>

          {workflowVersions.length === 0 ? (
            <Empty description="No version record yet" />
          ) : (
            <Timeline className="mt-4">
              {workflowVersions.map((version: WorkflowVersion) => (
                <Timeline.Item
                  key={version.id}
                  dot={
                    <Badge
                      status={version.status === 'active' ? 'success' : 'default'}
                      text={version.status === 'active' ? 'current' : ''}
                    />
                  }
                >
                  <div className="flex justify-between items-center ml-2">
                    <div>
                      <Text strong>Version {version.version}</Text>
                      <div className="text-sm text-gray-500 mt-1">{version.changeLog}</div>
                      <div className="text-xs text-gray-400 mt-1">
                        {formatDateTime(version.createdAt)} - {version.createdBy}
                      </div>
                    </div>
                    <Space>
                      <Button
                        size="small"
                        icon={<Eye className="w-3 h-3" />}
                        onClick={() => onSwitchVersion?.(version.id)}
                      >

                        View
                      </Button>
                      {version.status !== 'active' && (
                        <Button
                          size="small"
                          type="primary"
                          onClick={() => onSwitchVersion?.(version.id)}
                        >

                          Switch to this version
                        </Button>
                      )}
                    </Space>
                  </div>
                </Timeline.Item>
              ))}
            </Timeline>
          )}
        </Card>
      </div>
    );
  }

  // 流程配置模式：显示审批配置和SLA配置
  return (
    <div className="h-full">
      <Row gutter={[24, 24]}>
        <Col span={12}>
          <Card
            title="Approval"
            className="h-full rounded-lg shadow-sm border border-gray-200"
          >
            <div className="space-y-6">
              {/* 审批类型 */}
              <div>
                <Text strong className="block mb-2">
                  <Space size={4}>

                    Approval Type
                    <Tooltip
                      title={
                        <div>
                          <div><b>single</b>: It ends when any approver passes it.</div>
                          <div><b>serial</b>: Approval one by one in order.</div>
                          <div><b>parallel</b>：Everyone needs to pass at the same time.</div>
                          <div><b>conditional</b>: Dynamically determined by expression.</div>
                        </div>
                      }
                    >
                      <Info className="text-gray-400 cursor-help" />
                    </Tooltip>
                  </Space>
                </Text>
                <Select
                  value={approvalConfig.approvalType}
                  onChange={handleApprovalTypeChange}
                  className="w-full"
                  options={[
                    { value: "single", label: "Single approver" },
                    { value: "parallel", label: "Parallel" },
                    { value: "sequential", label: "Sequential" },
                    { value: "conditional", label: "Conditional" },
                  ]}
                />
              </div>

              {/* 审批人 */}
              <div>
                <Text strong className="block mb-2">

                  Approver
                </Text>
                <Select
                  mode="multiple"
                  placeholder="Select approvers"
                  value={approvalConfig.approvers}
                  onChange={handleApproversChange}
                  className="w-full"
                  loading={loadingUsers}
                  maxTagCount="responsive"
                  options={userList.map(user => ({ value: String(user.id), label: user.name }))}
                />
              </div>

              {/* 审批组说明（节点级） */}
              <div className="p-3 border border-dashed border-gray-300 rounded-md bg-gray-50">
                <div className="flex items-center mb-1">
                  <Users className="w-4 h-4 mr-1 text-gray-500" />
                  <Text strong>Approval Group</Text>
                  <Text type="secondary" className="ml-auto text-xs">

                    Node level configuration
                  </Text>
                </div>
                <Text type="secondary" className="text-xs block">

                  The approval group is at the node level and needs to be selected on the canvas.
                  <b>Approval node</b>After that, in the "Node Properties" panel on the right <b>"Candidate Group"</b>  Select in field.
The selected group will be written to the BPMN XML <code className="text-xs">candidateGroups</code>  attribute, The
backend engine automatically expands group members into specific approvers when a task is created.
                </Text>
              </div>

            </div>
          </Card>
        </Col>

        <Col span={12}>
          <Card
            title="SLA"
            className="h-full rounded-lg shadow-sm border border-gray-200"
          >
            <div className="space-y-6">
              {/* 响应时间 */}
              <div>
                <Text strong className="block mb-2">

                  Response Time
                </Text>
                <Input
                  type="number"
                  suffix="hours"
                  value={workflow?.slaConfig?.responseTimeHours}
                  onChange={e => handleResponseTimeChange(e.target.value)}
                />
              </div>

              {/* 解决时间 */}
              <div>
                <Text strong className="block mb-2">

                  Resolved At
                </Text>
                <Input
                  type="number"
                  suffix="hours"
                  value={workflow?.slaConfig?.resolutionTimeHours}
                  onChange={e => handleResolutionTimeChange(e.target.value)}
                />
              </div>

              {/* 工作时间设置 */}
              <div>
                <Text strong className="block mb-2">

                  Business Hours
                </Text>
                <div className="space-y-3">
                  <div className="flex items-center">
                    <Checkbox
                      checked={workflow?.slaConfig?.businessHoursOnly}
                      onChange={e => handleBusinessHoursChange(e.target.checked)}
                    >

                      Business hours only
                    </Checkbox>
                  </div>
                  <div className="flex items-center">
                    <Checkbox
                      checked={workflow?.slaConfig?.excludeWeekends}
                      onChange={e => handleExcludeWeekendsChange(e.target.checked)}
                    >

                      Exclude weekends
                    </Checkbox>
                  </div>
                  <div className="flex items-center">
                    <Checkbox
                      checked={workflow?.slaConfig?.excludeHolidays}
                      onChange={e => handleExcludeHolidaysChange(e.target.checked)}
                    >

                      Exclude holidays
                    </Checkbox>
                  </div>
                </div>
              </div>
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  );
}
