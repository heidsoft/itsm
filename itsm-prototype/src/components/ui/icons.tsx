import React from 'react';
import { 
  LucideIcon,
  CheckCircle,
  AlertTriangle,
  XCircle,
  Info,
  Clock,
  ArrowDown,
  Minus,
  ArrowUp,
  Loader2,
  Dot,
  Home,
  LayoutDashboard,
  Ticket,
  Bug,
  GitBranch,
  Package,
  BarChart3,
  Settings,
  Plus,
  Edit,
  Trash2,
  Save,
  X,
  Search,
  Filter,
  ArrowUpDown,
  RefreshCw,
  User,
  Users,
  UserCircle,
  LogIn,
  LogOut,
  Bell,
  HelpCircle,
  Shield,
  Lock,
  Unlock
} from 'lucide-react';
import { cn } from '@/lib/utils';

// 图标尺寸定义
export const iconSizes = {
  xs: 'w-3 h-3',
  sm: 'w-4 h-4',
  md: 'w-5 h-5',
  lg: 'w-6 h-6',
  xl: 'w-8 h-8',
  '2xl': 'w-10 h-10',
  '3xl': 'w-12 h-12',
} as const;

// 图标颜色定义
export const iconColors = {
  primary: 'text-blue-600',
  secondary: 'text-gray-600',
  success: 'text-green-600',
  warning: 'text-yellow-600',
  error: 'text-red-600',
  info: 'text-blue-500',
  muted: 'text-gray-400',
  white: 'text-white',
  current: 'text-current',
} as const;

// 图标状态定义
export const iconStates = {
  default: '',
  hover: 'hover:text-blue-700 transition-colors duration-200',
  active: 'text-blue-700',
  disabled: 'text-gray-300 cursor-not-allowed',
  interactive: 'hover:text-blue-700 hover:scale-110 transition-all duration-200 cursor-pointer',
} as const;

export type IconSize = keyof typeof iconSizes;
export type IconColor = keyof typeof iconColors;
export type IconState = keyof typeof iconStates;

interface IconProps {
  icon: LucideIcon;
  size?: IconSize;
  color?: IconColor;
  state?: IconState;
  className?: string;
  onClick?: () => void;
  'aria-label'?: string;
}

// 基础图标组件
export const Icon: React.FC<IconProps> = ({
  icon: IconComponent,
  size = 'md',
  color = 'current',
  state = 'default',
  className,
  onClick,
  'aria-label': ariaLabel,
  ...props
}) => {
  return (
    <IconComponent
      className={cn(
        iconSizes[size],
        iconColors[color],
        iconStates[state],
        onClick && 'cursor-pointer',
        className
      )}
      onClick={onClick}
      aria-label={ariaLabel}
      {...props}
    />
  );
};

// 状态图标组件
interface StatusIconProps {
  status: 'success' | 'warning' | 'error' | 'info' | 'pending';
  size?: IconSize;
  className?: string;
}

export const StatusIcon: React.FC<StatusIconProps> = ({
  status,
  size = 'md',
  className,
}) => {
  const statusConfig = {
    success: { icon: CheckCircle, color: 'success' as IconColor },
    warning: { icon: AlertTriangle, color: 'warning' as IconColor },
    error: { icon: XCircle, color: 'error' as IconColor },
    info: { icon: Info, color: 'info' as IconColor },
    pending: { icon: Clock, color: 'muted' as IconColor },
  };

  const config = statusConfig[status];

  return (
    <Icon
      icon={config.icon}
      size={size}
      color={config.color}
      className={className}
    />
  );
};

// 优先级图标组件
interface PriorityIconProps {
  priority: 'low' | 'medium' | 'high' | 'critical';
  size?: IconSize;
  className?: string;
}

export const PriorityIcon: React.FC<PriorityIconProps> = ({
  priority,
  size = 'md',
  className,
}) => {
  const priorityConfig = {
    low: { icon: ArrowDown, color: 'success' as IconColor },
    medium: { icon: Minus, color: 'warning' as IconColor },
    high: { icon: ArrowUp, color: 'error' as IconColor },
    critical: { icon: AlertTriangle, color: 'error' as IconColor },
  };

  const config = priorityConfig[priority];

  return (
    <Icon
      icon={config.icon}
      size={size}
      color={config.color}
      className={className}
    />
  );
};

// 加载图标组件
interface LoadingIconProps {
  size?: IconSize;
  className?: string;
}

export const LoadingIcon: React.FC<LoadingIconProps> = ({
  size = 'md',
  className,
}) => {
  return (
    <Icon
      icon={Loader2}
      size={size}
      className={cn('animate-spin', className)}
    />
  );
};

