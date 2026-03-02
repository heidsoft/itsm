'use client';

import React, { useState, useCallback } from 'react';
import { Form, Input, Select, Button, Card, Space, Spin, message } from 'antd';
import { Send, Bot, CheckCircle } from 'lucide-react';
import type { A2UIComponent, ValueDef, OptionItem } from '@/types/a2ui';
import { getValueByPath, setValueByPath } from '@/types/a2ui';

// 本地类型定义（避免循环引用）
interface A2UIDataModel {
  [path: string]: unknown;
}

// 解析值定义（字面量或路径）
function resolveValue(def: ValueDef | undefined, model: A2UIDataModel): unknown {
  if (!def) return undefined;

  if ('literalString' in def) return def.literalString;
  if ('literalNumber' in def) return def.literalNumber;
  if ('literalBoolean' in def) return def.literalBoolean;
  if ('path' in def) return getValueByPath(model, def.path);

  return undefined;
}

// 组件递归渲染器
const A2UIComponentRenderer: React.FC<{
  component: A2UIComponent;
  model: A2UIDataModel;
  onModelChange: (path: string, value: unknown) => void;
  onAction: (name: string, context: Record<string, unknown>) => void;
  allComponents: A2UIComponent[];
}> = ({ component, model, onModelChange, onAction, allComponents }) => {
  const compDef = component.component;
  const compType = Object.keys(compDef)[0];
  const props = (compDef as Record<string, unknown>)[compType];

  // 渲染子组件
  const renderChild = (childId: string) => {
    const child = allComponents.find(c => c.id === childId);
    if (!child) return null;
    return (
      <A2UIComponentRenderer
        key={child.id}
        component={child}
        model={model}
        onModelChange={onModelChange}
        onAction={onAction}
        allComponents={allComponents}
      />
    );
  };

  switch (compType) {
    case 'Text': {
      const textProps = props as { text?: ValueDef; usageHint?: string; color?: ValueDef | string };
      const textValue = resolveValue(textProps?.text, model);
      const hint = textProps?.usageHint;
      const classes: Record<string, string> = {
        h1: 'text-2xl font-bold',
        h2: 'text-xl font-semibold',
        h3: 'text-lg font-medium',
        body: 'text-base',
        caption: 'text-sm text-gray-500',
      };

      return (
        <div
          style={{ color: typeof textProps?.color === 'string' ? textProps.color : undefined }}
          className={hint ? classes[hint] : ''}
        >
          {String(textValue ?? '')}
        </div>
      );
    }

    case 'Column':
    case 'Row': {
      const children = props as { children?: { explicitList?: string[] } };
      const childIds = children?.children?.explicitList || [];
      return (
        <div className={compType === 'Row' ? 'flex gap-2 items-center' : 'flex flex-col gap-2'}>
          {childIds.map(id => renderChild(id))}
        </div>
      );
    }

    case 'Card': {
      const child = props as { child: string };
      return (
        <Card size="small" className="my-2">
          {renderChild(child.child)}
        </Card>
      );
    }

    case 'TextField': {
      const fieldProps = props as {
        label?: { literalString: string };
        text?: ValueDef;
        placeholder?: { literalString?: string };
        enabled?: ValueDef;
        error?: ValueDef;
      };

      // 提取路径
      const getPath = (def?: ValueDef) => {
        if (!def) return undefined;
        return (def as { path?: string }).path;
      };

      const path = getPath(fieldProps?.text)?.replace('/ticket/', '') || '';
      const isEnabled = fieldProps?.enabled
        ? resolveValue(fieldProps.enabled, model) !== false
        : true;
      const errorMsg = fieldProps?.error
        ? (resolveValue(fieldProps.error, model) as string)
        : undefined;

      return (
        <Form.Item
          name={path}
          label={fieldProps?.label?.literalString}
          validateStatus={errorMsg ? 'error' : ''}
          help={errorMsg}
        >
          <Input
            placeholder={fieldProps?.placeholder?.literalString}
            disabled={!isEnabled}
            onChange={e => {
              const dataPath = getPath(fieldProps?.text);
              if (dataPath) onModelChange(dataPath, e.target.value);
            }}
          />
        </Form.Item>
      );
    }

    case 'TextAreaField': {
      const fieldProps = props as {
        label?: { literalString: string };
        text?: ValueDef;
        rows?: number;
        enabled?: ValueDef;
      };

      // 提取路径
      const getPath = (def?: ValueDef) => {
        if (!def) return undefined;
        return (def as { path?: string }).path;
      };

      const path = getPath(fieldProps?.text)?.replace('/ticket/', '') || '';
      const isEnabled = fieldProps?.enabled
        ? resolveValue(fieldProps.enabled, model) !== false
        : true;

      return (
        <Form.Item name={path} label={fieldProps?.label?.literalString}>
          <Input.TextArea
            rows={fieldProps?.rows || 3}
            disabled={!isEnabled}
            onChange={e => {
              const dataPath = getPath(fieldProps?.text);
              if (dataPath) onModelChange(dataPath, e.target.value);
            }}
          />
        </Form.Item>
      );
    }

    case 'PickerSelect': {
      const selectProps = props as {
        label?: { literalString: string };
        selection?: ValueDef;
        options?: { explicitList?: OptionItem[] };
        enabled?: ValueDef;
      };

      // 提取路径
      const getPath = (def?: ValueDef) => {
        if (!def) return undefined;
        return (def as { path?: string }).path;
      };

      const path = getPath(selectProps?.selection)?.replace('/ticket/', '') || '';
      const options = selectProps?.options?.explicitList || [];
      const isEnabled = selectProps?.enabled
        ? resolveValue(selectProps.enabled, model) !== false
        : true;

      return (
        <Form.Item name={path} label={selectProps?.label?.literalString}>
          <Select
            disabled={!isEnabled}
            onChange={value => {
              const dataPath = getPath(selectProps?.selection);
              if (dataPath) onModelChange(dataPath, value);
            }}
          >
            {options.map(opt => (
              <Select.Option key={opt.id} value={opt.id}>
                {opt.text}
              </Select.Option>
            ))}
          </Select>
        </Form.Item>
      );
    }

    case 'Button': {
      const btnProps = props as {
        child?: string;
        action?: { name: string; context: { key: string; value: ValueDef }[] };
        enabled?: ValueDef;
      };

      const isEnabled = btnProps?.enabled ? resolveValue(btnProps.enabled, model) !== false : true;

      // 获取按钮文字（从子组件）
      let buttonText = '提交';
      if (btnProps?.child) {
        const childComp = allComponents.find(c => c.id === btnProps.child);
        if (childComp) {
          const childDef = childComp.component;
          const childType = Object.keys(childDef)[0];
          if (childType === 'Text') {
            buttonText = (resolveValue((childDef as any).Text?.text, model) as string) || '提交';
          }
        }
      }

      return (
        <Form.Item>
          <Button
            type="primary"
            disabled={!isEnabled}
            onClick={() => {
              if (btnProps?.action) {
                const context: Record<string, unknown> = {};
                btnProps.action.context.forEach(ctx => {
                  if ('path' in ctx.value) {
                    const path = ctx.value.path;
                    context[ctx.key] = getValueByPath(model, path);
                  }
                });
                onAction(btnProps.action.name, context);
              }
            }}
          >
            {buttonText}
          </Button>
        </Form.Item>
      );
    }

    default:
      return <div className="text-gray-400 text-xs">Unknown: {compType}</div>;
  }
};

