'use client';

import React, { useCallback, useEffect, useState } from 'react';
import { Alert, App, Button, Card, Col, Row, Spin, Statistic, Tag, Typography } from 'antd';
import { RotateCcw } from 'lucide-react';
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Legend,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { ChangeApi } from '@/lib/api/change-api';
import type { ChangeStatsResponse } from '@/lib/api/change-api';
import { ChangeStatus, ChangeStatusConfig, ChangeType, ChangeTypeConfig } from '@/constants/taxonomy';

const { Title, Text } = Typography;

// 键由 ChangeStatus 枚举计算而来，不手抄字面量：此前这里写着 implementing，
// 而后端存量值是 in_progress，「实施中」分片因此永远拿不到颜色也永远取不到计数。
const STATUS_COLORS: Record<ChangeStatus, string> = {
  [ChangeStatus.DRAFT]: '#d9d9d9',
  [ChangeStatus.PENDING]: '#faad14',
  [ChangeStatus.APPROVED]: '#52c41a',
  [ChangeStatus.REJECTED]: '#ff4d4f',
  [ChangeStatus.SCHEDULED]: '#13c2c2',
  [ChangeStatus.IN_PROGRESS]: '#1890ff',
  [ChangeStatus.COMPLETED]: '#722ed1',
  [ChangeStatus.FAILED]: '#f5222d',
  [ChangeStatus.ROLLED_BACK]: '#fa541c',
  [ChangeStatus.CANCELLED]: '#8c8c8c',
  [ChangeStatus.CLOSED]: '#595959',
};

const STATUS_ORDER: ChangeStatus[] = [
  ChangeStatus.DRAFT,
  ChangeStatus.PENDING,
  ChangeStatus.APPROVED,
  ChangeStatus.SCHEDULED,
  ChangeStatus.IN_PROGRESS,
  ChangeStatus.COMPLETED,
  ChangeStatus.FAILED,
  ChangeStatus.ROLLED_BACK,
  ChangeStatus.REJECTED,
  ChangeStatus.CANCELLED,
  ChangeStatus.CLOSED,
];

const TYPE_COLORS: Record<string, string> = {
  [ChangeType.STANDARD]: '#52c41a',
  [ChangeType.NORMAL]: '#1890ff',
  [ChangeType.EMERGENCY]: '#fa541c',
};

interface StatusSlice {
  status: ChangeStatus;
  name: string;
  value: number;
  color: string;
}

interface TypeSlice {
  type: string;
  name: string;
  count: number;
  color: string;
}

interface ReportData {
  byStatus: StatusSlice[];
  byType: TypeSlice[];
  total: number;
  // 成功率分母只算已有实施结果的变更（完成/失败/回滚）。
  // 用 total 做分母会把草稿和待审批一起算成失败，读数永远偏低。
  implemented: number;
  completed: number;
  successRate: number | null;
}

function toReportData(stats: ChangeStatsResponse): ReportData {
  const countOf: Record<string, number> = {
    [ChangeStatus.DRAFT]: stats.draft,
    [ChangeStatus.PENDING]: stats.pending,
    [ChangeStatus.APPROVED]: stats.approved,
    [ChangeStatus.SCHEDULED]: stats.scheduled,
    [ChangeStatus.IN_PROGRESS]: stats.inProgress,
    [ChangeStatus.COMPLETED]: stats.completed,
    [ChangeStatus.FAILED]: stats.failed,
    [ChangeStatus.ROLLED_BACK]: stats.rolledBack,
    [ChangeStatus.REJECTED]: stats.rejected,
    [ChangeStatus.CANCELLED]: stats.cancelled,
    [ChangeStatus.CLOSED]: stats.closed,
  };

  const byStatus = STATUS_ORDER.filter(status => countOf[status] > 0).map(status => ({
    status,
    name: ChangeStatusConfig[status].label,
    value: countOf[status],
    color: STATUS_COLORS[status],
  }));

  // 类型分布来自后端 GROUP BY type 的真实计数。
  // 枚举外的历史类型保留原始值展示，而不是丢弃后让总数与 total 对不上账。
  const byType = stats.byType.map(entry => ({
    type: entry.type,
    name:
      entry.type in ChangeTypeConfig
        ? ChangeTypeConfig[entry.type as ChangeType].label
        : entry.type,
    count: entry.count,
    color: TYPE_COLORS[entry.type] ?? '#d9d9d9',
  }));

  const implemented = stats.completed + stats.failed + stats.rolledBack;
  return {
    byStatus,
    byType,
    total: stats.total,
    implemented,
    completed: stats.completed,
    successRate: implemented > 0 ? (stats.completed / implemented) * 100 : null,
  };
}

const EMPTY_REPORT: ReportData = {
  byStatus: [],
  byType: [],
  total: 0,
  implemented: 0,
  completed: 0,
  successRate: null,
};

