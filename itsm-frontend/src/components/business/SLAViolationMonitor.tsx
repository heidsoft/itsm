'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Tag,
  Button,
  Space,
  Typography,
  Input,
  Select,
  DatePicker,
  Row,
  Col,
  Statistic,
  Progress,
  Modal,
  Form,
  message,
  Badge,
  Tooltip,
  Popconfirm,
  Drawer,
} from 'antd';
import {
  AlertTriangle,
  Bell,
  BellOff,
  Clock,
  CheckCircle,
  XCircle,
  Eye,
  Settings,
  RefreshCw,
} from 'lucide-react';
import { Switch } from 'antd';
import dayjs from 'dayjs';
import SLAApi from '@/lib/api/sla-api';
import type { SLAViolation } from '@/lib/api/sla-api';

const { Text } = Typography;
const { Option } = Select;
const { RangePicker } = DatePicker;
const { Search } = Input;

interface SLAViolationMonitorProps {
  autoRefresh?: boolean;
  refreshInterval?: number;
  onViolationUpdate?: (violation: SLAViolation) => void;
}

interface SLAAlertRule {
  id: number;
  name: string;
  sla_definition_id: number;
  alert_level: 'warning' | 'critical';
  trigger_conditions: {
    time_threshold_percent: number;
    violation_types: string[];
  };
  notification_channels: string[];
  is_active: boolean;
  created_at: string;
}

