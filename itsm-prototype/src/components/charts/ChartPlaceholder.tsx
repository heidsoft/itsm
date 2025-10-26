import React from 'react';
import { BarChart3, PieChart, TrendingUp, Activity } from 'lucide-react';

interface ChartPlaceholderProps {
  type: 'line' | 'bar' | 'pie' | 'area';
  title: string;
  description?: string;
  height?: number;
  className?: string;
}

export const ChartPlaceholder: React.FC<ChartPlaceholderProps> = ({
  type,
  title,
  description,
  height = 300,
  className = '',
}) => {
  const getIcon = () => {
    switch (type) {
      case 'line':
        return <TrendingUp className="mx-auto mb-2 text-4xl text-blue-500" />;
      case 'bar':
        return <BarChart3 className="mx-auto mb-2 text-4xl text-green-500" />;
      case 'pie':
        return <PieChart className="mx-auto mb-2 text-4xl text-purple-500" />;
      case 'area':
        return <Activity className="mx-auto mb-2 text-4xl text-orange-500" />;
      default:
        return <BarChart3 className="mx-auto mb-2 text-4xl text-gray-500" />;
    }
  };

  const getTypeText = () => {
    switch (type) {
      case 'line':
        return '折线图';
      case 'bar':
        return '柱状图';
      case 'pie':
        return '饼图';
      case 'area':
        return '面积图';
      default:
        return '图表';
    }
  };

  return (
    <div 
      className={`flex items-center justify-center bg-gray-50 rounded border ${className}`}
      style={{ height: `${height}px` }}
    >
      <div className="text-center text-gray-500">
        {getIcon()}
        <div className="text-lg font-medium mb-1">{title}</div>
        <div className="text-sm mb-2">{getTypeText()}</div>
        {description && (
          <div className="text-xs text-gray-400">{description}</div>
        )}
        <div className="text-xs text-gray-400 mt-2">
          图表组件暂未实现，显示占位符
        </div>
      </div>
    </div>
  );
};

export default ChartPlaceholder;
