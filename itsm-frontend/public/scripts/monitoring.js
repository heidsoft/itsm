// 性能监控
if ('performance' in window) {
  window.addEventListener('load', function () {
    try {
      var perfData = performance.getEntriesByType('navigation')[0];
      if (perfData) {
        var metrics = {
          'DNS查询':
            perfData.domainLookupEnd - perfData.domainLookupStart + 'ms',
          'TCP连接': perfData.connectEnd - perfData.connectStart + 'ms',
          '请求响应': perfData.responseEnd - perfData.requestStart + 'ms',
          'DOM解析':
            perfData.domContentLoadedEventEnd -
            perfData.domContentLoadedEventStart +
            'ms',
          '页面完全加载':
            perfData.loadEventEnd - perfData.loadEventStart + 'ms',
          '总加载时间': perfData.loadEventEnd - perfData.fetchStart + 'ms',
        };
        // 生产环境使用 console.debug（默认不显示），开发环境使用 console.log
        if (process.env.NODE_ENV !== 'production') {
          console.log('性能指标:', metrics);
        } else {
          console.debug('性能指标:', metrics);
        }
      }
    } catch (e) {
      // Ignore monitoring errors
    }
  });
}

// 错误监控
window.addEventListener('error', function (event) {
  // 安全过滤：避免敏感信息泄露
  var safeError = {
    message: event.message,
    filename: event.filename ? event.filename.split('/').pop() : 'unknown',
    lineno: event.lineno,
    colno: event.colno
  };
  console.error('全局错误:', safeError);
});

// 未处理的Promise拒绝
window.addEventListener('unhandledrejection', function () {
  console.error('未处理的Promise拒绝');
});
