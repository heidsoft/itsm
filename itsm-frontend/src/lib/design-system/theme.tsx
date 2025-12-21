"use client";

import React, {
  createContext,
  useContext,
  useState,
  useEffect,
  ReactNode,
} from "react";
import { ConfigProvider, theme } from "antd";
import { colors, darkColors } from "@/lib/design-system/colors";
import {
  spacing,
  borderRadius,
  boxShadow,
  fontSize,
  lineHeight,
  fontWeight,
} from "@/lib/design-system/spacing";

// 主题类型
export type ThemeMode = "light" | "dark" | "system";

// 主题上下文类型
interface ThemeContextType {
  mode: ThemeMode;
  setMode: (mode: ThemeMode) => void;
  isDark: boolean;
  toggleTheme: () => void;
}

// 创建主题上下文
const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

// 主题提供者属性
interface ThemeProviderProps {
  children: ReactNode;
  defaultMode?: ThemeMode;
  storageKey?: string;
}

// 主题提供者组件
export const ThemeProvider: React.FC<ThemeProviderProps> = ({
  children,
  defaultMode = "system",
  storageKey = "itsm-theme",
}) => {
  const [mode, setMode] = useState<ThemeMode>(defaultMode);
  const [isDark, setIsDark] = useState(false);

  // 从本地存储加载主题
  useEffect(() => {
    const savedMode = localStorage.getItem(storageKey) as ThemeMode;
    if (savedMode) {
      setMode(savedMode);
    }
  }, [storageKey]);

  // 监听系统主题变化
  useEffect(() => {
    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");

    const handleChange = () => {
      if (mode === "system") {
        setIsDark(mediaQuery.matches);
      }
    };

    mediaQuery.addEventListener("change", handleChange);
    handleChange(); // 初始检查

    return () => mediaQuery.removeEventListener("change", handleChange);
  }, [mode]);

  // 更新暗色状态
  useEffect(() => {
    if (mode === "dark") {
      setIsDark(true);
    } else if (mode === "light") {
      setIsDark(false);
    } else {
      // system mode
      setIsDark(window.matchMedia("(prefers-color-scheme: dark)").matches);
    }
  }, [mode]);

  // 保存主题到本地存储
  useEffect(() => {
    localStorage.setItem(storageKey, mode);
  }, [mode, storageKey]);

  // 切换主题
  const toggleTheme = () => {
    setMode((prev) => {
      switch (prev) {
        case "light":
          return "dark";
        case "dark":
          return "system";
        case "system":
          return "light";
        default:
          return "light";
      }
    });
  };

  // 设置主题
  const handleSetMode = (newMode: ThemeMode) => {
    setMode(newMode);
  };

  const contextValue: ThemeContextType = {
    mode,
    setMode: handleSetMode,
    isDark,
    toggleTheme,
  };

  return (
    <ThemeContext.Provider value={contextValue}>
      {children}
    </ThemeContext.Provider>
  );
};

// 使用主题钩子
export const useTheme = (): ThemeContextType => {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error("useTheme must be used within a ThemeProvider");
  }
  return context;
};

// Ant Design 主题配置
export const getAntdTheme = (isDark: boolean) => {
  const colorPalette = isDark ? darkColors : colors;

  return {
    algorithm: isDark ? theme.darkAlgorithm : theme.defaultAlgorithm,
    token: {
      // 颜色配置
      colorPrimary: colorPalette.primary[500],
      colorSuccess: "#22c55e",
      colorWarning: "#f59e0b",
      colorError: "#ef4444",
      colorInfo: "#0ea5e9",

      // 背景色
      colorBgContainer: colorPalette.functional.background.primary,
      colorBgLayout: colorPalette.functional.background.secondary,
      colorBgElevated: colorPalette.functional.background.elevated,

      // 文本色
      colorText: colorPalette.functional.text.primary,
      colorTextSecondary: colorPalette.functional.text.secondary,
      colorTextTertiary: colorPalette.functional.text.tertiary,
      colorTextQuaternary: colorPalette.functional.text.disabled,

      // 边框色
      colorBorder: colorPalette.functional.border.primary,
      colorBorderSecondary: colorPalette.functional.border.secondary,

      // 圆角
      borderRadius: borderRadius.lg,
      borderRadiusLG: borderRadius.xl,
      borderRadiusSM: borderRadius.md,

      // 阴影
      boxShadow: boxShadow.md,
      boxShadowSecondary: boxShadow.lg,
      boxShadowTertiary: boxShadow.xl,

      // 字体
      fontSize: fontSize.base,
      fontSizeLG: fontSize.lg,
      fontSizeSM: fontSize.sm,
      fontSizeXL: fontSize.xl,

      // 行高
      lineHeight: lineHeight.normal,
      lineHeightLG: lineHeight.relaxed,
      lineHeightSM: lineHeight.snug,

      // 字重
      fontWeight: 400,
      fontWeightStrong: 600,

      // 间距
      padding: spacing[4],
      paddingLG: spacing[6],
      paddingSM: spacing[2],
      paddingXL: spacing[8],

      margin: spacing[4],
      marginLG: spacing[6],
      marginSM: spacing[2],
      marginXL: spacing[8],

      // 控件高度
      controlHeight: 32,
      controlHeightLG: 40,
      controlHeightSM: 24,

      // 动画
      motionDurationSlow: "0.3s",
      motionDurationMid: "0.2s",
      motionDurationFast: "0.1s",
    },
    components: {
      // 按钮组件
      Button: {
        borderRadius: borderRadius.lg,
        controlHeight: 40,
        controlHeightLG: 48,
        controlHeightSM: 32,
        fontWeight: fontWeight.medium,
      },

      // 输入框组件
      Input: {
        borderRadius: borderRadius.lg,
        controlHeight: 40,
        controlHeightLG: 48,
        controlHeightSM: 32,
      },

      // 卡片组件
      Card: {
        borderRadius: borderRadius.lg,
        boxShadow: boxShadow.md,
      },

      // 模态框组件
      Modal: {
        borderRadius: borderRadius.xl,
        boxShadow: boxShadow.xl,
      },

      // 抽屉组件
      Drawer: {
        borderRadius: borderRadius.xl,
      },

      // 消息组件
      Message: {
        borderRadius: borderRadius.lg,
      },

      // 通知组件
      Notification: {
        borderRadius: borderRadius.lg,
      },
    },
  };
};

