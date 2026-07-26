import type { ThemeConfig } from 'antd';
import { getAntdTheme } from '@/lib/design-system/theme';

/**
 * Ant Design 静态主题配置（亮色）
 *
 * 说明：本文件不再自行定义令牌，统一从设计系统单一来源
 * `@/lib/design-system/theme` 的 getAntdTheme 派生，
 * 供认证页（login/register/sso）等无 ThemeProvider 上下文的场景使用，
 * 确保认证页与主应用（AntdProvider）视觉一致。
 */
export const antdTheme: ThemeConfig = getAntdTheme(false) as ThemeConfig;

export default antdTheme;
