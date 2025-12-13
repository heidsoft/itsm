"use client";

import React from 'react';
import { Card, Alert, List, Badge, Button } from 'antd';
import { AlertTriangle, Bell, X } from 'lucide-react';
import { SystemAlert } from '@/hooks/useDashboardData';

interface SystemAlertCardProps {
  alerts: SystemAlert[];
  title?: string;
  onDismiss?: (alertId: string) => void;
  onViewAll?: () => void;
  className?: string;
}

export const SystemAlertCard: React.FC<SystemAlertCardProps> = ({
  alerts,
  title = "系统警报",
  onDismiss,
  onViewAll,
  className = '',
}) => {
  const getSeverityColor = (type: string) => {
    switch (type) {
      case "error":
        return "error";
      case "warning":
        return "warning";
      case "info":
        return "info";
      case "success":
        return "success";
      default:
        return "info";
    }
  };

  const getSeverityText = (severity: string) => {
    switch (severity) {
      case "high":
        return "高";
      case "medium":
        return "中";
      case "low":
        return "低";
      default:
        return severity;
    }
  };

  const highCount = alerts.filter((alert) => alert.severity === "high").length;
  const mediumCount = alerts.filter((alert) => alert.severity === "medium").length;

  return (
    <Card
      title={
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <Bell size={16} className="text-orange-500" />
            <span className="font-semibold text-gray-800">{title}</span>
          </div>
          <div className="flex items-center space-x-2">
            {highCount > 0 && (
              <Badge count={highCount} className="bg-red-500" />
            )}
            {mediumCount > 0 && (
              <Badge count={mediumCount} className="bg-orange-500" />
            )}
            {onViewAll && (
              <Button type="link" size="small" onClick={onViewAll}>
                查看全部
              </Button>
            )}
          </div>
        </div>
      }
      className={`shadow-sm border-0 ${className}`}
    >
      <div className="space-y-3">
        {alerts.length === 0 ? (
          <div className="text-center py-8 text-gray-500">
            <AlertTriangle size={32} className="mx-auto mb-2 text-gray-300" />
            <p>暂无系统警报</p>
          </div>
        ) : (
          <List
            dataSource={alerts}
            renderItem={(alert) => (
              <List.Item className="border-0 px-0">
                <Alert
                  message={
                    <div className="flex items-center justify-between">
                      <div>
                        <span className="font-medium">{alert.message}</span>
                        <div className="text-xs text-gray-500 mt-1">
                          {alert.time} • {getSeverityText(alert.severity)}
                        </div>
                      </div>
                      {onDismiss && (
                        <Button
                          type="text"
                          size="small"
                          icon={<X size={14} />}
                          onClick={() => onDismiss(alert.message)}
                          className="text-gray-400 hover:text-gray-600"
                        />
                      )}
                    </div>
                  }
                  description={`严重程度: ${getSeverityText(alert.severity)}`}
                  type={getSeverityColor(alert.type)}
                  showIcon
                  className="mb-2"
                />
              </List.Item>
            )}
          />
        )}
      </div>
    </Card>
  );
};

export default SystemAlertCard;