// 图标按钮组件
interface IconButtonProps {
  icon: LucideIcon;
  size?: IconSize;
  variant?: 'default' | 'ghost' | 'outline';
  disabled?: boolean;
  loading?: boolean;
  onClick?: () => void;
  className?: string;
  'aria-label'?: string;
  children?: React.ReactNode;
}

export const IconButton: React.FC<IconButtonProps> = ({
  icon: IconComponent,
  size = 'md',
  variant = 'default',
  disabled = false,
  loading = false,
  onClick,
  className,
  'aria-label': ariaLabel,
  children,
}) => {
  const baseClasses = 'inline-flex items-center justify-center rounded-md transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2';
  
  const variantClasses = {
    default: 'bg-blue-600 text-white hover:bg-blue-700 active:bg-blue-800',
    ghost: 'text-gray-600 hover:text-gray-900 hover:bg-gray-100 active:bg-gray-200',
    outline: 'border border-gray-300 text-gray-700 hover:bg-gray-50 active:bg-gray-100',
  };

  const sizeClasses = {
    xs: 'p-1',
    sm: 'p-1.5',
    md: 'p-2',
    lg: 'p-2.5',
    xl: 'p-3',
    '2xl': 'p-4',
    '3xl': 'p-5',
  };

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled || loading}
      aria-label={ariaLabel}
      className={cn(
        baseClasses,
        variantClasses[variant],
        sizeClasses[size],
        disabled && 'opacity-50 cursor-not-allowed',
        loading && 'cursor-wait',
        className
      )}
    >
      {loading ? (
        <LoadingIcon size={size} />
      ) : (
        <Icon icon={IconComponent} size={size} />
      )}
      {children && <span className="ml-2">{children}</span>}
    </button>
  );
};

// 图标徽章组件
interface IconBadgeProps {
  icon: LucideIcon;
  count?: number;
  size?: IconSize;
  color?: IconColor;
  className?: string;
}

export const IconBadge: React.FC<IconBadgeProps> = ({
  icon: IconComponent,
  count,
  size = 'md',
  color = 'primary',
  className,
}) => {
  return (
    <div className={cn('relative inline-flex', className)}>
      <Icon icon={IconComponent} size={size} color={color} />
      {count !== undefined && count > 0 && (
        <span className="absolute -top-1 -right-1 inline-flex items-center justify-center px-1.5 py-0.5 text-xs font-bold leading-none text-white bg-red-600 rounded-full min-w-[1.25rem] h-5">
          {count > 99 ? '99+' : count}
        </span>
      )}
    </div>
  );
};

// 图标组合组件
interface IconGroupProps {
  icons: Array<{
    icon: LucideIcon;
    color?: IconColor;
    onClick?: () => void;
    'aria-label'?: string;
  }>;
  size?: IconSize;
  spacing?: 'tight' | 'normal' | 'loose';
  className?: string;
}

export const IconGroup: React.FC<IconGroupProps> = ({
  icons,
  size = 'md',
  spacing = 'normal',
  className,
}) => {
  const spacingClasses = {
    tight: 'space-x-1',
    normal: 'space-x-2',
    loose: 'space-x-4',
  };

  return (
    <div className={cn('flex items-center', spacingClasses[spacing], className)}>
      {icons.map((iconProps, index) => (
        <Icon
          key={index}
          {...iconProps}
          size={size}
          state={iconProps.onClick ? 'interactive' : 'default'}
        />
      ))}
    </div>
  );
};

// 图标分隔符组件
interface IconDividerProps {
  icon?: LucideIcon;
  size?: IconSize;
  className?: string;
}

export const IconDivider: React.FC<IconDividerProps> = ({
  icon,
  size = 'sm',
  className,
}) => {
  const IconComponent = icon || Dot;

  return (
    <div className={cn('flex items-center justify-center', className)}>
      <Icon icon={IconComponent} size={size} color="muted" />
    </div>
  );
};

// 导出常用图标配置
export const commonIcons = {
  // 导航图标
  home: Home,
  dashboard: LayoutDashboard,
  tickets: Ticket,
  incidents: AlertTriangle,
  problems: Bug,
  changes: GitBranch,
  assets: Package,
  reports: BarChart3,
  settings: Settings,
  
  // 操作图标
  add: Plus,
  edit: Edit,
  delete: Trash2,
  save: Save,
  cancel: X,
  search: Search,
  filter: Filter,
  sort: ArrowUpDown,
  refresh: RefreshCw,
  
  // 状态图标
  success: CheckCircle,
  warning: AlertTriangle,
  error: XCircle,
  info: Info,
  pending: Clock,
  
  // 用户图标
  user: User,
  users: Users,
  profile: UserCircle,
  login: LogIn,
  logout: LogOut,
  
  // 系统图标
  notification: Bell,
  help: HelpCircle,
  security: Shield,
  lock: Lock,
  unlock: Unlock,
};

export default Icon;