export const SLAViolationMonitor: React.FC<SLAViolationMonitorProps> = ({
  autoRefresh = true,
  refreshInterval = 30000,
  onViolationUpdate,
}) => {
  // 状态管理
  const [loading, setLoading] = useState(false);
  const [violations, setViolations] = useState<SLAViolation[]>([]);
  const [filteredViolations, setFilteredViolations] = useState<SLAViolation[]>([]);
  const [stats, setStats] = useState({
    total: 0,
    open: 0,
    resolved: 0,
    critical: 0,
  });
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
    const [filters, setFilters] = useState<{
      status: string;
      severity: string;
      type: string;
      dateRange: [dayjs.Dayjs, dayjs.Dayjs] | null;
      search: string;
    }>({
      status: '',
      severity: '',
      type: '',
      dateRange: null,
      search: '',
    });
  
  // 告警规则相关
  const [alertRules, setAlertRules] = useState<SLAAlertRule[]>([]);
  const [showRuleModal, setShowRuleModal] = useState(false);
  const [showRuleDrawer, setShowRuleDrawer] = useState(false);
  const [editingRule, setEditingRule] = useState<SLAAlertRule | null>(null);
  const [form] = Form.useForm();

  // 加载违规记录
  const loadViolations = useCallback(async () => {
    try {
      setLoading(true);
      const response = await SLAApi.getSLAViolations({
        page: 1,
        page_size: 100,
        status: filters.status || undefined,
      });

      const processedViolations: SLAViolation[] = response.items.map(violation => ({
        ...violation,
        expected_time: dayjs(violation.expected_time).format('YYYY-MM-DD HH:mm:ss'),
        actual_time: dayjs(violation.actual_time).format('YYYY-MM-DD HH:mm:ss'),
      }));

      setViolations(processedViolations);
      
      // 更新统计数据
      const total = processedViolations.length;
      const open = processedViolations.filter(v => v.status === 'open').length;
      const resolved = processedViolations.filter(v => v.status === 'resolved').length;
      const critical = processedViolations.filter(v => v.severity === 'critical').length;
      
      setStats({ total, open, resolved, critical });
    } catch (error) {
      console.error('加载SLA违规记录失败:', error);
      message.error('加载违规记录失败');
    } finally {
      setLoading(false);
    }
  }, [filters.status]);

  // 加载告警规则
  const loadAlertRules = async () => {
    try {
      const rules = await SLAApi.getSLAAlerts();
      // 这里需要转换API响应格式
      setAlertRules(rules.map((rule: any) => ({
        id: rule.id || Math.random(),
        name: rule.name || '默认规则',
        sla_definition_id: rule.sla_definition_id || 1,
        alert_level: rule.alert_level || 'warning',
        trigger_conditions: rule.trigger_conditions || {
          time_threshold_percent: 80,
          violation_types: ['response_time', 'resolution_time'],
        },
        notification_channels: rule.notification_channels || ['email'],
        is_active: rule.is_active !== false,
        created_at: rule.created_at || new Date().toISOString(),
      })));
    } catch (error) {
      console.error('加载告警规则失败:', error);
    }
  };

  // 应用过滤器
  useEffect(() => {
    let filtered = [...violations];

    if (filters.severity) {
      filtered = filtered.filter(v => v.severity === filters.severity);
    }
    if (filters.type) {
      filtered = filtered.filter(v => v.violation_type === filters.type);
    }
    if (filters.dateRange) {
      const [start, end] = filters.dateRange;
      filtered = filtered.filter(v => {
        const violationDate = dayjs(v.created_at);
        return violationDate.isAfter(start) && violationDate.isBefore(end);
      });
    }
    if (filters.search) {
      const search = filters.search.toLowerCase();
      filtered = filtered.filter(v => 
        v.ticket_id.toString().includes(search) ||
        v.violation_type.toLowerCase().includes(search) ||
        v.description.toLowerCase().includes(search)
      );
    }

    setFilteredViolations(filtered);
  }, [violations, filters]);

  // 初始化和自动刷新
  useEffect(() => {
    loadViolations();
    loadAlertRules();
  }, [loadViolations]);

  useEffect(() => {
    if (autoRefresh) {
      const timer = setInterval(loadViolations, refreshInterval);
      return () => clearInterval(timer);
    }
  }, [autoRefresh, refreshInterval, loadViolations]);

  // 批量更新违规状态
  const handleBatchUpdate = async (status: string) => {
    if (selectedRowKeys.length === 0) {
      message.warning('请选择要操作的违规记录');
      return;
    }

    try {
      await Promise.all(
        selectedRowKeys.map(id => 
          SLAApi.updateSLAViolationStatus(Number(id), status)
        )
      );
      message.success(`成功更新 ${selectedRowKeys.length} 条记录`);
      setSelectedRowKeys([]);
      loadViolations();
    } catch (error) {
      message.error('批量更新失败');
    }
  };

  // 创建或更新告警规则
  const handleSaveRule = async (values: any) => {
    try {
      if (editingRule) {
        // 更新规则
        await SLAApi.updateAlertRule(editingRule.id, values);
        message.success('规则更新成功');
      } else {
        // 创建规则
        await SLAApi.createAlertRule(values);
        message.success('规则创建成功');
      }
      setShowRuleModal(false);
      setEditingRule(null);
      form.resetFields();
      loadAlertRules();
    } catch (error) {
      message.error('规则保存失败');
    }
  };

  // 删除告警规则
  const handleDeleteRule = async (ruleId: number) => {
    try {
      await SLAApi.deleteAlertRule(ruleId);
      message.success('规则删除成功');
      loadAlertRules();
    } catch (error) {
      message.error('规则删除失败');
    }
  };

  // 获取违规级别颜色
  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical': return 'red';
      case 'high': return 'orange';
      case 'medium': return 'gold';
      case 'low': return 'blue';
      default: return 'default';
    }
  };

  // 获取状态颜色
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'open': return 'red';
      case 'investigating': return 'orange';
      case 'resolved': return 'green';
      default: return 'default';
    }
  };

  // 表格列定义
  const columns = [
    {
      title: '工单ID',
      dataIndex: 'ticket_id',
      key: 'ticket_id',
      render: (id: number) => (
        <Text code className="text-sm">#{String(id).padStart(6, '0')}</Text>
      ),
    },
    {
      title: '违规类型',
      dataIndex: 'violation_type',
      key: 'violation_type',
      render: (type: string) => (
        <Tag color={type === 'response_time' ? 'orange' : 'red'}>
          {type === 'response_time' ? '响应超时' : '解决超时'}
        </Tag>
      ),
    },
    {
      title: '延迟时间',
      dataIndex: 'delay_minutes',
      key: 'delay_minutes',
      render: (minutes: number) => (
        <Space>
          <Clock size={16} />
          <Text strong className={minutes > 60 ? 'text-red-600' : 'text-orange-600'}>
            {minutes}分钟
          </Text>
        </Space>
      ),
    },
    {
      title: '严重程度',
      dataIndex: 'severity',
      key: 'severity',
      render: (severity: string) => (
        <Tag color={getSeverityColor(severity)}>
          {severity === 'critical' ? '严重' : 
           severity === 'high' ? '高' : 
           severity === 'medium' ? '中' : '低'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>
          {status === 'open' ? '待处理' : 
           status === 'investigating' ? '处理中' : '已解决'}
        </Tag>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      render: (desc: string) => (
        <Tooltip title={desc}>
          <Text className="text-sm">{desc}</Text>
        </Tooltip>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (date: string) => (
        <Text className="text-sm">{dayjs(date).format('MM-DD HH:mm')}</Text>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: SLAViolation) => (
        <Space>
          <Button
            size="small"
            icon={<Eye size={14} />}
            onClick={() => {
              // 显示详情
              Modal.info({
                title: `违规详情 #${record.id}`,
                width: 600,
                content: (
                  <div className="space-y-4">
                    <div>
                      <Text strong>工单ID:</Text> #{record.ticket_id}
                    </div>
                    <div>
                      <Text strong>违规类型:</Text> {record.violation_type}
                    </div>
                    <div>
                      <Text strong>预期时间:</Text> {record.expected_time}
                    </div>
                    <div>
                      <Text strong>实际时间:</Text> {record.actual_time}
                    </div>
                    <div>
                      <Text strong>延迟:</Text> {record.delay_minutes} 分钟
                    </div>
                    <div>
                      <Text strong>描述:</Text> {record.description}
                    </div>
                  </div>
                ),
              });
            }}
          >
            详情
          </Button>
          {record.status === 'open' && (
            <Popconfirm
              title="确认标记为已解决？"
              onConfirm={() => {
                SLAApi.updateSLAViolationStatus(record.id, 'resolved')
                  .then(() => {
                    message.success('状态更新成功');
                    loadViolations();
                    onViolationUpdate?.(record);
                  })
                  .catch(() => message.error('状态更新失败'));
              }}
            >
              <Button size="small" type="primary">
                解决
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div className="sla-violation-monitor">
      {/* 统计卡片 */}
      <Row gutter={16} className="mb-6">
        <Col xs={24} sm={6}>
          <Card>
            <Statistic
              title="总违规数"
              value={stats.total}
              prefix={<AlertTriangle size={20} />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={6}>
          <Card>
            <Statistic
              title="待处理"
              value={stats.open}
              prefix={<XCircle size={20} />}
              styles={{ content: { color: '#ff4d4f' } }}
            />
            <Progress
              percent={stats.total > 0 ? (stats.open / stats.total) * 100 : 0}
              showInfo={false}
              strokeColor="#ff4d4f"
              size="small"
              className="mt-2"
            />
          </Card>
        </Col>
        <Col xs={24} sm={6}>
          <Card>
            <Statistic
              title="已解决"
              value={stats.resolved}
              prefix={<CheckCircle size={20} />}
              styles={{ content: { color: '#52c41a' } }}
            />
            <Progress
              percent={stats.total > 0 ? (stats.resolved / stats.total) * 100 : 0}
              showInfo={false}
              strokeColor="#52c41a"
              size="small"
              className="mt-2"
            />
          </Card>
        </Col>
        <Col xs={24} sm={6}>
          <Card>
            <Statistic
              title="严重违规"
              value={stats.critical}
              prefix={<Bell size={20} />}
              styles={{ content: { color: '#faad14' } }}
            />
          </Card>
        </Col>
      </Row>

      {/* 过滤器和操作 */}
      <Card className="mb-6">
        <Row justify="space-between" align="middle">
          <Col xs={24} lg={16}>
            <Space wrap>
              <Search
                placeholder="搜索工单ID、类型或描述"
                value={filters.search}
                onChange={(e) => setFilters(prev => ({ ...prev, search: e.target.value }))}
                style={{ width: 200 }}
              />
              <Select
                placeholder="状态"
                value={filters.status}
                onChange={(value) => setFilters(prev => ({ ...prev, status: value }))}
                style={{ width: 120 }}
                allowClear
              >
                <Option value="open">待处理</Option>
                <Option value="investigating">处理中</Option>
                <Option value="resolved">已解决</Option>
              </Select>
              <Select
                placeholder="严重程度"
                value={filters.severity}
                onChange={(value) => setFilters(prev => ({ ...prev, severity: value }))}
                style={{ width: 120 }}
                allowClear
              >
                <Option value="critical">严重</Option>
                <Option value="high">高</Option>
                <Option value="medium">中</Option>
                <Option value="low">低</Option>
              </Select>
              <Select
                placeholder="违规类型"
                value={filters.type}
                onChange={(value) => setFilters(prev => ({ ...prev, type: value }))}
                style={{ width: 120 }}
                allowClear
              >
                <Option value="response_time">响应超时</Option>
                <Option value="resolution_time">解决超时</Option>
              </Select>
              <RangePicker
                value={filters.dateRange}
                onChange={(dates) => setFilters(prev => ({ 
                  ...prev, 
                  dateRange: dates && dates[0] && dates[1] ? [dates[0], dates[1]] : null 
                }))}
              />
            </Space>
          </Col>
          <Col xs={24} lg={8}>
            <Space>
              <Button
                icon={<RefreshCw size={16} />}
                onClick={loadViolations}
                loading={loading}
              >
                刷新
              </Button>
              <Button
                icon={<Settings size={16} />}
                onClick={() => setShowRuleDrawer(true)}
              >
                告警设置
              </Button>
              {selectedRowKeys.length > 0 && (
                <>
                  <Button
                    onClick={() => handleBatchUpdate('resolved')}
                    type="primary"
                  >
                    批量解决 ({selectedRowKeys.length})
                  </Button>
                </>
              )}
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 违规记录表格 */}
      <Card>
        <Table
          dataSource={filteredViolations}
          columns={columns}
          rowKey="id"
          loading={loading}
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
            getCheckboxProps: (record) => ({
              disabled: record.status === 'resolved',
            }),
          }}
          pagination={{
            total: filteredViolations.length,
            pageSize: 20,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条记录`,
          }}
        />
      </Card>

      {/* 告警规则抽屉 */}
      <Drawer
        title={
          <Space>
            <Bell size={20} />
            SLA告警规则配置
          </Space>
        }
        size="large"
        style={{ width: 600 }}
        open={showRuleDrawer}
        onClose={() => setShowRuleDrawer(false)}
        extra={
          <Button
            type="primary"
            icon={<Bell size={16} />}
            onClick={() => setShowRuleModal(true)}
          >
            新建规则
          </Button>
        }
      >
        <div className="space-y-4">
          {alertRules.map(rule => (
            <Card key={rule.id} size="small">
              <Row justify="space-between" align="middle">
                <Col>
                  <div className="space-y-2">
                    <div className="flex items-center gap-2">
                      <Text strong>{rule.name}</Text>
                      <Tag color={rule.alert_level === 'critical' ? 'red' : 'orange'}>
                        {rule.alert_level === 'critical' ? '严重' : '警告'}
                      </Tag>
                      <Badge status={rule.is_active ? 'success' : 'default'} text={rule.is_active ? '启用' : '禁用'} />
                    </div>
                    <Text type="secondary" className="text-sm">
                      触发阈值: {rule.trigger_conditions.time_threshold_percent}% | 
                      通知方式: {rule.notification_channels.join(', ')}
                    </Text>
                  </div>
                </Col>
                <Col>
                  <Space>
                    <Button
                      size="small"
                      icon={<Settings size={14} />}
                      onClick={() => {
                        setEditingRule(rule);
                        form.setFieldsValue(rule);
                        setShowRuleModal(true);
                        setShowRuleDrawer(false);
                      }}
                    >
                      编辑
                    </Button>
                    <Popconfirm
                      title="确认删除此规则？"
                      onConfirm={() => handleDeleteRule(rule.id)}
                    >
                      <Button size="small" danger>
                        删除
                      </Button>
                    </Popconfirm>
                  </Space>
                </Col>
              </Row>
            </Card>
          ))}
          
          {alertRules.length === 0 && (
            <div className="text-center py-8">
              <BellOff size={48} className="mx-auto mb-4 text-gray-400" />
              <Text type="secondary">暂无告警规则</Text>
            </div>
          )}
        </div>
      </Drawer>

      {/* 告警规则模态框 */}
      <Modal
        title={editingRule ? '编辑告警规则' : '新建告警规则'}
        open={showRuleModal}
        onCancel={() => {
          setShowRuleModal(false);
          setEditingRule(null);
          form.resetFields();
        }}
        footer={null}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSaveRule}
        >
          <Form.Item
            name="name"
            label="规则名称"
            rules={[{ required: true, message: '请输入规则名称' }]}
          >
            <Input placeholder="例如：高优先级工单告警" />
          </Form.Item>

          <Form.Item
            name="alert_level"
            label="告警级别"
            rules={[{ required: true, message: '请选择告警级别' }]}
          >
            <Select placeholder="选择告警级别">
              <Option value="warning">警告</Option>
              <Option value="critical">严重</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name={['trigger_conditions', 'time_threshold_percent']}
            label="时间阈值 (%)"
            rules={[{ required: true, message: '请输入时间阈值' }]}
          >
            <Input
              type="number"
              min={1}
              max={100}
              placeholder="当SLA时间使用超过此百分比时触发告警"
              suffix="%"
            />
          </Form.Item>

          <Form.Item
            name={['trigger_conditions', 'violation_types']}
            label="违规类型"
            rules={[{ required: true, message: '请选择违规类型' }]}
          >
            <Select mode="multiple" placeholder="选择违规类型">
              <Option value="response_time">响应超时</Option>
              <Option value="resolution_time">解决超时</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="notification_channels"
            label="通知方式"
            rules={[{ required: true, message: '请选择通知方式' }]}
          >
            <Select mode="multiple" placeholder="选择通知方式">
              <Option value="email">邮件</Option>
              <Option value="sms">短信</Option>
              <Option value="webhook">Webhook</Option>
              <Option value="push">推送通知</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="is_active"
            label="启用状态"
            valuePropName="checked"
            initialValue={true}
          >
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {editingRule ? '更新' : '创建'}
              </Button>
              <Button
                onClick={() => {
                  setShowRuleModal(false);
                  setEditingRule(null);
                  form.resetFields();
                }}
              >
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default SLAViolationMonitor;