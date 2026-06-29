// 工作流设计器工具栏
// Workflow Designer Toolbar Component

'use client';

import React from 'react';
import { Button, Space, Tag, Breadcrumb, Typography, Dropdown, MenuProps, Tooltip } from 'antd';
import {
  SaveOutlined,
  CloudUploadOutlined,
  PlayCircleOutlined,
  ExportOutlined,
  HistoryOutlined,
  SettingOutlined,
  EditOutlined,
  BugOutlined,
  RocketOutlined,
  RobotOutlined
} from '@ant-design/icons';
import Link from 'next/link';
import type { WorkflowDefinition } from './WorkflowTypes';

const { Text } = Typography;

interface WorkflowToolbarProps {
  workflow: WorkflowDefinition | null;
  saving: boolean;
  deploying: boolean;
  onSave: (xml: string) => void;
  onSaveAndDeploy: (xml: string) => void;
  onDeploy: () => void;
  currentXML: string;
  onValidate?: () => void;
  validationIssues?: any[];
  onAIClick?: () => void;
}

export default function WorkflowToolbar({
  workflow,
  saving,
  deploying,
  onSave,
  onSaveAndDeploy,
  onDeploy,
  currentXML,
  onValidate,
  validationIssues = [],
  onAIClick
}: WorkflowToolbarProps) {
  // 导出XML
  const handleExportXML = () => {
    if (!currentXML) return;
    const blob = new Blob([currentXML], { type: 'application/xml' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${workflow?.name || 'workflow'}.bpmn`;
    link.click();
    URL.revokeObjectURL(url);
  };

  // 更多操作菜单
  const moreMenuItems: MenuProps['items'] = [
    {
      key: 'export',
      icon: <ExportOutlined />,
      label: '导出BPMN',
      onClick: handleExportXML
    },
    {
      key: 'history',
      icon: <HistoryOutlined />,
      label: '版本历史',
      onClick: () => {
        // 切换到版本标签页
        const event = new MouseEvent('click', { bubbles: true });
        const tab = document.querySelector('.ant-tabs-tab:nth-child(2)');
        if (tab) tab.dispatchEvent(event);
      }
    },
    {
      key: 'settings',
      icon: <SettingOutlined />,
      label: '流程设置',
      onClick: () => {
        // 切换到配置标签页
        const event = new MouseEvent('click', { bubbles: true });
        const tab = document.querySelector('.ant-tabs-tab:nth-child(3)');
        if (tab) tab.dispatchEvent(event);
      }
    },
    {
      type: 'divider'
    },
    {
      key: 'validate',
      icon: <BugOutlined />,
      label: (
        <Space>
          校验流程
          {validationIssues.length > 0 && (
            <Tag color={validationIssues.some(i => i.type === 'error') ? 'error' : 'warning'} size="small">
              {validationIssues.length}
            </Tag>
          )}
        </Space>
      ),
      onClick: onValidate
    }
  ];

  // AI操作菜单
  const aiMenuItems: MenuProps['items'] = [
    {
      key: 'generate',
      icon: <RocketOutlined />,
      label: 'AI生成流程',
      onClick: onAIClick
    },
    {
      key: 'optimize',
      icon: <RobotOutlined />,
      label: 'AI优化建议',
      onClick: onAIClick
    },
    {
      key: 'check',
      icon: <BugOutlined />,
      label: 'AI合规检查',
      onClick: onAIClick
    }
  ];

  return (
    <div className="bg-white border-b border-gray-200 px-6 py-3 flex justify-between items-center">
      <div className="flex items-center gap-4">
        <Breadcrumb>
          <Breadcrumb.Item>
            <Link href="/workflow">工作流管理</Link>
          </Breadcrumb.Item>
          <Breadcrumb.Item>
            {workflow?.name || '新工作流设计'}
          </Breadcrumb.Item>
        </Breadcrumb>

        {workflow?.version && (
          <Tag color="blue">v{workflow.version}</Tag>
        )}

        {workflow?.status && (
          <Tag color={workflow.status === 'active' ? 'success' : 'default'}>
            {workflow.status === 'active' ? '已部署' : '草稿'}
          </Tag>
        )}
      </div>

      <Space>
        <Dropdown menu={{ items: aiMenuItems }} placement="bottomRight">
          <Tooltip title="AI辅助功能">
            <Button icon={<RobotOutlined />}>
              AI助手
            </Button>
          </Tooltip>
        </Dropdown>

        <Dropdown menu={{ items: moreMenuItems }} placement="bottomRight">
          <Button>
            更多操作
          </Button>
        </Dropdown>

        <Button
          icon={<SaveOutlined />}
          loading={saving}
          onClick={() => onSave(currentXML)}
        >
          保存
        </Button>

        {workflow?.status !== 'active' && (
          <Button
            type="primary"
            icon={<CloudUploadOutlined />}
            loading={deploying}
            onClick={() => onSaveAndDeploy(currentXML)}
          >
            保存并部署
          </Button>
        )}

        {workflow?.status === 'active' && (
          <Button
            type="primary"
            icon={<PlayCircleOutlined />}
            loading={deploying}
            onClick={onDeploy}
          >
            重新部署
          </Button>
        )}
      </Space>
    </div>
  );
}
