'use client';

/**
 * SLA 升级矩阵页面
 *
 * 按优先级展示多级升级链：触发时长 / 通知角色 / 通知用户 / 说明
 */

import { Alert, Button, Card, Col, Row, Space, Statistic, Table, Tag, Typography } from 'antd';
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
  critical: 'red',
  high: 'orange',
  medium: 'gold',
  low: 'blue',
};

const priorityLabelMap: Record<string, string> = {
  critical: '致命',
  high: '高',
  medium: '中',
  low: '低',
};

// 后端 notifyRoles 取主角色词表原值，展示时补中文，避免运维对着 it_admin 猜职责。
const roleLabelMap: Record<string, string> = {
  admin: '租户管理员',
  it_admin: 'IT 管理员',
  manager: '部门经理',
  agent: '服务台坐席',
  requester: '申请人',
};

function formatMinutes(mins: number): string {
  if (mins < 60) return `${mins} 分钟`;
  if (mins < 1440) return `${(mins / 60).toFixed(1)} 小时`;
  return `${(mins / 1440).toFixed(1)} 天`;
}

export default function EscalationMatricesPage() {
  const [matrix, setMatrix] = useState<EscalationMatrix>({});
  const [loading, setLoading] = useState(true);
  // 加载失败时展示内置默认策略，并用错误 Alert 明确标注它不是生产配置。
  const [loadError, setLoadError] = useState<string | null>(null);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    setLoadError(null);
    EscalationMatrixApi.getMatrix()
      .then(m => {
        if (alive) setMatrix(m);
      })
      .catch((err: Error) => {
        if (!alive) return;
        setMatrix(DEFAULT_ESCALATION_MATRIX);
        setLoadError(err.message || '升级矩阵加载失败');
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
        totalTargets += l.notifyRoles.length + l.notifyUserIDs.length;
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
      width: 110,
      render: (v: string) => (
        <Tag color={priorityColorMap[v] ?? 'default'}>
          {priorityLabelMap[v] ? `${v} · ${priorityLabelMap[v]}` : v}
        </Tag>
      ),
    },
    {
      title: '级别',
      dataIndex: 'level',
      key: 'level',
      width: 80,
      render: (v: number) => <Tag color="blue">L{v}</Tag>,
    },
    {
      title: '触发时长',
      dataIndex: 'afterMinutes',
      key: 'afterMinutes',
      width: 130,
      render: (v: number) => (
        <Space>
          <Clock />
          <Text strong>{formatMinutes(v)}</Text>
        </Space>
      ),
    },
    {
      title: '通知角色',
      dataIndex: 'notifyRoles',
      key: 'notifyRoles',
      render: (roles: string[]) =>
        roles.length === 0 ? (
          <Text type="secondary">未指定角色</Text>
        ) : (
          <Space size={4} wrap>
            {roles.map(r => (
              <Tag key={r} color="geekblue">
                {roleLabelMap[r] ? `${r} · ${roleLabelMap[r]}` : r}
              </Tag>
            ))}
          </Space>
        ),
    },
    {
      title: '通知用户',
      dataIndex: 'notifyUserIDs',
      key: 'notifyUserIDs',
      render: (ids: number[]) =>
        ids.length === 0 ? (
          <Text type="secondary">未指定用户</Text>
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
      title: '升级说明',
      dataIndex: 'description',
      key: 'description',
      render: (v: string) => (v ? <Text>{v}</Text> : <Text type="secondary">无说明</Text>),
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
          按优先级配置的多级升级策略。SLA 阈值被突破后，系统按升级链逐级通知更高职责的处理人。
        </Paragraph>
      </div>

      {loadError && (
        <Alert
          title="升级矩阵加载失败"
          description={loadError + '。以下展示的是系统内置默认策略，并非生产配置，请勿据此判断线上升级行为。'}
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          action={
            <Button size="small" onClick={() => window.location.reload()}>
              重试
            </Button>
          }
        />
      )}

      <Alert
        title="本页展示后端当前生效的升级矩阵；租户自定义矩阵由运维按 SLA 策略维护"
        type="info"
        showIcon
      />

      <UsageGuideCard
        style={{ marginBottom: 16 }}
        title="怎么理解和使用升级矩阵"
        intro="升级矩阵是 SLA 超时的自动升级策略：工单超过阈值仍未处理时，系统按级别逐级通知更高职责的处理人。本页为只读查看，展示的是后端当前对本租户生效的矩阵。"
        steps={[
          '阅读方式：每个优先级（critical 最高）配置若干升级级别；「触发时长」是从告警开始计时累计的分钟数，达到即触发该级升级，按「通知角色 / 通知用户」定位升级对象。',
          '通知角色取系统主角色码（如 manager、it_admin、admin），升级时按用户主角色精确查人；角色为空且用户为空的那一级不会发出通知。',
          '生效范围：SLA 告警升级、工单超时升级与 BPMN 任务超时共用这套矩阵，无需单独启用。',
          '前置条件：SLA 时限来自「SLA 模板 / SLA 定义」，未安装 SLA 的优先级不会触发升级；站内信之外的渠道需在「系统配置」中配好 SMTP 等通知能力。',
          '建议用法：对照本表确认触发时长与组织值班响应能力是否匹配；如需调整矩阵本身，请通过后端升级矩阵接口（PUT /api/v1/escalation-matrices）提交，页面暂不提供在线编辑。',
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
            <Statistic title="升级级别总数" value={stats.totalLevels} suffix="级" styles={{ content: { color: '#1677ff' } }} />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title="通知目标数"
              value={stats.totalTargets}
              suffix="个"
              styles={{ content: { color: '#722ed1' } }}
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
                    <Tag color={priorityColorMap[priority] ?? 'default'}>
                      {priorityLabelMap[priority] ? `${priority} · ${priorityLabelMap[priority]}` : priority}
                    </Tag>
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
                      <Text>{formatMinutes(level.afterMinutes)}</Text>
                      {level.notifyRoles.map(r => (
                        <Tag key={r} color="geekblue" icon={<Bell />}>
                          {roleLabelMap[r] ? `${r} · ${roleLabelMap[r]}` : r}
                        </Tag>
                      ))}
                      {level.notifyUserIDs.map(id => (
                        <Tag key={id} color="purple" icon={<Users />}>
                          {id}
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