'use client';

import React, { useEffect, useState } from 'react';
import {
  App,
  Badge,
  Button,
  Card,
  DatePicker,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  TimePicker,
  Typography,
} from 'antd';
import {
  PlusOutlined,
  ReloadOutlined,
  ClockCircleOutlined,
  EditOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import {
  emailIntakeService,
  type OnCallSchedule,
  type OnCallShift,
} from '@/lib/services/emailIntakeService';
import { GroupAPI } from '@/lib/api/group-api';
import { UserApi } from '@/lib/api/user-api';
import { BACKEND_MAX_PAGE_SIZE } from '@/lib/api/pagination';

const { Title, Text } = Typography;

interface Group {
  id: number;
  name: string;
}
interface User {
  id: number;
  name: string;
  email?: string;
}

export default function OnCallPage() {
  const { message } = App.useApp();
  const [schedules, setSchedules] = useState<OnCallSchedule[]>([]);
  const [shifts, setShifts] = useState<OnCallShift[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [currentOnCall, setCurrentOnCall] = useState<
    Record<number, { userId: number; startAt: string; endAt: string } | null>
  >({});
  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染空表格。
  const [listLoading, setListLoading] = useState(true);
  const [scheduleModal, setScheduleModal] = useState(false);
  const [shiftModal, setShiftModal] = useState(false);
  const [editShiftModal, setEditShiftModal] = useState(false);
  const [selectedSchedule, setSelectedSchedule] = useState<OnCallSchedule | null>(null);
  const [editingShift, setEditingShift] = useState<OnCallShift | null>(null);
  const [scheduleForm] = Form.useForm();
  const [shiftForm] = Form.useForm();
  const [editShiftForm] = Form.useForm();

  const load = async () => {
    setListLoading(true);
    try {
      const [s, g, u, sh] = await Promise.all([
        emailIntakeService.schedules(),
        GroupAPI.getGroups({ pageSize: 100 }),
        UserApi.getUsers({ pageSize: BACKEND_MAX_PAGE_SIZE }),
        emailIntakeService.shifts(),
      ]);
      setSchedules(s.items);
      setGroups((g.items || []).map((gr) => ({ id: gr.id, name: gr.name })));
      setUsers((u.items || []).map((usr) => ({ id: usr.id, name: usr.name || usr.username, email: usr.email })));
      setShifts(sh.items || []);

      // Load current on-call for each schedule
      const onCallMap: Record<number, { userId: number; startAt: string; endAt: string } | null> =
        {};
      for (const sch of s.items) {
        try {
          const res = await emailIntakeService.currentOnCall(sch.groupId);
          if (res) onCallMap[sch.groupId] = res;
        } catch {
          /* no active shift */
        }
      }
      setCurrentOnCall(onCallMap);
    } catch {
      message.error('LoadingFailed');
    } finally {
      setListLoading(false);
    }
  };
  useEffect(() => {
    load();
  }, []);

  const saveSchedule = async () => {
    const v = await scheduleForm.validateFields();
    try {
      await emailIntakeService.createSchedule({
        ...v,
        timezone: v.timezone || 'Asia/Shanghai',
        status: 'active',
      });
      message.success('Shift schedule has been created');
      setScheduleModal(false);
      scheduleForm.resetFields();
      load();
    } catch {
      message.error('SaveFailed');
    }
  };

  const saveShift = async () => {
    const v = await shiftForm.validateFields();
    try {
      const startAt = v.range[0].toISOString();
      const endAt = v.range[1].toISOString();
      await emailIntakeService.createShift({
        scheduleId: selectedSchedule!.id,
        userId: v.userId,
        startAt,
        endAt,
      });
      message.success('Shift has been added');
      setShiftModal(false);
      shiftForm.resetFields();
      load();
    } catch {
      message.error('SaveFailed');
    }
  };

  const updateShift = async () => {
    const v = await editShiftForm.validateFields();
    try {
      const startAt = v.range[0].toISOString();
      const endAt = v.range[1].toISOString();
      await emailIntakeService.updateShift(editingShift!.id, {
        scheduleId: editingShift!.scheduleId,
        userId: v.userId,
        startAt,
        endAt,
      });
      message.success('Shift has been updated');
      setEditShiftModal(false);
      editShiftForm.resetFields();
      setEditingShift(null);
      load();
    } catch {
      message.error('Update failed');
    }
  };

  const removeShift = async (id: number) => {
    try {
      await emailIntakeService.deleteShift(id);
      message.success('Shift deleted');
      load();
    } catch {
      message.error('Delete failed');
    }
  };

  const openEditShift = (shift: OnCallShift) => {
    setEditingShift(shift);
    editShiftForm.setFieldsValue({
      userId: shift.userId,
      range: [dayjs(shift.startAt), dayjs(shift.endAt)],
    });
    setEditShiftModal(true);
  };

  const userName = (id: number) =>
    users.find(u => u.id === id)?.name || users.find(u => u.id === id)?.email || `User#${id}`;
  const groupName = (id: number) => groups.find(g => g.id === id)?.name || `Group#${id}`;

  const scheduleColumns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: 'Schedule name', dataIndex: 'name' },
    { title: 'Support Group', render: (_: unknown, r: OnCallSchedule) => groupName(r.groupId) },
    { title: 'Time zone', dataIndex: 'timezone' },
    {
      title: 'Current person on duty',
      render: (_: unknown, r: OnCallSchedule) => {
        const oc = currentOnCall[r.groupId];
        return oc ? (
          <Space orientation='vertical' size={0}>
            <Badge status='processing' text={<Text strong>{userName(oc.userId)}</Text>} />
            <Text type='secondary' style={{ fontSize: 12 }}>
              <ClockCircleOutlined /> {dayjs(oc.startAt).format('MM-DD HH:mm')} ~{' '}
              {dayjs(oc.endAt).format('MM-DD HH:mm')}
            </Text>
          </Space>
        ) : (
          <Tag color='default'>No one on duty</Tag>
        );
      },
    },
    {
      title: 'Status',
      dataIndex: 'status',
      render: (s: string) => (
        <Tag color={s === 'active' ? 'green' : 'default'}>{s === 'active' ? 'Active' : s}</Tag>
      ),
    },
    {
      title: 'Actions',
      width: 120,
      render: (_: unknown, r: OnCallSchedule) => (
        <Button
          size='small'
          icon={<PlusOutlined />}
          onClick={() => {
            setSelectedSchedule(r);
            setShiftModal(true);
          }}
        >

          Schedule a shift
        </Button>
      ),
    },
  ];

  const shiftColumns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    {
      title: 'Scheduling',
      render: (_: unknown, r: OnCallShift) => {
        const sch = schedules.find(s => s.id === r.scheduleId);
        return sch ? sch.name : `Scheduling#${r.scheduleId}`;
      },
    },
    {
      title: 'Engineer on duty',
      render: (_: unknown, r: OnCallShift) => userName(r.userId),
    },
    {
      title: 'Start Time',
      render: (_: unknown, r: OnCallShift) => dayjs(r.startAt).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: 'End Time',
      render: (_: unknown, r: OnCallShift) => dayjs(r.endAt).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: 'Actions',
      width: 120,
      render: (_: unknown, r: OnCallShift) => (
        <Space size='small'>
          <Button size='small' aria-label="Edit shift" icon={<EditOutlined />} onClick={() => openEditShift(r)} />
          <Popconfirm title="Are you sure you want to delete this shift?" onConfirm={() => removeShift(r.id)}>
            <Button size='small' aria-label="Delete shift" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Card className='mb-4'>
        <div className='flex items-center justify-between mb-4'>
          <Title level={4} className='!mb-0'>

            Duty schedule management
          </Title>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={load}>

              Refresh
            </Button>
            <Button
              type='primary'
              icon={<PlusOutlined />}
              onClick={() => {
                scheduleForm.resetFields();
                setScheduleModal(true);
              }}
            >

              Create a new shift
            </Button>
          </Space>
        </div>
        <Table
          rowKey='id'
          loading={listLoading}
          columns={scheduleColumns}
          dataSource={schedules}
          pagination={false}
        />
      </Card>

      <Card>
        <div className='flex items-center justify-between mb-4'>
          <Title level={4} className='!mb-0'>

            Shift List
          </Title>
        </div>
        <Table
          rowKey='id'
          loading={listLoading}
          columns={shiftColumns}
          dataSource={shifts}
          pagination={{ pageSize: 10 }}
        />
      </Card>

      <Modal
        title="Create a new shift"
        open={scheduleModal}
        onOk={saveSchedule}
        onCancel={() => setScheduleModal(false)}
      >
        <Form form={scheduleForm} layout='vertical'>
          <Form.Item name='name' label="Schedule name" rules={[{ required: true }]}>
            <Input placeholder="Please enter the schedule name (for example: NOC 7×24 schedule)" />
          </Form.Item>
          <Form.Item name='groupId' label="Support Group" rules={[{ required: true }]}>
            <Select
              placeholder={groups.length === 0 ? 'No optional support group yet' : 'Select support group'}
              showSearch
              optionFilterProp='label'
              notFoundContent={groups.length === 0 ? 'The current tenant does not have a support group. Please create one in the "Support Group" menu first.' : 'No match'}
              options={groups.map(g => ({ value: g.id, label: g.name }))}
            />
          </Form.Item>
          <Form.Item name='timezone' label="Time zone" initialValue='Asia/Shanghai'>
            <Select
              options={[
                { value: 'Asia/Shanghai', label: 'Asia/Shanghai (UTC+8)' },
                { value: 'Asia/Tokyo', label: 'Asia/Tokyo (UTC+9)' },
                { value: 'UTC', label: 'UTC' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`Schedule a shift - ${selectedSchedule?.name || ''}`}
        open={shiftModal}
        onOk={saveShift}
        onCancel={() => setShiftModal(false)}
      >
        <Form form={shiftForm} layout='vertical'>
          <Form.Item name='userId' label="Engineer on duty" rules={[{ required: true }]}>
            <Select
              placeholder="Select Engineer"
              showSearch
              optionFilterProp='label'
              options={users.map(u => ({
                value: u.id,
                label: u.name || u.email || `User#${u.id}`,
              }))}
            />
          </Form.Item>
          <Form.Item name='range' label="Duty period" rules={[{ required: true }]}>
            <DatePicker.RangePicker showTime className='w-full' format='YYYY-MM-DD HH:mm' />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="Edit shift"
        open={editShiftModal}
        onOk={updateShift}
        onCancel={() => {
          setEditShiftModal(false);
          editShiftForm.resetFields();
          setEditingShift(null);
        }}
      >
        <Form form={editShiftForm} layout='vertical'>
          <Form.Item name='userId' label="Engineer on duty" rules={[{ required: true }]}>
            <Select
              placeholder="Select Engineer"
              showSearch
              optionFilterProp='label'
              options={users.map(u => ({
                value: u.id,
                label: u.name || u.email || `User#${u.id}`,
              }))}
            />
          </Form.Item>
          <Form.Item name='range' label="Duty period" rules={[{ required: true }]}>
            <DatePicker.RangePicker showTime className='w-full' format='YYYY-MM-DD HH:mm' />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
