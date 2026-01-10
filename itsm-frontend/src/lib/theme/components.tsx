import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { theme, Theme, Colors } from './index';

type ThemeMode = 'light' | 'dark' | 'system';
type ColorScheme = 'light' | 'dark';

interface ThemeContextType {
  theme: Theme;
  mode: ThemeMode;
  colorScheme: ColorScheme;
  setMode: (mode: ThemeMode) => void;
  colors: Colors;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

interface ThemeProviderProps {
  children: ReactNode;
  defaultMode?: ThemeMode;
  storageKey?: string;
}

export function ThemeProvider({
  children,
  defaultMode = 'system',
  storageKey = 'itsm-theme-mode',
}: ThemeProviderProps) {
  const [mode, setModeState] = useState<ThemeMode>(defaultMode);
  const [colorScheme, setColorScheme] = useState<ColorScheme>('light');

  // Initialize theme mode from storage or system preference
  useEffect(() => {
    const stored = localStorage.getItem(storageKey) as ThemeMode;
    if (stored && ['light', 'dark', 'system'].includes(stored)) {
      setModeState(stored);
    }
  }, [storageKey]);

  // Update color scheme based on mode and system preference
  useEffect(() => {
    const updateColorScheme = () => {
      if (mode === 'system') {
        const systemPrefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
        setColorScheme(systemPrefersDark ? 'dark' : 'light');
      } else {
        setColorScheme(mode);
      }
    };

    updateColorScheme();

    if (mode === 'system') {
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
      mediaQuery.addEventListener('change', updateColorScheme);
      return () => mediaQuery.removeEventListener('change', updateColorScheme);
    }
  }, [mode]);

  // Apply theme to document
  useEffect(() => {
    const root = document.documentElement;
    
    if (colorScheme === 'dark') {
      root.classList.add('dark');
    } else {
      root.classList.remove('dark');
    }

    // Set CSS custom properties for easy access
    root.style.setProperty('--color-primary', theme.colors.primary[500]);
    root.style.setProperty('--color-primary-foreground', '#ffffff');
    root.style.setProperty('--color-background', colorScheme === 'dark' ? theme.colors.gray[900] : '#ffffff');
    root.style.setProperty('--color-foreground', colorScheme === 'dark' ? theme.colors.gray[100] : theme.colors.gray[900]);
    root.style.setProperty('--color-border', colorScheme === 'dark' ? theme.colors.gray[700] : theme.colors.gray[200]);
  }, [colorScheme]);

  const setMode = (newMode: ThemeMode) => {
    setModeState(newMode);
    localStorage.setItem(storageKey, newMode);
  };

  const value: ThemeContextType = {
    theme,
    mode,
    colorScheme,
    setMode,
    colors: theme.colors,
  };

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme() {
  const context = useContext(ThemeContext);
  if (context === undefined) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
}

// Theme-aware utility components
interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: keyof typeof theme.components.button.variant;
  size?: keyof typeof theme.components.button.size;
  children: ReactNode;
}

export function Button({
  variant = 'primary',
  size = 'md',
  className = '',
  children,
  ...props
}: ButtonProps) {
  const { theme: currentTheme } = useTheme();
  const buttonConfig = currentTheme.components.button;
  const variantStyles = buttonConfig.variant[variant];
  const sizeStyles = buttonConfig.size[size];

  const styles = {
    padding: sizeStyles.padding,
    fontSize: sizeStyles.fontSize,
    borderRadius: sizeStyles.borderRadius,
    backgroundColor: variantStyles.backgroundColor,
    color: variantStyles.color,
    border: variantStyles.border,
    transition: currentTheme.animation.transition.colors,
    fontWeight: currentTheme.typography.fontWeight.medium,
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    cursor: 'pointer',
  };

  return (
    <button
      className={`focus:outline-none focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed ${className}`}
      style={styles}
      onMouseEnter={(e) => {
        if (variantStyles.hover) {
          Object.assign(e.currentTarget.style, variantStyles.hover);
        }
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.backgroundColor = variantStyles.backgroundColor;
      }}
      onFocus={(e) => {
        if (variantStyles.focus) {
          e.currentTarget.style.boxShadow = variantStyles.focus.ring;
        }
      }}
      onBlur={(e) => {
        e.currentTarget.style.boxShadow = 'none';
      }}
      {...props}
    >
      {children}
    </button>
  );
}

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  variant?: keyof typeof theme.components.input.variant;
  inputSize?: keyof typeof theme.components.input.size;
  error?: boolean;
}

export function Input({
  variant = 'default',
  inputSize = 'md',
  error = false,
  className = '',
  ...props
}: InputProps) {
  const { theme: currentTheme } = useTheme();
  const inputConfig = currentTheme.components.input;
  const variantStyles = inputConfig.variant[error ? 'error' : variant];
  const sizeStyles = inputConfig.size[inputSize];

  const styles = {
    padding: sizeStyles.padding,
    fontSize: sizeStyles.fontSize,
    borderRadius: sizeStyles.borderRadius,
    backgroundColor: variantStyles.backgroundColor,
    border: variantStyles.border,
    color: variantStyles.color,
    transition: currentTheme.animation.transition.colors,
    width: '100%',
  };

  return (
    <input
      className={`focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed ${className}`}
      style={styles}
      onFocus={(e) => {
        if (variantStyles.focus) {
          e.currentTarget.style.borderColor = variantStyles.focus.borderColor;
          e.currentTarget.style.boxShadow = variantStyles.focus.ring;
        }
      }}
      onBlur={(e) => {
        e.currentTarget.style.borderColor = variantStyles.border.split(' ')[3]; // Extract border color
        e.currentTarget.style.boxShadow = 'none';
      }}
      {...props}
    />
  );
}

