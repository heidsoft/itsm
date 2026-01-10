// @ts-nocheck
/**
 * ITSM前端性能优化 - 性能监控和报告
 *
 * 实现性能监控、指标收集、报告生成等功能
 */

import { useState, useEffect, useCallback, useRef } from 'react';

// ==================== 性能指标类型 ====================

export interface PerformanceMetrics {
  // 页面加载指标
  loadTime: number;
  domContentLoaded: number;
  firstContentfulPaint: number;
  largestContentfulPaint: number;
  firstInputDelay: number;
  cumulativeLayoutShift: number;

  // 资源加载指标
  resourceLoadTime: number;
  resourceCount: number;
  resourceSize: number;

  // 内存使用指标
  memoryUsage: number;
  memoryLimit: number;

  // 网络指标
  networkLatency: number;
  networkThroughput: number;

  // 用户交互指标
  clickResponseTime: number;
  scrollResponseTime: number;

  // 自定义指标
  customMetrics: Record<string, number>;
}

export interface PerformanceReport {
  timestamp: number;
  url: string;
  userAgent: string;
  metrics: PerformanceMetrics;
  errors: Array<{
    message: string;
    stack?: string;
    timestamp: number;
  }>;
  warnings: Array<{
    message: string;
    timestamp: number;
  }>;
}

// ==================== 性能监控器 ====================

export class PerformanceMonitor {
  private metrics: PerformanceMetrics;
  private errors: Array<{ message: string; stack?: string; timestamp: number }> = [];
  private warnings: Array<{ message: string; timestamp: number }> = [];
  private observers: Map<string, PerformanceObserver> = new Map();
  private customMetrics: Map<string, number> = new Map();
  private isMonitoring: boolean = false;

  constructor() {
    this.metrics = this.initializeMetrics();
  }

  private initializeMetrics(): PerformanceMetrics {
    return {
      loadTime: 0,
      domContentLoaded: 0,
      firstContentfulPaint: 0,
      largestContentfulPaint: 0,
      firstInputDelay: 0,
      cumulativeLayoutShift: 0,
      resourceLoadTime: 0,
      resourceCount: 0,
      resourceSize: 0,
      memoryUsage: 0,
      memoryLimit: 0,
      networkLatency: 0,
      networkThroughput: 0,
      clickResponseTime: 0,
      scrollResponseTime: 0,
      customMetrics: {},
    };
  }

  /**
   * 开始监控
   */
  startMonitoring(): void {
    if (this.isMonitoring) {
      return;
    }

    this.isMonitoring = true;
    this.setupPerformanceObservers();
    this.setupErrorHandlers();
    this.setupNetworkMonitoring();
    this.setupMemoryMonitoring();
    this.setupUserInteractionMonitoring();
  }

  /**
   * 停止监控
   */
  stopMonitoring(): void {
    this.isMonitoring = false;
    this.observers.forEach(observer => observer.disconnect());
    this.observers.clear();
  }