// 主题配置组件
interface ThemeConfigProps {
  children: ReactNode;
}

export const ThemeConfig: React.FC<ThemeConfigProps> = ({ children }) => {
  const { isDark } = useTheme();
  const antdTheme = getAntdTheme(isDark);

  return <ConfigProvider theme={antdTheme}>{children}</ConfigProvider>;
};

// CSS 变量生成器
export const generateCSSVariables = (isDark: boolean) => {
  const colorPalette = isDark ? darkColors : colors;

  return {
    // 主色调
    "--color-primary-50": colorPalette.primary[50],
    "--color-primary-100": colorPalette.primary[100],
    "--color-primary-200": colorPalette.primary[200],
    "--color-primary-300": colorPalette.primary[300],
    "--color-primary-400": colorPalette.primary[400],
    "--color-primary-500": colorPalette.primary[500],
    "--color-primary-600": colorPalette.primary[600],
    "--color-primary-700": colorPalette.primary[700],
    "--color-primary-800": colorPalette.primary[800],
    "--color-primary-900": colorPalette.primary[900],

    // 中性色
    "--color-neutral-50": colorPalette.neutral[50],
    "--color-neutral-100": colorPalette.neutral[100],
    "--color-neutral-200": colorPalette.neutral[200],
    "--color-neutral-300": colorPalette.neutral[300],
    "--color-neutral-400": colorPalette.neutral[400],
    "--color-neutral-500": colorPalette.neutral[500],
    "--color-neutral-600": colorPalette.neutral[600],
    "--color-neutral-700": colorPalette.neutral[700],
    "--color-neutral-800": colorPalette.neutral[800],
    "--color-neutral-900": colorPalette.neutral[900],

    // 功能色
    "--color-background-primary": colorPalette.functional.background.primary,
    "--color-background-secondary":
      colorPalette.functional.background.secondary,
    "--color-background-tertiary": colorPalette.functional.background.tertiary,
    "--color-surface-primary": colorPalette.functional.surface.primary,
    "--color-surface-secondary": colorPalette.functional.surface.secondary,
    "--color-border-primary": colorPalette.functional.border.primary,
    "--color-border-secondary": colorPalette.functional.border.secondary,
    "--color-text-primary": colorPalette.functional.text.primary,
    "--color-text-secondary": colorPalette.functional.text.secondary,
    "--color-text-tertiary": colorPalette.functional.text.tertiary,

    // 语义色
    "--color-success": "#22c55e",
    "--color-warning": "#f59e0b",
    "--color-error": "#ef4444",
    "--color-info": "#0ea5e9",

    // 间距
    "--spacing-xs": spacing[1],
    "--spacing-sm": spacing[2],
    "--spacing-md": spacing[4],
    "--spacing-lg": spacing[6],
    "--spacing-xl": spacing[8],
    "--spacing-2xl": spacing[12],
    "--spacing-3xl": spacing[16],

    // 圆角
    "--border-radius-sm": borderRadius.sm,
    "--border-radius-md": borderRadius.md,
    "--border-radius-lg": borderRadius.lg,
    "--border-radius-xl": borderRadius.xl,

    // 阴影
    "--box-shadow-sm": boxShadow.sm,
    "--box-shadow-md": boxShadow.md,
    "--box-shadow-lg": boxShadow.lg,
    "--box-shadow-xl": boxShadow.xl,

    // 字体
    "--font-size-xs": fontSize.xs,
    "--font-size-sm": fontSize.sm,
    "--font-size-base": fontSize.base,
    "--font-size-lg": fontSize.lg,
    "--font-size-xl": fontSize.xl,
    "--font-size-2xl": fontSize["2xl"],

    // 行高
    "--line-height-tight": lineHeight.tight,
    "--line-height-normal": lineHeight.normal,
    "--line-height-relaxed": lineHeight.relaxed,

    // 字重
    "--font-weight-normal": fontWeight.normal,
    "--font-weight-medium": fontWeight.medium,
    "--font-weight-semibold": fontWeight.semibold,
    "--font-weight-bold": fontWeight.bold,
  };
};

// 应用 CSS 变量到文档
export const applyCSSVariables = (isDark: boolean) => {
  const variables = generateCSSVariables(isDark);
  const root = document.documentElement;

  Object.entries(variables).forEach(([key, value]) => {
    root.style.setProperty(key, value);
  });

  // 添加主题类名
  root.classList.toggle("dark", isDark);
  root.classList.toggle("light", !isDark);
};

export default ThemeProvider;
