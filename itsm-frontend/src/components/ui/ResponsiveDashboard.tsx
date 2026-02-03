'use client';

import React, { useState, useMemo, useEffect } from 'react';
import { Row, Col, Typography, Space, Button, Dropdown, Skeleton, Card } from 'antd';
import {
  SettingOutlined,
  ReloadOutlined,
  FullscreenOutlined,
  DownloadOutlined,
  FilterOutlined,
} from '@ant-design/icons';
import { motion, AnimatePresence } from 'framer-motion';
import { Responsive, WidthProvider } from 'react-grid-layout';
import 'react-grid-layout/css/styles.css';
import 'react-resizable/css/styles.css';

const { Title, Text } = Typography;
const ResponsiveGridLayout = WidthProvider(Responsive);

export interface DashboardWidget {
  id: string;
  title: string;
  type: 'stats' | 'chart' | 'table' | 'info' | 'custom';
  component: React.ComponentType<any>;
  props?: Record<string, any>;
  layout: {
    x: number;
    y: number;
    w: number;
    h: number;
    minW?: number;
    minH?: number;
    maxW?: number;
    maxH?: number;
  };
  responsive?: {
    lg?: Partial<DashboardWidget['layout']>;
    md?: Partial<DashboardWidget['layout']>;
    sm?: Partial<DashboardWidget['layout']>;
    xs?: Partial<DashboardWidget['layout']>;
  };
  refreshable?: boolean;
  configurable?: boolean;
  removable?: boolean;
}

export interface DashboardConfig {
  title: string;
  description?: string;
  refreshInterval?: number;
  autoRefresh?: boolean;
  theme?: 'light' | 'dark';
  layout?: 'grid' | 'masonry' | 'flex';
  columns?: {
    lg: number;
    md: number;
    sm: number;
    xs: number;
  };
}

export interface ResponsiveDashboardProps {
  widgets: DashboardWidget[];
  config?: Partial<DashboardConfig>;
  editable?: boolean;
  loading?: boolean;
  onWidgetRefresh?: (widgetId: string) => void;
  onWidgetRemove?: (widgetId: string) => void;
  onWidgetConfigure?: (widgetId: string) => void;
  onLayoutChange?: (layouts: any) => void;
  onConfigChange?: (config: DashboardConfig) => void;
  className?: string;
}

const defaultConfig: DashboardConfig = {
  title: '仪表盘',
  autoRefresh: false,
  refreshInterval: 30000,
  theme: 'light',
  layout: 'grid',
  columns: { lg: 12, md: 10, sm: 6, xs: 4 },
};

// 断点配置
const breakpoints = { lg: 1200, md: 996, sm: 768, xs: 480 };
const cols = { lg: 12, md: 10, sm: 6, xs: 4 };

// 动画变体
const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      duration: 0.3,
      staggerChildren: 0.1,
    },
  },
};

const widgetVariants = {
  hidden: { opacity: 0, scale: 0.9 },
  visible: {
    opacity: 1,
    scale: 1,
    transition: {
      duration: 0.4,
      ease: 'easeOut',
    },
  },
  exit: {
    opacity: 0,
    scale: 0.8,
    transition: {
      duration: 0.2,
    },
  },
};

