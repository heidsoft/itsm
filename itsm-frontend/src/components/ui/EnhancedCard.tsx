'use client';

import React, { forwardRef } from 'react';
import { Card as AntdCard, Typography, Space, Skeleton, Badge } from 'antd';
import { motion } from 'framer-motion';
import clsx from 'clsx';

const { Title, Text } = Typography;

export interface EnhancedCardProps extends Omit<React.HTMLAttributes<HTMLDivElement>, 'title'> {
  // 基础属性
  title?: React.ReactNode;
  subtitle?: React.ReactNode;
  description?: React.ReactNode;
  extra?: React.ReactNode;
  
  // 内容
  children?: React.ReactNode;
  
  // 样式变体
  variant?: 'default' | 'outlined' | 'filled' | 'elevated' | 'glass';
  size?: 'small' | 'medium' | 'large';
  
  // 状态
  loading?: boolean;
  disabled?: boolean;
  selected?: boolean;
  
  // 交互
  hoverable?: boolean;
  clickable?: boolean;
  onClick?: () => void;
  onDoubleClick?: () => void;
  
  // 视觉增强
  bordered?: boolean;
  shadow?: boolean | 'hover' | 'always';
  rounded?: boolean | 'small' | 'medium' | 'large';
  gradient?: boolean;
  
  // 状态指示
  status?: 'success' | 'warning' | 'error' | 'info';
  badge?: {
    count: number;
    color?: string;
    offset?: [number, number];
  };
  
  // 动画
  animation?: 'none' | 'fadeIn' | 'slideUp' | 'scaleIn' | 'bounceIn';
  animationDelay?: number;
  
  // 布局
  bodyStyle?: React.CSSProperties;
  headStyle?: React.CSSProperties;
  cover?: React.ReactNode;
  actions?: React.ReactNode[];
  
  // 可访问性
  role?: string;
  'aria-label'?: string;
  'aria-describedby'?: string;
  
  // 自定义样式
  className?: string;
  style?: React.CSSProperties;
}

// 动画变体
const animationVariants = {
  none: {},
  fadeIn: {
    initial: { opacity: 0 },
    animate: { opacity: 1 },
    exit: { opacity: 0 },
  },
  slideUp: {
    initial: { opacity: 0, y: 20 },
    animate: { opacity: 1, y: 0 },
    exit: { opacity: 0, y: -20 },
  },
  scaleIn: {
    initial: { opacity: 0, scale: 0.9 },
    animate: { opacity: 1, scale: 1 },
    exit: { opacity: 0, scale: 0.9 },
  },
  bounceIn: {
    initial: { opacity: 0, scale: 0.8 },
    animate: { 
      opacity: 1, 
      scale: 1,
      transition: {
        type: 'spring',
        damping: 20,
        stiffness: 300,
      }
    },
    exit: { opacity: 0, scale: 0.8 },
  },
};