  /**
   * 设置性能观察器
   */
  private setupPerformanceObservers(): void {
    // 监控导航时间
    if ('PerformanceObserver' in window) {
      const navObserver = new PerformanceObserver(list => {
        const entries = list.getEntries();
        entries.forEach(entry => {
          if (entry.entryType === 'navigation') {
            const navEntry = entry as PerformanceNavigationTiming;
            this.metrics.loadTime = navEntry.loadEventEnd - navEntry.loadEventStart;
            this.metrics.domContentLoaded =
              navEntry.domContentLoadedEventEnd - navEntry.domContentLoadedEventStart;
          }
        });
      });
      navObserver.observe({ entryTypes: ['navigation'] });
      this.observers.set('navigation', navObserver);
    }

    // 监控绘制指标
    if ('PerformanceObserver' in window) {
      const paintObserver = new PerformanceObserver(list => {
        const entries = list.getEntries();
        entries.forEach(entry => {
          if (entry.name === 'first-contentful-paint') {
            this.metrics.firstContentfulPaint = entry.startTime;
          }
        });
      });
      paintObserver.observe({ entryTypes: ['paint'] });
      this.observers.set('paint', paintObserver);
    }

    // 监控LCP
    if ('PerformanceObserver' in window) {
      const lcpObserver = new PerformanceObserver(list => {
        const entries = list.getEntries();
        const lastEntry = entries[entries.length - 1];
        this.metrics.largestContentfulPaint = lastEntry.startTime;
      });
      lcpObserver.observe({ entryTypes: ['largest-contentful-paint'] });
      this.observers.set('lcp', lcpObserver);
    }

    // 监控FID
    if ('PerformanceObserver' in window) {
      const fidObserver = new PerformanceObserver(list => {
        const entries = list.getEntries();
        entries.forEach(entry => {
          this.metrics.firstInputDelay = entry.processingStart - entry.startTime;
        });
      });
      fidObserver.observe({ entryTypes: ['first-input'] });
      this.observers.set('fid', fidObserver);
    }

    // 监控CLS
    if ('PerformanceObserver' in window) {
      let clsValue = 0;
      const clsObserver = new PerformanceObserver(list => {
        const entries = list.getEntries();
        entries.forEach((entry: any) => {
          if (!entry.hadRecentInput) {
            clsValue += entry.value;
          }
        });
        this.metrics.cumulativeLayoutShift = clsValue;
      });
      clsObserver.observe({ entryTypes: ['layout-shift'] });
      this.observers.set('cls', clsObserver);
    }

    // 监控资源加载
    if ('PerformanceObserver' in window) {
      const resourceObserver = new PerformanceObserver(list => {
        const entries = list.getEntries();
        let totalLoadTime = 0;
        let totalSize = 0;

        entries.forEach(entry => {
          totalLoadTime += entry.duration;
          totalSize += (entry as any).transferSize || 0;
        });

        this.metrics.resourceLoadTime = totalLoadTime;
        this.metrics.resourceSize = totalSize;
        this.metrics.resourceCount = entries.length;
      });
      resourceObserver.observe({ entryTypes: ['resource'] });
      this.observers.set('resource', resourceObserver);
    }
  }

  /**
   * 设置错误处理器
   */
  private setupErrorHandlers(): void {
    // 全局错误处理
    window.addEventListener('error', event => {
      this.errors.push({
        message: event.message,
        stack: event.error?.stack,
        timestamp: Date.now(),
      });
    });

    // Promise错误处理
    window.addEventListener('unhandledrejection', event => {
      this.errors.push({
        message: `Unhandled Promise Rejection: ${event.reason}`,
        stack: event.reason?.stack,
        timestamp: Date.now(),
      });
    });

    // 控制台警告
    const originalWarn = console.warn;
    console.warn = (...args) => {
      this.warnings.push({
        message: args.join(' '),
        timestamp: Date.now(),
      });
      originalWarn.apply(console, args);
    };
  }

  /**
   * 设置网络监控
   */
  private setupNetworkMonitoring(): void {
    if ('connection' in navigator) {
      const connection = (navigator as any).connection;
      this.metrics.networkLatency = connection.rtt || 0;
      this.metrics.networkThroughput = connection.downlink || 0;
    }
  }

  /**
   * 设置内存监控
   */
  private setupMemoryMonitoring(): void {
    if ('memory' in performance) {
      const memory = (performance as any).memory;
      this.metrics.memoryUsage = memory.usedJSHeapSize;
      this.metrics.memoryLimit = memory.jsHeapSizeLimit;
    }
  }

  /**
   * 设置用户交互监控
   */
  private setupUserInteractionMonitoring(): void {
    let clickStartTime = 0;
    let scrollStartTime = 0;

    // 监控点击响应时间
    document.addEventListener('click', () => {
      clickStartTime = performance.now();
    });

    document.addEventListener('click', () => {
      if (clickStartTime > 0) {
        this.metrics.clickResponseTime = performance.now() - clickStartTime;
        clickStartTime = 0;
      }
    });

    // 监控滚动响应时间
    let scrollTimeout: NodeJS.Timeout;
    document.addEventListener('scroll', () => {
      scrollStartTime = performance.now();
      clearTimeout(scrollTimeout);
      scrollTimeout = setTimeout(() => {
        if (scrollStartTime > 0) {
          this.metrics.scrollResponseTime = performance.now() - scrollStartTime;
          scrollStartTime = 0;
        }
      }, 100);
    });
  }

  /**
   * 添加自定义指标
   */
  addCustomMetric(name: string, value: number): void {
    this.customMetrics.set(name, value);
    this.metrics.customMetrics[name] = value;
  }

