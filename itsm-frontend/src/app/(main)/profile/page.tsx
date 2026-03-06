'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Form,
  Input,
  Button,
  Avatar,
  Typography,
  Space,
  Divider,
  message,
  Row,
  Col,
  Tabs,
  Upload,
  Select,
  Switch,
  Statistic,
  Progress,
  Tag,
  Table,
} from 'antd';
import {
  User,
  Mail,
  Phone,
  Building,
  Shield,
  Bell,
  Key,
  Camera,
  Save,
  Edit,
  Ticket,
  Clock,
  CheckCircle,
  Star,
  Settings,
  Lock,
  LogOut,
  Activity,
} from 'lucide-react';
import { PageHeader } from '@/components/layout/PageHeader';
import { UserApi } from '@/lib/api/user-api';
import { TicketApi } from '@/lib/api/ticket-api';
import { useI18n } from '@/lib/i18n';
import { useAuthStore, useAuthStoreHydration } from '@/lib/store/auth-store';

const { Title, Text } = Typography;
const { TabPane } = Tabs;

// 独特的设计系统
const DESIGN = {
  colors: {
    primary: '#0f172a',
    accent: '#3b82f6',
    success: '#10b981',
    warning: '#f59e0b',
    danger: '#ef4444',
    surface: '#ffffff',
    border: '#e2e8f0',
    text: '#1e293b',
    textMuted: '#64748b',
    bgSubtle: '#f8fafc',
    gradient: {
      primary: 'linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%)',
      success: 'linear-gradient(135deg, #10b981 0%, #059669 100%)',
      warning: 'linear-gradient(135deg, #f59e0b 0%, #d97706 100%)',
    },
  },
  shadows: {
    card: '0 1px 3px rgb(0 0 0 / 0.05)',
    cardHover: '0 10px 40px -10px rgb(0 0 0 / 0.15)',
    glow: (color: string) => `0 0 30px ${color}20`,
  },
  radius: {
    sm: '8px',
    md: '12px',
    lg: '16px',
    xl: '20px',
    full: '9999px',
  },
};

interface UserProfile {
  id: number;
  username: string;
  email: string;
  name: string;
  department?: string;
  phone?: string;
  role?: string;
  avatar?: string;
  tenant_id?: number;
  created_at?: string;
  last_login?: string;
}

interface UserStats {
  totalTickets: number;
  resolvedTickets: number;
  avgResolutionTime: number;
  satisfactionScore: number;
  responseRate: number;
}

interface ActivityItem {
  id: number;
  action: string;
  target: string;
  time: string;
  status: 'completed' | 'pending' | 'cancelled';
}

// Mock activity data
const mockActivities: ActivityItem[] = [
  { id: 1, action: '创建工单', target: '网络无法访问', time: '2小时前', status: 'completed' },
  { id: 2, action: '审批通过', target: '服务器申请', time: '5小时前', status: 'completed' },
  { id: 3, action: '更新工单', target: '数据库连接问题', time: '1天前', status: 'completed' },
  { id: 4, action: '评论', target: 'VPN权限申请', time: '2天前', status: 'completed' },
];