export const EnhancedCard = forwardRef<HTMLDivElement, EnhancedCardProps>(({
  title,
  subtitle,
  description,
  extra,
  children,
  variant = 'default',
  size = 'medium',
  loading = false,
  disabled = false,
  selected = false,
  hoverable = false,
  clickable = false,
  onClick,
  onDoubleClick,
  bordered = true,
  shadow = false,
  rounded = true,
  gradient = false,
  status,
  badge,
  animation = 'fadeIn',
  animationDelay = 0,
  bodyStyle,
  headStyle,
  cover,
  actions,
  role,
  'aria-label': ariaLabel,
  'aria-describedby': ariaDescribedBy,
  className,
  style,
  ...motionProps
}, ref) => {
  // 计算CSS类名
  const cardClasses = clsx(
    'enhanced-card',
    `enhanced-card--${variant}`,
    `enhanced-card--${size}`,
    {
      'enhanced-card--disabled': disabled,
      'enhanced-card--selected': selected,
      'enhanced-card--hoverable': hoverable || clickable,
      'enhanced-card--clickable': clickable,
      'enhanced-card--with-shadow': shadow === true || shadow === 'always',
      'enhanced-card--shadow-on-hover': shadow === 'hover',
      'enhanced-card--rounded': rounded === true,
      'enhanced-card--rounded-small': rounded === 'small',
      'enhanced-card--rounded-medium': rounded === 'medium',
      'enhanced-card--rounded-large': rounded === 'large',
      'enhanced-card--gradient': gradient,
      [`enhanced-card--status-${status}`]: status,
    },
    className
  );

  // 动画配置
  const animationConfig = animationVariants[animation as keyof typeof animationVariants];
  
  // 渲染标题区域
  const renderHeader = () => {
    if (!title && !subtitle && !extra) return null;

    return (
      <div className="enhanced-card__header" style={headStyle}>
        <div className="enhanced-card__header-content">
          {title && (
            <Title 
              level={size === 'large' ? 3 : size === 'medium' ? 4 : 5}
              className="enhanced-card__title"
            >
              {title}
            </Title>
          )}
          {subtitle && (
            <Text type="secondary" className="enhanced-card__subtitle">
              {subtitle}
            </Text>
          )}
          {description && (
            <Text className="enhanced-card__description">
              {description}
            </Text>
          )}
        </div>
        {extra && (
          <div className="enhanced-card__extra">
            {extra}
          </div>
        )}
      </div>
    );
  };

  // 渲染内容
  const renderContent = () => {
    if (loading) {
      return (
        <div className="enhanced-card__loading">
          <Skeleton
            active
            title={!!title}
            paragraph={{ rows: 3 }}
          />
        </div>
      );
    }

    return children;
  };

  // 渲染操作区域
  const renderActions = () => {
    if (!actions || actions.length === 0) return null;

    return (
      <div className="enhanced-card__actions">
        <Space size="small">
          {actions.map((action: React.ReactNode, index: number) => (
            <div key={index} className="enhanced-card__action">
              {action}
            </div>
          ))}
        </Space>
      </div>
    );
  };

  const cardContent = (
    <div
      ref={ref}
      className={cardClasses}
      style={style}
      role={role}
      aria-label={ariaLabel}
      aria-describedby={ariaDescribedBy}
      aria-disabled={disabled}
      aria-selected={selected}
      onClick={clickable && !disabled ? onClick : undefined}
      onDoubleClick={clickable && !disabled ? onDoubleClick : undefined}
    >
      {badge && (
        <Badge
          count={badge.count}
          color={badge.color}
          offset={badge.offset || [10, 10]}
          className="enhanced-card__badge"
        />
      )}
      
      {cover && (
        <div className="enhanced-card__cover">
          {cover}
        </div>
      )}
      
      {renderHeader()}
      
      <div className="enhanced-card__body" style={bodyStyle}>
        {renderContent()}
      </div>
      
      {renderActions()}
    </div>
  );

  if (animation === 'none') {
    return cardContent;
  }

  return (
    <motion.div
      {...animationConfig}
      transition={{
        duration: 0.3,
        delay: animationDelay,
        ease: 'easeOut',
      }}
      {...motionProps}
    >
      {cardContent}
    </motion.div>
  );
});

EnhancedCard.displayName = 'EnhancedCard';

// 预设卡片组件
export const StatsCard: React.FC<{
  title: string;
  value: string | number;
  change?: {
    value: number;
    type: 'increase' | 'decrease';
  };
  icon?: React.ReactNode;
  color?: string;
  loading?: boolean;
}> = ({ title, value, change, icon, color = '#1890ff', loading }) => (
  <EnhancedCard
    variant="filled"
    size="medium"
    hoverable
    shadow="hover"
    loading={loading}
    className="stats-card"
    gradient
  >
    <div className="flex items-center justify-between">
      <div className="flex-1">
        <Text type="secondary" className="text-sm">
          {title}
        </Text>
        <div className="mt-2">
          <Title level={2} className="!mb-0" style={{ color }}>
            {value}
          </Title>
          {change && (
            <Text
              type={change.type === 'increase' ? 'success' : 'danger'}
              className="text-sm"
            >
              {change.type === 'increase' ? '↗' : '↘'} {Math.abs(change.value)}%
            </Text>
          )}
        </div>
      </div>
      {icon && (
        <div 
          className="text-3xl opacity-60"
          style={{ color }}
        >
          {icon}
        </div>
      )}
    </div>
  </EnhancedCard>
);

