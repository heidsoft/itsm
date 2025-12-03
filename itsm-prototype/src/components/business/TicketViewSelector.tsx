/**
 * 工单视图选择器组件
 * 提供视图切换、保存、管理功能
 */

'use client';

import React, { useState, useEffect } from 'react';
import {
  Select,
  Button,
  Dropdown,
  Space,
  Modal,
  Form,
  Input,
  Switch,
  App,
  Popconfirm,
  Divider,
  Tag,
} from 'antd';
import {
  SaveOutlined,
  EditOutlined,
  DeleteOutlined,
  ShareAltOutlined,
  EyeOutlined,
  PlusOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import { TicketViewApi, TicketView, CreateTicketViewRequest } from '@/lib/api/ticket-view-api';
import { TicketFilterState } from './TicketFilters';

const { Option } = Select;
const { TextArea } = Input;

interface TicketViewSelectorProps {
  currentViewId?: number;
  filters: TicketFilterState;
  onViewChange: (view: TicketView | null) => void;
  onFiltersChange: (filters: Partial<TicketFilterState>) => void;
}

export const TicketViewSelector: React.FC<TicketViewSelectorProps> = ({
  currentViewId,
  filters,
  onViewChange,
  onFiltersChange,
}) => {
  const { message } = App.useApp();
  const [views, setViews] = useState<TicketView[]>([]);
  const [loading, setLoading] = useState(false);
  const [saveModalVisible, setSaveModalVisible] = useState(false);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editingView, setEditingView] = useState<TicketView | null>(null);
  const [form] = Form.useForm();

  // 加载视图列表
  const loadViews = async () => {
    setLoading(true);
    try {
      const response = await TicketViewApi.listViews();
      setViews(response?.views || []);
    } catch (error) {
      console.error('Failed to load views:', error);
      message.error('加载视图列表失败');
      setViews([]); // 确保 views 始终是数组
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadViews();
  }, []);

  // 应用视图
  const handleViewSelect = async (viewId: number) => {
    if (viewId === currentViewId) return;

    try {
      const view = await TicketViewApi.getView(viewId);
      if (!view) {
        message.error('视图不存在');
        return;
      }
      // 将视图的筛选条件应用到当前筛选器
      if (view?.filters) {
        onFiltersChange(view.filters as Partial<TicketFilterState>);
      }
      onViewChange(view);
      message.success(`已切换到视图: ${view?.name || '未知视图'}`);
    } catch (error) {
      console.error('Failed to load view:', error);
      message.error('加载视图失败');
    }
  };

  // 保存当前筛选条件为视图
  const handleSaveView = async (values: {
    name: string;
    description?: string;
    is_shared: boolean;
  }) => {
    try {
      const viewData: CreateTicketViewRequest = {
        name: values.name,
        description: values.description,
        filters: filters as any,
        columns: [], // 默认列，后续可以从表格配置中获取
        sort_config: {
          field: filters.sortBy?.split('_')[0] || 'createdAt',
          order: filters.sortBy?.split('_')[1] || 'desc',
        },
        is_shared: values.is_shared,
      };

      await TicketViewApi.createView(viewData);
      message.success('视图保存成功');
      setSaveModalVisible(false);
      form.resetFields();
      loadViews();
    } catch (error) {
      console.error('Failed to save view:', error);
      message.error('保存视图失败');
    }
  };

  // 更新视图
  const handleUpdateView = async (values: {
    name: string;
    description?: string;
    is_shared: boolean;
  }) => {
    if (!editingView) return;

    try {
      await TicketViewApi.updateView(editingView.id, {
        name: values.name,
        description: values.description,
        filters: filters as any,
        is_shared: values.is_shared,
      });
      message.success('视图更新成功');
      setEditModalVisible(false);
      setEditingView(null);
      form.resetFields();
      loadViews();
    } catch (error) {
      console.error('Failed to update view:', error);
      message.error('更新视图失败');
    }
  };

  // 删除视图
  const handleDeleteView = async (viewId: number) => {
    try {
      await TicketViewApi.deleteView(viewId);
      message.success('视图删除成功');
      if (currentViewId === viewId) {
        onViewChange(null);
      }
      loadViews();
    } catch (error) {
      console.error('Failed to delete view:', error);
      message.error('删除视图失败');
    }
  };

  // 打开编辑模态框
  const handleEditView = (view: TicketView) => {
    setEditingView(view);
    form.setFieldsValue({
      name: view.name,
      description: view.description,
      is_shared: view.is_shared,
    });
    setEditModalVisible(true);
  };

  // 视图操作菜单
  const getViewMenuItems = (view: TicketView): MenuProps['items'] => [
    {
      key: 'edit',
      label: '编辑',
      icon: <EditOutlined />,
      onClick: () => handleEditView(view),
    },
    {
      key: 'delete',
      label: '删除',
      icon: <DeleteOutlined />,
      danger: true,
      onClick: () => {
        Modal.confirm({
          title: '确认删除',
          content: `确定要删除视图"${view.name}"吗？`,
          onOk: () => handleDeleteView(view.id),
        });
      },
    },
    {
      key: 'share',
      label: view.is_shared ? '取消共享' : '共享',
      icon: <ShareAltOutlined />,
      onClick: async () => {
        try {
          await TicketViewApi.updateView(view.id, { is_shared: !view.is_shared });
          message.success(view.is_shared ? '已取消共享' : '已共享');
          loadViews();
        } catch (error) {
          message.error('操作失败');
        }
      },
    },
  ];

  const currentView = views.find(v => v.id === currentViewId);

  return (
    <>
      <Space>
        <Select
          style={{ width: 200 }}
          placeholder='选择视图'
          value={currentViewId}
          onChange={handleViewSelect}
          loading={loading}
          popupRender={menu => (
            <div>
              {menu}
              <Divider style={{ margin: '8px 0' }} />
              <Button
                type='text'
                icon={<PlusOutlined />}
                block
                onClick={() => {
                  form.resetFields();
                  setSaveModalVisible(true);
                }}
              >
                保存当前筛选为视图
              </Button>
            </div>
          )}
        >
          {views.map(view => (
            <Option key={view.id} value={view.id}>
              <Space>
                <span>{view.name}</span>
                {view.is_shared && (
                  <Tag color='blue' size='small'>
                    共享
                  </Tag>
                )}
                {view.id === currentViewId && (
                  <Tag color='green' size='small'>
                    当前
                  </Tag>
                )}
              </Space>
            </Option>
          ))}
        </Select>

        {currentView && (
          <Dropdown menu={{ items: getViewMenuItems(currentView) }} trigger={['click']}>
            <Button icon={<SettingOutlined />}>管理</Button>
          </Dropdown>
        )}
      </Space>

      {/* 保存视图模态框 */}
      <Modal
        title='保存视图'
        open={saveModalVisible}
        onOk={() => form.submit()}
        onCancel={() => {
          setSaveModalVisible(false);
          form.resetFields();
        }}
        okText='保存'
        cancelText='取消'
      >
        <Form
          form={form}
          layout='vertical'
          onFinish={handleSaveView}
          initialValues={{ is_shared: false }}
        >
          <Form.Item
            name='name'
            label='视图名称'
            rules={[{ required: true, message: '请输入视图名称' }]}
          >
            <Input placeholder='例如：我的待处理工单' />
          </Form.Item>
          <Form.Item name='description' label='描述'>
            <TextArea rows={3} placeholder='视图描述（可选）' />
          </Form.Item>
          <Form.Item name='is_shared' valuePropName='checked'>
            <Switch checkedChildren='共享' unCheckedChildren='私有' />
            <div style={{ marginTop: 8, color: '#999', fontSize: 12 }}>
              共享后，团队成员可以看到并使用此视图
            </div>
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑视图模态框 */}
      <Modal
        title='编辑视图'
        open={editModalVisible}
        onOk={() => form.submit()}
        onCancel={() => {
          setEditModalVisible(false);
          setEditingView(null);
          form.resetFields();
        }}
        okText='保存'
        cancelText='取消'
      >
        <Form form={form} layout='vertical' onFinish={handleUpdateView}>
          <Form.Item
            name='name'
            label='视图名称'
            rules={[{ required: true, message: '请输入视图名称' }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name='description' label='描述'>
            <TextArea rows={3} />
          </Form.Item>
          <Form.Item name='is_shared' valuePropName='checked'>
            <Switch checkedChildren='共享' unCheckedChildren='私有' />
            <div style={{ marginTop: 8, color: '#999', fontSize: 12 }}>
              共享后，团队成员可以看到并使用此视图
            </div>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};