interface BadgeProps {
  variant?: keyof typeof theme.components.badge.variant;
  size?: keyof typeof theme.components.badge.size;
  children: ReactNode;
  className?: string;
}

export function Badge({
  variant = 'default',
  size = 'md',
  children,
  className = '',
}: BadgeProps) {
  const { theme: currentTheme } = useTheme();
  const badgeConfig = currentTheme.components.badge;
  const variantStyles = badgeConfig.variant[variant];
  const sizeStyles = badgeConfig.size[size];

  const styles = {
    padding: sizeStyles.padding,
    fontSize: sizeStyles.fontSize,
    borderRadius: sizeStyles.borderRadius,
    backgroundColor: variantStyles.backgroundColor,
    color: variantStyles.color,
    fontWeight: currentTheme.typography.fontWeight.medium,
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
  };

  return (
    <span className={className} style={styles}>
      {children}
    </span>
  );
}

interface StatusBadgeProps {
  status: keyof typeof theme.colors.status;
  children?: ReactNode;
  size?: keyof typeof theme.components.badge.size;
  className?: string;
}

export function StatusBadge({
  status,
  children,
  size = 'sm',
  className = '',
}: StatusBadgeProps) {
  const { theme: currentTheme } = useTheme();
  const statusColor = currentTheme.colors.status[status];
  const sizeStyles = currentTheme.components.badge.size[size];

  // Generate lighter version for background
  const backgroundColor = statusColor + '20'; // Add transparency
  const textColor = statusColor;

  const styles = {
    padding: sizeStyles.padding,
    fontSize: sizeStyles.fontSize,
    borderRadius: sizeStyles.borderRadius,
    backgroundColor,
    color: textColor,
    border: `1px solid ${statusColor}40`,
    fontWeight: currentTheme.typography.fontWeight.medium,
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
  };

  const statusLabels = {
    new: '新建',
    open: '开放',
    in_progress: '处理中',
    pending: '待处理',
    resolved: '已解决',
    closed: '已关闭',
    cancelled: '已取消',
  };

  return (
    <span className={className} style={styles}>
      {children || statusLabels[status]}
    </span>
  );
}

interface PriorityBadgeProps {
  priority: keyof typeof theme.colors.priority;
  children?: ReactNode;
  size?: keyof typeof theme.components.badge.size;
  className?: string;
}

export function PriorityBadge({
  priority,
  children,
  size = 'sm',
  className = '',
}: PriorityBadgeProps) {
  const { theme: currentTheme } = useTheme();
  const priorityColor = currentTheme.colors.priority[priority];
  const sizeStyles = currentTheme.components.badge.size[size];

  // Generate lighter version for background
  const backgroundColor = priorityColor + '20'; // Add transparency
  const textColor = priorityColor;

  const styles = {
    padding: sizeStyles.padding,
    fontSize: sizeStyles.fontSize,
    borderRadius: sizeStyles.borderRadius,
    backgroundColor,
    color: textColor,
    border: `1px solid ${priorityColor}40`,
    fontWeight: currentTheme.typography.fontWeight.medium,
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
  };

  const priorityLabels = {
    low: '低',
    normal: '普通',
    high: '高',
    urgent: '紧急',
    critical: '关键',
  };

  return (
    <span className={className} style={styles}>
      {children || priorityLabels[priority]}
    </span>
  );
}

// Theme toggle component
export function ThemeToggle({ className = '' }: { className?: string }) {
  const { mode, setMode } = useTheme();

  const modes: Array<{ value: ThemeMode; label: string; icon: string }> = [
    { value: 'light', label: '浅色主题', icon: '☀️' },
    { value: 'dark', label: '深色主题', icon: '🌙' },
    { value: 'system', label: '跟随系统', icon: '💻' },
  ];

  return (
    <div className={`inline-flex rounded-md border border-gray-300 ${className}`}>
      {modes.map((item) => (
        <button
          key={item.value}
          onClick={() => setMode(item.value)}
          className={`px-3 py-1 text-sm font-medium transition-colors first:rounded-l-md last:rounded-r-md ${
            mode === item.value
              ? 'bg-blue-600 text-white'
              : 'bg-white text-gray-700 hover:bg-gray-50'
          }`}
          title={item.label}
        >
          <span className="mr-1">{item.icon}</span>
          {item.label}
        </button>
      ))}
    </div>
  );
}

// Utility hook for responsive design
export function useBreakpoint() {
  const [breakpoint, setBreakpoint] = useState<string>('');

  useEffect(() => {
    const getBreakpoint = () => {
      const width = window.innerWidth;
      if (width >= 1536) return '2xl';
      if (width >= 1280) return 'xl';
      if (width >= 1024) return 'lg';
      if (width >= 768) return 'md';
      if (width >= 640) return 'sm';
      return 'xs';
    };

    setBreakpoint(getBreakpoint());

    const handleResize = () => setBreakpoint(getBreakpoint());
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  return {
    breakpoint,
    isMobile: breakpoint === 'xs' || breakpoint === 'sm',
    isTablet: breakpoint === 'md',
    isDesktop: breakpoint === 'lg' || breakpoint === 'xl' || breakpoint === '2xl',
  };
}
