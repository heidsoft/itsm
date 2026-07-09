import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  reactStrictMode: true,

  // 图片优化配置
  images: {
    formats: ['image/webp', 'image/avif'],
    remotePatterns: [
      {
        protocol: 'https',
        hostname: '**',
      },
      {
        protocol: 'http',
        hostname: '**',
      },
    ],
    minimumCacheTTL: 86400, // 缓存 1 天
  },
  output: 'standalone',
  outputFileTracingRoot: process.cwd(),

  // P0 修复：强制 TypeScript 与 ESLint 在构建期暴露错误，避免缺 chunk 白屏雪崩。
  // 若需临时跳过某些历史告警，请在受影响的文件顶部用 `// @ts-expect-error <reason>` 或 `// eslint-disable-next-line <rule>` 局部处理。
  typescript: {
    ignoreBuildErrors: false,
  },
  eslint: {
    ignoreDuringBuilds: false,
  },
};

export default nextConfig;
