"use client";

import React from 'react';
import { Card, Statistic } from 'antd';
import { TrendingUp, TrendingDown, LucideIcon } from 'lucide-react';

interface KPICardProps {
  title: string;
  value: number;
  change: number;
  trend: 'up' | 'down';
  suffix?: string;
  icon: LucideIcon;
  gradient: string;
  className?: string;
}

export const KPICard: React.FC<KPICardProps> = ({
  title,
  value,
  change,
  trend,
  suffix,
  icon: Icon,
  gradient,
  className = '',
}) => {
  return (
    <Card className={`text-center hover:shadow-2xl hover:scale-105 transition-all duration-300 border-0 ${gradient} text-white overflow-hidden relative ${className}`}>
      <div className="absolute top-0 right-0 w-20 h-20 bg-white/10 rounded-full -mr-10 -mt-10"></div>
      <div className="absolute bottom-0 left-0 w-16 h-16 bg-white/5 rounded-full -ml-8 -mb-8"></div>
      <div className="relative z-10">
        <div className="flex items-center justify-center mb-4">
          <div className="w-16 h-16 bg-white/20 backdrop-blur-sm rounded-2xl flex items-center justify-center shadow-lg">
            <Icon className="w-8 h-8 text-white" />
          </div>
        </div>
        <Statistic
          title={
            <span className="text-white/80 font-medium text-sm">
              {title}
            </span>
          }
          value={value}
          suffix={suffix}
          valueStyle={{
            color: "white",
            fontSize: "32px",
            fontWeight: "700",
          }}
        />
        <div className="mt-3 flex items-center justify-center space-x-2 bg-white/10 rounded-full px-3 py-1">
          {trend === "up" ? (
            <TrendingUp size={14} className="text-green-300" />
          ) : (
            <TrendingDown size={14} className="text-red-300" />
          )}
          <span
            className={`text-sm font-medium ${
              trend === "up" ? "text-green-200" : "text-red-200"
            }`}
          >
            {change > 0 ? '+' : ''}{change}% 较上月
          </span>
        </div>
      </div>
    </Card>
  );
};

export default KPICard;