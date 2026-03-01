// 版本管理模态框
// Workflow Version Modal

'use client';

import React from 'react';
import { Modal, Alert, Typography, Tag } from 'antd';
import type { WorkflowDefinition } from './WorkflowTypes';

const { Text } = Typography;

interface WorkflowVersionModalProps {
  visible: boolean;
  onClose: () => void;
  onCreate: () => void;
  workflow: WorkflowDefinition | null;
}

export default function WorkflowVersionModal({
  visible,
  onClose,
  onCreate,
  workflow,
}: WorkflowVersionModalProps) {
  return (
    <Modal
      title="创建新版本"
      open={visible}
      onOk={onCreate}
      onCancel={onClose}
      okText="创建"
      cancelText="取消"
    >
      <div className="space-y-4">
        <Alert
          message="版本管理"
          description="创建新版本将保存当前的设计状态，不会影响已部署的版本。"
          type="info"
          showIcon
        />
        <div>
          <Text strong>当前版本</Text>
          <div className="mt-1">
            <Tag color="blue">{workflow?.version}</Tag>
          </div>
        </div>
        <div>
          <Text strong>新版本号</Text>
          <div className="mt-1">
            <Tag color="green">
              {workflow ? `${parseFloat(workflow.version) + 0.1}`.slice(0, 3) : '1.1'}
            </Tag>
          </div>
        </div>
      </div>
    </Modal>
  );
}
