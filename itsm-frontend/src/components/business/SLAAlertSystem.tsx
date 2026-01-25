'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Row,
  Col,
  Table,
  Tag,
  Button,
  Modal,
  Form,
  Input,
  Select,
  Switch,
  Space,
  Typography,
  Badge,
  Alert,
  message,
  App,
  Tooltip,
  Popconfirm,
} from 'antd';
import {
  SettingOutlined,
  BellOutlined,
  WarningOutlined,
  CloseCircleOutlined,
  CheckCircleOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  EyeOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import SLAApi from '@/lib/api/sla-api';

const { Title, Text } = Typography;
const { Option } = Select;
const { TextArea } = Input;

interface AlertRule {
  id: number;
  name: string;
  sla_definition_id: number;
  alert_level: 'warning' | 'critical' | 'severe';
  threshold_percentage: number; // 70, 85, 95
  notification_channels: string[]; // ['email', 'sms', 'in_app']
  escalation_enabled: boolean;
  escalation_levels: Array<{
    level: number;
    threshold: number;
    notify_users: number[];
  }>;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

interface AlertHistory {
  id: number;
  ticket_id: number;
  ticket_number: string;
  ticket_title: string;
  alert_rule_id: number;
  alert_rule_name: string;
  alert_level: string;
  threshold_percentage: number;
  actual_percentage: number;
  notification_sent: boolean;
  escalation_level: number;
  created_at: string;
  resolved_at?: string;
}

interface SLAAlertSystemProps {
  slaDefinitionId?: number;
  onAlertTriggered?: (alert: AlertHistory) => void;
}

export const SLAAlertSystem: React.FC<SLAAlertSystemProps> = ({
  slaDefinitionId,
  onAlertTriggered,
}) => {
  const { message: antMessage } = App.useApp();
  const [alertRules, setAlertRules] = useState<AlertRule[]>([]);
  const [alertHistory, setAlertHistory] = useState<AlertHistory[]>([]);
  const [loading, setLoading] = useState(false);
  const [ruleModalVisible, setRuleModalVisible] = useState(false);
  const [editingRule, setEditingRule] = useState<AlertRule | null>(null);
  const [form] = Form.useForm();

  // 加载预警规则
  const loadAlertRules = useCallback(async () => {
    try {
      setLoading(true);
      const data = await SLAApi.getAlertRules({ sla_definition_id: slaDefinitionId });
      setAlertRules(data);
    } catch (error) {
      console.error('Failed to load alert rules:', error);
      antMessage.error('加载预警规则失败');
      setAlertRules([]);
    } finally {
      setLoading(false);
    }
  }, [slaDefinitionId, antMessage]);

  // 加载预警历史
  const loadAlertHistory = useCallback(async () => {
    try {
      // 调用实际API
      const { default: SLAApi } = await import('@/lib/api/sla-api');
      const data = await SLAApi.getAlertHistory({ sla_definition_id: slaDefinitionId });
      setAlertHistory(data.items);
    } catch (error) {
      console.error('Failed to load alert history:', error);
      // 失败时设置为空数组，不使用Mock数据
      setAlertHistory([]);
    }
  }, [slaDefinitionId]);

  useEffect(() => {
    loadAlertRules();
    loadAlertHistory();
  }, [loadAlertRules, loadAlertHistory]);

  // 保存预警规则
  const handleSaveRule = useCallback(async () => {
    try {
      const values = await form.validateFields();
      const ruleData: Partial<AlertRule> = {
        ...values,
        sla_definition_id: slaDefinitionId || values.sla_definition_id,
      };

      const { default: SLAApi } = await import('@/lib/api/sla-api');

      if (editingRule) {
        await SLAApi.updateAlertRule(editingRule.id, ruleData);
        antMessage.success('预警规则已更新');
      } else {
        // 确保必要的字段存在
        const createData = {
          name: ruleData.name!,
          sla_definition_id: ruleData.sla_definition_id!,
          alert_level: ruleData.alert_level!,
          threshold_percentage: Number(ruleData.threshold_percentage),
          notification_channels: ruleData.notification_channels!,
          escalation_enabled: ruleData.escalation_enabled,
          escalation_levels: ruleData.escalation_levels,
          is_active: ruleData.is_active !== false, // 默认为 true
        };
        await SLAApi.createAlertRule(createData);
        antMessage.success('预警规则已创建');
      }

      setRuleModalVisible(false);
      setEditingRule(null);
      form.resetFields();
      loadAlertRules();
    } catch (error) {
      console.error('Failed to save alert rule:', error);
      antMessage.error('保存预警规则失败');
    }
  }, [form, editingRule, slaDefinitionId, antMessage, loadAlertRules]);

  // 删除预警规则
  const handleDeleteRule = useCallback(
    async (id: number) => {
      try {
        const { default: SLAApi } = await import('@/lib/api/sla-api');
        await SLAApi.deleteAlertRule(id);
        antMessage.success('预警规则已删除');
        loadAlertRules();
      } catch (error) {
        console.error('Failed to delete alert rule:', error);
        antMessage.error('删除预警规则失败');
      }
    },
    [antMessage, loadAlertRules]
  );

  // 切换规则状态
  const handleToggleRuleStatus = useCallback(
    async (id: number, isActive: boolean) => {
      try {
        const { default: SLAApi } = await import('@/lib/api/sla-api');
        await SLAApi.updateAlertRule(id, { is_active: !isActive });
        antMessage.success(`预警规则已${!isActive ? '启用' : '禁用'}`);
        loadAlertRules();
      } catch (error) {
        console.error('Failed to toggle rule status:', error);
        antMessage.error('更新规则状态失败');
      }
    },
    [antMessage, loadAlertRules]
  );

  // 打开编辑模态框
  const handleOpenRuleModal = useCallback(
    (rule?: AlertRule) => {
      if (rule) {
        setEditingRule(rule);
        form.setFieldsValue(rule);
      } else {
        setEditingRule(null);
        form.resetFields();
        form.setFieldsValue({
          threshold_percentage: 70,
          notification_channels: ['in_app'],
          escalation_enabled: false,
          is_active: true,
        });
      }
      setRuleModalVisible(true);
    },
    [form]
  );

  // 获取预警级别颜色
  const getAlertLevelColor = (level: string) => {
    const colors: Record<string, string> = {
      warning: 'orange',
      critical: 'red',
      severe: 'red',
    };
    return colors[level] || 'default';
  };

  // 获取预警级别文本
  const getAlertLevelText = (level: string) => {
    const texts: Record<string, string> = {
      warning: '警告',
      critical: '严重',
      severe: '严重',
    };
    return texts[level] || level;
  };

  // 预警规则表格列
  const ruleColumns: ColumnsType<AlertRule> = [
    {
      title: '规则名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => <Text strong>{text}</Text>,
    },
    {
      title: '预警级别',
      dataIndex: 'alert_level',
      key: 'alert_level',
      render: (level: string) => (
        <Tag color={getAlertLevelColor(level)}>{getAlertLevelText(level)}</Tag>
      ),
    },
    {
      title: '阈值',
      dataIndex: 'threshold_percentage',
      key: 'threshold_percentage',
      render: (percentage: number) => (
        <Text strong style={{ color: '#1890ff' }}>
          {percentage}%
        </Text>
      ),
    },
    {
      title: '通知渠道',
      dataIndex: 'notification_channels',
      key: 'notification_channels',
      render: (channels: string[]) => (
        <Space>
          {channels.map(channel => (
            <Tag key={channel} color='blue'>
              {channel === 'email' ? '邮件' : channel === 'sms' ? '短信' : '站内'}
            </Tag>
          ))}
        </Space>
      ),
    },
    {
      title: '升级机制',
      dataIndex: 'escalation_enabled',
      key: 'escalation_enabled',
      render: (enabled: boolean) => (
        <Tag color={enabled ? 'green' : 'default'}>{enabled ? '已启用' : '未启用'}</Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
      render: (isActive: boolean, record: AlertRule) => (
        <Switch
          checked={isActive}
          onChange={checked => handleToggleRuleStatus(record.id, !checked)}
        />
      ),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: AlertRule) => (
        <Space>
          <Button
            type='link'
            size='small'
            icon={<EditOutlined />}
            onClick={() => handleOpenRuleModal(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title='确定要删除这个预警规则吗？'
            onConfirm={() => handleDeleteRule(record.id)}
          >
            <Button type='link' size='small' danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  // 预警历史表格列
  const historyColumns: ColumnsType<AlertHistory> = [
    {
      title: '工单编号',
      dataIndex: 'ticket_number',
      key: 'ticket_number',
      render: (text: string) => (
        <Text strong style={{ color: '#1890ff' }}>
          {text}
        </Text>
      ),
    },
    {
      title: '工单标题',
      dataIndex: 'ticket_title',
      key: 'ticket_title',
    },
    {
      title: '预警规则',
      dataIndex: 'alert_rule_name',
      key: 'alert_rule_name',
    },
    {
      title: '预警级别',
      dataIndex: 'alert_level',
      key: 'alert_level',
      render: (level: string) => (
        <Tag color={getAlertLevelColor(level)}>{getAlertLevelText(level)}</Tag>
      ),
    },
    {
      title: '阈值/实际',
      key: 'threshold',
      render: (_: any, record: AlertHistory) => (
        <div>
          <Text type='secondary'>阈值: {record.threshold_percentage}%</Text>
          <br />
          <Text
            style={{
              color:
                record.actual_percentage >= record.threshold_percentage ? '#ff4d4f' : '#52c41a',
            }}
          >
            实际: {record.actual_percentage.toFixed(1)}%
          </Text>
        </div>
      ),
    },
    {
      title: '通知状态',
      dataIndex: 'notification_sent',
      key: 'notification_sent',
      render: (sent: boolean) => (
        <Badge status={sent ? 'success' : 'default'} text={sent ? '已发送' : '未发送'} />
      ),
    },
    {
      title: '升级级别',
      dataIndex: 'escalation_level',
      key: 'escalation_level',
      render: (level: number) => (level > 0 ? `L${level}` : '-'),
    },
    {
      title: '触发时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (date: string) => format(new Date(date), 'yyyy-MM-dd HH:mm:ss', { locale: zhCN }),
    },
    {
      title: '状态',
      key: 'status',
      render: (_: any, record: AlertHistory) =>
        record.resolved_at ? <Tag color='green'>已解决</Tag> : <Tag color='orange'>进行中</Tag>,
    },
  ];

  return (
    <div className='space-y-6'>
      {/* 预警规则配置 */}
      <Card
        title={
          <div className='flex items-center justify-between'>
            <div className='flex items-center gap-2'>
              <SettingOutlined />
              <span>预警规则配置</span>
            </div>
            <Button type='primary' icon={<PlusOutlined />} onClick={() => handleOpenRuleModal()}>
              创建预警规则
            </Button>
          </div>
        }
      >
        <Alert
          message='三级预警机制'
          description='系统支持三级预警：70%（提醒）、85%（警告）、95%（严重）。当SLA达成率低于阈值时，将自动触发预警并发送通知。'
          type='info'
          showIcon
          className='mb-4'
        />
        <Table
          columns={ruleColumns}
          dataSource={alertRules}
          rowKey='id'
          loading={loading}
          pagination={false}
        />
      </Card>

      {/* 预警历史 */}
      <Card
        title={
          <div className='flex items-center gap-2'>
            <BellOutlined />
            <span>预警历史记录</span>
            <Badge count={alertHistory.length} showZero className='ml-2' />
          </div>
        }
      >
        <Table
          columns={historyColumns}
          dataSource={alertHistory}
          rowKey='id'
          pagination={{
            pageSize: 10,
            showSizeChanger: true,
            showTotal: total => `共 ${total} 条记录`,
          }}
        />
      </Card>

      {/* 创建/编辑预警规则模态框 */}
      <Modal
        title={editingRule ? '编辑预警规则' : '创建预警规则'}
        open={ruleModalVisible}
        onOk={handleSaveRule}
        onCancel={() => {
          setRuleModalVisible(false);
          setEditingRule(null);
          form.resetFields();
        }}
        width={700}
      >
        <Form form={form} layout='vertical'>
          <Form.Item
            name='name'
            label='规则名称'
            rules={[{ required: true, message: '请输入规则名称' }]}
          >
            <Input placeholder='例如：P1工单-严重预警' />
          </Form.Item>
          <Form.Item
            name='alert_level'
            label='预警级别'
            rules={[{ required: true, message: '请选择预警级别' }]}
          >
            <Select placeholder='请选择预警级别'>
              <Option value='warning'>警告</Option>
              <Option value='critical'>严重</Option>
              <Option value='severe'>严重（最高）</Option>
            </Select>
          </Form.Item>
          <Form.Item
            name='threshold_percentage'
            label='阈值百分比'
            rules={[
              { required: true, message: '请输入阈值百分比' },
              { type: 'number', min: 0, max: 100, message: '阈值必须在0-100之间' },
            ]}
          >
            <Input type='number' placeholder='例如：70, 85, 95' addonAfter='%' />
          </Form.Item>
          <Form.Item
            name='notification_channels'
            label='通知渠道'
            rules={[{ required: true, message: '请至少选择一个通知渠道' }]}
          >
            <Select mode='multiple' placeholder='请选择通知渠道'>
              <Option value='email'>邮件</Option>
              <Option value='sms'>短信</Option>
              <Option value='in_app'>站内消息</Option>
            </Select>
          </Form.Item>
          <Form.Item name='escalation_enabled' label='启用升级机制' valuePropName='checked'>
            <Switch />
          </Form.Item>
          <Form.Item
            noStyle
            shouldUpdate={(prevValues, currentValues) =>
              prevValues.escalation_enabled !== currentValues.escalation_enabled
            }
          >
            {({ getFieldValue }) =>
              getFieldValue('escalation_enabled') ? (
                <Form.Item
                  name='escalation_levels'
                  label='升级级别配置'
                  tooltip='配置多级升级，当达到更高阈值时自动升级并通知更多人员'
                >
                  <TextArea
                    rows={4}
                    placeholder='JSON格式，例如：[{"level": 1, "threshold": 95, "notify_users": [1,2]}]'
                  />
                </Form.Item>
              ) : null
            }
          </Form.Item>
          <Form.Item name='is_active' label='启用状态' valuePropName='checked'>
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};
