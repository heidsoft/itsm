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
  P1: { color: "red", icon: Zap, label: "P1 紧急" },
  P2: { color: "orange", icon: AlertCircle, label: "P2 高" },
  P3: { color: "yellow", icon: Info, label: "P3 中" },
  P4: { color: "blue", icon: ChevronsDown, label: "P4 低" },
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
}

export const TicketCard: React.FC<TicketCardProps> = ({
  id,
  title,
  status,
  priority,
  lastUpdate,
  type = "事件",
}) => {
  const config = priorityConfig[priority] || priorityConfig.P4; // 默认P4
  const Icon = config.icon;

  const cardRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (priority === "P1") {
      // 播放声音告警
      playAlertSound();
      // 添加闪烁动画类
      cardRef.current?.classList.add("animate-pulse-strong"); // 自定义动画类
    } else {
      cardRef.current?.classList.remove("animate-pulse-strong");
    }
  }, [priority]);

  return (
    <div
      ref={cardRef}
      className={`relative bg-white p-6 rounded-lg shadow-md border-l-4 
            border-${config.color}-500 
            hover:shadow-xl transition-shadow duration-300 ease-in-out 
            flex flex-col justify-between`}
    >
      {/* 优先级徽章 */}
      <div
        className={`absolute top-3 right-3 px-3 py-1 rounded-full text-xs font-bold 
                bg-${config.color}-100 text-${config.color}-800`}
      >
        {config.label}
      </div>

      <h3 className="text-lg font-semibold text-gray-800 mb-2">{title}</h3>
      <p className="text-sm text-gray-600 mb-4">
        {type}ID: {id}
      </p>

      <div className="flex items-center justify-between text-sm text-gray-500">
        <div className="flex items-center">
          <Icon className={`w-4 h-4 mr-2 text-${config.color}-600`} />
          <span>状态: {status}</span>
        </div>
        <div className="flex items-center">
          <Clock className="w-4 h-4 mr-1" />
          <span>{lastUpdate}</span>
        </div>
      </div>
    </div>
  );
};
