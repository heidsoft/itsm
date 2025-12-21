'use client';

import React, { useState, useEffect, useRef } from 'react';
import { Skeleton } from 'antd';
import { FileImageOutlined } from '@ant-design/icons';

interface LazyImageProps {
  src: string;
  alt: string;
  width?: number | string;
  height?: number | string;
  className?: string;
  placeholder?: React.ReactNode;
  fallback?: React.ReactNode;
  threshold?: number;
  rootMargin?: string;
  onLoad?: () => void;
  onError?: (error: Error) => void;
  style?: React.CSSProperties;
}

/**
 * 懒加载图片组件
 * 使用Intersection Observer API实现高性能图片懒加载
 */
export const LazyImage: React.FC<LazyImageProps> = ({
  src,
  alt,
  width,
  height,
  className = '',
  placeholder,
  fallback,
  threshold = 0.1,
  rootMargin = '50px',
  onLoad,
  onError,
  style,
}) => {
  const [isLoaded, setIsLoaded] = useState(false);
  const [isInView, setIsInView] = useState(false);
  const [hasError, setHasError] = useState(false);
  const imgRef = useRef<HTMLImageElement>(null);
  const observerRef = useRef<IntersectionObserver | null>(null);

  useEffect(() => {
    // 检查浏览器是否支持Intersection Observer
    if (!('IntersectionObserver' in window)) {
      // 如果不支持，直接加载图片
      setIsInView(true);
      return;
    }

    const currentImg = imgRef.current;
    if (!currentImg) return;

    // 创建Intersection Observer
    observerRef.current = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            setIsInView(true);
            // 图片进入视口后，停止观察
            if (observerRef.current && currentImg) {
              observerRef.current.unobserve(currentImg);
            }
          }
        });
      },
      {
        threshold,
        rootMargin,
      }
    );

    // 开始观察
    observerRef.current.observe(currentImg);

    // 清理函数
    return () => {
      if (observerRef.current && currentImg) {
        observerRef.current.unobserve(currentImg);
      }
    };
  }, [threshold, rootMargin]);

  const handleLoad = () => {
    setIsLoaded(true);
    if (onLoad) {
      onLoad();
    }
  };

  const handleError = () => {
    setHasError(true);
    if (onError) {
      onError(new Error(`Failed to load image: ${src}`));
    }
  };

  // 默认占位符
  const defaultPlaceholder = (
    <Skeleton.Image
      active
      style={{ width: width || '100%', height: height || 'auto' }}
    />
  );

  // 默认错误回退
  const defaultFallback = (
    <div
      style={{
        width: width || '100%',
        height: height || 200,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#f0f0f0',
        color: '#999',
        borderRadius: 4,
        ...style,
      }}
      className={className}
    >
      <div style={{ textAlign: 'center' }}>
        <FileImageOutlined style={{ fontSize: 48, marginBottom: 8 }} />
        <div style={{ fontSize: 12 }}>图片加载失败</div>
      </div>
    </div>
  );

  // 显示错误状态
  if (hasError) {
    return <>{fallback || defaultFallback}</>;
  }

  // 显示占位符（图片未进入视口或未加载完成）
  if (!isInView || !isLoaded) {
    return (
      <div
        ref={imgRef}
        style={{
          width: width || '100%',
          height: height || 'auto',
          ...style,
        }}
        className={className}
      >
        {!isLoaded && (placeholder || defaultPlaceholder)}
        {isInView && (
          <img
            src={src}
            alt={alt}
            onLoad={handleLoad}
            onError={handleError}
            style={{
              ...style,
              display: isLoaded ? 'block' : 'none',
              width: width || '100%',
              height: height || 'auto',
            }}
            className={className}
          />
        )}
      </div>
    );
  }

  // 显示已加载的图片
  return (
    <img
      ref={imgRef}
      src={src}
      alt={alt}
      style={{
        ...style,
        width: width || '100%',
        height: height || 'auto',
      }}
      className={className}
    />
  );
};

/**
 * 响应式懒加载图片组件
 * 支持根据屏幕大小加载不同尺寸的图片
 */
interface ResponsiveLazyImageProps extends Omit<LazyImageProps, 'src'> {
  srcSet: {
    sm?: string;
    md?: string;
    lg?: string;
    xl?: string;
  };
  defaultSrc: string;
}

export const ResponsiveLazyImage: React.FC<ResponsiveLazyImageProps> = ({
  srcSet,
  defaultSrc,
  ...props
}) => {
  const [currentSrc, setCurrentSrc] = useState(defaultSrc);

  useEffect(() => {
    const updateSrc = () => {
      const width = window.innerWidth;

      if (width < 640 && srcSet.sm) {
        setCurrentSrc(srcSet.sm);
      } else if (width < 768 && srcSet.md) {
        setCurrentSrc(srcSet.md);
      } else if (width < 1024 && srcSet.lg) {
        setCurrentSrc(srcSet.lg);
      } else if (srcSet.xl) {
        setCurrentSrc(srcSet.xl);
      } else {
        setCurrentSrc(defaultSrc);
      }
    };

    updateSrc();

    window.addEventListener('resize', updateSrc);
    return () => window.removeEventListener('resize', updateSrc);
  }, [srcSet, defaultSrc]);

  return <LazyImage {...props} src={currentSrc} />;
};

/**
 * 懒加载背景图片组件
 */
interface LazyBackgroundImageProps {
  src: string;
  children?: React.ReactNode;
  className?: string;
  style?: React.CSSProperties;
  threshold?: number;
  rootMargin?: string;
}

export const LazyBackgroundImage: React.FC<LazyBackgroundImageProps> = ({
  src,
  children,
  className = '',
  style,
  threshold = 0.1,
  rootMargin = '50px',
}) => {
  const [isInView, setIsInView] = useState(false);
  const [isLoaded, setIsLoaded] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!('IntersectionObserver' in window)) {
      setIsInView(true);
      return;
    }

    const currentContainer = containerRef.current;
    if (!currentContainer) return;

    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            setIsInView(true);
            observer.unobserve(currentContainer);
          }
        });
      },
      {
        threshold,
        rootMargin,
      }
    );

    observer.observe(currentContainer);

    return () => {
      observer.unobserve(currentContainer);
    };
  }, [threshold, rootMargin]);

  useEffect(() => {
    if (isInView && !isLoaded) {
      const img = new Image();
      img.src = src;
      img.onload = () => setIsLoaded(true);
    }
  }, [isInView, isLoaded, src]);

  return (
    <div
      ref={containerRef}
      className={className}
      style={{
        ...style,
        backgroundImage: isLoaded ? `url(${src})` : 'none',
        backgroundSize: 'cover',
        backgroundPosition: 'center',
        transition: 'background-image 0.3s ease-in-out',
      }}
    >
      {children}
    </div>
  );
};

export default LazyImage;