const ChangeSuccessReport = () => {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<ReportData>(EMPTY_REPORT);
  const [error, setError] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const stats = await ChangeApi.getChangeStats();
      setData(toReportData(stats));
    } catch (err) {
      // 失败必须如实呈现：修复前这里把异常吞成空数据并打印「使用演示数据」，
      // 页面因此永远显示全 0 卡片，运维无法区分「没有变更」和「接口失败」。
      setData(EMPTY_REPORT);
      const reason = err instanceof Error ? err.message : '未知错误';
      setError(reason);
      message.error(`加载变更统计失败：${reason}`);
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  const isEmpty = data.byStatus.length === 0;

  return (
    <div className="p-6 bg-gray-50 min-h-full">
      <header className="mb-6">
        <Title level={2}>变更成功率报表</Title>
        <Text className="text-gray-500">展示变更管理的状态分布和成功率统计</Text>
      </header>

      <Card className="mb-6">
        <Row justify="space-between" align="middle">
          <Col>
            <Text className="text-gray-600">变更执行情况监控</Text>
          </Col>
          <Col>
            <Button icon={<RotateCcw />} onClick={loadData} loading={loading}>
              刷新数据
            </Button>
          </Col>
        </Row>
      </Card>

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <Spin description="加载报表数据..." />
        </div>
      ) : error ? (
        // 失败态不得再渲染 0 值卡片或「还没有变更记录」：那会把接口故障
        // 伪装成空数据，正好是本次修复要消除的那类误导。
        <Alert
          type="error"
          showIcon
          title="变更统计加载失败"
          description={`${error}。请确认已登录且有变更读取权限，或稍后重试。`}
          action={
            <Button size="small" icon={<RotateCcw />} onClick={loadData}>
              重试
            </Button>
          }
        />
      ) : (
        <>
          <Row gutter={[16, 16]} className="mb-6">
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic title="变更总数" value={data.total} styles={{ content: { color: '#1890ff' } }} />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic
                  title="已完成"
                  value={data.completed}
                  styles={{ content: { color: '#52c41a' } }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic
                  title="已实施变更"
                  value={data.implemented}
                  suffix="/ 完成+失败+回滚"
                  styles={{ content: { color: '#722ed1' } }}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                {data.successRate === null ? (
                  // 空分母不是 0%：显示「暂无实施结果」而不是伪装成一个真实的低成功率。
                  <Statistic
                    title="成功率"
                    value="暂无实施结果"
                    styles={{ content: { fontSize: 18, color: '#8c8c8c' } }}
                  />
                ) : (
                  <Statistic
                    title="成功率"
                    value={data.successRate}
                    precision={1}
                    suffix="%"
                    styles={{
                      content: { color: data.successRate >= 80 ? '#52c41a' : '#ff4d4f' },
                    }}
                  />
                )}
              </Card>
            </Col>
          </Row>

          {isEmpty ? (
            <Card>
              <EmptyChangeStats />
            </Card>
          ) : (
            <Row gutter={[16, 16]}>
              <Col xs={24} lg={12}>
                <Card title="变更状态分布">
                  <ResponsiveContainer width="100%" height={300}>
                    <PieChart>
                      <Pie
                        data={data.byStatus}
                        cx="50%"
                        cy="50%"
                        outerRadius={100}
                        dataKey="value"
                        nameKey="name"
                        label={({ name, percent }) => `${name} ${((percent ?? 0) * 100).toFixed(0)}%`}
                      >
                        {data.byStatus.map(entry => (
                          <Cell key={entry.status} fill={entry.color} />
                        ))}
                      </Pie>
                      <Tooltip />
                      <Legend />
                    </PieChart>
                  </ResponsiveContainer>
                </Card>
              </Col>

              <Col xs={24} lg={12}>
                <Card title="变更类型分布">
                  <ResponsiveContainer width="100%" height={300}>
                    <BarChart data={data.byType}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis dataKey="name" />
                      <YAxis allowDecimals={false} />
                      <Tooltip />
                      <Legend />
                      <Bar dataKey="count" name="变更数量" fill="#1890ff">
                        {data.byType.map(entry => (
                          <Cell key={entry.type} fill={entry.color} />
                        ))}
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                </Card>
              </Col>
            </Row>
          )}

          {!isEmpty && (
            <Card title="状态说明" className="mt-6">
              <Row gutter={[16, 16]}>
                {data.byStatus.map(entry => (
                  <Col xs={12} sm={8} md={6} key={entry.status}>
                    <div className="flex items-center gap-2">
                      <Tag color={entry.color} className="m-0">
                        {entry.name}
                      </Tag>
                      <span className="text-lg font-semibold">{entry.value}</span>
                    </div>
                  </Col>
                ))}
              </Row>
            </Card>
          )}
        </>
      )}
    </div>
  );
};

function EmptyChangeStats() {
  return (
    <div className="py-8 text-center">
      <p className="text-gray-600">当前租户还没有变更记录</p>
      <p className="text-gray-400 text-sm mt-1">
        创建变更并流转到相应状态后，这里会显示真实的状态分布与类型分布。
      </p>
    </div>
  );
}

export default ChangeSuccessReport;
