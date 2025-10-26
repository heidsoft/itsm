"use client";

import React from 'react';
import { Card, List, Tag, Button, Badge, Dropdown, Menu, Avatar, Space, Tooltip } from 'antd';
import { MoreHorizontal, Eye, Clock, User } from 'lucide-react';
import { RecentTicket } from '../../hooks/useDashboardData';

interface TicketListCardProps {
  tickets: RecentTicket[];
  title?: string;
  showActions?: boolean;
  onViewAll?: () => void;
  onViewTicket?: (ticketId: string) => void;
  className?: string;
}

export const TicketListCard: React.FC<TicketListCardProps> = ({
  tickets,
  title = "工单管理",
  showActions = true,
  onViewAll,
  onViewTicket,
  className = '',
}) => {
  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case "high":
        return "red";
      case "medium":
        return "orange";
      case "low":
        return "blue";
      default:
        return "default";
    }
  };

  const getPriorityText = (priority: string) => {
    switch (priority) {
      case "high":
        return "高";
      case "medium":
        return "中";
      case "low":
        return "低";
      default:
        return priority;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case "processing":
        return "processing";
      case "pending":
        return "warning";
      case "resolved":
        return "success";
      default:
        return "default";
    }
  };

  const getStatusText = (status: string) => {
    switch (status) {
      case "processing":
        return "处理中";
      case "pending":
        return "待处理";
      case "resolved":
        return "已解决";
      default:
        return status;
    }
  };

  const pendingCount = tickets.filter((t) => t.status === "pending").length;

  return (
    <Card
      title={
        <div className="flex items-center justify-between">
          <span className="font-semibold text-gray-800">{title}</span>
          <div className="flex items-center space-x-2">
            <Badge count={pendingCount} />
            {onViewAll && (
              <Button type="link" size="small" onClick={onViewAll}>
                查看全部
              </Button>
            )}
          </div>
        </div>
      }
      className={`shadow-sm border-0 ${className}`}
      extra={
        showActions ? (
          <Dropdown
            overlay={
              <Menu>
                <Menu.Item key="1">按优先级排序</Menu.Item>
                <Menu.Item key="2">按状态排序</Menu.Item>
                <Menu.Item key="3">按时间排序</Menu.Item>
              </Menu>
            }
          >
            <Button type="text" icon={<MoreHorizontal size={16} />} />
          </Dropdown>
        ) : null
      }
    >
      <List
        dataSource={tickets}
        renderItem={(item) => (
          <List.Item
            className="hover:bg-gray-50 transition-colors duration-200 rounded-lg px-3 py-2 cursor-pointer"
            onClick={() => onViewTicket?.(item.id)}
            actions={[
              <Button
                key="view"
                type="text"
                icon={<Eye size={14} />}
                size="small"
                className="text-blue-500 hover:text-blue-600"
                onClick={(e) => {
                  e.stopPropagation();
                  onViewTicket?.(item.id);
                }}
              />,
            ]}
          >
            <List.Item.Meta
              avatar={
                <Avatar
                  size="small"
                  className="bg-gradient-to-r from-blue-500 to-purple-600"
                >
                  {item.id.split('-')[2]}
                </Avatar>
              }
              title={
                <div className="flex items-center space-x-2">
                  <Tooltip title={item.title}>
                    <span className="font-medium text-gray-800 text-sm truncate" style={{ maxWidth: '200px' }}>
                      {item.title}
                    </span>
                  </Tooltip>
                  <Tag
                    color={getPriorityColor(item.priority)}
                  >
                    {getPriorityText(item.priority)}
                  </Tag>
                </div>
              }
              description={
                <div className="space-y-1">
                  <div className="flex items-center space-x-4 text-xs text-gray-500">
                    <Space size={4}>
                      <User size={12} />
                      <span>{item.assignee}</span>
                    </Space>
                    <Space size={4}>
                      <Clock size={12} />
                      <span>{item.time}</span>
                    </Space>
                  </div>
                  <div className="flex items-center space-x-2">
                    <Tag
                      color={getStatusColor(item.status)}
                    >
                      {getStatusText(item.status)}
                    </Tag>
                    <span className="text-xs text-gray-400">
                      SLA: {item.sla}
                    </span>
                  </div>
                </div>
              }
            />
          </List.Item>
        )}
      />
    </Card>
  );
};

export default TicketListCard;