import React from 'react';
import Image from 'next/image';
import type { ImageProps } from 'next/image';

interface AppImageProps extends Omit<ImageProps, 'alt'> {
  alt: string;
  /** 是否启用懒加载，默认 true */
  lazy?: boolean;
  /** 加载失败时的占位图 */
  fallbackSrc?: string;
}

export default function AppImage({
  alt,
  lazy = true,
  fallbackSrc = '/images/placeholder.png',
  quality = 80,
  priority = false,
  ...props
}: AppImageProps) {
  const [imgSrc, setImgSrc] = React.useState(props.src);

  const handleError = () => {
    setImgSrc(fallbackSrc);
  };

  return (
    <Image
      alt={alt}
      quality={quality}
      priority={priority}
      loading={lazy && !priority ? 'lazy' : 'eager'}
      onError={handleError}
      {...props}
      src={imgSrc}
    />
  );
}
