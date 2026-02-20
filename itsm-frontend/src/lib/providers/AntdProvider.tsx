'use client';

import { AntdRegistry } from '@ant-design/nextjs-registry';
import { ConfigProvider, App } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import antdTheme from '@/lib/antd-theme';

interface AntdProviderProps {
  children: React.ReactNode;
}

export const AntdProvider: React.FC<AntdProviderProps> = ({ children }) => {
  return (
    <AntdRegistry>
      <ConfigProvider theme={antdTheme} locale={zhCN}>
        <App>
          {children}
        </App>
      </ConfigProvider>
    </AntdRegistry>
  );
};