// ==================== 主渲染器 ====================

export function A2UIFormRenderer() {
  const [form] = Form.useForm();
  const [components, setComponents] = useState<A2UIComponent[]>([]);
  const [model, setModel] = useState<A2UIDataModel>({
    ticket: { title: '', type: 'hardware', priority: 'medium', description: '' },
    ui: { canSubmit: false, status: '', statusColor: 'gray', errors: {} },
  });
  const [userIntent, setUserIntent] = useState('');
  const [loading, setLoading] = useState(false);
  const [surfaceId, setSurfaceId] = useState<string | null>(null);
  const [submitted, setSubmitted] = useState(false);

  // 渲染 ID 列表
  const getRenderableIds = (): string[] => {
    const root = components.find(c => c.id === 'root');
    if (!root) return [];

    const compDef = root.component;
    const compType = Object.keys(compDef)[0];
    const children = (compDef as Record<string, unknown>)[compType] as {
      children?: { explicitList?: string[] };
    };

    return children?.children?.explicitList || [];
  };

  // 处理数据模型变化（双向绑定）
  const handleModelChange = useCallback((path: string, value: unknown) => {
    setModel(prev => {
      const newModel = JSON.parse(JSON.stringify(prev));
      setValueByPath(newModel, path, value);
      return newModel;
    });
  }, []);

  // 获取认证token
  const getAuthHeaders = (): HeadersInit => {
    const token = typeof window !== 'undefined' ? localStorage.getItem('access_token') : null;
    return {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    };
  };

  // 检查是否已登录
  const checkAuth = (): boolean => {
    const token = typeof window !== 'undefined' ? localStorage.getItem('access_token') : null;
    if (!token) {
      message.error('请先登录后再使用此功能');
      return false;
    }
    return true;
  };

  // 发送意图到 AI
  const sendToAI = async () => {
    if (!userIntent.trim()) {
      message.warning('请描述您的需求');
      return;
    }
    if (!checkAuth()) {
      return;
    }
    setLoading(true);
    try {
      const res = await fetch('/api/v1/a2ui/ticket/form', {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({ intent: userIntent, surfaceId }),
      });

      // 检查HTTP状态码
      if (res.status === 401) {
        message.error('登录已过期，请重新登录');
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
        return;
      }
      if (res.status === 403) {
        message.error('权限不足，请联系管理员');
        return;
      }

      const data = await res.json();

      if (data.code !== 0) {
        message.error(data.message || '生成表单失败');
        return;
      }

      // 解析消息
      (data.messages || []).forEach((msg: string) => {
        try {
          const parsed = JSON.parse(msg);

          if ('surfaceUpdate' in parsed) {
            setSurfaceId(parsed.surfaceUpdate.surfaceId);
            const newComps = parsed.surfaceUpdate.components;
            setComponents(prev => {
              const existingIds = new Set(prev.map((c: A2UIComponent) => c.id));
              return [...prev, ...newComps.filter((c: A2UIComponent) => !existingIds.has(c.id))];
            });
          }

          if ('dataModelUpdate' in parsed) {
            const updates = parsed.dataModelUpdate.contents;
            setModel(prev => {
              const newModel = JSON.parse(JSON.stringify(prev));
              updates.forEach(
                (u: {
                  key: string;
                  valueString?: string;
                  valueNumber?: number;
                  valueBoolean?: boolean;
                }) => {
                  const value = u.valueString ?? u.valueNumber ?? u.valueBoolean;
                  setValueByPath(newModel, `${parsed.dataModelUpdate.path}/${u.key}`, value);
                }
              );
              return newModel;
            });

            // 同步到 Ant Design Form
            updates.forEach((u: { key: string; valueString?: string }) => {
              if (u.valueString !== undefined) {
                form.setFieldValue(u.key, u.valueString);
              }
            });
          }

          if ('deleteSurface' in parsed) {
            setComponents([]);
            setSurfaceId(null);
          }
        } catch (e) {
          console.error('Parse message error:', e);
        }
      });

      setUserIntent('');
    } catch (e) {
      console.error(e);
      message.error('请求失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  // 处理用户操作
  const handleAction = async (actionName: string, context: Record<string, unknown>) => {
    if (actionName === 'submit' && surfaceId) {
      if (!checkAuth()) {
        return;
      }
      setLoading(true);
      try {
        const res = await fetch('/api/v1/a2ui/ticket/action', {
          method: 'POST',
          headers: getAuthHeaders(),
          body: JSON.stringify({ action: actionName, surfaceId, context }),
        });

        // 检查HTTP状态码
        if (res.status === 401) {
          message.error('登录已过期，请重新登录');
          localStorage.removeItem('access_token');
          localStorage.removeItem('refresh_token');
          return;
        }
        if (res.status === 403) {
          message.error('权限不足，请联系管理员');
          return;
        }

        const data = await res.json();

        if (data.code !== 0) {
          message.error(data.message || '提交失败');
          return;
        }

        // 处理响应消息
        (data.messages || []).forEach((msg: string) => {
          try {
            const parsed = JSON.parse(msg);
            if ('dataModelUpdate' in parsed) {
              const updates = parsed.dataModelUpdate.contents;
              const status = updates.find((u: { key: string }) => u.key === 'status')?.valueString;
              if (status === 'created') {
                setSubmitted(true);
                message.success('工单创建成功！');
              }
            }
            if ('deleteSurface' in parsed) {
              setComponents([]);
              setSurfaceId(null);
            }
          } catch (e) {
            console.error('Parse message error:', e);
          }
        });
      } catch (e) {
        message.error('提交失败');
      } finally {
        setLoading(false);
      }
    }
  };

  const renderableIds = getRenderableIds();

  // 成功提交后的显示
  if (submitted) {
    return (
      <Card className="max-w-2xl mx-auto">
        <div className="text-center py-8">
          <CheckCircle className="w-16 h-16 text-green-500 mx-auto mb-4" />
          <h2 className="text-xl font-semibold mb-2">工单创建成功</h2>
          <p className="text-gray-500 mb-4">您的工单已成功提交</p>
          <Button
            type="primary"
            onClick={() => {
              setSubmitted(false);
              setComponents([]);
              setModel({
                ticket: { title: '', type: 'hardware', priority: 'medium', description: '' },
                ui: { canSubmit: false, status: '', statusColor: 'gray', errors: {} },
              });
            }}
          >
            创建新工单
          </Button>
        </div>
      </Card>
    );
  }

  return (
    <Card className="max-w-2xl mx-auto">
      {/* AI 输入区 */}
      <div className="mb-4 p-3 bg-gradient-to-r from-blue-50 to-purple-50 rounded-lg">
        <div className="flex items-center gap-2 mb-2">
          <Bot className="w-4 h-4 text-purple-500" />
          <span className="text-sm font-medium">描述需求，AI 自动生成表单</span>
        </div>
        <Space.Compact style={{ width: '100%' }}>
          <Input
            placeholder="例如：帮我申请一台 ThinkPad 笔记本，用于 Java 开发..."
            value={userIntent}
            onChange={e => setUserIntent(e.target.value)}
            onPressEnter={sendToAI}
          />
          <Button icon={<Send />} onClick={sendToAI} loading={loading}>
            发送
          </Button>
        </Space.Compact>
      </div>

      {/* A2UI 表单渲染 */}
      {components.length > 0 ? (
        <Spin spinning={loading}>
          <Form form={form} layout="vertical">
            {renderableIds.map(id => {
              const comp = components.find(c => c.id === id);
              if (!comp) return null;
              return (
                <A2UIComponentRenderer
                  key={comp.id}
                  component={comp}
                  model={model}
                  onModelChange={handleModelChange}
                  onAction={handleAction}
                  allComponents={components}
                />
              );
            })}
          </Form>
        </Spin>
      ) : (
        <div className="text-center text-gray-400 py-8">
          <Bot className="w-12 h-12 mx-auto mb-2 opacity-50" />
          <p>描述您的需求，AI 将生成工单表单</p>
          <p className="text-xs mt-2">例如：帮我申请一台 ThinkPad 笔记本</p>
        </div>
      )}

      {/* 调试信息 */}
      {process.env.NODE_ENV === 'development' && components.length > 0 && (
        <details className="mt-4 text-xs text-gray-400">
          <summary>A2UI 调试信息</summary>
          <pre className="mt-2 p-2 bg-gray-100 rounded overflow-auto max-h-40">
            {JSON.stringify(components, null, 2)}
          </pre>
        </details>
      )}
    </Card>
  );
}

export default A2UIFormRenderer;
