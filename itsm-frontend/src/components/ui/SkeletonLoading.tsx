'use client';

import React, { useState, useEffect } from 'react';
import { Card, Skeleton, SkeletonProps } from 'antd';
import { cn } from '@/lib/utils';

/**
 * 骨架屏表格
 * 
 * 用于表格加载状态的骨架屏
 */
export interface SkeletonTableProps {
  rows?: number;
  columns?: number;
  avatar?: boolean;
  showHeader?: boolean;
}

export const SkeletonTable: React.FC<SkeletonTableProps> = ({
  rows = 5,
  columns = 4,
  avatar = false,
  showHeader = true,
}) => {
  return (
    <div className="skeleton-table space-y-4">
      {/* 表头 */}
      {showHeader && (
        <div className="flex gap-4 pb-3 border-b border-gray-200">
          {avatar && <div className="w-8 h-4 bg-gray-200 rounded animate-pulse" />}
          {Array.from({ length: columns }).map((_, i) => (
            <div
              key={i}
              className="flex-1 h-4 bg-gray-200 rounded animate-pulse"
            />
          ))}
        </div>
      )}
      
      {/* 表格行 */}
      {Array.from({ length: rows }).map((_, rowIndex) => (
        <Card key={rowIndex} className="shadow-sm">
          <Skeleton
            active
            avatar={avatar ? { size: 'default', shape: 'circle' } : false}
            paragraph={{ rows: Math.min(columns - 1, 2) }}
          />
        </Card>
      ))}
    </div>
  );
};

/**
 * 骨架屏卡片网格
 * 
 * 用于卡片网格布局的骨架屏
 */
export interface SkeletonCardGridProps {
  count?: number;
  columns?: 1 | 2 | 3 | 4 | 6;
  avatar?: boolean;
}

export const SkeletonCardGrid: React.FC<SkeletonCardGridProps> = ({
  count = 6,
  columns = 3,
  avatar = true,
}) => {
  const gridColsMap = {
    1: 'grid-cols-1',
    2: 'grid-cols-1 sm:grid-cols-2',
    3: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3',
    4: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-4',
    6: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6',
  };

  return (
    <div className={cn('grid gap-6', gridColsMap[columns])}>
      {Array.from({ length: count }).map((_, index) => (
        <Card key={index} className="shadow-sm">
          <Skeleton
            active
            avatar={avatar}
            paragraph={{ rows: 2 }}
          />
        </Card>
      ))}
    </div>
  );
};

/**
 * 骨架屏表单
 * 
 * 用于表单加载状态的骨架屏
 */
export interface SkeletonFormProps {
  fields?: number;
  showSubmit?: boolean;
}

export const SkeletonForm: React.FC<SkeletonFormProps> = ({
  fields = 5,
  showSubmit = true,
}) => {
  return (
    <div className="skeleton-form space-y-6">
      {Array.from({ length: fields }).map((_, index) => (
        <div key={index} className="space-y-2">
          {/* 标签 */}
          <div className="w-24 h-4 bg-gray-200 rounded animate-pulse" />
          {/* 输入框 */}
          <div className="w-full h-10 bg-gray-200 rounded animate-pulse" />
        </div>
      ))}
      
      {/* 提交按钮 */}
      {showSubmit && (
        <div className="flex gap-3 pt-4">
          <div className="w-24 h-10 bg-blue-200 rounded animate-pulse" />
          <div className="w-24 h-10 bg-gray-200 rounded animate-pulse" />
        </div>
      )}
    </div>
  );
};

/**
 * 骨架屏统计卡片
 * 
 * 用于KPI统计卡片的骨架屏
 */
export interface SkeletonStatCardProps {
  count?: number;
  columns?: 2 | 3 | 4 | 6;
}

export const SkeletonStatCard: React.FC<SkeletonStatCardProps> = ({
  count = 6,
  columns = 3,
}) => {
  const gridColsMap = {
    2: 'grid-cols-1 sm:grid-cols-2',
    3: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3',
    4: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-4',
    6: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6',
  };

  return (
    <div className={cn('grid gap-6', gridColsMap[columns])}>
      {Array.from({ length: count }).map((_, index) => (
        <Card key={index} className="shadow-sm">
          <div className="space-y-3">
            <div className="w-20 h-4 bg-gray-200 rounded animate-pulse" />
            <div className="w-32 h-8 bg-gray-200 rounded animate-pulse" />
            <div className="w-16 h-3 bg-gray-200 rounded animate-pulse" />
          </div>
        </Card>
      ))}
    </div>
  );
};

