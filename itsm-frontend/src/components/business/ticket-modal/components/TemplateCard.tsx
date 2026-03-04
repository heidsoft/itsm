/**
 * 工单模板卡片组件
 */

import React from 'react';
import { Button } from 'antd';
import { Edit, Trash2 } from 'lucide-react';
import type { TicketTemplate } from '../types';

interface TemplateCardProps {
  template: TicketTemplate;
  onEdit: (template: TicketTemplate) => void;
  onDelete: (templateId: number) => void;
}

export const TemplateCard: React.FC<TemplateCardProps> = ({
  template,
  onEdit,
  onDelete,
}) => {
  return (
    <div className="p-4 border border-gray-200 rounded-lg hover:border-blue-300 transition-colors">
      <div className="flex justify-between items-start">
        <div>
          <h4 className="font-semibold text-gray-900">{template.name}</h4>
          <p className="text-sm text-gray-600 mt-1">{template.description}</p>
          <div className="flex gap-4 mt-2 text-xs text-gray-500">
            <span>类型: {String(template.type)}</span>
            <span>分类: {template.category}</span>
            <span>优先级: {String(template.priority)}</span>
            <span>SLA: {template.sla}</span>
          </div>
        </div>
        <div className="flex gap-2">
          <Button
            size="small"
            icon={<Edit size={14} />}
            onClick={() => onEdit(template)}
          >
            编辑
          </Button>
          <Button size="small" danger onClick={() => onDelete(template.id)}>
            删除
          </Button>
        </div>
      </div>
    </div>
  );
};

TemplateCard.displayName = 'TemplateCard';
