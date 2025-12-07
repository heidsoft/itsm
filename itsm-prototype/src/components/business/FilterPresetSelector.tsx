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
  App,
  Popconfirm,
  Tag,
  Checkbox,
} from 'antd';
import {
  SaveOutlined,
  EditOutlined,
  DeleteOutlined,
  StarOutlined,
  StarFilled,
  PlusOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import { TicketFilterState } from './TicketFilters';

const { Option } = Select;

// 筛选预设接口
export interface FilterPreset {
  id: number;
  name: string;
  filters: Partial<TicketFilterState>;
  is_favorite?: boolean;
  created_at?: string;
}

// 本地存储的预设（实际应该从API获取）
const STORAGE_KEY = 'ticket_filter_presets';

// 从本地存储加载预设
const loadPresetsFromStorage = (): FilterPreset[] => {
  if (typeof window === 'undefined') return [];
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    return stored ? JSON.parse(stored) : [];
  } catch {
    return [];
  }
};

// 保存预设到本地存储
const savePresetsToStorage = (presets: FilterPreset[]) => {
  if (typeof window === 'undefined') return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(presets));
  } catch {
    console.error('Failed to save presets to storage');
  }
};

interface FilterPresetSelectorProps {
  filters: TicketFilterState;
  onFiltersChange: (filters: Partial<TicketFilterState>) => void;
}

