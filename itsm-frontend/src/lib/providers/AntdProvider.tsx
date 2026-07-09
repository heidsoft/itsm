'use client';

import { AntdRegistry } from '@ant-design/nextjs-registry';
import { ConfigProvider, App } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { useTheme, getAntdTheme } from '@/lib/design-system/theme';

interface AntdProviderProps {
  children: React.ReactNode;
}

export const AntdProvider: React.FC<AntdProviderProps> = ({ children }) => {
  const { isDark } = useTheme();
  const antdTheme = getAntdTheme(isDark);
  
  return (
    <AntdRegistry>
      <ConfigProvider theme={antdTheme} locale={zhCN}>
        <App>{children}</App>
      </ConfigProvider>
    </AntdRegistry>
  );
};