  /**
   * 获取性能指标
   */
  getMetrics(): PerformanceMetrics {
    return { ...this.metrics };
  }

  /**
   * 获取性能报告
   */
  getReport(): PerformanceReport {
    return {
      timestamp: Date.now(),
      url: window.location.href,
      userAgent: navigator.userAgent,
      metrics: this.getMetrics(),
      errors: [...this.errors],
      warnings: [...this.warnings],
    };
  }

  /**
   * 清除数据
   */
  clear(): void {
    this.metrics = this.initializeMetrics();
    this.errors = [];
    this.warnings = [];
    this.customMetrics.clear();
  }
}

export const performanceMonitor = new PerformanceMonitor();

// ==================== 性能分析器 ====================

export class PerformanceAnalyzer {
  /**
   * 分析性能指标
   */
  analyzeMetrics(metrics: PerformanceMetrics): {
    score: number;
    recommendations: string[];
    issues: string[];
  } {
    const recommendations: string[] = [];
    const issues: string[] = [];
    let score = 100;

    // 分析加载时间
    if (metrics.loadTime > 3000) {
      score -= 20;
      issues.push('页面加载时间过长');
      recommendations.push('优化资源加载，使用代码分割和懒加载');
    }

    // 分析FCP
    if (metrics.firstContentfulPaint > 1800) {
      score -= 15;
      issues.push('首次内容绘制时间过长');
      recommendations.push('优化关键渲染路径，减少阻塞资源');
    }

    // 分析LCP
    if (metrics.largestContentfulPaint > 2500) {
      score -= 15;
      issues.push('最大内容绘制时间过长');
      recommendations.push('优化图片和字体加载，使用预加载');
    }

    // 分析FID
    if (metrics.firstInputDelay > 100) {
      score -= 10;
      issues.push('首次输入延迟过长');
      recommendations.push('减少JavaScript执行时间，使用Web Workers');
    }

    // 分析CLS
    if (metrics.cumulativeLayoutShift > 0.1) {
      score -= 10;
      issues.push('累积布局偏移过大');
      recommendations.push('为图片和广告预留空间，避免动态插入内容');
    }

    // 分析内存使用
    if (metrics.memoryUsage > metrics.memoryLimit * 0.8) {
      score -= 10;
      issues.push('内存使用率过高');
      recommendations.push('检查内存泄漏，优化对象创建和销毁');
    }

    // 分析资源大小
    if (metrics.resourceSize > 5 * 1024 * 1024) {
      // 5MB
      score -= 10;
      issues.push('资源总大小过大');
      recommendations.push('压缩资源，使用CDN，移除未使用的代码');
    }

    // 分析资源数量
    if (metrics.resourceCount > 100) {
      score -= 10;
      issues.push('资源数量过多');
      recommendations.push('合并资源，减少HTTP请求数量');
    }

    return {
      score: Math.max(0, score),
      recommendations,
      issues,
    };
  }

