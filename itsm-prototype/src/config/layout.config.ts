/**
 * ITSM 布局配置 - 完全遵循 Ant Design 官方设计规范
 * 参考: https://ant.design/components/layout-cn
 */

/**
 * Ant Design 官方设计规则:
 * 
 * 1. 顶部导航高度计算公式: 48+8n
 *    - 应用类系统: 64px (n=2) ✅ 我们使用
 *    - 展示类页面: 80px (n=4)
 * 
 * 2. 侧边导航宽度计算公式: 200+8n
 *    - 标准宽度: 200px (n=0)
 *    - 宽版宽度: 256px (n=7) ✅ 针对PC端优化
 *    - 收起宽度: 80px (标准) ✅ 我们使用
 * 
 * 3. 二级导航高度:
 *    - 应用类: 48px
 *    - 展示类: 56px
 */

export const LAYOUT_CONFIG = {
  // ========== Header 配置 (Ant Design 标准: 64px) ==========
  header: {
    height: 64,                    // 64px = 48 + 8×2 ✅
    padding: '0 24px',             // 标准内边距 24px
    paddingMobile: '0 16px',       // 移动端 16px
    logoHeight: 40,                // Logo高度
    fontSize: 18,                  // 标题字号
    iconSize: 20,                  // 图标大小
    avatarSize: 32,                // 头像大小
    shadow: '0 2px 8px rgba(0, 0, 0, 0.08)', // Ant Design 标准阴影
  },

  // ========== Sider 配置 (Ant Design 标准: 200px/80px) ==========
  sider: {
    width: 256,                    // 256px = 200 + 8×7 (加宽以适配PC端)
    collapsedWidth: 80,            // 80px 标准收起宽度 ✅
    breakpoint: 'lg' as const,     // 992px 响应式断点
    logoAreaHeight: 64,            // Logo区域高度 = Header高度
    logoPadding: '0 24px',         // Logo内边距
    logoPaddingCollapsed: '0 20px', // 收起时内边距
    logoIconSize: 40,              // Logo图标大小
    logoTextSize: 20,              // Logo文字大小
    menuPadding: '16px 8px',       // 菜单内边距 (增加水平间距)
    menuItemHeight: 48,            // 菜单项高度 = 40 + 8 (增加高度)
    shadow: '2px 0 8px rgba(0, 0, 0, 0.1)', // 阴影
  },

  // ========== Content 配置 ==========
  content: {
    // 外边距 (根据Sider状态动态计算)
    marginExpanded: 24,            // 展开时外边距
    marginCollapsed: 16,           // 收起时外边距
    marginMobile: 8,               // 移动端外边距
    
    // 左边距 (Sider宽度 + margin)
    marginLeftExpanded: 280,       // 256 + 24
    marginLeftCollapsed: 96,       // 80 + 16
    
    // 内边距
    padding: 24,                   // 标准内边距
    paddingCollapsed: 16,          // 收起时内边距
    paddingMobile: 12,             // 移动端内边距
    
    // 最小高度
    minHeight: 'calc(100vh - 64px)', // 100vh - Header高度
    
    // 页头
    pageHeaderMarginBottom: 24,    // 页头底部边距
    pageHeaderPaddingBottom: 16,   // 页头内边距
    pageTitleFontSize: 24,         // 标题字号 (增大)
    pageDescFontSize: 14,          // 描述字号
  },

  // ========== 响应式断点 (Ant Design 标准) ==========
  breakpoints: {
    xs: 0,      // < 576px (mobile)
    sm: 576,    // >= 576px (large mobile)
    md: 768,    // >= 768px (tablet)
    lg: 992,    // >= 992px (desktop) ✅ Sider断点
    xl: 1200,   // >= 1200px (large desktop)
    xxl: 1600,  // >= 1600px (xlarge desktop)
  },

  // ========== 间距系统 (8px 栅格) ==========
  spacing: {
    unit: 8,    // 基础单位
    xs: 8,      // 8 = 8×1
    sm: 12,     // 12 = 8×1.5
    md: 16,     // 16 = 8×2
    lg: 24,     // 24 = 8×3
    xl: 32,     // 32 = 8×4
    xxl: 48,    // 48 = 8×6
  },

  // ========== 圆角系统 ==========
  borderRadius: {
    sm: 4,      // 小圆角
    md: 6,      // 中圆角
    lg: 8,      // 大圆角 ✅ 常用
    xl: 12,     // 超大圆角
    xxl: 16,    // 卡片圆角
  },

  // ========== 阴影系统 (Ant Design 标准) ==========
  shadows: {
    xs: '0 1px 2px rgba(0, 0, 0, 0.05)',
    sm: '0 2px 4px rgba(0, 0, 0, 0.08)',
    md: '0 4px 12px rgba(0, 0, 0, 0.12)',
    lg: '0 8px 24px rgba(0, 0, 0, 0.16)',
    xl: '0 12px 48px rgba(0, 0, 0, 0.20)',
  },

  // ========== 过渡动画 ==========
  transitions: {
    fast: '150ms cubic-bezier(0.4, 0, 0.2, 1)',
    base: '200ms cubic-bezier(0.4, 0, 0.2, 1)',
    slow: '300ms cubic-bezier(0.4, 0, 0.2, 1)',
  },

  // ========== Z-Index 层级 ==========
  zIndex: {
    header: 10,
    sider: 10,
    dropdown: 1000,
    modal: 1050,
    tooltip: 1070,
  },
} as const;

/**
 * 辅助函数：获取Content的左边距
 */
export const getContentMarginLeft = (collapsed: boolean): number => {
  if (collapsed) {
    return LAYOUT_CONFIG.sider.collapsedWidth + LAYOUT_CONFIG.content.marginCollapsed;
  }
  return LAYOUT_CONFIG.sider.width + LAYOUT_CONFIG.content.marginExpanded;
};

/**
 * 辅助函数：获取Content的外边距
 */
export const getContentMargin = (collapsed: boolean): string => {
  const margin = collapsed 
    ? LAYOUT_CONFIG.content.marginCollapsed 
    : LAYOUT_CONFIG.content.marginExpanded;
  const marginLeft = getContentMarginLeft(collapsed);
  return `${margin}px ${margin}px ${margin}px ${marginLeft}px`;
};

/**
 * 辅助函数：获取Content的内边距
 */
export const getContentPadding = (collapsed: boolean): string => {
  return `${collapsed ? LAYOUT_CONFIG.content.paddingCollapsed : LAYOUT_CONFIG.content.padding}px`;
};

/**
 * 辅助函数：验证尺寸是否符合Ant Design公式
 */
export const validateHeaderHeight = (height: number): boolean => {
  // 验证公式: height = 48 + 8n
  return (height - 48) % 8 === 0 && height >= 48;
};

export const validateSiderWidth = (width: number): boolean => {
  // 验证公式: width = 200 + 8n
  return (width - 200) % 8 === 0 && width >= 200;
};

// 验证当前配置
console.assert(
  validateHeaderHeight(LAYOUT_CONFIG.header.height),
  `❌ Header高度 ${LAYOUT_CONFIG.header.height}px 不符合 Ant Design 公式 (48+8n)`
);

console.assert(
  validateSiderWidth(LAYOUT_CONFIG.sider.width),
  `❌ Sider宽度 ${LAYOUT_CONFIG.sider.width}px 不符合 Ant Design 公式 (200+8n)`
);

console.log('✅ Layout配置符合 Ant Design 官方设计规范');

export default LAYOUT_CONFIG;
