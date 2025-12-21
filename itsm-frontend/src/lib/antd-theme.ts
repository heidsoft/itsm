import type { ThemeConfig } from 'antd';

/**
 * Ant Design 主题配置
 * 这个文件定义了 Ant Design 组件的主题样式，确保与我们的企业级设计系统保持一致
 * 
 * 主要功能：
 * 1. 覆盖 Ant Design 的默认样式
 * 2. 确保我们的自定义 CSS 类不被覆盖
 * 3. 提供一致的颜色和间距系统
 */

export const antdTheme: ThemeConfig = {
  token: {
    // 主色调 - 与我们的设计系统保持一致
    colorPrimary: '#3b82f6', // blue-600
    colorPrimaryHover: '#2563eb', // blue-700
    colorPrimaryActive: '#1d4ed8', // blue-800
    
    // 成功、警告、错误色
    colorSuccess: '#10b981', // green-500
    colorWarning: '#f59e0b', // amber-500
    colorError: '#ef4444', // red-500
    
    // 中性色
    colorText: '#1e293b', // slate-800
    colorTextSecondary: '#64748b', // slate-500
    colorTextTertiary: '#94a3b8', // slate-400
    colorTextQuaternary: '#cbd5e1', // slate-300
    
    // 背景色
    colorBgContainer: '#ffffff',
    colorBgElevated: '#ffffff',
    colorBgLayout: '#f8fafc', // slate-50
    colorBgSpotlight: '#f1f5f9', // slate-100
    
    // 边框色
    colorBorder: '#e2e8f0', // slate-200
    colorBorderSecondary: '#f1f5f9', // slate-100
    
    // 圆角
    borderRadius: 6, // 与我们的 --radius-md 一致
    borderRadiusLG: 8, // 与我们的 --radius-lg 一致
    borderRadiusSM: 4, // 与我们的 --radius-sm 一致
    
    // 间距
    padding: 16, // 与我们的 --spacing-lg 一致
    paddingLG: 24, // 与我们的 --spacing-xl 一致
    paddingSM: 12, // 与我们的 --spacing-md 一致
    paddingXS: 8, // 与我们的 --spacing-sm 一致
    
    // 字体
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "Helvetica Neue", Helvetica, Arial, sans-serif',
    fontSize: 14,
    fontSizeLG: 16,
    fontSizeSM: 12,
    
    // 行高
    lineHeight: 1.5,
    
    // 阴影
    boxShadow: '0 1px 2px 0 rgb(0 0 0 / 0.05)', // 与我们的 --shadow-sm 一致
    boxShadowSecondary: '0 4px 6px -1px rgb(0 0 0 / 0.08)', // 与我们的 --shadow-md 一致
    boxShadowTertiary: '0 10px 15px -3px rgb(0 0 0 / 0.08)', // 与我们的 --shadow-lg 一致
    
    // 动画
    motionDurationFast: '0.15s',
    motionDurationMid: '0.2s',
    motionDurationSlow: '0.3s',
  },
  
  components: {
    // 按钮组件
    Button: {
      borderRadius: 6,
      controlHeight: 32,
      controlHeightLG: 40,
      controlHeightSM: 24,
      paddingInline: 12,
      paddingBlock: 6,
    },
    
    // 卡片组件
    Card: {
      borderRadiusLG: 8,
      borderRadiusSM: 6,
      paddingLG: 16,
      paddingSM: 12,
    },
    
    // 表格组件
    Table: {
      borderRadius: 6,
      headerBg: '#f8fafc', // slate-50
      headerColor: '#475569', // slate-600
      rowHoverBg: '#f1f5f9', // slate-100
    },
    
    // 表单组件
    Form: {
      labelColor: '#374151', // gray-700
      labelFontSize: 14,
      labelHeight: 32,
    },
    
    Input: {
      borderRadius: 6,
      controlHeight: 32,
      controlHeightLG: 40,
      controlHeightSM: 24,
    },
    
    Select: {
      borderRadius: 6,
      controlHeight: 32,
      controlHeightLG: 40,
      controlHeightSM: 24,
    },
    
    // 菜单组件
    Menu: {
      itemBorderRadius: 6,
      itemMarginInline: 6,
      itemMarginBlock: 1,
      itemHeight: 32,
      subMenuItemBg: 'transparent',
      itemSelectedBg: 'linear-gradient(to right, #dbeafe, #e0e7ff)', // blue-50 to indigo-50
      itemSelectedColor: '#2563eb', // blue-700
      itemHoverBg: 'linear-gradient(to right, #f1f5f9, #dbeafe)', // slate-50 to blue-50
      itemHoverColor: '#2563eb', // blue-700
    },
    
    // 模态框组件
    Modal: {
      borderRadiusLG: 8,
      borderRadiusSM: 6,
      paddingLG: 16,
      paddingSM: 12,
    },
    
    // 下拉菜单组件
    Dropdown: {
      borderRadiusLG: 6,
      borderRadiusSM: 4,
      paddingLG: 8,
      paddingSM: 4,
    },
    
    // 分页组件
    Pagination: {
      borderRadius: 6,
      itemSize: 32,
    },
    
    // 标签组件
    Tag: {
      borderRadiusSM: 4,
      paddingXS: 4,
      paddingSM: 8,
    },
    
    // 统计组件
    Statistic: {
      titleFontSize: 14,
      contentFontSize: 24,
      fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "Helvetica Neue", Helvetica, Arial, sans-serif',
    },
    
    // 进度条组件
    Progress: {
      borderRadius: 6,
      defaultColor: '#3b82f6', // blue-600
      remainingColor: '#e2e8f0', // slate-200
    },
    
    // 警告组件
    Alert: {
      borderRadiusLG: 8,
      borderRadiusSM: 6,
      paddingLG: 16,
      paddingSM: 12,
    },
    
    // 面包屑组件
    Breadcrumb: {
      fontSize: 14,
      itemColor: '#64748b', // slate-500
      lastItemColor: '#1e293b', // slate-800
      separatorColor: '#cbd5e1', // slate-300
    },
  },
};

/**
 * 导出主题配置的默认值
 * 这个配置确保 Ant Design 组件的样式与我们的企业级设计系统保持一致
 */
export default antdTheme; 