  /**
   * 生成性能报告
   */
  generateReport(report: PerformanceReport): string {
    const analysis = this.analyzeMetrics(report.metrics);

    let reportText = `# 性能报告\n\n`;
    reportText += `**生成时间**: ${new Date(report.timestamp).toLocaleString()}\n`;
    reportText += `**页面URL**: ${report.url}\n`;
    reportText += `**性能评分**: ${analysis.score}/100\n\n`;

    reportText += `## 核心指标\n\n`;
    reportText += `- **页面加载时间**: ${report.metrics.loadTime.toFixed(2)}ms\n`;
    reportText += `- **首次内容绘制**: ${report.metrics.firstContentfulPaint.toFixed(2)}ms\n`;
    reportText += `- **最大内容绘制**: ${report.metrics.largestContentfulPaint.toFixed(2)}ms\n`;
    reportText += `- **首次输入延迟**: ${report.metrics.firstInputDelay.toFixed(2)}ms\n`;
    reportText += `- **累积布局偏移**: ${report.metrics.cumulativeLayoutShift.toFixed(4)}\n\n`;

    reportText += `## 资源指标\n\n`;
    reportText += `- **资源加载时间**: ${report.metrics.resourceLoadTime.toFixed(2)}ms\n`;
    reportText += `- **资源数量**: ${report.metrics.resourceCount}\n`;
    reportText += `- **资源大小**: ${(report.metrics.resourceSize / 1024 / 1024).toFixed(2)}MB\n\n`;

    reportText += `## 内存指标\n\n`;
    reportText += `- **内存使用**: ${(report.metrics.memoryUsage / 1024 / 1024).toFixed(2)}MB\n`;
    reportText += `- **内存限制**: ${(report.metrics.memoryLimit / 1024 / 1024).toFixed(2)}MB\n\n`;

    if (analysis.issues.length > 0) {
      reportText += `## 发现的问题\n\n`;
      analysis.issues.forEach(issue => {
        reportText += `- ${issue}\n`;
      });
      reportText += `\n`;
    }

    if (analysis.recommendations.length > 0) {
      reportText += `## 优化建议\n\n`;
      analysis.recommendations.forEach(recommendation => {
        reportText += `- ${recommendation}\n`;
      });
      reportText += `\n`;
    }

    if (report.errors.length > 0) {
      reportText += `## 错误信息\n\n`;
      report.errors.forEach(error => {
        reportText += `- **${new Date(error.timestamp).toLocaleString()}**: ${error.message}\n`;
        if (error.stack) {
          reportText += `  \`\`\`\n${error.stack}\n\`\`\`\n`;
        }
      });
      reportText += `\n`;
    }

    if (report.warnings.length > 0) {
      reportText += `## 警告信息\n\n`;
      report.warnings.forEach(warning => {
        reportText += `- **${new Date(warning.timestamp).toLocaleString()}**: ${warning.message}\n`;
      });
    }

    return reportText;
  }
}

export const performanceAnalyzer = new PerformanceAnalyzer();

// ==================== 性能监控Hook ====================

/**
 * 使用性能监控
 */
export function usePerformanceMonitoring() {
  const [metrics, setMetrics] = useState<PerformanceMetrics | null>(null);
  const [isMonitoring, setIsMonitoring] = useState(false);
  const [report, setReport] = useState<PerformanceReport | null>(null);

  const startMonitoring = useCallback(() => {
    performanceMonitor.startMonitoring();
    setIsMonitoring(true);
  }, []);

  const stopMonitoring = useCallback(() => {
    performanceMonitor.stopMonitoring();
    setIsMonitoring(false);
  }, []);

  const updateMetrics = useCallback(() => {
    const currentMetrics = performanceMonitor.getMetrics();
    setMetrics(currentMetrics);
  }, []);

  const generateReport = useCallback(() => {
    const currentReport = performanceMonitor.getReport();
    setReport(currentReport);
    return currentReport;
  }, []);

  const addCustomMetric = useCallback(
    (name: string, value: number) => {
      performanceMonitor.addCustomMetric(name, value);
      updateMetrics();
    },
    [updateMetrics]
  );

  const clearData = useCallback(() => {
    performanceMonitor.clear();
    setMetrics(null);
    setReport(null);
  }, []);

  useEffect(() => {
    if (isMonitoring) {
      const interval = setInterval(updateMetrics, 1000);
      return () => clearInterval(interval);
    }
  }, [isMonitoring, updateMetrics]);

  return {
    metrics,
    isMonitoring,
    report,
    startMonitoring,
    stopMonitoring,
    updateMetrics,
    generateReport,
    addCustomMetric,
    clearData,
  };
}

// ==================== 性能仪表板组件 ====================

