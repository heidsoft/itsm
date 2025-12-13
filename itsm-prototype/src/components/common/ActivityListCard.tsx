"use client";

import React from 'react';
import { Card, List, Avatar, Typography } from 'antd';
import { Clock, Activity } from 'lucide-react';
import { RecentActivity } from '@/hooks/useDashboardData';

const { Text } = Typography;

interface ActivityListCardProps {
  activities: RecentActivity[];
  title?: string;
  maxItems?: number;
  className?: string;
}

export const ActivityListCard: React.FC<ActivityListCardProps> = ({
  activities,
  title = "最近活动",
  maxItems = 10,
  className = '',
}) => {
  const displayActivities = activities.slice(0, maxItems);

  return (
    <Card
      title={
        <div className="flex items-center space-x-2">
          <Activity size={16} className="text-blue-500" />
          <span className="font-semibold text-gray-800">{title}</span>
        </div>
      }
      className={`shadow-sm border-0 ${className}`}
    >
      <List
        dataSource={displayActivities}
        renderItem={(item) => (
          <List.Item className="hover:bg-gray-50 transition-colors duration-200 rounded-lg px-3 py-2">
            <List.Item.Meta
              avatar={
                <Avatar
                  src={item.avatar}
                  size="small"
                  className="shadow-sm"
                >
                  {item.operator.charAt(0)}
                </Avatar>
              }
              title={
                <div className="flex items-center justify-between">
                  <Text className="text-sm font-medium text-gray-800">
                    <span className="text-blue-600">{item.operator}</span>
                    <span className="mx-1">{item.action}</span>
                    <span className="text-purple-600">{item.target}</span>
                  </Text>
                </div>
              }
              description={
                <div className="flex items-center space-x-1 text-xs text-gray-500">
                  <Clock size={12} />
                  <span>{item.time}</span>
                </div>
              }
            />
          </List.Item>
        )}
      />
    </Card>
  );
};

export default ActivityListCard;