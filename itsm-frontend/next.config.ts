import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  reactStrictMode: true,
  output: 'standalone',
  outputFileTracingRoot: process.cwd(),

  // TODO: 逐步修复类型错误后移除此配置，启用构建时类型检查
  typescript: {
    ignoreBuildErrors: true,
  },
  // TODO: 修复历史 ESLint 警告后移除此配置。Release 构建不应被已有 lint 债务阻塞。
  eslint: {
    ignoreDuringBuilds: true,
  },
};

export default nextConfig;
