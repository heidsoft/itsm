'use client';

import { useState, useEffect, useRef } from 'react';
import { useIsTouchDevice } from '@/hooks/useResponsive';

export interface TouchGestureConfig {
  /** 最小滑动距离 */
  minSwipeDistance?: number;
  /** 最大滑动时间(ms) */
  maxSwipeTime?: number;
  /** 点击区域最小大小(px) */
  minTouchArea?: number;
  /** 触摸反馈时间(ms) */
  touchFeedbackDuration?: number;
}

export interface TouchHandlers {
  /** 点击事件 */
  onTap?: (event: TouchEvent | MouseEvent) => void;
  /** 长按事件 */
  onLongPress?: (event: TouchEvent | MouseEvent) => void;
  /** 滑动开始 */
  onSwipeStart?: (direction: 'left' | 'right' | 'up' | 'down') => void;
  /** 滑动中 */
  onSwipeMove?: (direction: 'left' | 'right' | 'up' | 'down', distance: number) => void;
  /** 滑动结束 */
  onSwipeEnd?: (direction: 'left' | 'right' | 'up' | 'down', velocity: number) => void;
  /** 缩放事件 */
  onPinch?: (scale: number) => void;
}

export interface TouchFeedbackProps {
  children: React.ReactNode;
  /** 启用触摸反馈 */
  enableFeedback?: boolean;
  /** 反馈样式类名 */
  feedbackClass?: string;
  /** 是否禁用默认行为 */
  preventDefault?: boolean;
  /** 手势配置 */
  config?: TouchGestureConfig;
  /** 手势处理器 */
  handlers?: TouchHandlers;
  /** 自定义类名 */
  className?: string;
  /** 触摸区域大小 */
  touchArea?: { width: number; height: number };
}

/**
 * 移动端手势和触摸反馈组件
 * 优化移动端交互体验，提供手势识别和触摸反馈
 */
