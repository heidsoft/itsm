'use client';

/**
 * SLA 升级矩阵页面
 *
 * 按优先级展示多级升级链：阈值 / 目标类型 / 目标 ID / 通知渠道
 */

import { Alert, Card, Col, Row, Space, Statistic, Table, Tag, Typography } from 'antd';
import { Users, Clock, Bell, AlertTriangle } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import {
  EscalationMatrixApi,
  DEFAULT_ESCALATION_MATRIX,
  type EscalationLevel,
  type EscalationMatrix,
} from '@/lib/api/escalation-matrix-api';
import { UsageGuideCard } from '@/components/common/UsageGuideCard';

const { Title, Paragraph, Text } = Typography;

const priorityColorMap: Record<string, string> = {
  P1: 'red',
  P2: 'orange',
  P3: 'gold',
  P4: 'blue',
};

function formatMinutes(mins: number): string {
  if (mins < 60) return `${mins} 分钟`;
  if (mins < 1440) return `${(mins / 60).toFixed(1)} 小时`;
  return `${(mins / 1440).toFixed(1)} 天`;
}

export default function EscalationMatricesPage() {
  const [matrix, setMatrix] = useState<EscalationMatrix>(DEFAULT_ESCALATION_MATRIX);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    EscalationMatrixApi.getMatrix()
      .then(m => {
        if (alive) setMatrix(m);
      })
      .finally(() => {
        if (alive) setLoading(false);
      });
    return () => {
      alive = false;
    };
  }, []);

  const stats = useMemo(() => {
    let totalLevels = 0;
    let totalTargets = 0;
    Object.values(matrix).forEach(levels => {
      totalLevels += levels.length;
      levels.forEach(l => {
        totalTargets += l.targetIds.length;
      });
    });
    return { priorities: Object.keys(matrix).length, totalLevels, totalTargets };
  }, [matrix]);

  const allRows = useMemo(() => {
    const rows: Array<{ priority: string } & EscalationLevel> = [];
    Object.entries(matrix).forEach(([priority, levels]) => {
      levels.forEach(level => {
        rows.push({ priority, ...level });
      });
    });
    return rows;
  }, [matrix]);

  const columns = [
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      render: (v: string) => <Tag color={priorityColorMap[v] ?? 'default'}>{v}</Tag>,
    },
    {
      title: '级别',
      dataIndex: 'level',
      key: 'level',
      width: 80,
      render: (v: number) => <Tag color="blue">L{v}</Tag>,
    },
    {
      title: '触发阈值',
      dataIndex: 'thresholdMinutes',
      key: 'thresholdMinutes',
      width: 130,
      render: (v: number) => (
        <Space>
          <Clock />
          <Text strong>{formatMinutes(v)}</Text>
        </Space>
      ),
    },
    {
      title: '目标类型',
      dataIndex: 'targetType',
      key: 'targetType',
      width: 110,
      render: (v: string) => <Tag color="geekblue">{v}</Tag>,
    },
    {
      title: '目标 ID',
      dataIndex: 'targetIds',
      key: 'targetIds',
      render: (ids: number[]) =>
        ids.length === 0 ? (
          <Text type="secondary">未指定</Text>
        ) : (
          <Space size={4} wrap>
            {ids.map(id => (
              <Tag key={id} color="purple" icon={<Users />}>
                {id}
              </Tag>
            ))}
          </Space>
        ),
    },
    {
      title: '通知渠道',
      dataIndex: 'notifyChannels',
      key: 'notifyChannels',
      render: (channels: string[]) => (
        <Space size={4} wrap>
          {channels.map(c => (
            <Tag key={c} color="cyan" icon={<Bell />}>
              {c}
            </Tag>
          ))}
        </Space>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <Title level={3} style={{ marginBottom: 4 }}>
          <AlertTriangle style={{ marginRight: 8 }} />
          SLA 升级矩阵
        </Title>
        <Paragraph type="secondary">
          按优先级配置的多级升级策略。当 SLA 阈值被突破时，系统会按升级链逐级通知更高级别的处理人。
        </Paragraph>
      </div>

      <Alert
        message="只读预览"
        description="升级矩阵为系统内置策略，当前仅支持查看，暂不提供在线编辑。后续版本将开放租户级自定义配置。"
        type="info"
        showIcon
      />

      <UsageGuideCard
        style={{ marginBottom: 16 }}
        title="怎么理解和使用升级矩阵"
        intro="升级矩阵是 SLA 超时的自动升级策略：工单超过阈值仍未处理时，系统按级别逐级通知更高级别的处理人。本页为只读查看，不需要（也不支持）在线编辑。"
        steps={[
          '阅读方式：每个优先级（P1 最高）配置若干升级级别；待处理时长超过某级“阈值时间”即触发该级升级，按“目标类型 + ID”（用户 / 角色 / 群组）定位升级对象，并通过“通知渠道”（邮件、站内信等）发送。',
          '默认策略中优先级越高阈值越短、级别越多：如 P1 为 5 / 15 / 30 分钟三级，P3 仅 240 分钟一级。',
          '生效范围：SLA 告警升级、工单超时升级、BPMN 任务超时共用这套矩阵，无需单独启用。',
          '前置条件：SLA 时限来自“SLA 模板 / SLA 定义”，未安装 SLA 的优先级不会触发升级；邮件渠道需要在“系统配置”中配好 SMTP。',
          '建议用法：对照本表确认阈值与组织值班响应能力是否匹配；如需调整，当前可通过修改 SLA 定义与通知渠道间接影响，矩阵本身的自定义编辑将在后续版本开放。',
        ]}
      />

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic title="覆盖优先级" value={stats.priorities} suffix="个" />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic title="升级级别总数" value={stats.totalLevels} suffix="级" valueStyle={{ color: '#1677ff' }} />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title="通知目标数"
              value={stats.totalTargets}
              suffix="个"
              valueStyle={{ color: '#722ed1' }}
            />
          </Card>
        </Col>
      </Row>

      <Card title="升级矩阵（按优先级和级别）">
        <Table
          rowKey={r => `${r.priority}-${r.level}`}
          loading={loading}
          dataSource={allRows}
          columns={columns}
          pagination={false}
          scroll={{ x: 'max-content' }}
          expandable={{
            expandedRowRender: () => null,
          }}
        />
      </Card>

      <Card title="按优先级分组视图">
        <Row gutter={[16, 16]}>
          {Object.entries(matrix).map(([priority, levels]) => (
            <Col xs={24} md={12} lg={8} key={priority}>
              <Card
                size="small"
                title={
                  <Space>
                    <Tag color={priorityColorMap[priority] ?? 'default'}>{priority}</Tag>
                    <Text type="secondary">{levels.length} 级升级</Text>
                  </Space>
                }
              >
                {levels.map(level => (
                  <div
                    key={level.level}
                    style={{
                      padding: '8px 0',
                      borderBottom: '1px dashed #f0f0f0',
                    }}
                  >
                    <Space size="small" wrap>
                      <Tag color="blue">L{level.level}</Tag>
                      <Clock />
                      <Text>{formatMinutes(level.thresholdMinutes)}</Text>
                      <Tag color="geekblue">{level.targetType}</Tag>
                      {level.notifyChannels.map(c => (
                        <Tag key={c} color="cyan">
                          {c}
                        </Tag>
                      ))}
                    </Space>
                  </div>
                ))}
              </Card>
            </Col>
          ))}
        </Row>
      </Card>
    </div>
  );
}