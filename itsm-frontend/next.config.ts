import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // 可以关闭的功能
  poweredByHeader: false,        // 关闭 X-Powered-By 头
  reactStrictMode: true,         // 开启/关闭严格模式
  
  // Docker部署配置
  output: 'standalone',
  
  // 修复后的开发指示器配置
  devIndicators: {
    position: 'bottom-right',     // 新的位置配置选项
  },
  
  // 实验性功能开关
  experimental: {
    // 关闭开发者偏好设置面板
    clientRouterFilter: false,
    // 优化包导入 - 减少打包体积
    optimizePackageImports: ['antd', '@ant-design/icons', 'lucide-react'],
  },
  
  // 图片优化配置
  images: {
    unoptimized: false,         // 关闭图片优化
  },
  
  // 开发模式配置
  eslint: {
    ignoreDuringBuilds: true,  // 构建时忽略 ESLint 错误
  },
  
  typescript: {
    ignoreBuildErrors: true,   // 构建时忽略 TypeScript 错误
  },

  // 配置反向代理，解决跨域问题
  async rewrites() {
    return [
      {
        source: '/api/v1/:path*',
        destination: 'http://localhost:8090/api/v1/:path*',
      },
      {
        source: '/api/:path*',
        destination: 'http://localhost:8090/api/:path*',
      },
    ];
  },
};

export default nextConfig;
