
module.exports = {
  content: [
    './src/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {
      // 可以移除不需要的动画
      // animation: {},
      // keyframes: {},
    },
  },
  plugins: [],
  // 关闭某些功能
  corePlugins: {
    preflight: true,    // 关闭基础样式重置
    container: true,    // 关闭容器类
  }
};
