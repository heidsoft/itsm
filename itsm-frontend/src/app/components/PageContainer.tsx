import React from 'react';
import { Layout, Typography, Breadcrumb, Space, Divider } from 'antd';
import type { BreadcrumbProps } from 'antd';

const { Content } = Layout;
const { Title } = Typography;

interface PageContainerProps {
  header?: {
    title?: React.ReactNode;
    breadcrumb?: {
      items?: BreadcrumbProps['items'];
    };
  };
  extra?: React.ReactNode;
  children?: React.ReactNode;
}

export const PageContainer: React.FC<PageContainerProps> = ({
  header,
  extra,
  children,
}) => {
  return (
    <div style={{ padding: '24px', background: '#fff', minHeight: '100%' }}>
      {(header?.title || header?.breadcrumb?.items) && (
        <>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
            <div>
              {header?.breadcrumb?.items && (
                <Breadcrumb items={header.breadcrumb.items} style={{ marginBottom: '8px' }} />
              )}
              {header?.title && <Title level={4} style={{ margin: 0 }}>{header.title}</Title>}
            </div>
            {extra && <Space>{extra}</Space>}
          </div>
          <Divider style={{ margin: '0 0 24px 0' }} />
        </>
      )}
      <Content>{children}</Content>
    </div>
  );
};
