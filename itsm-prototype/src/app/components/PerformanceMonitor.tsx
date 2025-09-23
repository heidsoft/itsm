'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { 
  Card, 
  Progress, 
  Statistic, 
  Row, 
  Col, 
  Tag, 
  Timeline,
  Typography,
  Space,
  Button
} from 'antd';
import { 
  Activity, 
  Cpu, 
  HardDrive, 
  Wifi, 
  Clock, 
  AlertTriangle,
  CheckCircle,
  TrendingUp,
  TrendingDown,
  RefreshCw
} from 'lucide-react';

const { Text } = Typography;

interface PerformanceMetrics {
  cpu: {
    usage: number;
    cores: number;
    temperature?: number;
  };
  memory: {
    used: number;
    total: number;
    usage: number;
  };
  disk: {
    used: number;
    total: number;
    usage: number;
  };
  network: {
    inbound: number;
    outbound: number;
    latency: number;
  };
  response: {
    avgResponseTime: number;
    p95ResponseTime: number;
    errorRate: number;
  };
  database: {
    connections: number;
    maxConnections: number;
    queryTime: number;
  };
}

interface SystemAlert {
  id: string;
  type: 'warning' | 'error' | 'info';
  message: string;
  timestamp: string;
  resolved: boolean;
}

interface PerformanceMonitorProps {
  refreshInterval?: number;
  showAlerts?: boolean;
  compact?: boolean;
  className?: string;
}

