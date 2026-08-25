'use client';

import React, { useEffect, useState } from 'react';
import {
  App, Badge, Button, Card, DatePicker, Form, Modal, Select, Space, Table, Tag, TimePicker, Typography,
} from 'antd';
import { PlusOutlined, ReloadOutlined, ClockCircleOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import {
  emailIntakeService, type OnCallSchedule,
} from '@/lib/services/emailIntakeService';

const { Title, Text } = Typography;

interface Group { id: number; name: string; }
interface User { id: number; name: string; email?: string; }

export default function OnCallPage() {
  const { message } = App.useApp();
  const [schedules, setSchedules] = useState<OnCallSchedule[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [currentOnCall, setCurrentOnCall] = useState<Record<number, { userId: number; startAt: string; endAt: string } | null>>({});
  const [loading, setLoading] = useState(false);
  const [scheduleModal, setScheduleModal] = useState(false);
  const [shiftModal, setShiftModal] = useState(false);
  const [selectedSchedule, setSelectedSchedule] = useState<OnCallSchedule | null>(null);
  const [scheduleForm] = Form.useForm();
  const [shiftForm] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const [s, g, u] = await Promise.all([
        emailIntakeService.schedules(),
        fetch('/api/v1/groups?pageSize=100').then(r => r.json()),
        fetch('/api/v1/users?pageSize=200').then(r => r.json()),
      ]);
      setSchedules(s.items);
      setGroups(g.items || g.data?.items || []);
      setUsers(u.items || u.data?.items || []);

      // Load current on-call for each schedule
      const onCallMap: Record<number, { userId: number; startAt: string; endAt: string } | null> = {};
      for (const sch of s.items) {
        try {
          const res = await emailIntakeService.currentOnCall(sch.groupId);
          if (res) onCallMap[sch.groupId] = res;
        } catch { /* no active shift */ }
      }
      setCurrentOnCall(onCallMap);
    } catch { message.error('加载失败'); } finally { setLoading(false); }
  };
  useEffect(() => { load(); }, []);

  const saveSchedule = async () => {
    const v = await scheduleForm.validateFields();
    try {
      await emailIntakeService.createSchedule({ ...v, timezone: v.timezone || 'Asia/Shanghai', status: 'active' });
      message.success('排班已创建');
      setScheduleModal(false); scheduleForm.resetFields();
      load();
    } catch { message.error('保存失败'); }
  };

  const saveShift = async () => {
    const v = await shiftForm.validateFields();
    try {
      const startAt = v.range[0].toISOString();
      const endAt = v.range[1].toISOString();
      await emailIntakeService.createShift({
        scheduleId: selectedSchedule!.id,
        userId: v.userId,
        startAt, endAt,
      });
      message.success('班次已添加');
      setShiftModal(false); shiftForm.resetFields();
      load();
    } catch { message.error('保存失败'); }
  };

  const userName = (id: number) => users.find(u => u.id === id)?.name || users.find(u => u.id === id)?.email || `用户#${id}`;
  const groupName = (id: number) => groups.find(g => g.id === id)?.name || `组#${id}`;

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '排班名称', dataIndex: 'name' },
    { title: '支持组', render: (_: unknown, r: OnCallSchedule) => groupName(r.groupId) },
    { title: '时区', dataIndex: 'timezone' },
    {
      title: '当前值班人', render: (_: unknown, r: OnCallSchedule) => {
        const oc = currentOnCall[r.groupId];
        return oc ? (
          <Space direction="vertical" size={0}>
            <Badge status="processing" text={<Text strong>{userName(oc.userId)}</Text>} />
            <Text type="secondary" style={{ fontSize: 12 }}>
              <ClockCircleOutlined /> {dayjs(oc.startAt).format('MM-DD HH:mm')} ~ {dayjs(oc.endAt).format('MM-DD HH:mm')}
            </Text>
          </Space>
        ) : <Tag color="default">无人值班</Tag>;
      }
    },
    { title: '状态', dataIndex: 'status', render: (s: string) => <Tag color={s === 'active' ? 'green' : 'default'}>{s === 'active' ? '活跃' : s}</Tag> },
    {
      title: '操作', width: 120,
      render: (_: unknown, r: OnCallSchedule) => (
        <Button size="small" icon={<PlusOutlined />} onClick={() => { setSelectedSchedule(r); setShiftModal(true); }}>
          安排班次
        </Button>
      ),
    },
  ];

  return (
    <div>
      <Card>
        <div className="flex items-center justify-between mb-4">
          <Title level={4} className="!mb-0">值班排班管理</Title>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={load}>刷新</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => { scheduleForm.resetFields(); setScheduleModal(true); }}>
              新建排班
            </Button>
          </Space>
        </div>
        <Table rowKey="id" loading={loading} columns={columns} dataSource={schedules} pagination={false} />
      </Card>

      <Modal title="新建排班" open={scheduleModal} onOk={saveSchedule} onCancel={() => setScheduleModal(false)}>
        <Form form={scheduleForm} layout="vertical">
          <Form.Item name="name" label="排班名称" rules={[{ required: true }]}>
            <Select placeholder="选择支持组" options={groups.map(g => ({ value: g.name + ' 排班', label: g.name }))} />
          </Form.Item>
          <Form.Item name="groupId" label="支持组" rules={[{ required: true }]}>
            <Select placeholder="选择支持组" showSearch optionFilterProp="label"
              options={groups.map(g => ({ value: g.id, label: g.name }))}
            />
          </Form.Item>
          <Form.Item name="timezone" label="时区" initialValue="Asia/Shanghai">
            <Select options={[
              { value: 'Asia/Shanghai', label: 'Asia/Shanghai (UTC+8)' },
              { value: 'Asia/Tokyo', label: 'Asia/Tokyo (UTC+9)' },
              { value: 'UTC', label: 'UTC' },
            ]} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`安排班次 - ${selectedSchedule?.name || ''}`}
        open={shiftModal} onOk={saveShift} onCancel={() => setShiftModal(false)}
      >
        <Form form={shiftForm} layout="vertical">
          <Form.Item name="userId" label="值班工程师" rules={[{ required: true }]}>
            <Select placeholder="选择工程师" showSearch optionFilterProp="label"
              options={users.map(u => ({ value: u.id, label: u.name || u.email || `用户#${u.id}` }))}
            />
          </Form.Item>
          <Form.Item name="range" label="值班时段" rules={[{ required: true }]}>
            <DatePicker.RangePicker showTime className="w-full" format="YYYY-MM-DD HH:mm" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