export default function ProfilePage() {
  const { t } = useI18n();
  const { user } = useAuthStore();
  useAuthStoreHydration();

  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [editing, setEditing] = useState(false);
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState<UserStats | null>(null);
  const [activities, setActivities] = useState<ActivityItem[]>([]);
  const [profileForm] = Form.useForm();
  const [preferencesForm] = Form.useForm();

  useEffect(() => {
    loadProfile();
    loadStats();
    loadActivities();
  }, []);

  const loadProfile = async () => {
    try {
      setLoading(true);
      if (user) {
        const userData = {
          id: user.id,
          username: user.username || '',
          email: user.email || '',
          name: user.name || '',
          department: user.department || '',
          phone: '',
          role: user.role || 'user',
          created_at: user.createdAt || new Date().toISOString(),
          last_login: undefined,
        };
        setProfile(userData);
        profileForm.setFieldsValue(userData);
      }
    } catch (error) {
      console.error('Failed to load profile:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadStats = async () => {
    try {
      const ticketStats = await TicketApi.getTicketStats();
      setStats({
        totalTickets: ticketStats.total || 0,
        resolvedTickets: ticketStats.resolved || 0,
        avgResolutionTime: 0, // 需要后端支持
        satisfactionScore: 0, // 需要后端支持
        responseRate: ticketStats.total > 0
          ? Math.round(((ticketStats.total - ticketStats.open) / ticketStats.total) * 100)
          : 0,
      });
    } catch (error) {
      console.error('Failed to load stats:', error);
      // 使用默认值
      setStats({
        totalTickets: 0,
        resolvedTickets: 0,
        avgResolutionTime: 0,
        satisfactionScore: 0,
        responseRate: 0,
      });
    }
  };

  const loadActivities = async () => {
    try {
      // 从 localStorage 获取用户活动记录
      const savedActivities = localStorage.getItem('user_activities');
      if (savedActivities) {
        setActivities(JSON.parse(savedActivities));
      } else {
        // 如果没有记录，使用空数组
        setActivities([]);
      }
    } catch (error) {
      console.error('Failed to load activities:', error);
      setActivities([]);
    }
  };

  const handleSaveProfile = async (values: any) => {
    if (!profile?.id) {
      message.error('用户信息不完整');
      return;
    }
    setLoading(true);
    try {
      await UserApi.updateUser(profile.id, {
        name: values.name,
        email: values.email,
        phone: values.phone,
        department: values.department,
      });
      message.success('个人信息更新成功');
      setEditing(false);
 => prev ? {      setProfile(prev ...prev, ...values } : null);

      // 更新 auth store 中的用户信息
      const { updateUser } = useAuthStore.getState();
      updateUser({ ...values });
    } catch (error) {
      console.error('Failed to update profile:', error);
      message.error('更新失败，请重试');
    } finally {
      setLoading(false);
    }
  };

  const handleSavePreferences = async (values: any) => {
    try {
      message.success('偏好设置保存成功');
    } catch (error) {
      message.error('保存失败，请重试');
    }
  };

  // 获取用户首字母
  const userInitial = (profile?.name || profile?.username || 'U').charAt(0).toUpperCase();

  // 角色标签颜色
  const roleColor = profile?.role === 'admin' || profile?.role === 'super_admin' ? DESIGN.colors.accent : DESIGN.colors.textMuted;

  return (
    <div className="min-h-screen" style={{ background: DESIGN.colors.bgSubtle }}>
      <PageHeader
        title="个人中心"
      />

      <div style={{ padding: '24px', maxWidth: 1200, margin: '0 auto' }}>
        <Row gutter={[24, 24]}>
          {/* 左侧 - 用户信息卡片 */}
          <Col xs={24} lg={8}>
            <Card
              style={{
                borderRadius: DESIGN.radius.lg,
                border: 'none',
                boxShadow: DESIGN.shadows.card,
                overflow: 'hidden',
              }}
              styles={{ body: { padding: 0 } }}
            >
              {/* 头部背景 */}
              <div
                style={{
                  height: 120,
                  background: DESIGN.colors.gradient.primary,
                  position: 'relative',
                }}
              >
                {/* 编辑按钮 */}
                <Button
                  type="text"
                  icon={<Settings size={18} />}
                  onClick={() => setEditing(!editing)}
                  style={{
                    position: 'absolute',
                    top: 16,
                    right: 16,
                    color: 'rgba(255,255,255,0.8)',
                    background: 'rgba(255,255,255,0.2)',
                  }}
                />
              </div>

              {/* 头像和信息 */}
              <div style={{ padding: '0 24px 24px', marginTop: -50 }}>
                <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 16 }}>
                  <Avatar
                    size={100}
                    style={{
                      background: DESIGN.colors.gradient.primary,
                      fontSize: 40,
                      fontWeight: 700,
                      border: '4px solid white',
                      boxShadow: DESIGN.shadows.cardHover,
                    }}
                  >
                    {userInitial}
                  </Avatar>
                </div>

                <div style={{ textAlign: 'center', marginBottom: 16 }}>
                  <Title level={4} style={{ margin: 0, marginBottom: 4 }}>
                    {profile?.name || profile?.username || '用户'}
                  </Title>
                  <Text style={{ color: DESIGN.colors.textMuted }}>
                    {profile?.email || 'user@example.com'}
                  </Text>
                </div>

                <div style={{ display: 'flex', justifyContent: 'center', gap: 8, marginBottom: 20 }}>
                  <Tag
                    color={roleColor}
                    style={{
                      padding: '4px 12px',
                      borderRadius: DESIGN.radius.full,
                      fontWeight: 600,
                    }}
                  >
                    {profile?.role === 'admin' ? '管理员' : profile?.role === 'super_admin' ? '超级管理员' : '用户'}
                  </Tag>
                  <Tag
                    style={{
                      padding: '4px 12px',
                      borderRadius: DESIGN.radius.full,
                      background: `${DESIGN.colors.success}15`,
                      color: DESIGN.colors.success,
                      border: 'none',
                    }}
                  >
                    <CheckCircle size={12} /> 已激活
                  </Tag>
                </div>

                <Divider style={{ margin: '16px 0' }} />

                {/* 用户详情 */}
                <Space direction="vertical" style={{ width: '100%' }} size="middle">
                  <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                    <Building size={16} style={{ color: DESIGN.colors.textMuted }} />
                    <Text>{profile?.department || '技术部'}</Text>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                    <Phone size={16} style={{ color: DESIGN.colors.textMuted }} />
                    <Text>{profile?.phone || '未设置'}</Text>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                    <Clock size={16} style={{ color: DESIGN.colors.textMuted }} />
                    <Text>上次登录: {profile?.last_login || '刚刚'}</Text>
                  </div>
                </Space>
              </div>
            </Card>

            {/* 统计卡片 */}
            <Card
              style={{
                marginTop: 24,
                borderRadius: DESIGN.radius.lg,
                border: 'none',
                boxShadow: DESIGN.shadows.card,
              }}
            >
              <Title level={5} style={{ marginBottom: 20 }}>工作统计</Title>
              <Row gutter={[16, 16]}>
                <Col span={12}>
                  <Statistic
                    title="提交工单"
                    value={stats?.totalTickets || 0}
                    prefix={<Ticket size={16} style={{ color: DESIGN.colors.accent }} />}
                  />
                </Col>
                <Col span={12}>
                  <Statistic
                    title="已解决"
                    value={stats?.resolvedTickets || 0}
                    prefix={<CheckCircle size={16} style={{ color: DESIGN.colors.success }} />}
                  />
                </Col>
                <Col span={12}>
                  <Statistic
                    title="满意度"
                    value={stats?.satisfactionScore || 0}
                    suffix="/5"
                    prefix={<Star size={16} style={{ color: DESIGN.colors.warning }} />}
                  />
                </Col>
                <Col span={12}>
                  <Statistic
                    title="响应率"
                    value={stats?.responseRate || 0}
                    suffix="%"
                    prefix={<Activity size={16} style={{ color: DESIGN.colors.accent }} />}
                  />
                </Col>
              </Row>
            </Card>
          </Col>

          {/* 右侧 - 内容区域 */}
          <Col xs={24} lg={16}>
            <Card
              style={{
                borderRadius: DESIGN.radius.lg,
                border: 'none',
                boxShadow: DESIGN.shadows.card,
              }}
            >
              <Tabs defaultActiveKey="profile" size="large">
                {/* 基本信息 */}
                <TabPane
                  tab={
                    <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <User size={16} />
                      基本信息
                    </span>
                  }
                >
                  <Form
                    form={profileForm}
                    layout="vertical"
                    onFinish={handleSaveProfile}
                    initialValues={profile || undefined}
                  >
                    <Row gutter={24}>
                      <Col xs={24} md={12}>
                        <Form.Item
                          label="姓名"
                          name="name"
                          rules={[{ required: true, message: '请输入姓名' }]}
                        >
                          <Input prefix={<User size={16} style={{ color: DESIGN.colors.textMuted }} />} />
                        </Form.Item>
                      </Col>
                      <Col xs={24} md={12}>
                        <Form.Item
                          label="用户名"
                          name="username"
                        >
                          <Input disabled prefix={<User size={16} style={{ color: DESIGN.colors.textMuted }} />} />
                        </Form.Item>
                      </Col>
                      <Col xs={24} md={12}>
                        <Form.Item
                          label="邮箱"
                          name="email"
                          rules={[{ required: true, type: 'email', message: '请输入有效邮箱' }]}
                        >
                          <Input prefix={<Mail size={16} style={{ color: DESIGN.colors.textMuted }} />} />
                        </Form.Item>
                      </Col>
                      <Col xs={24} md={12}>
                        <Form.Item
                          label="电话"
                          name="phone"
                        >
                          <Input prefix={<Phone size={16} style={{ color: DESIGN.colors.textMuted }} />} />
                        </Form.Item>
                      </Col>
                      <Col xs={24} md={12}>
                        <Form.Item
                          label="部门"
                          name="department"
                        >
                          <Input prefix={<Building size={16} style={{ color: DESIGN.colors.textMuted }} />} />
                        </Form.Item>
                      </Col>
                      <Col xs={24} md={12}>
                        <Form.Item label="角色">
                          <Tag color={roleColor} style={{ padding: '4px 12px' }}>
                            {profile?.role === 'admin' ? '管理员' : profile?.role === 'super_admin' ? '超级管理员' : '用户'}
                          </Tag>
                        </Form.Item>
                      </Col>
                    </Row>

                    <div style={{ marginTop: 24, textAlign: 'right' }}>
                      <Space>
                        {editing && (
                          <Button onClick={() => setEditing(false)}>取消</Button>
                        )}
                        <Button
                          type="primary"
                          htmlType="submit"
                          icon={<Save size={16} />}
                          style={{
                            background: DESIGN.colors.gradient.primary,
                            boxShadow: DESIGN.shadows.glow(DESIGN.colors.accent),
                          }}
                        >
                          保存修改
                        </Button>
                      </Space>
                    </div>
                  </Form>
                </TabPane>

                {/* 偏好设置 */}
                <TabPane
                  tab={
                    <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <Settings size={16} />
                      偏好设置
                    </span>
                  }
                >
                  <Form
                    form={preferencesForm}
                    layout="vertical"
                    onFinish={handleSavePreferences}
                    initialValues={{
                      language: 'zh-CN',
                      timezone: 'Asia/Shanghai',
                      emailNotify: true,
                      desktopNotify: true,
                    }}
                  >
                    <Title level={5}>通知设置</Title>
                    <Row gutter={24}>
                      <Col xs={24}>
                        <Form.Item label="语言" name="language">
                          <Select>
                            <Select.Option value="zh-CN">简体中文</Select.Option>
                            <Select.Option value="en-US">English</Select.Option>
                          </Select>
                        </Form.Item>
                      </Col>
                      <Col xs={24}>
                        <Form.Item label="时区" name="timezone">
                          <Select>
                            <Select.Option value="Asia/Shanghai">中国标准时间 (UTC+8)</Select.Option>
                            <Select.Option value="UTC">UTC</Select.Option>
                          </Select>
                        </Form.Item>
                      </Col>
                      <Col xs={24}>
                        <Form.Item label="邮件通知" name="emailNotify" valuePropName="checked">
                          <Switch checkedChildren="开启" unCheckedChildren="关闭" />
                        </Form.Item>
                      </Col>
                      <Col xs={24}>
                        <Form.Item label="桌面通知" name="desktopNotify" valuePropName="checked">
                          <Switch checkedChildren="开启" unCheckedChildren="关闭" />
                        </Form.Item>
                      </Col>
                    </Row>

                    <div style={{ marginTop: 24, textAlign: 'right' }}>
                      <Button
                        type="primary"
                        htmlType="submit"
                        icon={<Save size={16} />}
                        style={{
                          background: DESIGN.colors.gradient.primary,
                        }}
                      >
                        保存设置
                      </Button>
                    </div>
                  </Form>
                </TabPane>

                {/* 最近活动 */}
                <TabPane
                  tab={
                    <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <Activity size={16} />
                      最近活动
                    </span>
                  }
                >
                  <div>
                    {mockActivities.map((activity, index) => (
                      <div
                        key={activity.id}
                        style={{
                          display: 'flex',
                          gap: 16,
                          padding: '16px 0',
                          borderBottom: index < mockActivities.length - 1 ? `1px solid ${DESIGN.colors.border}` : 'none',
                        }}
                      >
                        <div
                          style={{
                            width: 40,
                            height: 40,
                            borderRadius: DESIGN.radius.md,
                            background: `${DESIGN.colors.success}15`,
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            color: DESIGN.colors.success,
                          }}
                        >
                          <CheckCircle size={18} />
                        </div>
                        <div style={{ flex: 1 }}>
                          <div style={{ fontWeight: 500, marginBottom: 4 }}>
                            {activity.action}
                          </div>
                          <div style={{ color: DESIGN.colors.textMuted, fontSize: 13 }}>
                            {activity.target}
                          </div>
                        </div>
                        <div style={{ color: DESIGN.colors.textMuted, fontSize: 13 }}>
                          {activity.time}
                        </div>
                      </div>
                    ))}
                  </div>
                </TabPane>

                {/* 安全设置 */}
                <TabPane
                  tab={
                    <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <Lock size={16} />
                      安全设置
                    </span>
                  }
                >
                  <Space direction="vertical" style={{ width: '100%' }} size="large">
                    <Card
                      size="small"
                      style={{
                        borderRadius: DESIGN.radius.md,
                        border: `1px solid ${DESIGN.colors.border}`,
                      }}
                    >
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <div>
                          <div style={{ fontWeight: 600, marginBottom: 4 }}>修改密码</div>
                          <div style={{ color: DESIGN.colors.textMuted, fontSize: 13 }}>
                            定期修改密码可以保护账户安全
                          </div>
                        </div>
                        <Button icon={<Key size={16} />}>修改</Button>
                      </div>
                    </Card>

                    <Card
                      size="small"
                      style={{
                        borderRadius: DESIGN.radius.md,
                        border: `1px solid ${DESIGN.colors.border}`,
                      }}
                    >
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <div>
                          <div style={{ fontWeight: 600, marginBottom: 4 }}>两步验证</div>
                          <div style={{ color: DESIGN.colors.textMuted, fontSize: 13 }}>
                            为账户添加额外的安全保护
                          </div>
                        </div>
                        <Button type="primary" ghost>
                          启用
                        </Button>
                      </div>
                    </Card>

                    <Card
                      size="small"
                      style={{
                        borderRadius: DESIGN.radius.md,
                        border: `1px solid ${DESIGN.colors.border}`,
                      }}
                    >
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <div>
                          <div style={{ fontWeight: 600, marginBottom: 4 }}>登录历史</div>
                          <div style={{ color: DESIGN.colors.textMuted, fontSize: 13 }}>
                            查看账户的登录历史记录
                          </div>
                        </div>
                        <Button>查看</Button>
                      </div>
                    </Card>
                  </Space>
                </TabPane>
              </Tabs>
            </Card>
          </Col>
        </Row>
      </div>
    </div>
  );
}
