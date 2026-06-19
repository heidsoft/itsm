import type { Metadata, Viewport } from 'next';
import Script from 'next/script';
// 禁用 Google Fonts (build 离线环境) - 改用系统字体
// import { Inter, Noto_Sans_SC } from 'next/font/google';
import './globals.css';
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import { AuthGuard } from '@/components/auth/AuthGuard';
import { AntdProvider } from '@/lib/providers/AntdProvider';
import { ThemeProvider, ThemeConfig } from '@/lib/design-system/theme';

dayjs.locale('zh-cn');
import ErrorBoundary from '@/components/common/ErrorBoundary';
import { QueryProvider } from '@/lib/providers/QueryProvider';

// 使用系统字体代替 Google Fonts
const inter = { variable: '--font-inter' };
const notoSansSC = { variable: '--font-noto-sans-sc' };

export const metadata: Metadata = {
  title: 'AI-Native ITSM - AI驱动的IT服务管理系统',
  description: 'AI-Native ITSM 是一款开源的AI驱动IT服务管理系统，提供工单管理、CMDB、知识库RAG、BPMN工作流、SLA监控、AI智能分诊等核心功能',
  keywords: 'ITSM, AI, 工单管理, CMDB, 知识库, BPMN, SLA, IT服务管理, 开源',
  authors: [{ name: 'AI-Native ITSM Team' }],
  creator: 'AI-Native ITSM',
  publisher: 'AI-Native ITSM',
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
    title: 'AI-Native ITSM - AI驱动的IT服务管理系统',
    description: 'AI-Native ITSM 是一款开源的AI驱动IT服务管理系统，提供工单管理、CMDB、知识库RAG、BPMN工作流、SLA监控、AI智能分诊等核心功能',
    type: 'website',
    locale: 'zh_CN',
    siteName: 'AI-Native ITSM',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'AI-Native ITSM - AI驱动的IT服务管理系统',
    description: 'AI-Native ITSM 是一款开源的AI驱动IT服务管理系统，提供工单管理、CMDB、知识库RAG、BPMN工作流、SLA监控、AI智能分诊等核心功能',
  },
  icons: {
    icon: '/file.svg',
    shortcut: '/file.svg',
    apple: '/file.svg',
  },
  manifest: '/site.webmanifest',
};

export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  maximumScale: 5,
  userScalable: true,
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN" suppressHydrationWarning>
      <head>
        {/* 安全相关头部 */}
        <meta httpEquiv="X-UA-Compatible" content="IE=edge" />
        <meta name="theme-color" content="#1890ff" />

        {/* PWA相关 */}
        <meta name="mobile-web-app-capable" content="yes" />
        <meta name="apple-mobile-web-app-capable" content="yes" />
        <meta name="apple-mobile-web-app-status-bar-style" content="default" />
        <meta name="apple-mobile-web-app-title" content="AI-Native ITSM" />
      </head>
      <body
        className={`${inter.variable} ${notoSansSC.variable} antialiased`}
        style={{
          fontFamily: `var(--font-noto-sans-sc), var(--font-inter), -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, 'Noto Sans', sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol', 'Noto Color Emoji'`,
        }}
      >
        <ThemeProvider>
          <ThemeConfig>
            <AntdProvider>
              <QueryProvider>
                <ErrorBoundary>{children}</ErrorBoundary>
              </QueryProvider>
            </AntdProvider>
          </ThemeConfig>
        </ThemeProvider>

        {/* 性能监控脚本 */}
        <Script src="/scripts/monitoring.js" strategy="afterInteractive" />
      </body>
    </html>
  );
}