export const InfoCard: React.FC<{
  title: string;
  description?: string;
  items: Array<{
    label: string;
    value: React.ReactNode;
    icon?: React.ReactNode;
  }>;
  extra?: React.ReactNode;
}> = ({ title, description, items, extra }) => (
  <EnhancedCard
    title={title}
    description={description}
    extra={extra}
    variant="outlined"
    size="medium"
    hoverable
    shadow="hover"
  >
    <Space direction="vertical" size="middle" className="w-full">
      {items.map((item, index) => (
        <div key={index} className="flex items-center justify-between">
          <div className="flex items-center space-x-2">
            {item.icon}
            <Text>{item.label}</Text>
          </div>
          <div>{item.value}</div>
        </div>
      ))}
    </Space>
  </EnhancedCard>
);

export default EnhancedCard;

// CSS样式
export const enhancedCardStyles = `
.enhanced-card {
  position: relative;
  background: #fff;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
}

.enhanced-card--default {
  border: 1px solid #f0f0f0;
}

.enhanced-card--outlined {
  border: 2px solid #d9d9d9;
  background: transparent;
}

.enhanced-card--filled {
  background: linear-gradient(135deg, #f6f8fa 0%, #ffffff 100%);
  border: 1px solid #e8e8e8;
}

.enhanced-card--elevated {
  border: none;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.enhanced-card--glass {
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.enhanced-card--small {
  padding: 12px;
}

.enhanced-card--medium {
  padding: 16px;
}

.enhanced-card--large {
  padding: 24px;
}

.enhanced-card--hoverable:hover {
  transform: translateY(-2px);
}

.enhanced-card--clickable {
  cursor: pointer;
}

.enhanced-card--clickable:active {
  transform: translateY(0);
}

.enhanced-card--disabled {
  opacity: 0.6;
  cursor: not-allowed;
  pointer-events: none;
}

.enhanced-card--selected {
  border-color: #1890ff;
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2);
}

.enhanced-card--with-shadow {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.enhanced-card--shadow-on-hover:hover {
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
}

.enhanced-card--rounded {
  border-radius: 8px;
}

.enhanced-card--rounded-small {
  border-radius: 4px;
}

.enhanced-card--rounded-medium {
  border-radius: 8px;
}

.enhanced-card--rounded-large {
  border-radius: 12px;
}

.enhanced-card--gradient {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.enhanced-card--gradient .enhanced-card__title {
  color: white !important;
}

.enhanced-card--status-success {
  border-left: 4px solid #52c41a;
}

.enhanced-card--status-warning {
  border-left: 4px solid #faad14;
}

.enhanced-card--status-error {
  border-left: 4px solid #ff4d4f;
}

.enhanced-card--status-info {
  border-left: 4px solid #1890ff;
}

.enhanced-card__header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
}

.enhanced-card__header-content {
  flex: 1;
}

.enhanced-card__title {
  margin-bottom: 4px !important;
}

.enhanced-card__subtitle {
  display: block;
  margin-bottom: 8px;
}

.enhanced-card__description {
  display: block;
  font-size: 14px;
  line-height: 1.5;
}

.enhanced-card__extra {
  margin-left: 16px;
}

.enhanced-card__body {
  flex: 1;
}

.enhanced-card__actions {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
}

.enhanced-card__cover {
  margin: -16px -16px 16px -16px;
  overflow: hidden;
}

.enhanced-card__badge {
  position: absolute;
  top: 0;
  right: 0;
  z-index: 1;
}

.enhanced-card__loading {
  padding: 16px 0;
}

.stats-card .ant-typography {
  margin-bottom: 0 !important;
}

@media (max-width: 768px) {
  .enhanced-card {
    margin-bottom: 16px;
  }
  
  .enhanced-card__header {
    flex-direction: column;
    align-items: stretch;
  }
  
  .enhanced-card__extra {
    margin-left: 0;
    margin-top: 8px;
  }
}
`;
