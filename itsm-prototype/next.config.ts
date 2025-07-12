import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // 可以关闭的功能
  poweredByHeader: false,        // 关闭 X-Powered-By 头
  reactStrictMode: true,         // 开启/关闭严格模式
  
  // 修复后的开发指示器配置
  devIndicators: {
    // buildActivity: false,        // 已移除：此选项已弃用
    // buildActivityPosition: 'bottom-right',  // 已移除：已重命名为 position
    position: 'bottom-right',     // 新的位置配置选项
  },
  
  // serverExternalPackages 替代了 experimental.serverComponentsExternalPackages
  serverExternalPackages: [],
  
  // 实验性功能开关
  experimental: {
    // 关闭开发者偏好设置面板
    clientRouterFilter: false,
  },
  
  // 图片优化配置
  images: {
    unoptimized: false,         // 关闭图片优化
  },
  
  // 开发模式配置
  eslint: {
    ignoreDuringBuilds: false,  // 构建时忽略 ESLint 错误
  },
  
  typescript: {
    ignoreBuildErrors: false,   // 构建时忽略 TypeScript 错误
  }
};

export default nextConfig;
