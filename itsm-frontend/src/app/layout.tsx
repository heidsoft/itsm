import type { Metadata } from 'next';
import { Inter, Noto_Sans_SC } from 'next/font/google';
import './globals.css';
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import { AuthGuard } from '@/components/auth/AuthGuard';
import { AntdProvider } from '@/app/lib/providers/AntdProvider';

dayjs.locale('zh-cn');
import ErrorBoundary from '@/components/common/ErrorBoundary';
import { QueryProvider } from '@/app/lib/providers/QueryProvider';

// 使用更适合中文的字体组合
const inter = Inter({
  variable: '--font-inter',
  subsets: ['latin'],
  display: 'swap',
});

const notoSansSC = Noto_Sans_SC({
  variable: '--font-noto-sans-sc',
  subsets: ['latin'],
  weight: ['400', '500', '600', '700'],
  display: 'swap',
});

export const metadata: Metadata = {
  title: 'ITSM Platform - IT服务管理平台',
  description: '专业的IT服务管理平台，提供工单管理、事件管理、问题管理、变更管理等核心功能',
  keywords: 'ITSM, 工单管理, 事件管理, 问题管理, 变更管理, IT服务管理',
  authors: [{ name: 'ITSM Team' }],
  creator: 'ITSM Platform',
  publisher: 'ITSM Platform',
  formatDetection: {
    email: false,
    address: false,
    telephone: false,
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      'max-video-preview': -1,
      'max-image-preview': 'large',
      'max-snippet': -1,
    },
  },
  openGraph: {
    title: 'ITSM Platform - IT服务管理平台',
    description: '专业的IT服务管理平台，提供工单管理、事件管理、问题管理、变更管理等核心功能',
    type: 'website',
    locale: 'zh_CN',
    siteName: 'ITSM Platform',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'ITSM Platform - IT服务管理平台',
    description: '专业的IT服务管理平台，提供工单管理、事件管理、问题管理、变更管理等核心功能',
  },
  icons: {
    icon: '/favicon.ico',
    shortcut: '/favicon-16x16.png',
    apple: '/apple-touch-icon.png',
  },
  manifest: '/site.webmanifest',
};

export const viewport = {
  width: 'device-width',
  initialScale: 1,
  maximumScale: 1,
  userScalable: false,
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang='zh-CN' suppressHydrationWarning>
      <head>
        {/* 预加载关键资源 */}
        <link rel='preconnect' href='https://fonts.googleapis.com' />
        <link rel='preconnect' href='https://fonts.gstatic.com' crossOrigin='anonymous' />

        {/* 安全相关头部 */}
        <meta httpEquiv='X-UA-Compatible' content='IE=edge' />
        <meta name='theme-color' content='#1890ff' />

        {/* PWA相关 */}
        <meta name='mobile-web-app-capable' content='yes' />
        <meta name='apple-mobile-web-app-capable' content='yes' />
        <meta name='apple-mobile-web-app-status-bar-style' content='default' />
        <meta name='apple-mobile-web-app-title' content='ITSM Platform' />

        {/* 性能优化 */}
        <link rel='dns-prefetch' href='//fonts.googleapis.com' />
        <link rel='dns-prefetch' href='//fonts.gstatic.com' />
      </head>
      <body
        className={`${inter.variable} ${notoSansSC.variable} antialiased`}
        style={{
          fontFamily: `var(--font-noto-sans-sc), var(--font-inter), -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, 'Noto Sans', sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol', 'Noto Color Emoji'`,
        }}
      >
        <AntdProvider>
          <QueryProvider>
            <ErrorBoundary>{children}</ErrorBoundary>
          </QueryProvider>
        </AntdProvider>

        {/* 性能监控脚本 */}
        <script
          dangerouslySetInnerHTML={{
            __html: `
              // 性能监控
              if ('performance' in window) {
                window.addEventListener('load', () => {
                  const perfData = performance.getEntriesByType('navigation')[0];
                  if (perfData) {
                    console.log('页面加载性能:', {
                      'DNS查询': perfData.domainLookupEnd - perfData.domainLookupStart + 'ms',
                      'TCP连接': perfData.connectEnd - perfData.connectStart + 'ms',
                      '请求响应': perfData.responseEnd - perfData.requestStart + 'ms',
                      'DOM解析': perfData.domContentLoadedEventEnd - perfData.domContentLoadedEventStart + 'ms',
                      '页面完全加载': perfData.loadEventEnd - perfData.loadEventStart + 'ms',
                      '总加载时间': perfData.loadEventEnd - perfData.fetchStart + 'ms'
                    });
                  }
                });
              }
              
              // 错误监控
              window.addEventListener('error', (event) => {
                console.error('全局错误:', {
                  message: event.message,
                  filename: event.filename,
                  lineno: event.lineno,
                  colno: event.colno,
                  error: event.error
                });
              });
              
              // 未处理的Promise拒绝
              window.addEventListener('unhandledrejection', (event) => {
                console.error('未处理的Promise拒绝:', event.reason);
              });
            `,
          }}
        />
      </body>
    </html>
  );
}
