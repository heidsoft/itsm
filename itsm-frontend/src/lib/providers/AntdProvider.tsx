'use client';

// test-coverage-guard: skip — ConfigProvider 语言切换的薄封装,providers.test.tsx 覆盖渲染。
import { AntdRegistry } from '@ant-design/nextjs-registry';
import { ConfigProvider, App } from 'antd';
import enUS from 'antd/locale/en_US';
import { useTheme, getAntdTheme } from '@/lib/design-system/theme';

interface AntdProviderProps {
  children: React.ReactNode;
}

export const AntdProvider: React.FC<AntdProviderProps> = ({ children }) => {
  const { isDark } = useTheme();
  const antdTheme = getAntdTheme(isDark);

  const antdLocale = enUS;

  return (
    <AntdRegistry>
      <ConfigProvider theme={antdTheme} locale={antdLocale}>
        <App>{children}</App>
      </ConfigProvider>
    </AntdRegistry>
  );
};
