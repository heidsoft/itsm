import type { Metadata, Viewport } from 'next';
import Script from 'next/script';
// 禁用 Google Fonts (build 离线环境) - 改用系统字体
// import { Inter, Noto_Sans_SC } from 'next/font/google';
import './globals.css';
import { AntdProvider } from '@/lib/providers/AntdProvider';
import { ThemeProvider, ThemeConfig, useTheme } from '@/lib/design-system/theme';
import { RecentVisitTracker } from '@/components/layout/RecentVisitTracker';
import { ThemeHtmlClassSync } from '@/components/layout/ThemeHtmlClassSync';
import { DayjsLocaleSync } from '@/components/layout/DayjsLocaleSync';
import ErrorBoundary from '@/components/common/ErrorBoundary';
import { QueryProvider } from '@/lib/providers/QueryProvider';
import GlobalShortcutProvider from '@/components/common/GlobalShortcutProvider';

// 使用系统字体代替 Google Fonts
const inter = { variable: '--font-inter' };
const notoSansSC = { variable: '--font-noto-sans-sc' };

export const metadata: Metadata = {
  title: 'AI-Native ITSM - AI-driven IT service management system',
  description: 'AI-Native ITSM is an open source AI-driven IT service management system that provides core functions such as work order management, CMDB, knowledge base RAG, BPMN workflow, SLA monitoring, and AI-assisted triage.',
  keywords: 'ITSM, AI, work order management, CMDB, knowledge base, BPMN, SLA, IT service management, open source',
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
    title: 'AI-Native ITSM - AI-driven IT service management system',
    description: 'AI-Native ITSM is an open source AI-driven IT service management system that provides core functions such as work order management, CMDB, knowledge base RAG, BPMN workflow, SLA monitoring, and AI-assisted triage.',
    type: 'website',
    locale: 'en_US',
    siteName: 'AI-Native ITSM',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'AI-Native ITSM - AI-driven IT service management system',
    description: 'AI-Native ITSM is an open source AI-driven IT service management system that provides core functions such as work order management, CMDB, knowledge base RAG, BPMN workflow, SLA monitoring, and AI-assisted triage.',
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
    <html lang="en-US" suppressHydrationWarning>
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
          <RecentVisitTracker />
          <ThemeHtmlClassSync />
          <DayjsLocaleSync />
          <ThemeConfig>
            <AntdProvider>
              <QueryProvider>
                <GlobalShortcutProvider><ErrorBoundary>{children}</ErrorBoundary></GlobalShortcutProvider>
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
