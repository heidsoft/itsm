'use client';

import React from 'react';
import { Card, Skeleton, Row, Col, Space } from 'antd';

/**
 * 卡片骨架屏
 */
export const CardSkeleton: React.FC<{ count?: number }> = ({ count = 1 }) => {
  return (
    <>
      {Array.from({ length: count }).map((_, index) => (
        <Card key={index} style={{ marginBottom: 16 }}>
          <Skeleton active paragraph={{ rows: 3 }} />
        </Card>
      ))}
    </>
  );
};

/**
 * 表格骨架屏
 */
export const TableSkeleton: React.FC<{ rows?: number }> = ({ rows = 5 }) => {
  return (
    <Card>
      <Space orientation="vertical" style={{ width: '100%' }} size="middle">
        {/* 表格头部 */}
        <Skeleton.Button active style={{ width: '100%', height: 40 }} />
        
        {/* 表格行 */}
        {Array.from({ length: rows }).map((_, index) => (
          <Skeleton key={index} active paragraph={false} />
        ))}
      </Space>
    </Card>
  );
};

/**
 * 统计卡片骨架屏
 */
export const StatCardSkeleton: React.FC<{ count?: number }> = ({ count = 4 }) => {
  return (
    <Row gutter={[12, 12]}>
      {Array.from({ length: count }).map((_, index) => (
        <Col xs={24} sm={12} md={6} lg={6} xl={6} key={index}>
          <Card
            className="border-0 shadow-sm"
            styles={{ body: { padding: '16px' } }}
          >
            <div className="flex items-center justify-between">
              <div style={{ flex: 1 }}>
                <Skeleton.Button
                  active
                  size="small"
                  style={{ width: 80, height: 16, marginBottom: 8 }}
                />
                <Skeleton.Button
                  active
                  size="default"
                  style={{ width: 100, height: 24 }}
                />
              </div>
              <Skeleton.Avatar active size="large" shape="square" />
            </div>
          </Card>
        </Col>
      ))}
    </Row>
  );
};

/**
 * 表单骨架屏
 */
export const FormSkeleton: React.FC<{ fields?: number }> = ({ fields = 5 }) => {
  return (
    <Card>
      <Space orientation="vertical" style={{ width: '100%' }} size="large">
        {Array.from({ length: fields }).map((_, index) => (
          <div key={index}>
            <Skeleton.Button
              active
              size="small"
              style={{ width: 100, height: 16, marginBottom: 8 }}
            />
            <Skeleton.Input active style={{ width: '100%' }} />
          </div>
        ))}
        <Skeleton.Button active style={{ width: 120 }} />
      </Space>
    </Card>
  );
};

/**
 * 详情页骨架屏
 */
export const DetailSkeleton: React.FC = () => {
  return (
    <Space orientation="vertical" style={{ width: '100%' }} size="large">
      {/* 标题区域 */}
      <Card>
        <Skeleton
          active
          avatar
          paragraph={{ rows: 2 }}
          title={{ width: '60%' }}
        />
      </Card>

      {/* 内容区域 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={16}>
          <Card>
            <Skeleton active paragraph={{ rows: 6 }} />
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card>
            <Skeleton active paragraph={{ rows: 4 }} />
          </Card>
        </Col>
      </Row>
    </Space>
  );
};

/**
 * 列表项骨架屏
 */
export const ListSkeleton: React.FC<{ count?: number }> = ({ count = 5 }) => {
  return (
    <Card>
      <Space orientation="vertical" style={{ width: '100%' }} size="middle">
        {Array.from({ length: count }).map((_, index) => (
          <Skeleton key={index} active avatar paragraph={{ rows: 1 }} />
        ))}
      </Space>
    </Card>
  );
};

/**
 * 图表骨架屏
 */
export const ChartSkeleton: React.FC<{ height?: number }> = ({ height = 300 }) => {
  return (
    <Card>
      <Skeleton.Node
        active
        style={{
          width: '100%',
          height: height,
        }}
      >
        <div
          style={{
            width: '100%',
            height: '100%',
            background: 'linear-gradient(90deg, #f0f0f0 25%, #e0e0e0 50%, #f0f0f0 75%)',
            backgroundSize: '200% 100%',
            animation: 'loading 1.5s ease-in-out infinite',
          }}
        />
      </Skeleton.Node>
    </Card>
  );
};

/**
 * 仪表板骨架屏（组合型）
 */
export const DashboardSkeleton: React.FC = () => {
  return (
    <Space direction="vertical" style={{ width: '100%' }} size="large">
      {/* 统计卡片 */}
      <StatCardSkeleton count={4} />

      {/* 图表区域 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={16}>
          <ChartSkeleton height={350} />
        </Col>
        <Col xs={24} lg={8}>
          <ChartSkeleton height={350} />
        </Col>
      </Row>

      {/* 表格区域 */}
      <TableSkeleton rows={5} />
    </Space>
  );
};

/**
 * 页面加载骨架屏（通用）
 */
export const PageSkeleton: React.FC<{ type?: 'list' | 'detail' | 'form' | 'dashboard' | 'table' }> = ({
  type = 'list',
}) => {
  const skeletonMap = {
    list: <ListSkeleton />,
    detail: <DetailSkeleton />,
    form: <FormSkeleton />,
    dashboard: <DashboardSkeleton />,
    table: <TableSkeleton />,
  };

  return (
    <div style={{ padding: '24px' }}>
      {skeletonMap[type] || <CardSkeleton />}
    </div>
  );
};

/**
 * 带动画的加载骨架屏容器
 */
export const AnimatedSkeletonContainer: React.FC<{
  loading: boolean;
  skeleton?: React.ReactNode;
  children: React.ReactNode;
  delay?: number;
}> = ({ loading, skeleton, children, delay = 0 }) => {
  const [showSkeleton, setShowSkeleton] = React.useState(false);

  React.useEffect(() => {
    if (loading) {
      const timer = setTimeout(() => {
        setShowSkeleton(true);
      }, delay);
      return () => clearTimeout(timer);
    } else {
      setShowSkeleton(false);
    }
  }, [loading, delay]);

  if (!loading) {
    return <>{children}</>;
  }

  if (!showSkeleton) {
    return null;
  }

  return <>{skeleton || <CardSkeleton />}</>;
};

// CSS动画（添加到全局样式或Tailwind配置中）
const skeletonStyles = `
@keyframes loading {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}
`;

// 导出所有组件
export default {
  Card: CardSkeleton,
  Table: TableSkeleton,
  StatCard: StatCardSkeleton,
  Form: FormSkeleton,
  Detail: DetailSkeleton,
  List: ListSkeleton,
  Chart: ChartSkeleton,
  Dashboard: DashboardSkeleton,
  Page: PageSkeleton,
  AnimatedContainer: AnimatedSkeletonContainer,
};

