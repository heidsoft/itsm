'use client';

import React, { useEffect, useRef, useState } from 'react';
import { Tooltip } from 'antd';
import { Zap, AlertCircle, Info, ChevronsDown, Clock, Volume2, VolumeX } from 'lucide-react';

const priorityConfig = {
  P1: {
    color: 'red',
    icon: Zap,
    label: 'P1 紧急',
    borderColor: 'border-red-500',
    bgColor: 'bg-red-100',
    textColor: 'text-red-800',
    iconColor: 'text-red-600',
  },
  P2: {
    color: 'orange',
    icon: AlertCircle,
    label: 'P2 高',
    borderColor: 'border-orange-500',
    bgColor: 'bg-orange-100',
    textColor: 'text-orange-800',
    iconColor: 'text-orange-600',
  },
  P3: {
    color: 'yellow',
    icon: Info,
    label: 'P3 中',
    borderColor: 'border-yellow-500',
    bgColor: 'bg-yellow-100',
    textColor: 'text-yellow-800',
    iconColor: 'text-yellow-600',
  },
  P4: {
    color: 'blue',
    icon: ChevronsDown,
    label: 'P4 低',
    borderColor: 'border-blue-500',
    bgColor: 'bg-blue-100',
    textColor: 'text-blue-800',
    iconColor: 'text-blue-600',
  },
};

interface TicketCardProps {
  id: string | number;
  title: string;
  status: string;
  priority: 'P1' | 'P2' | 'P3' | 'P4';
  lastUpdate: string;
  type?: string;
  playSound?: boolean;
  onClick?: () => void;
}

export const TicketCard: React.FC<TicketCardProps> = ({
  id,
  title,
  status,
  priority,
  lastUpdate,
  type = '事件',
  playSound = false,
  onClick,
}) => {
  const config = priorityConfig[priority] || priorityConfig.P4;
  const Icon = config.icon;
  const cardRef = useRef<HTMLDivElement>(null);
  const [soundEnabled, setSoundEnabled] = useState(false);
  const [prefersReducedMotion, setPrefersReducedMotion] = useState(false);

  // Detect reduced motion preference
  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)');
    setPrefersReducedMotion(mediaQuery.matches);
    const handler = (e: MediaQueryListEvent) => setPrefersReducedMotion(e.matches);
    mediaQuery.addEventListener('change', handler);
    return () => mediaQuery.removeEventListener('change', handler);
  }, []);

  // Handle pulse animation with reduced motion consideration
  useEffect(() => {
    if (priority === 'P1' && !prefersReducedMotion) {
      cardRef.current?.classList.add('animate-pulse-strong');
    } else {
      cardRef.current?.classList.remove('animate-pulse-strong');
    }
  }, [priority, prefersReducedMotion]);

  // Keyboard handler
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      onClick?.();
    }
  };

  // Sound toggle for P1
  const toggleSound = (e: React.MouseEvent) => {
    e.stopPropagation();
    setSoundEnabled(!soundEnabled);
    if (!soundEnabled) {
      const audio = new Audio('/alert.mp3');
      audio.play().catch(() => {});
    }
  };

  return (
    <div
      ref={cardRef}
      role="button"
      tabIndex={0}
      aria-label={`工单 ${id}: ${title}, 状态: ${status}, 优先级: ${config.label}`}
      onClick={onClick}
      onKeyDown={handleKeyDown}
      className={`relative flex flex-col justify-between rounded-lg border-l-4 bg-white p-6 shadow-md transition-all duration-300 ease-in-out hover:shadow-xl hover:-translate-y-1 cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 ${config.borderColor}`}
    >
      {/* 优先级徽章 */}
      <Tooltip title={`优先级: ${config.label}`}>
        <div
          className={`absolute top-3 right-3 rounded-full px-3 py-1 text-xs font-bold ${config.bgColor} ${config.textColor}`}
        >
          {config.label}
        </div>
      </Tooltip>

      {/* P1 声音开关 */}
      {priority === 'P1' && (
        <Tooltip title={soundEnabled ? '关闭声音提醒' : '开启声音提醒'}>
          <button
            type="button"
            onClick={toggleSound}
            aria-label={soundEnabled ? '关闭声音提醒' : '开启声音提醒'}
            className="absolute top-3 left-3 p-1.5 rounded-full bg-white shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary-500"
          >
            {soundEnabled ? (
              <Volume2 className="h-4 w-4 text-red-500" />
            ) : (
              <VolumeX className="h-4 w-4 text-gray-400" />
            )}
          </button>
        </Tooltip>
      )}

      <h3 className="mb-2 text-lg font-semibold text-gray-800 truncate pt-6" title={title}>
        {title}
      </h3>
      <p className="mb-4 text-sm text-gray-600">
        {type}ID: {id}
      </p>

      <div className="flex items-center justify-between text-sm text-gray-500">
        <div className="flex items-center">
          <Icon className={`mr-2 h-4 w-4 ${config.iconColor}`} />
          <span>状态: {status}</span>
        </div>
        <div className="flex items-center">
          <Clock className="mr-1 h-4 w-4" />
          <span>{lastUpdate}</span>
        </div>
      </div>
    </div>
  );
};
