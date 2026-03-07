import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  reactStrictMode: true,
  // 使用环境变量控制输出模式：
  // - 'export' 用于 GitHub Pages 静态部署
  // - 'standalone' 用于 Docker/Node.js 部署
  output: process.env.NEXT_OUTPUT_MODE === 'standalone' ? 'standalone' : 'export',
  
  // 忽略错误
  eslint: {
    ignoreDuringBuilds: true,
  },
  typescript: {
    ignoreBuildErrors: true,
  },
  
  // webpack 优化
  webpack: (config, { isServer }) => {
    if (!isServer) {
      config.optimization = {
        ...config.optimization,
        splitChunks: {
          chunks: 'all',
          cacheGroups: {
            default: false,
            vendors: false,
          },
        },
      };
    }
    return config;
  },
};

export default nextConfig;
