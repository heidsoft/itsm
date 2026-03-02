// 工作流属性面板组件
// Workflow Properties Component - 流程配置 Tab

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
} from 'antd';

const { Text } = Typography;
import { Eye, GitBranch } from 'lucide-react';
import type {
  WorkflowDefinition,
  WorkflowVersion,
  ApprovalConfig,
  UserInfo,
  RoleInfo,
  SLAConfig,
} from './WorkflowTypes';

const { Option } = Select;

interface WorkflowPropertiesProps {
  workflow: WorkflowDefinition | null;
  approvalConfig: ApprovalConfig;
  setApprovalConfig: (config: ApprovalConfig) => void;
  workflowVersions: WorkflowVersion[];
  userList: UserInfo[];
  roleList: RoleInfo[];
  loadingUsers: boolean;
  loadingRoles: boolean;
  activeTab?: string;
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
  loadingUsers,
  loadingRoles,
  onSwitchVersion,
  onShowVersionModal,
  onUpdateSLA,
}: WorkflowPropertiesProps) {
  // 审批类型变更
  const handleApprovalTypeChange = (value: string) => {
    setApprovalConfig({
      ...approvalConfig,
      approval_type: value as 'single' | 'parallel' | 'sequential' | 'conditional',
    });
  };

  // 审批人变更
  const handleApproversChange = (value: string[]) => {
    setApprovalConfig({
      ...approvalConfig,
      approvers: value,
    });
  };

  // 自动审批角色变更
  const handleAutoApproveRolesChange = (value: string[]) => {
    setApprovalConfig({
      ...approvalConfig,
      auto_approve_roles: value,
    });
  };

  // 响应时间变更
  const handleResponseTimeChange = (value: string) => {
    onUpdateSLA?.({
      response_time_hours: parseInt(value) || 24,
    });
  };

  // 解决时间变更
  const handleResolutionTimeChange = (value: string) => {
    onUpdateSLA?.({
      resolution_time_hours: parseInt(value) || 72,
    });
  };

  // 仅工作时间变更
  const handleBusinessHoursChange = (checked: boolean) => {
    onUpdateSLA?.({
      business_hours_only: checked,
    });
  };

  // 排除周末变更
  const handleExcludeWeekendsChange = (checked: boolean) => {
    onUpdateSLA?.({
      exclude_weekends: checked,
    });
  };

  // 排除节假日变更
  const handleExcludeHolidaysChange = (checked: boolean) => {
    onUpdateSLA?.({
      exclude_holidays: checked,
    });
  };

  return (
    <div className='h-full'>
      {/* 版本历史内容 */}
      <Card className='rounded-lg shadow-sm border border-gray-200' variant='borderless'>
        <div className='flex justify-between items-center mb-6'>
          <h3 className='text-base font-semibold mb-0'>版本历史</h3>
          <Button
            type='primary'
            icon={<GitBranch className='w-4 h-4' />}
            onClick={onShowVersionModal}
          >
            创建新版本
          </Button>
        </div>

        <Timeline className='mt-4'>
          {workflowVersions.map((version: WorkflowVersion) => (
            <Timeline.Item
              key={version.id}
              dot={
                <Badge
                  status={version.status === 'active' ? 'success' : 'default'}
                  text={version.status === 'active' ? '当前' : ''}
                />
              }
            >
              <div className='flex justify-between items-center ml-2'>
                <div>
                  <Text strong>版本 {version.version}</Text>
                  <div className='text-sm text-gray-500 mt-1'>{version.change_log}</div>
                  <div className='text-xs text-gray-400 mt-1'>
                    {new Date(version.created_at).toLocaleString()} - {version.created_by}
                  </div>
                </div>
                <Space>
                  <Button
                    size='small'
                    icon={<Eye className='w-3 h-3' />}
                    onClick={() => onSwitchVersion?.(version.id)}
                  >
                    查看
                  </Button>
                  {version.status !== 'active' && (
                    <Button
                      size='small'
                      type='primary'
                      onClick={() => onSwitchVersion?.(version.id)}
                    >
                      切换到此版本
                    </Button>
                  )}
                </Space>
              </div>
            </Timeline.Item>
          ))}
        </Timeline>
      </Card>

      {/* 流程配置内容 */}
      <Row gutter={[24, 24]} className='mt-4'>
        <Col span={12}>
          <Card
            title='审批配置'
            className='h-full rounded-lg shadow-sm border border-gray-200'
            variant='borderless'
          >
            <div className='space-y-6'>
              {/* 审批类型 */}
              <div>
                <Text strong className='block mb-2'>
                  审批类型
                </Text>
                <Select
                  value={approvalConfig.approval_type}
                  onChange={handleApprovalTypeChange}
                  className='w-full'
                >
                  <Option value='single'>单人审批</Option>
                  <Option value='parallel'>并行审批</Option>
                  <Option value='sequential'>串行审批</Option>
                  <Option value='conditional'>条件审批</Option>
                </Select>
              </div>

              {/* 审批人 */}
              <div>
                <Text strong className='block mb-2'>
                  审批人
                </Text>
                <Select
                  mode='multiple'
                  placeholder='选择审批人'
                  value={approvalConfig.approvers}
                  onChange={handleApproversChange}
                  className='w-full'
                  loading={loadingUsers}
                >
                  {userList.map(user => (
                    <Option key={user.id} value={String(user.id)}>
                      {user.name}
                    </Option>
                  ))}
                </Select>
              </div>

              {/* 自动审批角色 */}
              <div>
                <Text strong className='block mb-2'>
                  自动审批角色
                </Text>
                <Select
                  mode='multiple'
                  placeholder='选择角色'
                  value={approvalConfig.auto_approve_roles}
                  onChange={handleAutoApproveRolesChange}
                  className='w-full'
                  loading={loadingRoles}
                >
                  {roleList.map(role => (
                    <Option key={role.code} value={role.code}>
                      {role.name}
                    </Option>
                  ))}
                </Select>
              </div>
            </div>
          </Card>
        </Col>

        <Col span={12}>
          <Card
            title='SLA配置'
            className='h-full rounded-lg shadow-sm border border-gray-200'
            variant='borderless'
          >
            <div className='space-y-6'>
              {/* 响应时间 */}
              <div>
                <Text strong className='block mb-2'>
                  响应时间
                </Text>
                <Input
                  type='number'
                  suffix='小时'
                  value={workflow?.sla_config?.response_time_hours}
                  onChange={e => handleResponseTimeChange(e.target.value)}
                />
              </div>

              {/* 解决时间 */}
              <div>
                <Text strong className='block mb-2'>
                  解决时间
                </Text>
                <Input
                  type='number'
                  suffix='小时'
                  value={workflow?.sla_config?.resolution_time_hours}
                  onChange={e => handleResolutionTimeChange(e.target.value)}
                />
              </div>

              {/* 工作时间设置 */}
              <div>
                <Text strong className='block mb-2'>
                  工作时间设置
                </Text>
                <div className='space-y-3'>
                  <div className='flex items-center'>
                    <Checkbox
                      checked={workflow?.sla_config?.business_hours_only}
                      onChange={e => handleBusinessHoursChange(e.target.checked)}
                    >
                      仅工作时间
                    </Checkbox>
                  </div>
                  <div className='flex items-center'>
                    <Checkbox
                      checked={workflow?.sla_config?.exclude_weekends}
                      onChange={e => handleExcludeWeekendsChange(e.target.checked)}
                    >
                      排除周末
                    </Checkbox>
                  </div>
                  <div className='flex items-center'>
                    <Checkbox
                      checked={workflow?.sla_config?.exclude_holidays}
                      onChange={e => handleExcludeHolidaysChange(e.target.checked)}
                    >
                      排除节假日
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