/**
 * 渐进式图片加载
 * 
 * 支持模糊占位图和懒加载
 */
export interface ProgressiveImageProps {
  src: string;
  placeholder?: string;
  alt?: string;
  width?: number | string;
  height?: number | string;
  className?: string;
  objectFit?: 'contain' | 'cover' | 'fill' | 'none' | 'scale-down';
  onLoad?: () => void;
  onError?: () => void;
}

export const ProgressiveImage: React.FC<ProgressiveImageProps> = ({
  src,
  placeholder,
  alt = '',
  width = '100%',
  height = 'auto',
  className,
  objectFit = 'cover',
  onLoad,
  onError,
}) => {
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState(false);

  const handleLoad = () => {
    setLoaded(true);
    onLoad?.();
  };

  const handleError = () => {
    setError(true);
    onError?.();
  };

  return (
    <div
      className={cn('progressive-image-wrapper', 'relative overflow-hidden', className)}
      style={{ width, height }}
    >
      {/* 占位图 - 模糊效果 */}
      {placeholder && !loaded && !error && (
        <img
          src={placeholder}
          alt={alt}
          className="absolute inset-0 w-full h-full blur-lg scale-110 transition-opacity duration-300"
          style={{ objectFit }}
        />
      )}

      {/* 骨架屏 - 无占位图时显示 */}
      {!placeholder && !loaded && !error && (
        <div className="absolute inset-0 bg-gray-200 animate-pulse" />
      )}

      {/* 主图 */}
      {!error && (
        <img
          src={src}
          alt={alt}
          loading="lazy"
          onLoad={handleLoad}
          onError={handleError}
          className={cn(
            'absolute inset-0 w-full h-full transition-opacity duration-500',
            loaded ? 'opacity-100' : 'opacity-0'
          )}
          style={{ objectFit }}
        />
      )}

      {/* 错误状态 */}
      {error && (
        <div className="absolute inset-0 flex items-center justify-center bg-gray-100 text-gray-400">
          <div className="text-center">
            <div className="text-4xl mb-2">📷</div>
            <div className="text-sm">图片加载失败</div>
          </div>
        </div>
      )}
    </div>
  );
};

/**
 * 懒加载图片
 * 
 * 使用Intersection Observer实现的懒加载图片
 */
export interface LazyImageProps extends ProgressiveImageProps {
  /**
   * 触发加载的阈值
   * @default 0.1
   */
  threshold?: number;
  
  /**
   * 根元素边距
   * @default '50px'
   */
  rootMargin?: string;
}

export const LazyImage: React.FC<LazyImageProps> = ({
  threshold = 0.1,
  rootMargin = '50px',
  ...props
}) => {
  const [inView, setInView] = useState(false);
  const imgRef = React.useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!imgRef.current) return;

    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            setInView(true);
            observer.disconnect();
          }
        });
      },
      { threshold, rootMargin }
    );

    observer.observe(imgRef.current);

    return () => observer.disconnect();
  }, [threshold, rootMargin]);

  return (
    <div ref={imgRef}>
      {inView ? (
        <ProgressiveImage {...props} />
      ) : (
        <div
          className="bg-gray-200 animate-pulse"
          style={{ width: props.width, height: props.height }}
        />
      )}
    </div>
  );
};

/**
 * 通用骨架屏包装器
 * 
 * 根据loading状态自动显示骨架屏或内容
 */
export interface SkeletonWrapperProps {
  loading: boolean;
  skeleton?: React.ReactNode;
  skeletonProps?: SkeletonProps;
  children: React.ReactNode;
}

export const SkeletonWrapper: React.FC<SkeletonWrapperProps> = ({
  loading,
  skeleton,
  skeletonProps,
  children,
}) => {
  if (!loading) return <>{children}</>;

  if (skeleton) return <>{skeleton}</>;

  return <Skeleton active {...skeletonProps} />;
};

export default SkeletonTable;