export const ResponsiveDashboard: React.FC<ResponsiveDashboardProps> = ({
  widgets,
  config = {},
  editable = false,
  loading = false,
  onWidgetRefresh,
  onWidgetRemove,
  onWidgetConfigure,
  onLayoutChange,
  onConfigChange,
  className,
}) => {
  const [currentBreakpoint, setCurrentBreakpoint] = useState<string>('lg');
  const [layouts, setLayouts] = useState<any>({});
  const [mounted, setMounted] = useState(false);

  const mergedConfig = { ...defaultConfig, ...config };

  // 组件挂载状态
  useEffect(() => {
    setMounted(true);
  }, []);

  // 生成布局配置
  const generateLayouts = useMemo(() => {
    const layoutsMap: any = {};

    Object.keys(cols).forEach(breakpoint => {
      layoutsMap[breakpoint] = widgets.map(widget => ({
        i: widget.id,
        x: widget.responsive?.[breakpoint as keyof typeof widget.responsive]?.x ?? widget.layout.x,
        y: widget.responsive?.[breakpoint as keyof typeof widget.responsive]?.y ?? widget.layout.y,
        w: widget.responsive?.[breakpoint as keyof typeof widget.responsive]?.w ?? widget.layout.w,
        h: widget.responsive?.[breakpoint as keyof typeof widget.responsive]?.h ?? widget.layout.h,
        minW: widget.layout.minW ?? 2,
        minH: widget.layout.minH ?? 2,
        maxW: widget.layout.maxW ?? Infinity,
        maxH: widget.layout.maxH ?? Infinity,
      }));
    });

    return layoutsMap;
  }, [widgets]);

  // 处理布局变化
  const handleLayoutChange = (layout: any, layouts: any) => {
    setLayouts(layouts);
    onLayoutChange?.(layouts);
  };

  // 处理断点变化
  const handleBreakpointChange = (breakpoint: string) => {
    setCurrentBreakpoint(breakpoint);
  };

  // 渲染widget操作按钮
  const renderWidgetActions = (widget: DashboardWidget) => {
    const actions = [];

    if (widget.refreshable && onWidgetRefresh) {
      actions.push(
        <Button
          key='refresh'
          type='text'
          size='small'
          icon={<ReloadOutlined />}
          onClick={() => onWidgetRefresh(widget.id)}
          title='刷新'
        />
      );
    }

    if (widget.configurable && onWidgetConfigure) {
      actions.push(
        <Button
          key='configure'
          type='text'
          size='small'
          icon={<SettingOutlined />}
          onClick={() => onWidgetConfigure(widget.id)}
          title='配置'
        />
      );
    }

    if (editable && widget.removable && onWidgetRemove) {
      actions.push(
        <Button
          key='remove'
          type='text'
          size='small'
          danger
          onClick={() => onWidgetRemove(widget.id)}
          title='移除'
        >
          ×
        </Button>
      );
    }

    return actions.length > 0 ? <Space size='small'>{actions}</Space> : null;
  };

  // 渲染单个widget
  const renderWidget = (widget: DashboardWidget) => {
    const Component = widget.component;
    const actions = renderWidgetActions(widget);

    if (loading) {
      return (
        <Card title={widget.title} extra={actions} className='h-full'>
          <Skeleton active paragraph={{ rows: 4 }} />
        </Card>
      );
    }

    return (
      <motion.div
        key={widget.id}
        variants={widgetVariants}
        initial='hidden'
        animate='visible'
        exit='exit'
        className='h-full'
      >
        <Card
          title={widget.title}
          extra={actions}
          hoverable
          className='h-full'
          styles={{ body: { height: 'calc(100% - 60px)', overflow: 'auto' } }}
        >
          <Component {...widget.props} />
        </Card>
      </motion.div>
    );
  };

  // 渲染工具栏
  const renderToolbar = () => (
    <div className='flex justify-between items-center mb-6'>
      <div>
        <Title level={2} className='!mb-1'>
          {mergedConfig.title}
        </Title>
        {mergedConfig.description && <Text type='secondary'>{mergedConfig.description}</Text>}
      </div>
      <Space>
        <Button
          icon={<FilterOutlined />}
          onClick={() => {
            /* 打开过滤器 */
          }}
        >
          过滤
        </Button>
        <Button
          icon={<ReloadOutlined />}
          onClick={() => {
            /* 刷新所有 */
          }}
          loading={loading}
        >
          刷新
        </Button>
        <Button
          icon={<DownloadOutlined />}
          onClick={() => {
            /* 导出数据 */
          }}
        >
          导出
        </Button>
        {editable && (
          <Dropdown
            menu={{
              items: [
                {
                  key: 'addWidget',
                  label: '添加组件',
                  icon: <SettingOutlined />,
                },
                {
                  key: 'editLayout',
                  label: '编辑布局',
                  icon: <SettingOutlined />,
                },
                {
                  key: 'settings',
                  label: '仪表盘设置',
                  icon: <SettingOutlined />,
                },
              ],
            }}
          >
            <Button icon={<SettingOutlined />}>设置</Button>
          </Dropdown>
        )}
        <Button
          icon={<FullscreenOutlined />}
          onClick={() => {
            /* 进入全屏 */
          }}
        >
          全屏
        </Button>
      </Space>
    </div>
  );

  // 网格布局渲染
  const renderGridLayout = () => {
    if (!mounted) {
      return (
        <Row gutter={[16, 16]}>
          {widgets.map(widget => (
            <Col key={widget.id} span={24 / Math.ceil(widgets.length / 2)}>
              <Skeleton active />
            </Col>
          ))}
        </Row>
      );
    }

    return (
      <ResponsiveGridLayout
        className='responsive-dashboard-grid'
        layouts={layouts.lg ? layouts : generateLayouts}
        breakpoints={breakpoints}
        cols={cols}
        rowHeight={60}
        isDraggable={editable}
        isResizable={editable}
        onLayoutChange={handleLayoutChange}
        onBreakpointChange={handleBreakpointChange}
        margin={[16, 16]}
        containerPadding={[0, 0]}
        useCSSTransforms={true}
        preventCollision={false}
        compactType='vertical'
      >
        {widgets.map(renderWidget)}
      </ResponsiveGridLayout>
    );
  };

  // 瀑布流布局渲染
  const renderMasonryLayout = () => (
    <div className='masonry-layout'>
      <Row gutter={[16, 16]}>
        {widgets.map(widget => (
          <Col key={widget.id} xs={24} sm={12} md={8} lg={6} xl={6}>
            {renderWidget(widget)}
          </Col>
        ))}
      </Row>
    </div>
  );

  // Flex布局渲染
  const renderFlexLayout = () => (
    <motion.div
      variants={containerVariants}
      initial='hidden'
      animate='visible'
      className='flex-layout space-y-4'
    >
      {widgets.map(renderWidget)}
    </motion.div>
  );

  // 根据布局类型渲染
  const renderLayout = () => {
    switch (mergedConfig.layout) {
      case 'masonry':
        return renderMasonryLayout();
      case 'flex':
        return renderFlexLayout();
      case 'grid':
      default:
        return renderGridLayout();
    }
  };

  return (
    <div className={`responsive-dashboard ${className || ''}`}>
      {renderToolbar()}

      <AnimatePresence mode='wait'>{renderLayout()}</AnimatePresence>

      <style jsx global>{`
        .responsive-dashboard-grid {
          position: relative;
          transition: all 0.3s ease;
        }

        .responsive-dashboard-grid .react-grid-item {
          transition: all 200ms ease;
          transition-property: left, top, width, height;
        }

        .responsive-dashboard-grid .react-grid-item.cssTransforms {
          transition-property: transform, width, height;
        }

        .responsive-dashboard-grid .react-grid-item > .react-resizable-handle {
          position: absolute;
          width: 20px;
          height: 20px;
          bottom: 0;
          right: 0;
          background: url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNiIgaGVpZ2h0PSI2IiB2aWV3Qm94PSIwIDAgNiA2IiBmaWxsPSJub25lIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPgo8ZG90cyBmaWxsPSIjOTk5Ij4KPGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiLz4KPGNpcmNsZSBjeD0iMSIgY3k9IjMiIHI9IjEiLz4KPGNpcmNsZSBjeD0iMSIgY3k9IjUiIHI9IjEiLz4KPGNpcmNsZSBjeD0iMyIgY3k9IjMiIHI9IjEiLz4KPGNpcmNsZSBjeD0iMyIgY3k9IjUiIHI9IjEiLz4KPGNpcmNsZSBjeD0iNSIgY3k9IjUiIHI9IjEiLz4KPC9kb3RzPgo8L3N2Zz4K');
          background-size: contain;
          background-repeat: no-repeat;
          cursor: se-resize;
        }

        .responsive-dashboard-grid .react-grid-placeholder {
          background: #1890ff !important;
          opacity: 0.2;
          border-radius: 8px;
          transition-duration: 100ms;
        }

        .masonry-layout .ant-col {
          margin-bottom: 16px;
        }

        .flex-layout > * {
          width: 100%;
        }

        @media (max-width: 768px) {
          .responsive-dashboard .ant-typography-h2 {
            font-size: 1.5rem !important;
          }

          .responsive-dashboard .ant-btn {
            padding: 4px 8px;
          }
        }
      `}</style>
    </div>
  );
};

export default ResponsiveDashboard;