export const FilterPresetSelector: React.FC<FilterPresetSelectorProps> = ({
  filters,
  onFiltersChange,
}) => {
  const { message } = App.useApp();
  const [presets, setPresets] = useState<FilterPreset[]>([]);
  const [saveModalVisible, setSaveModalVisible] = useState(false);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editingPreset, setEditingPreset] = useState<FilterPreset | null>(null);
  const [form] = Form.useForm();

  // 加载预设列表
  useEffect(() => {
    const loaded = loadPresetsFromStorage();
    setPresets(loaded);
  }, []);

  // 应用预设
  const handlePresetSelect = (presetId: number) => {
    const preset = presets.find(p => p.id === presetId);
    if (preset) {
      onFiltersChange(preset.filters);
      message.success(`已应用预设: ${preset.name}`);
    }
  };

  // 保存当前筛选条件为预设
  const handleSavePreset = async () => {
    try {
      const values = await form.validateFields();
      const newPreset: FilterPreset = {
        id: Date.now(),
        name: values.name,
        filters: { ...filters },
        is_favorite: values.is_favorite || false,
        created_at: new Date().toISOString(),
      };

      const updated = [...presets, newPreset];
      setPresets(updated);
      savePresetsToStorage(updated);
      setSaveModalVisible(false);
      form.resetFields();
      message.success('预设已保存');
    } catch (error) {
      console.error('Failed to save preset:', error);
    }
  };

  // 编辑预设
  const handleEditPreset = async () => {
    if (!editingPreset) return;

    try {
      const values = await form.validateFields();
      const updated = presets.map(p =>
        p.id === editingPreset.id
          ? { ...p, name: values.name, is_favorite: values.is_favorite || false }
          : p
      );
      setPresets(updated);
      savePresetsToStorage(updated);
      setEditModalVisible(false);
      setEditingPreset(null);
      form.resetFields();
      message.success('预设已更新');
    } catch (error) {
      console.error('Failed to update preset:', error);
    }
  };

  // 删除预设
  const handleDeletePreset = (presetId: number) => {
    const updated = presets.filter(p => p.id !== presetId);
    setPresets(updated);
    savePresetsToStorage(updated);
    message.success('预设已删除');
  };

  // 切换收藏状态
  const handleToggleFavorite = (presetId: number) => {
    const updated = presets.map(p =>
      p.id === presetId ? { ...p, is_favorite: !p.is_favorite } : p
    );
    setPresets(updated);
    savePresetsToStorage(updated);
    message.success('收藏状态已更新');
  };

  // 获取预设菜单项
  const getPresetMenuItems = (preset: FilterPreset): MenuProps['items'] => [
    {
      key: 'favorite',
      label: preset.is_favorite ? '取消收藏' : '收藏',
      icon: preset.is_favorite ? <StarFilled /> : <StarOutlined />,
      onClick: () => handleToggleFavorite(preset.id),
    },
    {
      key: 'edit',
      label: '编辑',
      icon: <EditOutlined />,
      onClick: () => {
        form.setFieldsValue({
          name: preset.name,
          is_favorite: preset.is_favorite || false,
        });
        setEditingPreset(preset);
        setEditModalVisible(true);
      },
    },
    {
      type: 'divider',
    },
    {
      key: 'delete',
      label: '删除',
      icon: <DeleteOutlined />,
      danger: true,
      onClick: () => {
        Modal.confirm({
          title: '确认删除',
          content: `确定要删除预设"${preset.name}"吗？`,
          onOk: () => handleDeletePreset(preset.id),
        });
      },
    },
  ];

  // 收藏的预设
  const favoritePresets = presets.filter(p => p.is_favorite);
  // 其他预设
  const otherPresets = presets.filter(p => !p.is_favorite);

  return (
    <>
      <Space>
        <Select
          style={{ width: 180 }}
          placeholder='筛选预设'
          onSelect={handlePresetSelect}
          popupRender={menu => (
            <div>
              {favoritePresets.length > 0 && (
                <>
                  <div style={{ padding: '8px 12px', fontSize: '12px', color: '#999' }}>
                    收藏的预设
                  </div>
                  {favoritePresets.map(preset => (
                    <div
                      key={preset.id}
                      style={{
                        padding: '8px 12px',
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        cursor: 'pointer',
                      }}
                      onClick={() => handlePresetSelect(preset.id)}
                    >
                      <Space>
                        <StarFilled style={{ color: '#faad14' }} />
                        <span>{preset.name}</span>
                      </Space>
                      <Dropdown menu={{ items: getPresetMenuItems(preset) }} trigger={['click']}>
                        <Button
                          type='text'
                          size='small'
                          onClick={e => e.stopPropagation()}
                          style={{ padding: '0 4px' }}
                        >
                          ...
                        </Button>
                      </Dropdown>
                    </div>
                  ))}
                  {otherPresets.length > 0 && (
                    <div style={{ borderTop: '1px solid #f0f0f0', margin: '4px 0' }} />
                  )}
                </>
              )}
              {otherPresets.length > 0 && (
                <>
                  {favoritePresets.length > 0 && (
                    <div style={{ padding: '8px 12px', fontSize: '12px', color: '#999' }}>
                      其他预设
                    </div>
                  )}
                  {otherPresets.map(preset => (
                    <div
                      key={preset.id}
                      style={{
                        padding: '8px 12px',
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        cursor: 'pointer',
                      }}
                      onClick={() => handlePresetSelect(preset.id)}
                    >
                      <span>{preset.name}</span>
                      <Dropdown menu={{ items: getPresetMenuItems(preset) }} trigger={['click']}>
                        <Button
                          type='text'
                          size='small'
                          onClick={e => e.stopPropagation()}
                          style={{ padding: '0 4px' }}
                        >
                          ...
                        </Button>
                      </Dropdown>
                    </div>
                  ))}
                </>
              )}
              <div style={{ borderTop: '1px solid #f0f0f0', margin: '4px 0' }} />
              <Button
                type='text'
                icon={<PlusOutlined />}
                block
                onClick={() => {
                  form.resetFields();
                  setSaveModalVisible(true);
                }}
                style={{ textAlign: 'left', padding: '8px 12px' }}
              >
                保存当前筛选为预设
              </Button>
            </div>
          )}
        >
          {presets.length === 0 && (
            <Option value='' disabled>
              暂无预设
            </Option>
          )}
        </Select>
      </Space>

      {/* 保存预设模态框 */}
      <Modal
        title='保存筛选预设'
        open={saveModalVisible}
        onOk={handleSavePreset}
        onCancel={() => {
          setSaveModalVisible(false);
          form.resetFields();
        }}
      >
        <Form form={form} layout='vertical'>
          <Form.Item
            name='name'
            label='预设名称'
            rules={[{ required: true, message: '请输入预设名称' }]}
          >
            <Input placeholder='例如：我的待处理工单' />
          </Form.Item>
          <Form.Item name='is_favorite' valuePropName='checked'>
            <Input type='checkbox' /> 收藏此预设
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑预设模态框 */}
      <Modal
        title='编辑筛选预设'
        open={editModalVisible}
        onOk={handleEditPreset}
        onCancel={() => {
          setEditModalVisible(false);
          setEditingPreset(null);
          form.resetFields();
        }}
      >
        <Form form={form} layout='vertical'>
          <Form.Item
            name='name'
            label='预设名称'
            rules={[{ required: true, message: '请输入预设名称' }]}
          >
            <Input placeholder='例如：我的待处理工单' />
          </Form.Item>
          <Form.Item name='is_favorite' valuePropName='checked'>
            <Input type='checkbox' /> 收藏此预设
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

