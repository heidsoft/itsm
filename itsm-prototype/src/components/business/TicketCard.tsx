"use client";

import React, { useEffect, useRef } from "react";
import {
  Zap,
  AlertCircle,
  Info,
  ChevronsDown,
  Clock,
} from "lucide-react";

const priorityConfig = {
  P1: {
    color: "red",
    icon: Zap,
    label: "P1 紧急",
    borderColor: "border-red-500",
    bgColor: "bg-red-100",
    textColor: "text-red-800",
    iconColor: "text-red-600",
  },
  P2: {
    color: "orange",
    icon: AlertCircle,
    label: "P2 高",
    borderColor: "border-orange-500",
    bgColor: "bg-orange-100",
    textColor: "text-orange-800",
    iconColor: "text-orange-600",
  },
  P3: {
    color: "yellow",
    icon: Info,
    label: "P3 中",
    borderColor: "border-yellow-500",
    bgColor: "bg-yellow-100",
    textColor: "text-yellow-800",
    iconColor: "text-yellow-600",
  },
  P4: {
    color: "blue",
    icon: ChevronsDown,
    label: "P4 低",
    borderColor: "border-blue-500",
    bgColor: "bg-blue-100",
    textColor: "text-blue-800",
    iconColor: "text-blue-600",
  },
};

const playAlertSound = () => {
  const audio = new Audio("/alert.mp3"); // 确保public目录下有alert.mp3文件
  audio.play().catch((e) => console.error("Error playing sound:", e));
};

interface TicketCardProps {
  id: string | number;
  title: string;
  status: string;
  priority: "P1" | "P2" | "P3" | "P4";
  lastUpdate: string;
  type?: string;
  playSound?: boolean;
}

export const TicketCard: React.FC<TicketCardProps> = ({
  id,
  title,
  status,
  priority,
  lastUpdate,
  type = "事件",
  playSound = false,
}) => {
  const config = priorityConfig[priority] || priorityConfig.P4; // 默认P4
  const Icon = config.icon;

  const cardRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (priority === "P1") {
      if (playSound) {
        playAlertSound();
      }
      cardRef.current?.classList.add("animate-pulse-strong"); // 自定义动画类
    } else {
      cardRef.current?.classList.remove("animate-pulse-strong");
    }
  }, [priority, playSound]);

  return (
    <div
      ref={cardRef}
      className={`relative flex flex-col justify-between rounded-lg border-l-4 bg-white p-6 shadow-md transition-shadow duration-300 ease-in-out hover:shadow-xl ${config.borderColor}`}
    >
      {/* 优先级徽章 */}
      <div
        className={`absolute top-3 right-3 rounded-full px-3 py-1 text-xs font-bold ${config.bgColor} ${config.textColor}`}
      >
        {config.label}
      </div>

      <h3 className="mb-2 text-lg font-semibold text-gray-800">{title}</h3>
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
