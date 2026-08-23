// 工作流设计器工具栏
// Workflow Designer Toolbar Component

'use client';

import React from 'react';
import type { MenuProps } from 'antd';
import { Button, Space, Tag, Breadcrumb, Typography, Dropdown, Tooltip } from 'antd';
import { Save, Download, Settings, History, Bug, Rocket, PlayCircle, CloudUpload, Bot } from 'lucide-react';
import Link from 'next/link';
import { useI18n } from '@/lib/i18n/useI18n';
import type { WorkflowDefinition } from './WorkflowTypes';

const { Text } = Typography;

interface WorkflowToolbarProps {
  workflow: WorkflowDefinition | null;
  saving: boolean;
  deploying: boolean;
  hasChanges?: boolean;
  onSave: (xml: string) => void;
  onSaveAndDeploy: (xml: string) => void;
  onDeploy: () => void;
  currentXML: string;
  onValidate?: () => void;
  validationIssues?: any[];
  onAIClick?: () => void;
  onTabChange?: (key: string) => void;
}

export default function WorkflowToolbar({
  workflow,
  saving,
  deploying,
  hasChanges = false,
  onSave,
  onSaveAndDeploy,
  onDeploy,
  currentXML,
  onValidate,
  validationIssues = [],
  onAIClick,
  onTabChange,
}: WorkflowToolbarProps) {
  const { t } = useI18n();

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
      icon: <Download />,
      label: t('workflow.designer.toolbarExportBpmn'),
      onClick: handleExportXML
    },
    {
      key: 'history',
      icon: <History />,
      label: t('workflow.designer.toolbarVersionHistory'),
      onClick: () => onTabChange?.('versions')
    },
    {
      key: 'settings',
      icon: <Settings />,
      label: t('workflow.designer.toolbarProcessSettings'),
      onClick: () => onTabChange?.('config')
    },
    {
      type: 'divider'
    },
    {
      key: 'validate',
      icon: <Bug />,
      label: (
        <Space>
          {t('workflow.designer.toolbarValidate')}
          {validationIssues.length > 0 && (
            <Tag color={validationIssues.some(i => i.type === 'error') ? 'error' : 'warning'}>
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
      icon: <Rocket />,
      label: t('workflow.designer.toolbarAIGenerate'),
      onClick: onAIClick
    },
    {
      key: 'optimize',
      icon: <Bot />,
      label: t('workflow.designer.toolbarAIOptimize'),
      onClick: onAIClick
    },
    {
      key: 'check',
      icon: <Bug />,
      label: t('workflow.designer.toolbarAICheck'),
      onClick: onAIClick
    }
  ];

  return (
    <div className="bg-white border-b border-gray-200 px-6 py-3 flex justify-between items-center">
      <div className="flex items-center gap-4">
        <Breadcrumb
          items={[
            {
              title: <Link href="/workflow">{t('workflow.designer.toolbarBreadcrumb')}</Link>,
            },
            {
              title: workflow?.name || t('workflow.designer.toolbarNewWorkflow'),
            },
          ]}
        />

        {workflow?.version && (
          <Tag color="blue">v{workflow.version}</Tag>
        )}

        {workflow?.status && (
          <Tag color={workflow.status === 'active' ? 'success' : 'default'}>
            {workflow.status === 'active' ? t('workflow.designer.toolbarDeployed') : t('workflow.designer.toolbarDraft')}
          </Tag>
        )}

        {hasChanges && (
          <Tag color="warning">{t('workflow.designer.toolbarUnsaved')}</Tag>
        )}
      </div>

      <Space>
        <Dropdown menu={{ items: aiMenuItems }} placement="bottomRight">
          <Tooltip title={t('workflow.designer.toolbarAITooltip')}>
            <Button icon={<Bot />}>
              {t('workflow.designer.toolbarAI')}
            </Button>
          </Tooltip>
        </Dropdown>

        <Dropdown menu={{ items: moreMenuItems }} placement="bottomRight">
          <Button>
            {t('workflow.designer.toolbarMore')}
          </Button>
        </Dropdown>

        <Button
          icon={<Save />}
          loading={saving}
          onClick={() => onSave(currentXML)}
        >
          {t('workflow.designer.toolbarSave')}
        </Button>

        {workflow?.status !== 'active' && (
          <Button
            type="primary"
            icon={<CloudUpload />}
            loading={deploying}
            onClick={() => onSaveAndDeploy(currentXML)}
          >
            {t('workflow.designer.toolbarSaveDeploy')}
          </Button>
        )}

        {workflow?.status === 'active' && (
          <Button
            type="primary"
            icon={<PlayCircle />}
            loading={deploying}
            onClick={onDeploy}
          >
            {t('workflow.designer.toolbarRedeploy')}
          </Button>
        )}
      </Space>
    </div>
  );
}
