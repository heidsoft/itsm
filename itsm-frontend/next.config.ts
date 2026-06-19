import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  reactStrictMode: true,
  output: 'standalone',

  // TODO: 逐步修复类型错误后移除此配置，启用构建时类型检查
  typescript: {
    ignoreBuildErrors: true,
  },
};

export default nextConfig;
