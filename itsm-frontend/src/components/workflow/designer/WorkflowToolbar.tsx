// 工作流工具栏组件
// Workflow Toolbar Component - 顶部操作栏

'use client';

import React from 'react';
import { Button, Space, Tag, Typography } from 'antd';
import {
  ArrowLeft,
  Save,
  PlayCircle,
  GitBranch,
  Settings,
  Edit3,
  CheckCircle,
  Clock,
  AlertCircle,
} from 'lucide-react';
import { useRouter } from 'next/navigation';
import type { WorkflowDefinition } from './WorkflowTypes';

const { Title, Text } = Typography;

interface WorkflowToolbarProps {
  workflow: WorkflowDefinition | null;
  saving: boolean;
  deploying: boolean;
  onSave: (xml: string) => void;
  onSaveAndDeploy: (xml: string) => void;
  onDeploy: () => void;
  currentXML: string;
}

export default function WorkflowToolbar({
  workflow,
  saving,
  deploying,
  onSave,
  onSaveAndDeploy,
  onDeploy,
  currentXML,
}: WorkflowToolbarProps) {
  const router = useRouter();

  // 状态颜色映射
  const getStatusColor = (status: string): string => {
    switch (status) {
      case 'active':
        return 'success';
      case 'draft':
        return 'processing';
      case 'inactive':
        return 'default';
      case 'archived':
        return 'warning';
      default:
        return 'default';
    }
  };

  // 状态图标映射
  const getStatusIcon = (status: string): React.ReactNode => {
    switch (status) {
      case 'active':
        return <CheckCircle className="w-4 h-4" />;
      case 'draft':
        return <Edit3 className="w-4 h-4" />;
      case 'inactive':
        return <Clock className="w-4 h-4" />;
      case 'archived':
        return <AlertCircle className="w-4 h-4" />;
      default:
        return <Clock className="w-4 h-4" />;
    }
  };

  // 状态文本映射
  const getStatusText = (status: string): string => {
    switch (status) {
      case 'active':
        return '已激活';
      case 'draft':
        return '草稿';
      case 'inactive':
        return '未激活';
      case 'archived':
        return '已归档';
      default:
        return '未知';
    }
  };

  return (
    <div className="bg-white px-6 border-b border-gray-100 flex items-center justify-between h-16 leading-none">
      <div className="flex items-center justify-between w-full h-full">
        {/* 左侧 - 返回按钮和工作流信息 */}
        <div className="flex items-center">
          <Button
            type="text"
            icon={<ArrowLeft className="w-4 h-4" />}
            onClick={() => router.back()}
            className="mr-4"
          >
            返回
          </Button>
          <div>
            <Title level={4} className="!mb-0 !text-base">
              {workflow?.name || '工作流设计器'}
            </Title>
            <div className="flex items-center gap-2 mt-1">
              <Tag color={getStatusColor(workflow?.status || 'draft')} className="mr-0">
                <span className="flex items-center gap-1">
                  {getStatusIcon(workflow?.status || 'draft')}
                  {getStatusText(workflow?.status || 'draft')}
                </span>
              </Tag>
              <Text type="secondary" className="text-xs">
                版本 {workflow?.version}
              </Text>
              {workflow?.category && (
                <Tag color="blue" className="ml-2">
                  {workflow.category}
                </Tag>
              )}
            </div>
          </div>
        </div>

        {/* 右侧 - 操作按钮 */}
        <Space>
          <Button icon={<GitBranch className="w-4 h-4" />}>版本管理</Button>
          <Button icon={<Settings className="w-4 h-4" />}>流程设置</Button>
          <Button
            icon={<Save className="w-4 h-4" />}
            loading={saving}
            onClick={() => onSave(currentXML)}
          >
            保存
          </Button>
          <Button
            type="primary"
            icon={<PlayCircle className="w-4 h-4" />}
            loading={saving || deploying}
            onClick={() => onSaveAndDeploy(currentXML)}
            disabled={!workflow}
          >
            保存并部署
          </Button>
          <Button
            icon={<PlayCircle className="w-4 h-4" />}
            loading={deploying}
            onClick={onDeploy}
            disabled={!workflow || workflow.status === 'active'}
          >
            部署
          </Button>
        </Space>
      </div>
    </div>
  );
}