export const PerformanceMonitor: React.FC<PerformanceMonitorProps> = ({
  refreshInterval = 30000, // 30秒刷新一次
  showAlerts = true,
  compact = false,
  className = ''
}) => {
  const [metrics, setMetrics] = useState<PerformanceMetrics | null>(null);
  const [alerts, setAlerts] = useState<SystemAlert[]>([]);
  const [loading, setLoading] = useState(false);
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null);

  // 模拟获取性能指标数据
  const fetchMetrics = useCallback(async () => {
    setLoading(true);
    try {
      // 模拟API调用延迟
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      // 生成模拟数据
      const mockMetrics: PerformanceMetrics = {
        cpu: {
          usage: Math.random() * 80 + 10, // 10-90%
          cores: 8,
          temperature: Math.random() * 20 + 45 // 45-65°C
        },
        memory: {
          used: Math.random() * 6 + 2, // 2-8GB
          total: 16,
          usage: 0
        },
        disk: {
          used: Math.random() * 200 + 50, // 50-250GB
          total: 500,
          usage: 0
        },
        network: {
          inbound: Math.random() * 100 + 10, // 10-110 Mbps
          outbound: Math.random() * 50 + 5, // 5-55 Mbps
          latency: Math.random() * 50 + 10 // 10-60ms
        },
        response: {
          avgResponseTime: Math.random() * 200 + 50, // 50-250ms
          p95ResponseTime: Math.random() * 500 + 200, // 200-700ms
          errorRate: Math.random() * 2 // 0-2%
        },
        database: {
          connections: Math.floor(Math.random() * 80 + 10), // 10-90
          maxConnections: 100,
          queryTime: Math.random() * 100 + 10 // 10-110ms
        }
      };

      // 计算使用率百分比
      mockMetrics.memory.usage = (mockMetrics.memory.used / mockMetrics.memory.total) * 100;
      mockMetrics.disk.usage = (mockMetrics.disk.used / mockMetrics.disk.total) * 100;

      setMetrics(mockMetrics);
      setLastUpdate(new Date());

      // 生成告警
      const newAlerts: SystemAlert[] = [];
      
      if (mockMetrics.cpu.usage > 80) {
        newAlerts.push({
          id: `cpu-${Date.now()}`,
          type: 'warning',
          message: `CPU使用率过高: ${mockMetrics.cpu.usage.toFixed(1)}%`,
          timestamp: new Date().toISOString(),
          resolved: false
        });
      }

      if (mockMetrics.memory.usage > 85) {
        newAlerts.push({
          id: `memory-${Date.now()}`,
          type: 'error',
          message: `内存使用率过高: ${mockMetrics.memory.usage.toFixed(1)}%`,
          timestamp: new Date().toISOString(),
          resolved: false
        });
      }

      if (mockMetrics.response.errorRate > 1) {
        newAlerts.push({
          id: `error-${Date.now()}`,
          type: 'error',
          message: `错误率过高: ${mockMetrics.response.errorRate.toFixed(2)}%`,
          timestamp: new Date().toISOString(),
          resolved: false
        });
      }

      setAlerts(prev => [...newAlerts, ...prev.slice(0, 10)]); // 保留最新10条告警
    } catch (error) {
      console.error('获取性能指标失败:', error);
    } finally {
      setLoading(false);
    }
  }, []);

  // 自动刷新
  useEffect(() => {
    fetchMetrics();
    const interval = setInterval(fetchMetrics, refreshInterval);
    return () => clearInterval(interval);
  }, [fetchMetrics, refreshInterval]);

  // 获取状态颜色
  const getStatusColor = (usage: number, type: 'cpu' | 'memory' | 'disk' = 'cpu') => {
    const thresholds = {
      cpu: { warning: 70, danger: 85 },
      memory: { warning: 75, danger: 90 },
      disk: { warning: 80, danger: 95 }
    };
    
    const threshold = thresholds[type];
    if (usage >= threshold.danger) return '#ff4d4f';
    if (usage >= threshold.warning) return '#faad14';
    return '#52c41a';
  };

  // 获取趋势图标
  const getTrendIcon = (value: number, threshold: number) => {
    if (value > threshold) {
      return <TrendingUp size={14} className="text-red-500" />;
    }
    return <TrendingDown size={14} className="text-green-500" />;
  };

  if (!metrics) {
    return (
      <Card loading={loading} className={className}>
        <div className="text-center py-8">
          <Activity size={48} className="text-gray-400 mx-auto mb-4" />
          <Text className="text-gray-500">正在加载性能数据...</Text>
        </div>
      </Card>
    );
  }

  if (compact) {
    return (
      <Card 
        size="small" 
        className={className}
        title={
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <Activity size={16} />
              <span>系统监控</span>
            </div>
            <Button 
              type="text" 
              size="small" 
              icon={<RefreshCw size={14} />}
              loading={loading}
              onClick={fetchMetrics}
            />
          </div>
        }
      >
        <Row gutter={[16, 16]}>
          <Col span={6}>
            <Statistic
              title="CPU"
              value={metrics.cpu.usage}
              precision={1}
              suffix="%"
              valueStyle={{ color: getStatusColor(metrics.cpu.usage, 'cpu') }}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="内存"
              value={metrics.memory.usage}
              precision={1}
              suffix="%"
              valueStyle={{ color: getStatusColor(metrics.memory.usage, 'memory') }}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="磁盘"
              value={metrics.disk.usage}
              precision={1}
              suffix="%"
              valueStyle={{ color: getStatusColor(metrics.disk.usage, 'disk') }}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title="响应时间"
              value={metrics.response.avgResponseTime}
              precision={0}
              suffix="ms"
              valueStyle={{ 
                color: metrics.response.avgResponseTime > 200 ? '#faad14' : '#52c41a' 
              }}
            />
          </Col>
        </Row>
      </Card>
    );
  }

  return (
    <div className={`performance-monitor ${className}`}>
      <Row gutter={[16, 16]}>
        {/* 系统资源监控 */}
        <Col xs={24} lg={16}>
          <Card 
            title={
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <Activity size={20} />
                  <span>系统资源监控</span>
                </div>
                <div className="flex items-center space-x-4">
                  {lastUpdate && (
                    <Text className="text-sm text-gray-500">
                      最后更新: {lastUpdate.toLocaleTimeString()}
                    </Text>
                  )}
                  <Button 
                    type="text" 
                    size="small" 
                    icon={<RefreshCw size={14} />}
                    loading={loading}
                    onClick={fetchMetrics}
                  />
                </div>
              </div>
            }
          >
            <Row gutter={[24, 24]}>
              {/* CPU监控 */}
              <Col xs={24} sm={12} lg={8}>
                <div className="text-center">
                  <div className="flex items-center justify-center mb-2">
                    <Cpu size={20} className="text-blue-600 mr-2" />
                    <Text className="font-semibold">CPU使用率</Text>
                    {getTrendIcon(metrics.cpu.usage, 70)}
                  </div>
                  <Progress
                    type="circle"
                    percent={metrics.cpu.usage}
                    format={(percent) => `${percent?.toFixed(1)}%`}
                    strokeColor={getStatusColor(metrics.cpu.usage, 'cpu')}
                    size={120}
                  />
                  <div className="mt-3 space-y-1">
                    <div className="flex justify-between text-sm">
                      <span>核心数:</span>
                      <span>{metrics.cpu.cores}</span>
                    </div>
                    {metrics.cpu.temperature && (
                      <div className="flex justify-between text-sm">
                        <span>温度:</span>
                        <span>{metrics.cpu.temperature.toFixed(1)}°C</span>
                      </div>
                    )}
                  </div>
                </div>
              </Col>

              {/* 内存监控 */}
              <Col xs={24} sm={12} lg={8}>
                <div className="text-center">
                  <div className="flex items-center justify-center mb-2">
                    <HardDrive size={20} className="text-green-600 mr-2" />
                    <Text className="font-semibold">内存使用率</Text>
                    {getTrendIcon(metrics.memory.usage, 75)}
                  </div>
                  <Progress
                    type="circle"
                    percent={metrics.memory.usage}
                    format={(percent) => `${percent?.toFixed(1)}%`}
                    strokeColor={getStatusColor(metrics.memory.usage, 'memory')}
                    size={120}
                  />
                  <div className="mt-3 space-y-1">
                    <div className="flex justify-between text-sm">
                      <span>已用:</span>
                      <span>{metrics.memory.used.toFixed(1)}GB</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span>总计:</span>
                      <span>{metrics.memory.total}GB</span>
                    </div>
                  </div>
                </div>
              </Col>

              {/* 磁盘监控 */}
              <Col xs={24} sm={12} lg={8}>
                <div className="text-center">
                  <div className="flex items-center justify-center mb-2">
                    <HardDrive size={20} className="text-purple-600 mr-2" />
                    <Text className="font-semibold">磁盘使用率</Text>
                    {getTrendIcon(metrics.disk.usage, 80)}
                  </div>
                  <Progress
                    type="circle"
                    percent={metrics.disk.usage}
                    format={(percent) => `${percent?.toFixed(1)}%`}
                    strokeColor={getStatusColor(metrics.disk.usage, 'disk')}
                    size={120}
                  />
                  <div className="mt-3 space-y-1">
                    <div className="flex justify-between text-sm">
                      <span>已用:</span>
                      <span>{metrics.disk.used.toFixed(0)}GB</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span>总计:</span>
                      <span>{metrics.disk.total}GB</span>
                    </div>
                  </div>
                </div>
              </Col>
            </Row>
          </Card>
        </Col>

        {/* 网络和性能指标 */}
        <Col xs={24} lg={8}>
          <Space direction="vertical" size="middle" className="w-full">
            {/* 网络监控 */}
            <Card size="small" title={
              <div className="flex items-center space-x-2">
                <Wifi size={16} />
                <span>网络状态</span>
              </div>
            }>
              <Row gutter={[16, 16]}>
                <Col span={12}>
                  <Statistic
                    title="入站流量"
                    value={metrics.network.inbound}
                    precision={1}
                    suffix="Mbps"
                    valueStyle={{ fontSize: '16px' }}
                  />
                </Col>
                <Col span={12}>
                  <Statistic
                    title="出站流量"
                    value={metrics.network.outbound}
                    precision={1}
                    suffix="Mbps"
                    valueStyle={{ fontSize: '16px' }}
                  />
                </Col>
                <Col span={24}>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-gray-600">网络延迟:</span>
                    <Tag color={metrics.network.latency > 100 ? 'red' : 'green'}>
                      {metrics.network.latency.toFixed(0)}ms
                    </Tag>
                  </div>
                </Col>
              </Row>
            </Card>

            {/* 应用性能 */}
            <Card size="small" title={
              <div className="flex items-center space-x-2">
                <Clock size={16} />
                <span>应用性能</span>
              </div>
            }>
              <Space direction="vertical" size="small" className="w-full">
                <div className="flex justify-between items-center">
                  <span className="text-sm">平均响应时间:</span>
                  <Tag color={metrics.response.avgResponseTime > 200 ? 'orange' : 'green'}>
                    {metrics.response.avgResponseTime.toFixed(0)}ms
                  </Tag>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-sm">P95响应时间:</span>
                  <Tag color={metrics.response.p95ResponseTime > 500 ? 'red' : 'blue'}>
                    {metrics.response.p95ResponseTime.toFixed(0)}ms
                  </Tag>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-sm">错误率:</span>
                  <Tag color={metrics.response.errorRate > 1 ? 'red' : 'green'}>
                    {metrics.response.errorRate.toFixed(2)}%
                  </Tag>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-sm">数据库连接:</span>
                  <Tag color={metrics.database.connections > 80 ? 'orange' : 'blue'}>
                    {metrics.database.connections}/{metrics.database.maxConnections}
                  </Tag>
                </div>
              </Space>
            </Card>

            {/* 系统告警 */}
            {showAlerts && alerts.length > 0 && (
              <Card size="small" title={
                <div className="flex items-center space-x-2">
                  <AlertTriangle size={16} />
                  <span>系统告警</span>
                  <Tag color="red">{alerts.filter(a => !a.resolved).length}</Tag>
                </div>
              }>
                <Timeline>
                  {alerts.slice(0, 5).map((alert) => (
                    <Timeline.Item
                      key={alert.id}
                      dot={
                        alert.type === 'error' ? 
                          <AlertTriangle size={12} className="text-red-500" /> :
                          <CheckCircle size={12} className="text-yellow-500" />
                      }
                    >
                      <div className="text-sm">
                        <div className={`font-medium ${
                          alert.type === 'error' ? 'text-red-600' : 'text-yellow-600'
                        }`}>
                          {alert.message}
                        </div>
                        <div className="text-gray-500 text-xs">
                          {new Date(alert.timestamp).toLocaleTimeString()}
                        </div>
                      </div>
                    </Timeline.Item>
                  ))}
                </Timeline>
              </Card>
            )}
          </Space>
        </Col>
      </Row>
    </div>
  );
};

export default PerformanceMonitor;