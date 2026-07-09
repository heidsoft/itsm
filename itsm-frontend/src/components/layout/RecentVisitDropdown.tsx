'use client';

import React from 'react';
import { Dropdown, Button, List, Empty, Tooltip } from 'antd';
import { Clock, X } from 'lucide-react';
import { useRecentVisitStore } from '@/lib/store/recent-visit-store';
import { useRouter } from 'next/navigation';

export const RecentVisitDropdown: React.FC = () => {
  const router = useRouter();
  const { visits, removeVisit, clearVisits } = useRecentVisitStore();

  const items = [
    {
      key: 'header',
      label: (
        <div className="flex items-center justify-between px-2 py-1 border-b">
          <span className="font-medium">最近访问</span>
          {visits.length > 0 && (
            <Button
              type="text"
              size="small"
              onClick={(e) => {
                e.stopPropagation();
                clearVisits();
              }}
              className="text-xs text-gray-500 hover:text-gray-700"
            >
              清空
            </Button>
          )}
        </div>
      ),
      disabled: true,
    },
    ...(visits.length === 0
      ? [
          {
            key: 'empty',
            label: <Empty description="暂无访问记录" image={Empty.PRESENTED_IMAGE_SIMPLE} />,
            disabled: true,
          },
        ]
      : visits.map((visit) => ({
          key: visit.id,
          label: (
            <div className="flex items-center justify-between group">
              <div
                className="flex-1 truncate py-1"
                onClick={(e) => {
                  e.stopPropagation();
                  router.push(visit.path);
                }}
                style={{ cursor: 'pointer' }}
              >
                {visit.title}
              </div>
              <Button
                type="text"
                size="small"
                icon={<X size={12} />}
                onClick={(e) => {
                  e.stopPropagation();
                  removeVisit(visit.path);
                }}
                className="opacity-0 group-hover:opacity-100 text-gray-400 hover:text-red-500"
              />
            </div>
          ),
        }))),
  ];

  return (
    <Dropdown menu={{ items }} placement="bottomRight" trigger={['click']}>
      <Tooltip title="最近访问">
        <Button
          type="text"
          className="mr-2"
          aria-label="最近访问"
          title="最近访问"
        >
          <Clock size={18} />
        </Button>
      </Tooltip>
    </Dropdown>
  );
};