export const PerformanceDashboard: React.FC = () => {
  const {
    metrics,
    isMonitoring,
    report,
    startMonitoring,
    stopMonitoring,
    generateReport,
    clearData,
  } = usePerformanceMonitoring();

  const [analysis, setAnalysis] = useState<{
    score: number;
    recommendations: string[];
    issues: string[];
  } | null>(null);

  useEffect(() => {
    if (metrics) {
      const currentAnalysis = performanceAnalyzer.analyzeMetrics(metrics);
      setAnalysis(currentAnalysis);
    }
  }, [metrics]);

  const handleGenerateReport = () => {
    const currentReport = generateReport();
    const reportText = performanceAnalyzer.generateReport(currentReport);

    // 下载报告
    const blob = new Blob([reportText], { type: 'text/markdown' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `performance-report-${Date.now()}.md`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  if (!isMonitoring) {
    return (
      <div style={{ padding: '20px', textAlign: 'center' }}>
        <h2>性能监控仪表板</h2>
        <p>点击开始监控按钮开始收集性能数据</p>
        <button onClick={startMonitoring} style={{ padding: '10px 20px', fontSize: '16px' }}>
          开始监控
        </button>
      </div>
    );
  }

  return (
    <div style={{ padding: '20px' }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: '20px',
        }}
      >
        <h2>性能监控仪表板</h2>
        <div>
          <button onClick={stopMonitoring} style={{ marginRight: '10px', padding: '8px 16px' }}>
            停止监控
          </button>
          <button
            onClick={handleGenerateReport}
            style={{ marginRight: '10px', padding: '8px 16px' }}
          >
            生成报告
          </button>
          <button onClick={clearData} style={{ padding: '8px 16px' }}>
            清除数据
          </button>
        </div>
      </div>

      {analysis && (
        <div
          style={{
            marginBottom: '20px',
            padding: '16px',
            backgroundColor: '#f5f5f5',
            borderRadius: '8px',
          }}
        >
          <h3>性能评分: {analysis.score}/100</h3>
          {analysis.issues.length > 0 && (
            <div>
              <h4>发现的问题:</h4>
              <ul>
                {analysis.issues.map((issue, index) => (
                  <li key={index} style={{ color: '#ff4d4f' }}>
                    {issue}
                  </li>
                ))}
              </ul>
            </div>
          )}
          {analysis.recommendations.length > 0 && (
            <div>
              <h4>优化建议:</h4>
              <ul>
                {analysis.recommendations.map((recommendation, index) => (
                  <li key={index} style={{ color: '#1890ff' }}>
                    {recommendation}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}

      {metrics && (
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))',
            gap: '20px',
          }}
        >
          <div style={{ padding: '16px', border: '1px solid #d9d9d9', borderRadius: '8px' }}>
            <h3>页面加载指标</h3>
            <p>加载时间: {metrics.loadTime.toFixed(2)}ms</p>
            <p>DOM内容加载: {metrics.domContentLoaded.toFixed(2)}ms</p>
            <p>首次内容绘制: {metrics.firstContentfulPaint.toFixed(2)}ms</p>
            <p>最大内容绘制: {metrics.largestContentfulPaint.toFixed(2)}ms</p>
            <p>首次输入延迟: {metrics.firstInputDelay.toFixed(2)}ms</p>
            <p>累积布局偏移: {metrics.cumulativeLayoutShift.toFixed(4)}</p>
          </div>

          <div style={{ padding: '16px', border: '1px solid #d9d9d9', borderRadius: '8px' }}>
            <h3>资源指标</h3>
            <p>资源加载时间: {metrics.resourceLoadTime.toFixed(2)}ms</p>
            <p>资源数量: {metrics.resourceCount}</p>
            <p>资源大小: {(metrics.resourceSize / 1024 / 1024).toFixed(2)}MB</p>
          </div>

          <div style={{ padding: '16px', border: '1px solid #d9d9d9', borderRadius: '8px' }}>
            <h3>内存指标</h3>
            <p>内存使用: {(metrics.memoryUsage / 1024 / 1024).toFixed(2)}MB</p>
            <p>内存限制: {(metrics.memoryLimit / 1024 / 1024).toFixed(2)}MB</p>
            <p>使用率: {((metrics.memoryUsage / metrics.memoryLimit) * 100).toFixed(2)}%</p>
          </div>

          <div style={{ padding: '16px', border: '1px solid #d9d9d9', borderRadius: '8px' }}>
            <h3>网络指标</h3>
            <p>网络延迟: {metrics.networkLatency}ms</p>
            <p>网络吞吐量: {metrics.networkThroughput}Mbps</p>
          </div>

          <div style={{ padding: '16px', border: '1px solid #d9d9d9', borderRadius: '8px' }}>
            <h3>交互指标</h3>
            <p>点击响应时间: {metrics.clickResponseTime.toFixed(2)}ms</p>
            <p>滚动响应时间: {metrics.scrollResponseTime.toFixed(2)}ms</p>
          </div>
        </div>
      )}
    </div>
  );
};

export default {
  PerformanceMonitor,
  performanceMonitor,
  PerformanceAnalyzer,
  performanceAnalyzer,
  usePerformanceMonitoring,
  PerformanceDashboard,
};