export const TouchFeedback: React.FC<TouchFeedbackProps> = ({
  children,
  enableFeedback = true,
  feedbackClass = 'active:scale-95 active:bg-blue-50',
  preventDefault = false,
  config = {},
  handlers = {},
  className = '',
  touchArea
}) => {
  const isTouchDevice = useIsTouchDevice();
  const elementRef = useRef<HTMLDivElement>(null);
  const [isPressed, setIsPressed] = useState(false);
  const [isLongPressing, setIsLongPressing] = useState(false);
  
  // 默认配置
  const {
    minSwipeDistance = 50,
    maxSwipeTime = 300,
    minTouchArea = 44,
    touchFeedbackDuration = 150
  } = config;

  // 手势状态
  const touchStateRef = useRef({
    startTime: 0,
    startX: 0,
    startY: 0,
    lastX: 0,
    lastY: 0,
    isSwiping: false,
    isPinching: false,
    initialDistance: 0,
    longPressTimer: null as NodeJS.Timeout | null
  });

  // 检测滑动方向
  const getSwipeDirection = (deltaX: number, deltaY: number): 'left' | 'right' | 'up' | 'down' => {
    const absDeltaX = Math.abs(deltaX);
    const absDeltaY = Math.abs(deltaY);
    
    if (absDeltaX > absDeltaY) {
      return deltaX > 0 ? 'right' : 'left';
    } else {
      return deltaY > 0 ? 'down' : 'up';
    }
  };

  // 计算两点距离
  const getDistance = (touches: TouchList | React.TouchList): number => {
    if (touches.length < 2) return 0;
    
    const touch1 = touches[0];
    const touch2 = touches[1];
    const dx = touch2.clientX - touch1.clientX;
    const dy = touch2.clientY - touch1.clientY;
    
    return Math.sqrt(dx * dx + dy * dy);
  };

  // 触摸开始
  const handleTouchStart = (e: React.TouchEvent) => {
    if (!isTouchDevice) return;
    
    const touch = e.touches[0];
    const state = touchStateRef.current;
    const now = Date.now();
    
    state.startTime = now;
    state.startX = touch.clientX;
    state.startY = touch.clientY;
    state.lastX = touch.clientX;
    state.lastY = touch.clientY;
    state.isSwiping = false;
    state.isPinching = false;
    
    // 长按检测
    if (handlers.onLongPress) {
      state.longPressTimer = setTimeout(() => {
        setIsLongPressing(true);
        handlers.onLongPress?.(e.nativeEvent);
      }, 500);
    }
    
    setIsPressed(true);
    
    if (preventDefault) {
      e.preventDefault();
    }
  };

  // 触摸移动
  const handleTouchMove = (e: React.TouchEvent) => {
    if (!isTouchDevice) return;
    
    const touches = e.touches;
    const state = touchStateRef.current;
    const now = Date.now();
    
    // 多指手势 - 缩放
    if (touches.length === 2 && handlers.onPinch) {
      e.preventDefault();
      
      const distance = getDistance(touches);
      
      if (!state.isPinching) {
        state.isPinching = true;
        state.initialDistance = distance;
      } else {
        const scale = distance / state.initialDistance;
        handlers.onPinch(scale);
      }
      return;
    }
    
    // 单指手势 - 滑动
    if (touches.length === 1) {
      const touch = touches[0];
      const deltaX = touch.clientX - state.startX;
      const deltaY = touch.clientY - state.startY;
      const deltaTime = now - state.startTime;
      
      // 清除长按定时器
      if (state.longPressTimer) {
        clearTimeout(state.longPressTimer);
        state.longPressTimer = null;
      }
      
      // 检测是否为滑动
      const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
      if (distance > 10 && !state.isSwiping) {
        state.isSwiping = true;
        setIsLongPressing(false);
        
        const direction = getSwipeDirection(deltaX, deltaY);
        handlers.onSwipeStart?.(direction);
      }
      
      // 滑动中
      if (state.isSwiping && handlers.onSwipeMove) {
        const direction = getSwipeDirection(deltaX, deltaY);
        handlers.onSwipeMove(direction, distance);
      }
      
      state.lastX = touch.clientX;
      state.lastY = touch.clientY;
    }
  };

  // 触摸结束
  const handleTouchEnd = (e: React.TouchEvent) => {
    if (!isTouchDevice) return;
    
    const state = touchStateRef.current;
    const now = Date.now();
    const deltaTime = now - state.startTime;
    
    // 清除长按定时器
    if (state.longPressTimer) {
      clearTimeout(state.longPressTimer);
      state.longPressTimer = null;
    }
    
    const deltaX = state.lastX - state.startX;
    const deltaY = state.lastY - state.startY;
    const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
    
    // 滑动结束
    if (state.isSwiping && handlers.onSwipeEnd && deltaTime <= maxSwipeTime) {
      const direction = getSwipeDirection(deltaX, deltaY);
      const velocity = distance / deltaTime * 1000; // px/s
      handlers.onSwipeEnd(direction, velocity);
    }
    // 点击
    else if (distance < 10 && deltaTime < 300 && handlers.onTap && !isLongPressing) {
      handlers.onTap(e.nativeEvent);
    }
    
    // 重置状态
    setIsPressed(false);
    setIsLongPressing(false);
    state.isSwiping = false;
    state.isPinching = false;
  };

  // 鼠标事件（桌面端兼容）
  const handleMouseDown = (e: React.MouseEvent) => {
    if (isTouchDevice) return;
    
    const state = touchStateRef.current;
    state.startTime = Date.now();
    state.startX = e.clientX;
    state.startY = e.clientY;
    
    setIsPressed(true);
  };

  const handleMouseUp = (e: React.MouseEvent) => {
    if (isTouchDevice) return;
    
    const state = touchStateRef.current;
    const deltaTime = Date.now() - state.startTime;
    const deltaX = e.clientX - state.startX;
    const deltaY = e.clientY - state.startY;
    const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
    
    if (distance < 10 && deltaTime < 300 && handlers.onTap) {
      handlers.onTap(e.nativeEvent);
    }
    
    setIsPressed(false);
  };

  useEffect(() => {
    return () => {
      if (touchStateRef.current.longPressTimer) {
        clearTimeout(touchStateRef.current.longPressTimer);
      }
    };
  }, []);

  // 构建样式
  const touchAreaStyle = touchArea ? {
    minWidth: `${Math.max(touchArea.width, minTouchArea)}px`,
    minHeight: `${Math.max(touchArea.height, minTouchArea)}px`,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center'
  } : {};

  const computedClassName = `
    ${className}
    ${enableFeedback && isPressed ? feedbackClass : ''}
    ${isLongPressing ? 'bg-gray-100 scale-105' : ''}
    ${isTouchDevice ? 'touch-manipulation' : ''}
  `.trim().replace(/\s+/g, ' ');

  return (
    <div
      ref={elementRef}
      className={computedClassName}
      style={touchAreaStyle}
      onTouchStart={handleTouchStart}
      onTouchMove={handleTouchMove}
      onTouchEnd={handleTouchEnd}
      onMouseDown={handleMouseDown}
      onMouseUp={handleMouseUp}
      role="button"
      tabIndex={0}
    >
      {children}
    </div>
  );
};

export default TouchFeedback;
