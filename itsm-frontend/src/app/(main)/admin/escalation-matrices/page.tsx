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
  critical: 'Fatal',
  high: 'High',
  medium: 'Medium',
  low: 'Low',
};

// 后端 notifyRoles 取主角色词表原值，展示时补中文，避免运维对着 it_admin 猜职责。
const roleLabelMap: Record<string, string> = {
  admin: 'Tenant Administrator',
  it_admin: 'IT Administrator',
  manager: 'Manager',
  agent: 'Help Desk Agent',
  requester: 'Requester',
};

function formatMinutes(mins: number): string {
  if (mins < 60) return `${mins} minutes`;
  if (mins < 1440) return `${(mins / 60).toFixed(1)} hours`;
  return `${(mins / 1440).toFixed(1)} days`;
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
        setLoadError(err.message || 'Upgrade matrix loading failed');
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
      title: 'Priority',
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
      title: 'Level',
      dataIndex: 'level',
      key: 'level',
      width: 80,
      render: (v: number) => <Tag color="blue">L{v}</Tag>,
    },
    {
      title: 'Trigger duration',
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
      title: 'Notification role',
      dataIndex: 'notifyRoles',
      key: 'notifyRoles',
      render: (roles: string[]) =>
        roles.length === 0 ? (
          <Text type="secondary">No role specified</Text>
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
      title: 'Notify user',
      dataIndex: 'notifyUserIDs',
      key: 'notifyUserIDs',
      render: (ids: number[]) =>
        ids.length === 0 ? (
          <Text type="secondary">No user specified</Text>
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
      title: 'Upgrade instructions',
      dataIndex: 'description',
      key: 'description',
      render: (v: string) => (v ? <Text>{v}</Text> : <Text type="secondary">No description</Text>),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <Title level={3} style={{ marginBottom: 4 }}>
          <AlertTriangle style={{ marginRight: 8 }} />

          SLA upgrade matrix
        </Title>
        <Paragraph type="secondary">

          Multi-level upgrade strategy configured by priority. After the SLA threshold is exceeded, the system notifies processors with higher responsibilities step by step according to the escalation chain.
        </Paragraph>
      </div>

      {loadError && (
        <Alert
          title="Upgrade matrix loading failed"
          description={loadError + '. What is shown below is the default policy built into the system, not the production configuration. Please do not judge online upgrade behavior based on this.'}
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          action={
            <Button size="small" onClick={() => window.location.reload()}>

              Retry
            </Button>
          }
        />
      )}

      <Alert
        title="This page displays the upgrade matrix currently in effect on the backend; the tenant's custom matrix is maintained by operation and maintenance according to the SLA policy"
        type="info"
        showIcon
      />

      <UsageGuideCard
        style={{ marginBottom: 16 }}
        title="How to understand and use the upgrade matrix"
        intro="The escalation matrix is an automatic escalation strategy for SLA timeout: when a work order exceeds the threshold and remains unprocessed, the system notifies the processor with higher responsibilities step by step. This page is read-only and displays the backend matrix currently in effect for this tenant."
        steps={[
          'Reading method: Configure several upgrade levels for each priority (critical is the highest); "Trigger duration" is the accumulated number of minutes from the start of the alarm. Once reached, an upgrade of this level is triggered. Press "Notify Role/Notify User" to locate the upgrade target.',
          'The notification role takes the system main role code (such as manager, it_admin, admin). When upgrading, the person will be accurately checked according to the user\'s main role; no notification will be issued at the level where the role is empty and the user is empty.',
          'Effective scope: SLA alarm upgrade, work order timeout upgrade and BPMN task timeout share this matrix and do not need to be enabled separately.',
          'Prerequisite: SLA time limit comes from "SLA Template/SLA Definition". Priorities without SLA installed will not trigger upgrades; channels other than intra-site messaging need to configure notification capabilities such as SMTP in "System Configuration".',
          'Recommended usage: Compare this table to confirm whether the trigger duration matches the organization\'s on-duty response capability; if you need to adjust the matrix itself, please submit it through the back-end upgrade matrix interface (PUT /api/v1/escalation-matrices). The page does not currently provide online editing.',
        ]}
      />

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic title="Override priority" value={stats.priorities} suffix="items" />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic title="Total number of upgrade levels" value={stats.totalLevels} suffix="level" styles={{ content: { color: '#1677ff' } }} />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title="Number of notification targets"
              value={stats.totalTargets}
              suffix="items"
              styles={{ content: { color: '#722ed1' } }}
            />
          </Card>
        </Col>
      </Row>

      <Card title="Upgrade matrix (by priority and level)">
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

      <Card title="Group views by priority">
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
                    <Text type="secondary">{levels.length}  level upgrade</Text>
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