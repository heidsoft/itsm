'use client';

import React, { useEffect } from 'react';

/**
 * 根级错误边界 (global-error.tsx)
 *
 * 关键约束：不能使用 Ant Design 或任何依赖 root layout Provider 的组件，
 * 因为 root layout 可能已经挂掉。这里只使用原生 HTML + 内联样式。
 */
export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error('[GlobalError]', error);
  }, [error]);

  const containerStyle: React.CSSProperties = {
    minHeight: '100vh',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background: 'linear-gradient(135deg, #f5f7fa 0%, #e9ecef 100%)',
    padding: '24px',
    fontFamily:
      "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
  };

  const cardStyle: React.CSSProperties = {
    maxWidth: '480px',
    width: '100%',
    background: '#ffffff',
    borderRadius: '12px',
    padding: '40px',
    textAlign: 'center',
    boxShadow: '0 8px 32px rgba(0, 0, 0, 0.1)',
  };

  const iconStyle: React.CSSProperties = {
    width: '64px',
    height: '64px',
    margin: '0 auto 24px',
    background: '#fff2f0',
    borderRadius: '50%',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    fontSize: '32px',
  };

  const titleStyle: React.CSSProperties = {
    fontSize: '24px',
    fontWeight: 700,
    color: '#1f2937',
    marginBottom: '12px',
  };

  const subtitleStyle: React.CSSProperties = {
    fontSize: '15px',
    color: '#6b7280',
    lineHeight: '1.6',
    marginBottom: '32px',
  };

  const buttonContainerStyle: React.CSSProperties = {
    display: 'flex',
    gap: '12px',
    justifyContent: 'center',
    flexWrap: 'wrap',
  };

  const primaryButtonStyle: React.CSSProperties = {
    display: 'inline-flex',
    alignItems: 'center',
    gap: '8px',
    background: '#3b82f6',
    color: '#ffffff',
    border: 'none',
    borderRadius: '8px',
    padding: '10px 24px',
    fontSize: '14px',
    fontWeight: 500,
    cursor: 'pointer',
    transition: 'background 0.2s',
  };

  const secondaryButtonStyle: React.CSSProperties = {
    display: 'inline-flex',
    alignItems: 'center',
    gap: '8px',
    background: '#ffffff',
    color: '#374151',
    border: '1px solid #d1d5db',
    borderRadius: '8px',
    padding: '10px 24px',
    fontSize: '14px',
    fontWeight: 500,
    cursor: 'pointer',
    transition: 'background 0.2s',
  };

  const errorIdStyle: React.CSSProperties = {
    marginTop: '24px',
    fontSize: '12px',
    color: '#9ca3af',
  };

  return (
    <html>
      <body>
        <div style={containerStyle}>
          <div style={cardStyle}>
            <div style={iconStyle}>⚠️</div>
            <h1 style={titleStyle}>System error occurred</h1>
            <p style={subtitleStyle}>

              Sorry, the application encountered a critical error. Please try again or contact technical support if the problem persists.
            </p>
            <div style={buttonContainerStyle}>
              <button style={primaryButtonStyle} onClick={() => reset()}>

                🔄 Try again
              </button>
              <button
                style={secondaryButtonStyle}
                onClick={() => {
                  window.location.href = '/dashboard';
                }}
              >

                🏠 Return to dashboard
              </button>
            </div>
            {error?.digest && (
              <div style={errorIdStyle}>Error ID: {error.digest}</div>
            )}
          </div>
        </div>
      </body>
    </html>
  );
}
