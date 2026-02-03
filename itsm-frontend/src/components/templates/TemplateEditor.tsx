/**
 * 模板编辑器组件
 * 提供完整的模板创建和编辑功能，包括基础信息、字段设计、权限配置等
 */

'use client';

import React, { useState, useCallback, useEffect } from 'react';
import {
  Card,
  Form,
  Input,
  Select,
  Button,
  Space,
  Steps,
  Tabs,
  Row,
  Col,
  Switch,
  Upload,
  Empty,
  message,
  Alert,
  Tag,
  Divider,
  Collapse,
  Radio,
  InputNumber,
  Tooltip,
  Modal,
  Badge,
  ColorPicker,
  type UploadFile,
} from 'antd';
import {
  SaveOutlined,
  EyeOutlined,
  CloseOutlined,
  PlusOutlined,
  UploadOutlined,
  InfoCircleOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  RocketOutlined,
  HistoryOutlined,
  SettingOutlined,
  LockOutlined,
  ThunderboltOutlined,
  FileTextOutlined,
} from '@ant-design/icons';
import { FieldDesigner } from './FieldDesigner';
import type {
  TicketTemplate,
  CreateTemplateRequest,
  UpdateTemplateRequest,
  TemplateField,
  TemplatePermission,
  TemplateAutomation,
  TemplateDefaults,
  TemplateVisibility,
} from '@/types/template';
import {
  useCreateTemplateMutation,
  useUpdateTemplateMutation,
  usePublishTemplateMutation,
  useTemplateCategoriesQuery,
} from '@/lib/hooks/useTemplateQuery';

const { TextArea } = Input;
const { Option } = Select;
const { Step } = Steps;
const { Panel } = Collapse;

// ==================== 步骤定义 ====================

const STEPS = [
  {
    key: 'basic',
    title: '基础信息',
    icon: <FileTextOutlined />,
    description: '模板名称、描述、分类等',
  },
  {
    key: 'fields',
    title: '字段设计',
    icon: <SettingOutlined />,
    description: '配置工单字段',
  },
  {
    key: 'defaults',
    title: '默认配置',
    icon: <InfoCircleOutlined />,
    description: '设置默认值',
  },
  {
    key: 'permissions',
    title: '权限配置',
    icon: <LockOutlined />,
    description: '控制可见性',
  },
  {
    key: 'automation',
    title: '自动化',
    icon: <ThunderboltOutlined />,
    description: '配置自动化规则',
  },
];

// ==================== 接口定义 ====================

export interface TemplateEditorProps {
  template?: TicketTemplate;
  mode?: 'create' | 'edit' | 'duplicate';
  onSave?: (template: TicketTemplate) => void;
  onCancel?: () => void;
}

// ==================== 主组件 ====================

