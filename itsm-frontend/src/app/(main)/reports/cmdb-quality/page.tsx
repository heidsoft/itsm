'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Alert, App, Button, Card, Col, Empty, Row, Skeleton, Statistic, Table, Tag, Typography } from 'antd';
import { Bar, BarChart, CartesianGrid, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { RotateCcw } from 'lucide-react';

import { CMDBApi } from '@/lib/api/cmdb-api';
import type { GovernanceDuplicate, GovernanceReport } from '@/lib/api/cmdb-api';

const { Title, Text } = Typography;

const COLORS = {
  completeness: '#1890ff',
  duplicate: '#fa541c',
  stale: '#faad14',
  orphan: '#722ed1',
};

// CMDB 数据质量报表。
//
// 🔴 所有指标都由后端治理看板计算，本页**不再自算**。
// 改造前这里用硬编码 9 个字段（name/type/status/environment/criticality/
// assetTag/serialNumber/assignedTo/ownedBy）在前端算完整度，与后端
// computeQualityMetrics 的 owner 覆盖率口径互不相通，同一租户在两处能读到
// 两个不同的完整度；而「哪些字段算必填」本该由租户在「属性定义」里配置，
// 前端根本读不到那份配置。完整率的事实来源必须是后端的必填属性定义。
//
// 因此本页只剩「呈现」职责：拉 /cmdb/governance/report，按返回结构画出图表。
// 若要新增质量维度，改后端纯函数 + DTO，不要在前端重组口径。

const CMDBQualityReport = () => {
  const { message } = App.useApp();
  // 初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染
  // 「暂无数据」结论，再被真实数据覆盖。
  const [loading, setLoading] = useState(true);
  const [report, setReport] = useState<GovernanceReport | null>(null);
  const [failed, setFailed] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    setFailed(false);
    try {
      const data = await CMDBApi.getGovernanceReport();
      setReport(data);
    } catch (error) {
      console.warn('获取 CMDB 治理数据失败:', error);
      message.error('加载CMDB质量数据失败');
      setReport(null);
      setFailed(true);
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  if (loading) {
    return (
      <div className="space-y-4">
        <Row gutter={[16, 16]}>
          {[0, 1, 2, 3].map((i) => (
            <Col xs={24} sm={12} lg={6} key={`skeleton-${i}`}>
              <Card>
                <Skeleton active />
              </Card>
            </Col>
          ))}
        </Row>
        <Skeleton active paragraph={{ rows: 6 }} />
      </div>
    );
  }

  if (failed || !report) {
    return (
      <Card>
        <Empty description="无法获取 CMDB 治理数据" />
      </Card>
    );
  }

  const { quality, certification, diff, duplicateGroups, duplicateRatePct } = report;
  // 没有任何必填属性定义时，完整率不可评估——必须显式区分，不能显示 0%。
  const notAssessable = quality.assessable === 0;
  // 认证率同构的不可评估态：没有在用资产时 freshnessPct 恒为 0，
  // 显示「0%」会被读成「所有资产都过期」，实际是「还没有资产要认证」。
  const noCertifiable = (certification?.verified ?? 0) === 0;

  // 图表数据：把「最缺的必填属性」画成横向条形图，这是唯一可行动的视图
  const missingData = quality.missingByAttribute.slice(0, 10).map((item) => ({
    name: `${item.attribute} (类型${item.ciTypeId})`,
    missing: item.missing,
    applicable: item.applicable,
    rate: item.applicable > 0 ? Math.round((item.missing / item.applicable) * 100) : 0,
  }));

  const diffData = Object.entries(diff).map(([action, count]) => ({ name: action, count }));

  const duplicateColumns = [
    { title: '判重依据', dataIndex: 'reason', key: 'reason', width: 140,
      render: (reason: string) => {
        const LABELS: Record<string, { text: string; color: string }> = {
          cloud_identity: { text: '云身份相同', color: 'blue' },
          asset_tag: { text: '资产标签相同', color: 'orange' },
          serial_name: { text: '序列号+名称相同', color: 'purple' },
        };
        const meta = LABELS[reason] ?? { text: reason, color: 'default' };
        return <Tag color={meta.color}>{meta.text}</Tag>;
      } },
    { title: '判重键', dataIndex: 'key', key: 'key', ellipsis: true },
    { title: '重复 CI', dataIndex: 'ciNumbers', key: 'ciNumbers',
      render: (numbers: string[]) => (
        <div className="flex flex-wrap gap-1">
          {numbers.map((n) => <Tag key={n}>{n}</Tag>)}
        </div>
      ) },
  ];

  return (
    <>
      <header className="mb-6">
        <Title level={2}>CMDB数据质量报表</Title>
        <p className="text-gray-500 mt-1">
          由后端治理看板（/cmdb/governance/report）统一计算：完整率按各 CI 类型配置的必填属性算，
          重复率按 CI 之间的身份键聚类算。本页只呈现，不再自算口径。
        </p>
      </header>

      <Card className="mb-6">
        <Row justify="space-between" align="middle">
          <Col>
            <Text className="text-gray-600">配置项数据质量监控</Text>
          </Col>
          <Col>
            <Button icon={<RotateCcw />} onClick={loadData} loading={loading}>
              刷新数据
            </Button>
          </Col>
        </Row>
      </Card>

      {/* 未配置必填属性时明确告知，而不是显示 0% 完整 */}
      {notAssessable && (
        <Alert
          className="mb-6"
          type="info"
          showIcon
          title="尚未配置必填属性，完整率不可评估"
          description="空值不代表数据质量差；请在 CMDB「属性定义」中标注必填项。"
        />
      )}

      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="配置项总数" value={quality.total} suffix="项" />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title={notAssessable ? '整体完整度（不可评估）' : '整体完整度'}
              value={notAssessable ? '—' : quality.completenessPct}
              suffix={notAssessable ? undefined : '%'}
              styles={{ content: { color: COLORS.completeness } }}
            />
            <Text type="secondary" className="text-xs">
              {notAssessable ? '未配置必填属性' : `可评估 ${quality.assessable} 项`}
            </Text>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="重复率"
              value={duplicateRatePct}
              suffix="%"
              styles={{ content: { color: COLORS.duplicate } }}
            />
            <Text type="secondary" className="text-xs">
              {report.duplicateCiCount} 项参与重复组
            </Text>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="陈旧 / 孤儿"
              value={`${quality.stale} / ${quality.orphan}`}
              styles={{ content: { color: COLORS.stale } }}
            />
            <Text type="secondary" className="text-xs">
              超 7 天未同步 / 无负责人且无来源
            </Text>
          </Card>
        </Col>
      </Row>

      {/* 确认新鲜度：CMDB Health 的 Compliance 维度。
          与完整率是两件事——完整率问「必填属性填齐了吗」，认证问「这些值还有人担保吗」。
          一条属性全填齐但三年没人确认过的 CI，完整率高而可信度为零。 */}
      <Card
        title="确认新鲜度（数据认证）"
        className="mb-6"
        extra={
          <Text type="secondary" className="text-xs">
            确认于 {certification?.freshThresholdDays ?? 0} 天内视为新鲜
          </Text>
        }
      >
        {noCertifiable ? (
          <Empty description="暂无可评估的在用资产" />
        ) : (
          <Row gutter={[16, 16]} align="middle">
            <Col xs={24} lg={8}>
              <Statistic
                title="认证率"
                value={certification.freshnessPct}
                suffix="%"
                styles={{ content: { color: COLORS.completeness } }}
              />
              <Text type="secondary" className="text-xs">
                分母为在用且可评估的 {certification.verified} 项
              </Text>
            </Col>
            <Col xs={24} lg={16}>
              {/* 三桶互斥完备（和恒等于 verified），所以用堆叠条而不是三个独立数字：
                  运维一眼能看出「空白区（从未确认）」占多大比例。 */}
              <div className="flex h-8 w-full overflow-hidden rounded">
                {[
                  { key: 'fresh', value: certification.freshConfirmed, color: '#52c41a' },
                  { key: 'stale', value: certification.staleConfirmed, color: '#faad14' },
                  { key: 'never', value: certification.neverConfirmed, color: '#f5222d' },
                ].map((seg) => {
                  const width = `${(seg.value / certification.verified) * 100}%`;
                  return seg.value > 0 ? (
                    <div
                      key={seg.key}
                      style={{ width, backgroundColor: seg.color }}
                      className="flex items-center justify-center text-xs text-white"
                      title={`${seg.value} 项`}
                    >
                      {seg.value}
                    </div>
                  ) : null;
                })}
              </div>
              <div className="mt-2 flex flex-wrap gap-4 text-xs">
                <Text>
                  <span style={{ color: '#52c41a' }}>■</span> 新鲜 {certification.freshConfirmed}
                </Text>
                <Text>
                  <span style={{ color: '#faad14' }}>■</span> 已过期 {certification.staleConfirmed}
                </Text>
                <Text>
                  <span style={{ color: '#f5222d' }}>■</span> 从未确认 {certification.neverConfirmed}
                </Text>
              </div>
              {certification.neverConfirmed > 0 && (
                <Alert
                  className="mt-3"
                  type="warning"
                  showIcon
                  title={`${certification.neverConfirmed} 项资产从未被任何人确认过`}
                  description="「从未确认」与「已过期」不是一回事：过期是确认过但太久，从未确认说明这条资产从来没人认领责任。建议到 CMDB「数据认证」页发起认证任务。"
                />
              )}
            </Col>
          </Row>
        )}
      </Card>

      {/* 必填属性缺口：唯一能直接指导补齐动作的视图 */}
      <Card title="最缺的必填属性" className="mb-6">
        {missingData.length === 0 ? (
          <Empty description={notAssessable ? '未配置必填属性定义' : '必填属性均已填写'} />
        ) : (
          <ResponsiveContainer width="100%" height={Math.max(220, missingData.length * 46)}>
            <BarChart data={missingData} layout="vertical" margin={{ left: 100, right: 32 }}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis type="number" domain={[0, 'dataMax']} />
              <YAxis type="category" dataKey="name" width={110} />
              <Tooltip
                formatter={(value) => [`${value ?? 0} 项`, '']}
              />
              <Legend />
              <Bar dataKey="missing" name="缺失数" fill={COLORS.completeness} />
              <Bar dataKey="applicable" name="适用数" fill="#d9d9d9" />
            </BarChart>
          </ResponsiveContainer>
        )}
      </Card>

      {/* 重复组明细 */}
      <Card title="重复配置项（同实体多笔账）" className="mb-6">
        {duplicateGroups.length === 0 ? (
          <Empty description="未发现重复配置项" />
        ) : (
          <Table
            rowKey={(record: GovernanceDuplicate) => record.key}
            columns={duplicateColumns}
            dataSource={duplicateGroups}
            pagination={duplicateGroups.length > 10 ? { pageSize: 10 } : false}
            size="small"
          />
        )}
      </Card>

      {/* 发现差异分桶 */}
      <Card title="发现差异分类">
        <ResponsiveContainer width="100%" height={260}>
          <BarChart data={diffData}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="name" />
            <YAxis />
            <Tooltip />
            <Legend />
            <Bar dataKey="count" name="条目数" fill={COLORS.orphan} />
          </BarChart>
        </ResponsiveContainer>
        <Text type="secondary" className="text-xs">
          云资源 {report.cloudResources} 项 / 配置项 {report.cis} 项。add=未建账、noop=已关联、
          retire_confirm=云上已消失待确认退役、duplicate=同批重复。
        </Text>
      </Card>
    </>
  );
};

export default CMDBQualityReport;
