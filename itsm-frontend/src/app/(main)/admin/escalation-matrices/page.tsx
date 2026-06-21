'use client';

/**
 * SLA 升级矩阵页面
 *
 * 对应后端 service/escalation_matrix.go（service 层，无 HTTP API）
 *
 * 功能：
 *   1. 按优先级（P1/P2/P3/P4）展示默认升级矩阵
 *   2. 每个优先级下的多级升级链：阈值 / 目标类型 / 目标 ID / 通知渠道
 *   3. 提示：当前矩阵为只读预览，后端尚未暴露 HTTP 编辑接口
 */

import { Alert, Card, Col, Row, Space, Statistic, Table, Tag, Typography } from 'antd';
import { AlertOutlined, ClockCircleOutlined, NotificationOutlined, TeamOutlined } from '@ant-design/icons';
import { useEffect, useMemo, useState } from 'react';
import {
  EscalationMatrixApi,
  DEFAULT_ESCALATION_MATRIX,
  type EscalationLevel,
  type EscalationMatrix,
} from '@/lib/api/escalation-matrix-api';

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
          <ClockCircleOutlined />
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
              <Tag key={id} color="purple" icon={<TeamOutlined />}>
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
            <Tag key={c} color="cyan" icon={<NotificationOutlined />}>
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
          <AlertOutlined style={{ marginRight: 8 }} />
          SLA 升级矩阵
        </Title>
        <Paragraph type="secondary">
          按优先级配置的多级升级策略。当 SLA 阈值被突破时，系统会按升级链逐级通知更高级别的处理人。
        </Paragraph>
      </div>

      <Alert
        message="只读预览模式"
        description="升级矩阵当前在后端为进程内缓存（service/escalation_matrix.go），尚未提供 HTTP 编辑接口。后续版本会开放租户级自定义能力。"
        type="warning"
        showIcon
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

      <Card title="完整升级矩阵（按优先级 + 级别）">
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
                      <ClockCircleOutlined />
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