/**
 * 安全内容渲染组件
 * 用于安全地渲染用户输入的内容，防止 XSS 攻击
 */

'use client';

import React from 'react';
import { xssProtection, contentSecurity } from '@/lib/security';

interface SafeContentProps {
  /** 要渲染的内容 */
  content: string | null | undefined;
  /** 内容为空时显示的占位符 */
  fallback?: string;
  /** 是否启用 HTML 净化（用于富文本内容） */
  sanitizeHtml?: boolean;
  /** 最大长度限制 */
  maxLength?: number;
  /** 自定义类名 */
  className?: string;
  /** 自定义样式 */
  style?: React.CSSProperties;
  /** 是否显示为多行（保留换行） */
  preserveNewlines?: boolean;
}

/**
 * 安全内容渲染组件
 *
 * 使用示例：
 * ```tsx
 * // 基本使用
 * <SafeContent content={data.description} />
 *
 * // 带占位符
 * <SafeContent content={data.notes} fallback="暂无内容" />
 *
 * // 保留换行
 * <SafeContent content={data.message} preserveNewlines />
 *
 * // 富文本内容（需要净化 HTML）
 * <SafeContent content={richText} sanitizeHtml />
 * ```
 */
export const SafeContent: React.FC<SafeContentProps> = ({
  content,
  fallback = '',
  sanitizeHtml = false,
  maxLength,
  className,
  style,
  preserveNewlines = false,
}) => {
  // 处理空值
  if (content === null || content === undefined || content === '') {
    return (
      <span className={className} style={{ color: 'var(--color-text-muted, #8c8c8c)', ...style }}>
        {fallback}
      </span>
    );
  }

  // 处理内容
  let processedContent = String(content);

  // 应用长度限制
  if (maxLength && processedContent.length > maxLength) {
    processedContent = processedContent.substring(0, maxLength) + '...';
  }

  // 安全处理
  let safeContent: string;
  if (sanitizeHtml) {
    // 净化 HTML（移除危险标签和属性）
    safeContent = contentSecurity.sanitizeContent(processedContent);
  } else {
    // 转义 HTML（默认行为）
    safeContent = xssProtection.escapeHtml(processedContent);
  }

  // 渲染
  if (preserveNewlines) {
    // 将换行符转换为 <br />
    const lines = safeContent.split('\n');
    return (
      <span className={className} style={style}>
        {lines.map((line, index) => (
          <React.Fragment key={index}>
            {line}
            {index < lines.length - 1 && <br />}
          </React.Fragment>
        ))}
      </span>
    );
  }

  return (
    <span
      className={className}
      style={style}
      dangerouslySetInnerHTML={{ __html: safeContent }}
    />
  );
};

/**
 * 安全文本块组件（用于多行内容）
 */
export const SafeTextBlock: React.FC<
  SafeContentProps & {
    /** 是否显示为段落 */
    asParagraph?: boolean;
  }
> = ({ content, fallback = '暂无内容', asParagraph = true, ...props }) => {
  if (content === null || content === undefined || content === '') {
    return (
      <span style={{ color: 'var(--color-text-muted, #8c8c8c)' }}>
        {fallback}
      </span>
    );
  }

  const safeContent = xssProtection.escapeHtml(String(content));
  const lines = safeContent.split('\n');

  if (asParagraph) {
    return (
      <div className={props.className} style={props.style}>
        {lines.map((line, index) => (
          <p key={index} style={{ marginBottom: index < lines.length - 1 ? '0.5em' : 0 }}>
            {line || '\u00A0'}
          </p>
        ))}
      </div>
    );
  }

  return (
    <SafeContent content={content} preserveNewlines fallback={fallback} {...props} />
  );
};

/**
 * 安全描述项组件（用于 Descriptions）
 */
export const SafeDescriptionItem: React.FC<{
  label: string;
  content: string | null | undefined;
  fallback?: string;
}> = ({ label, content, fallback = '-' }) => {
  return (
    <span>
      <SafeContent content={content} fallback={fallback} />
    </span>
  );
};

export default SafeContent;
