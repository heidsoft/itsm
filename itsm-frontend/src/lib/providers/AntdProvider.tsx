'use client';

// test-coverage-guard: skip — ConfigProvider 语言切换的薄封装,providers.test.tsx 覆盖渲染。
import { AntdRegistry } from '@ant-design/nextjs-registry';
import { ConfigProvider, App } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import enUS from 'antd/locale/en_US';
import { useTheme, getAntdTheme } from '@/lib/design-system/theme';
import { useI18n } from '@/lib/i18n/useI18n';

interface AntdProviderProps {
  children: React.ReactNode;
}

export const AntdProvider: React.FC<AntdProviderProps> = ({ children }) => {
  const { isDark } = useTheme();
  const { language } = useI18n();
  const antdTheme = getAntdTheme(isDark);

  const antdLocale = language === 'en-US' ? enUS : zhCN;

  return (
    <AntdRegistry>
      <ConfigProvider theme={antdTheme} locale={antdLocale}>
        <App>{children}</App>
      </ConfigProvider>
    </AntdRegistry>
  );
};
