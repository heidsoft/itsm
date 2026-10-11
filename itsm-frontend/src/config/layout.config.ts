/**
 * ITSM 布局尺寸配置。Header 高度按 48+8n、Sider 宽度按 200+8n 取值，
 * 开发环境由文件末尾的 validate* 自检。
 */

export const LAYOUT_CONFIG = {
  // Header
  header: {
    height: 64,
    padding: '0 24px',
    paddingMobile: '0 16px',
    logoHeight: 40,
    fontSize: 18,
    iconSize: 20,
    avatarSize: 32,
    shadow: '0 2px 8px rgba(0, 0, 0, 0.08)',
  },

  // Sider
  sider: {
    width: 256,
    collapsedWidth: 80,
    breakpoint: 'lg' as const,
    logoAreaHeight: 64, // 与 header.height 对齐
    logoPadding: '0 24px',
    logoPaddingCollapsed: '0 20px',
    logoIconSize: 40,
    logoTextSize: 20,
    menuPadding: '16px 8px',
    menuItemHeight: 48,
    shadow: '2px 0 8px rgba(0, 0, 0, 0.1)',
  },

  // Content
  content: {
    marginExpanded: 24,
    marginCollapsed: 16,
    marginMobile: 8,

    marginLeftExpanded: 280, // sider.width + marginExpanded
    marginLeftCollapsed: 96, // sider.collapsedWidth + marginCollapsed

    padding: 24,
    paddingCollapsed: 16,
    paddingMobile: 12,

    minHeight: 'calc(100vh - 64px)', // 扣除 Header 高度

    pageHeaderMarginBottom: 24,
    pageHeaderPaddingBottom: 16,
    pageTitleFontSize: 24,
    pageDescFontSize: 14,
  },

  // 响应式断点
  breakpoints: {
    xs: 0,
    sm: 576,
    md: 768,
    lg: 992, // Sider 收起断点，对应 sider.breakpoint
    xl: 1200,
    xxl: 1600,
  },

  // 间距系统（8px 栅格）
  spacing: {
    unit: 8,
    xs: 8,
    sm: 12,
    md: 16,
    lg: 24,
    xl: 32,
    xxl: 48,
  },

  // 圆角
  borderRadius: {
    sm: 4,
    md: 6,
    lg: 8,
    xl: 12,
    xxl: 16,
  },

  // 阴影
  shadows: {
    xs: '0 1px 2px rgba(0, 0, 0, 0.05)',
    sm: '0 2px 4px rgba(0, 0, 0, 0.08)',
    md: '0 4px 12px rgba(0, 0, 0, 0.12)',
    lg: '0 8px 24px rgba(0, 0, 0, 0.16)',
    xl: '0 12px 48px rgba(0, 0, 0, 0.20)',
  },

  // 过渡动画
  transitions: {
    fast: '150ms cubic-bezier(0.4, 0, 0.2, 1)',
    base: '200ms cubic-bezier(0.4, 0, 0.2, 1)',
    slow: '300ms cubic-bezier(0.4, 0, 0.2, 1)',
  },

  // Z-Index 层级
  zIndex: {
    header: 10,
    sider: 10,
    dropdown: 1000,
    modal: 1050,
    tooltip: 1070,
  },
} as const;

export const getContentMarginLeft = (collapsed: boolean): number => {
  if (collapsed) {
    return LAYOUT_CONFIG.sider.collapsedWidth + LAYOUT_CONFIG.content.marginCollapsed;
  }
  return LAYOUT_CONFIG.sider.width + LAYOUT_CONFIG.content.marginExpanded;
};

export const getContentMargin = (collapsed: boolean): string => {
  const margin = collapsed
    ? LAYOUT_CONFIG.content.marginCollapsed
    : LAYOUT_CONFIG.content.marginExpanded;
  const marginLeft = getContentMarginLeft(collapsed);
  return `${margin}px ${margin}px ${margin}px ${marginLeft}px`;
};

export const getContentPadding = (collapsed: boolean): string => {
  return `${collapsed ? LAYOUT_CONFIG.content.paddingCollapsed : LAYOUT_CONFIG.content.padding}px`;
};

export const validateHeaderHeight = (height: number): boolean => {
  return (height - 48) % 8 === 0 && height >= 48;
};

export const validateSiderWidth = (width: number): boolean => {
  return (width - 200) % 8 === 0 && width >= 200;
};

if (process.env.NODE_ENV === 'development') {
  console.assert(
    validateHeaderHeight(LAYOUT_CONFIG.header.height),
    `❌ Header高度 ${LAYOUT_CONFIG.header.height}px 不符合 Ant Design 公式 (48+8n)`
  );

  console.assert(
    validateSiderWidth(LAYOUT_CONFIG.sider.width),
    `❌ Sider宽度 ${LAYOUT_CONFIG.sider.width}px 不符合 Ant Design 公式 (200+8n)`
  );
}

export default LAYOUT_CONFIG;