export const TemplateEditor: React.FC<TemplateEditorProps> = ({
  template,
  mode = 'create',
  onSave,
  onCancel,
}) => {
  const [form] = Form.useForm();
  const [currentStep, setCurrentStep] = useState(0);
  const [fields, setFields] = useState<TemplateField[]>(template?.fields || []);
  const [isDraft, setIsDraft] = useState(template?.isDraft ?? true);
  const [coverImage, setCoverImage] = useState<UploadFile[]>([]);
  const [activeTab, setActiveTab] = useState('editor');

  // Mutations
  const createMutation = useCreateTemplateMutation();
  const updateMutation = useUpdateTemplateMutation();
  const publishMutation = usePublishTemplateMutation();

  // Queries
  const { data: categoriesData } = useTemplateCategoriesQuery();
  const categories = categoriesData || [];

  // 初始化表单
  useEffect(() => {
    if (template) {
      form.setFieldsValue({
        name: template.name,
        description: template.description,
        categoryId: template.categoryId,
        color: template.color,
        tags: template.tags,
        defaults: template.defaults,
        permission: template.permission,
        automation: template.automation,
      });
      setFields(template.fields);
      setIsDraft(template.isDraft);
    }
  }, [template, form]);

  // 处理字段变更
  const handleFieldsChange = useCallback((newFields: TemplateField[]) => {
    setFields(newFields);
  }, []);

  // 验证当前步骤
  const validateCurrentStep = async (): Promise<boolean> => {
    try {
      const step = STEPS[currentStep];
      
      if (step.key === 'basic') {
        await form.validateFields(['name', 'description', 'categoryId']);
      } else if (step.key === 'fields') {
        if (fields.length === 0) {
          message.error('请至少添加一个字段');
          return false;
        }
      }
      
      return true;
    } catch (error) {
      return false;
    }
  };

  // 下一步
  const handleNext = async () => {
    const isValid = await validateCurrentStep();
    if (isValid) {
      setCurrentStep(currentStep + 1);
    }
  };

  // 上一步
  const handlePrevious = () => {
    setCurrentStep(currentStep - 1);
  };

  // 保存草稿
  const handleSaveDraft = async () => {
    try {
      const values = await form.validateFields();
      const data: CreateTemplateRequest | UpdateTemplateRequest = {
        ...values,
        fields,
        isDraft: true,
      };

      if (mode === 'edit' && template) {
        await updateMutation.mutateAsync({ id: template.id, data });
      } else {
        await createMutation.mutateAsync(data as CreateTemplateRequest);
      }

      message.success('草稿已保存');
    } catch (error) {
      console.error('保存草稿失败:', error);
    }
  };

  // 保存并发布
  const handlePublish = async () => {
    try {
      // 验证所有步骤
      await form.validateFields();
      
      if (fields.length === 0) {
        message.error('请至少添加一个字段');
        return;
      }

      const values = await form.validateFields();
      const data: CreateTemplateRequest = {
        ...values,
        fields,
      };

      let templateId: string;

      if (mode === 'edit' && template) {
        const result = await updateMutation.mutateAsync({
          id: template.id,
          data: { ...data, isDraft: false },
        });
        templateId = result.id;
      } else {
        const result = await createMutation.mutateAsync(data);
        templateId = result.id;
      }

      // 发布模板
      if (isDraft) {
        await publishMutation.mutateAsync({
          templateId,
          changelog: '初始版本',
        });
      }

      message.success('模板已发布');
      onSave?.(template!);
    } catch (error) {
      console.error('发布失败:', error);
    }
  };

  // 取消
  const handleCancel = () => {
    Modal.confirm({
      title: '确认取消',
      content: '确定要取消编辑吗？未保存的更改将丢失。',
      onOk: () => {
        onCancel?.();
      },
    });
  };

  // 渲染基础信息步骤
  const renderBasicStep = () => (
    <Card>
      <Form form={form} layout="vertical">
        <Row gutter={24}>
          <Col span={16}>
            <Form.Item
              label="模板名称"
              name="name"
              rules={[
                { required: true, message: '请输入模板名称' },
                { min: 2, max: 100, message: '名称长度为2-100个字符' },
              ]}
            >
              <Input
                placeholder="如：软件安装申请"
                size="large"
                prefix={<FileTextOutlined />}
              />
            </Form.Item>

            <Form.Item
              label="模板描述"
              name="description"
              rules={[
                { required: true, message: '请输入模板描述' },
                { min: 10, max: 500, message: '描述长度为10-500个字符' },
              ]}
            >
              <TextArea
                rows={4}
                placeholder="详细描述此模板的用途和适用场景..."
                showCount
                maxLength={500}
              />
            </Form.Item>

            <Row gutter={16}>
              <Col span={12}>
                <Form.Item
                  label="模板分类"
                  name="categoryId"
                  rules={[{ required: true, message: '请选择分类' }]}
                >
                  <Select
                    placeholder="选择模板分类"
                    showSearch
                    optionFilterProp="children"
                  >
                    {categories.map((cat: any) => (
                      <Option key={cat.id} value={cat.id}>
                        {cat.name}
                      </Option>
                    ))}
                  </Select>
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item label="主题颜色" name="color">
                  <ColorPicker showText />
                </Form.Item>
              </Col>
            </Row>

            <Form.Item label="标签" name="tags">
              <Select
                mode="tags"
                placeholder="添加标签（回车确认）"
                style={{ width: '100%' }}
              />
            </Form.Item>
          </Col>

          <Col span={8}>
            <Form.Item label="封面图片">
              <Upload
                listType="picture-card"
                fileList={coverImage}
                onChange={({ fileList }) => setCoverImage(fileList)}
                beforeUpload={() => false}
                maxCount={1}
              >
                {coverImage.length === 0 && (
                  <div>
                    <PlusOutlined />
                    <div style={{ marginTop: 8 }}>上传封面</div>
                  </div>
                )}
              </Upload>
              <div className="text-xs text-gray-500 mt-2">
                建议尺寸：400x200，支持 JPG/PNG
              </div>
            </Form.Item>

            <Alert
              message="模板状态"
              description={
                isDraft ? (
                  <Space>
                    <Badge status="warning" />
                    <span>草稿模式</span>
                  </Space>
                ) : (
                  <Space>
                    <Badge status="success" />
                    <span>已发布</span>
                  </Space>
                )
              }
              type={isDraft ? 'warning' : 'success'}
              showIcon
            />
          </Col>
        </Row>
      </Form>
    </Card>
  );

  // 渲染字段设计步骤
  const renderFieldsStep = () => (
    <Card>
      <Alert
        message="拖拽字段进行排序，点击字段进行配置"
        type="info"
        showIcon
        closable
        className="mb-4"
      />
      <FieldDesigner value={fields} onChange={handleFieldsChange} />
    </Card>
  );

  // 渲染默认配置步骤
  const renderDefaultsStep = () => (
    <Card>
      <Form form={form} layout="vertical">
        <Tabs>
          <Tabs.TabPane tab="工单默认值" key="ticket">
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item label="默认类型" name={['defaults', 'type']}>
                  <Select placeholder="选择工单类型">
                    <Option value="incident">事件</Option>
                    <Option value="request">服务请求</Option>
                    <Option value="problem">问题</Option>
                    <Option value="change">变更</Option>
                  </Select>
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item label="默认优先级" name={['defaults', 'priority']}>
                  <Select placeholder="选择优先级">
                    <Option value="low">低</Option>
                    <Option value="medium">中</Option>
                    <Option value="high">高</Option>
                    <Option value="urgent">紧急</Option>
                  </Select>
                </Form.Item>
              </Col>
            </Row>

            <Row gutter={16}>
              <Col span={12}>
                <Form.Item label="默认处理人" name={['defaults', 'assigneeId']}>
                  <Select
                    placeholder="选择默认处理人"
                    showSearch
                    allowClear
                  >
                    {/* 实际应该从用户列表获取 */}
                    <Option value="user1">张三</Option>
                    <Option value="user2">李四</Option>
                  </Select>
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item label="默认团队" name={['defaults', 'teamId']}>
                  <Select placeholder="选择默认团队" showSearch allowClear>
                    {/* 实际应该从团队列表获取 */}
                    <Option value="team1">技术支持团队</Option>
                    <Option value="team2">运维团队</Option>
                  </Select>
                </Form.Item>
              </Col>
            </Row>

            <Form.Item label="默认标签" name={['defaults', 'tags']}>
              <Select mode="tags" placeholder="添加默认标签" />
            </Form.Item>

            <Form.Item label="SLA配置" name={['defaults', 'slaId']}>
              <Select placeholder="选择SLA配置" allowClear>
                <Option value="sla1">标准SLA (4小时响应)</Option>
                <Option value="sla2">高优先级SLA (1小时响应)</Option>
              </Select>
            </Form.Item>
          </Tabs.TabPane>

          <Tabs.TabPane tab="字段默认值" key="fields">
            <Alert
              message="为自定义字段设置默认值"
              type="info"
              showIcon
              className="mb-4"
            />
            {fields.length === 0 ? (
              <Empty description="请先在字段设计步骤中添加字段" />
            ) : (
              <Space orientation="vertical" style={{ width: '100%' }}>
                {fields.map((field) => (
                  <Form.Item
                    key={field.id}
                    label={field.label}
                    name={['defaults', field.name]}
                  >
                    {field.type === 'text' || field.type === 'textarea' ? (
                      <Input placeholder={`${field.label}的默认值`} />
                    ) : field.type === 'number' ? (
                      <InputNumber style={{ width: '100%' }} />
                    ) : field.type === 'select' ? (
                      <Select placeholder={`选择${field.label}默认值`}>
                        {field.options?.map((opt) => (
                          <Option key={opt.value} value={opt.value}>
                            {opt.label}
                          </Option>
                        ))}
                      </Select>
                    ) : (
                      <Input placeholder={`${field.label}的默认值`} />
                    )}
                  </Form.Item>
                ))}
              </Space>
            )}
          </Tabs.TabPane>
        </Tabs>
      </Form>
    </Card>
  );

  // 渲染权限配置步骤
  const renderPermissionsStep = () => (
    <Card>
      <Form form={form} layout="vertical">
        <Alert
          message="控制哪些用户可以使用此模板"
          type="info"
          showIcon
          closable
          className="mb-4"
        />

        <Form.Item
          label="可见性"
          name={['permission', 'visibility']}
          rules={[{ required: true, message: '请选择可见性' }]}
        >
          <Radio.Group>
            <Space orientation="vertical">
              <Radio value="public">
                <strong>公开</strong> - 所有用户都可以使用
              </Radio>
              <Radio value="private">
                <strong>私有</strong> - 仅创建人可以使用
              </Radio>
              <Radio value="department">
                <strong>部门</strong> - 指定部门的用户可以使用
              </Radio>
              <Radio value="role">
                <strong>角色</strong> - 指定角色的用户可以使用
              </Radio>
              <Radio value="team">
                <strong>团队</strong> - 指定团队的成员可以使用
              </Radio>
            </Space>
          </Radio.Group>
        </Form.Item>

        <Form.Item
          noStyle
          shouldUpdate={(prevValues, currentValues) =>
            prevValues.permission?.visibility !==
            currentValues.permission?.visibility
          }
        >
          {({ getFieldValue }) => {
            const visibility = getFieldValue(['permission', 'visibility']);

            if (visibility === 'department') {
              return (
                <Form.Item
                  label="允许的部门"
                  name={['permission', 'allowedDepartments']}
                >
                  <Select mode="multiple" placeholder="选择部门">
                    <Option value="dept1">技术部</Option>
                    <Option value="dept2">运维部</Option>
                    <Option value="dept3">产品部</Option>
                  </Select>
                </Form.Item>
              );
            }

            if (visibility === 'role') {
              return (
                <Form.Item
                  label="允许的角色"
                  name={['permission', 'allowedRoles']}
                >
                  <Select mode="multiple" placeholder="选择角色">
                    <Option value="admin">管理员</Option>
                    <Option value="agent">工程师</Option>
                    <Option value="user">普通用户</Option>
                  </Select>
                </Form.Item>
              );
            }

            if (visibility === 'team') {
              return (
                <Form.Item
                  label="允许的团队"
                  name={['permission', 'allowedTeams']}
                >
                  <Select mode="multiple" placeholder="选择团队">
                    <Option value="team1">技术支持团队</Option>
                    <Option value="team2">运维团队</Option>
                  </Select>
                </Form.Item>
              );
            }

            return null;
          }}
        </Form.Item>

        <Divider>高级权限</Divider>

        <Form.Item label="拒绝的用户" name={['permission', 'denyUsers']}>
          <Select
            mode="multiple"
            placeholder="选择要排除的用户"
            allowClear
          >
            <Option value="user1">用户1</Option>
            <Option value="user2">用户2</Option>
          </Select>
        </Form.Item>
      </Form>
    </Card>
  );

  // 渲染自动化配置步骤
  const renderAutomationStep = () => (
    <Card>
      <Form form={form} layout="vertical">
        <Collapse
          defaultActiveKey={['assign']}
          expandIconPosition="end"
          className="mb-4"
        >
          {/* 自动分配 */}
          <Panel
            header={
              <Space>
                <ThunderboltOutlined />
                <strong>自动分配</strong>
              </Space>
            }
            key="assign"
          >
            <Form.Item
              label="启用自动分配"
              name={['automation', 'autoAssign']}
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>

            <Form.Item
              noStyle
              shouldUpdate={(prevValues, currentValues) =>
                prevValues.automation?.autoAssign !==
                currentValues.automation?.autoAssign
              }
            >
              {({ getFieldValue }) => {
                const autoAssign = getFieldValue(['automation', 'autoAssign']);

                if (!autoAssign) return null;

                return (
                  <>
                    <Form.Item
                      label="分配规则"
                      name={['automation', 'assignmentRule', 'type']}
                    >
                      <Select placeholder="选择分配规则">
                        <Option value="round_robin">轮流分配</Option>
                        <Option value="load_balance">负载均衡</Option>
                        <Option value="skill_based">技能匹配</Option>
                        <Option value="random">随机分配</Option>
                      </Select>
                    </Form.Item>

                    <Form.Item
                      label="目标团队"
                      name={['automation', 'assignmentRule', 'targetTeamId']}
                    >
                      <Select placeholder="选择团队">
                        <Option value="team1">技术支持团队</Option>
                        <Option value="team2">运维团队</Option>
                      </Select>
                    </Form.Item>
                  </>
                );
              }}
            </Form.Item>
          </Panel>

          {/* 审批流程 */}
          <Panel
            header={
              <Space>
                <CheckCircleOutlined />
                <strong>审批流程</strong>
              </Space>
            }
            key="approval"
          >
            <Form.Item
              label="需要审批"
              name={['automation', 'requiresApproval']}
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>

            <Form.Item
              noStyle
              shouldUpdate={(prevValues, currentValues) =>
                prevValues.automation?.requiresApproval !==
                currentValues.automation?.requiresApproval
              }
            >
              {({ getFieldValue }) => {
                const requiresApproval = getFieldValue([
                  'automation',
                  'requiresApproval',
                ]);

                if (!requiresApproval) return null;

                return (
                  <>
                    <Form.Item
                      label="审批级别"
                      name={[
                        'automation',
                        'approvalWorkflow',
                        'approvalLevel',
                      ]}
                    >
                      <Select placeholder="选择审批级别">
                        <Option value="manager">经理审批</Option>
                        <Option value="director">总监审批</Option>
                        <Option value="executive">高管审批</Option>
                        <Option value="custom">自定义审批</Option>
                      </Select>
                    </Form.Item>

                    <Form.Item
                      label="审批类型"
                      name={[
                        'automation',
                        'approvalWorkflow',
                        'approvalType',
                      ]}
                    >
                      <Radio.Group>
                        <Radio value="sequential">顺序审批</Radio>
                        <Radio value="parallel">并行审批</Radio>
                        <Radio value="any_one">任一人审批</Radio>
                      </Radio.Group>
                    </Form.Item>
                  </>
                );
              }}
            </Form.Item>
          </Panel>

          {/* 自动通知 */}
          <Panel
            header={
              <Space>
                <InfoCircleOutlined />
                <strong>自动通知</strong>
              </Space>
            }
            key="notify"
          >
            <Form.Item
              label="启用自动通知"
              name={['automation', 'autoNotify']}
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>
          </Panel>

          {/* 自动标签 */}
          <Panel
            header={
              <Space>
                <Tag>自动标签</Tag>
              </Space>
            }
            key="tag"
          >
            <Form.Item
              label="启用自动标签"
              name={['automation', 'autoTag']}
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>
          </Panel>
        </Collapse>
      </Form>
    </Card>
  );

  // 渲染当前步骤内容
  const renderStepContent = () => {
    const step = STEPS[currentStep];

    switch (step.key) {
      case 'basic':
        return renderBasicStep();
      case 'fields':
        return renderFieldsStep();
      case 'defaults':
        return renderDefaultsStep();
      case 'permissions':
        return renderPermissionsStep();
      case 'automation':
        return renderAutomationStep();
      default:
        return null;
    }
  };

  return (
    <div className="template-editor">
      {/* 顶部导航 */}
      <Card className="mb-4">
        <div className="flex items-center justify-between">
          <Space size="large">
            <h2 className="text-xl font-bold m-0">
              {mode === 'create'
                ? '创建模板'
                : mode === 'edit'
                ? '编辑模板'
                : '复制模板'}
            </h2>
            {template && (
              <Badge
                status={isDraft ? 'warning' : 'success'}
                text={isDraft ? '草稿' : '已发布'}
              />
            )}
          </Space>

          <Space>
            <Button
              icon={<EyeOutlined />}
              onClick={() => setActiveTab('preview')}
            >
              预览
            </Button>
            <Button icon={<SaveOutlined />} onClick={handleSaveDraft}>
              保存草稿
            </Button>
            <Button
              type="primary"
              icon={<RocketOutlined />}
              onClick={handlePublish}
              loading={
                createMutation.isPending ||
                updateMutation.isPending ||
                publishMutation.isPending
              }
            >
              发布模板
            </Button>
            <Button icon={<CloseOutlined />} onClick={handleCancel}>
              取消
            </Button>
          </Space>
        </div>
      </Card>

      {/* 步骤导航 */}
      <Card className="mb-4">
        <Steps current={currentStep} onChange={setCurrentStep}>
          {STEPS.map((step, index) => (
            <Step
              key={step.key}
              title={step.title}
              description={step.description}
              icon={step.icon}
            />
          ))}
        </Steps>
      </Card>

      {/* 步骤内容 */}
      <div className="mb-4">{renderStepContent()}</div>

      {/* 底部操作按钮 */}
      <Card>
        <div className="flex justify-between">
          <Button
            onClick={handlePrevious}
            disabled={currentStep === 0}
          >
            上一步
          </Button>

          <Space>
            <Button onClick={handleSaveDraft}>保存草稿</Button>
            {currentStep === STEPS.length - 1 ? (
              <Button
                type="primary"
                icon={<RocketOutlined />}
                onClick={handlePublish}
                loading={
                  createMutation.isPending ||
                  updateMutation.isPending ||
                  publishMutation.isPending
                }
              >
                发布模板
              </Button>
            ) : (
              <Button type="primary" onClick={handleNext}>
                下一步
              </Button>
            )}
          </Space>
        </div>
      </Card>
    </div>
  );
};

export default TemplateEditor;

