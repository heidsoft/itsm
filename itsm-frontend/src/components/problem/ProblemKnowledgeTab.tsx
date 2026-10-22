'use client';

/**
 * 问题详情「知识沉淀」Tab（2026-10-22 沉淀闭环落地）
 *
 * 数据源：
 *   - POST /api/v1/problems/:id/knowledge 沉淀（draft，发布须过知识评审流）
 *   - GET  /api/v1/problems/:id/knowledge 反查（含草稿，创建时间倒序）
 *
 * 渲染红线：
 *   - 沉淀产物恒为 draft，必须如实标注「草稿」——沉淀降低的是记录成本，不是审核成本，
 *     不能把 draft 渲染成已发布；
 *   - 422（问题既无根因也无解决方案）要原样呈现后端理由，这是质量门不是流程绊脚石；
 *   - 403/503 分别呈现「无知识库写权限」与「沉淀能力未装配」，不静默假成功；
 *   - 空列表用 Empty 明示「尚未沉淀」，不用 null/undefined 蒙混。
 */

import React, { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  Button,
  Empty,
  Form,
  Input,
  List,
  Modal,
  Select,
  Tag,
  Typography,
  message,
} from 'antd';
import { BookOpen, Plus } from 'lucide-react';
import dayjs from 'dayjs';
import { useRouter } from 'next/navigation';
import {
  ProblemApi,
  type ProblemKnowledgeArticle,
  type PromoteProblemKnowledgeRequest,
} from '@/lib/api/problem-api';
import { ApiRequestError } from '@/lib/api/http-client';

const { Text } = Typography;

export interface ProblemKnowledgeTabProps {
  problemId: number;
}

/** 错误码 → 人话。与后端契约一一对应，新增码时同步。 */
function describePromoteError(err: unknown): string {
  if (err instanceof ApiRequestError) {
    if (err.status === 422 || err.code === 4220) {
      return '问题尚无根因或解决方案，沉淀为知识没有可复用价值（请先补充根因或解决方案）';
    }
    if (err.status === 403 || err.code === 2003) {
      return '没有知识库写权限（knowledge:write），无法沉淀';
    }
    if (err.status === 503 || err.code === 5003) {
      return '知识沉淀能力未装配，请联系管理员';
    }
    if (err.status === 404 || err.code === 4004) {
      return '问题不存在或不属于当前租户';
    }
  }
  return '沉淀失败，请稍后重试';
}

export const ProblemKnowledgeTab: React.FC<ProblemKnowledgeTabProps> = ({ problemId }) => {
  const router = useRouter();
  const [articles, setArticles] = useState<ProblemKnowledgeArticle[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [modalOpen, setModalOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm<PromoteProblemKnowledgeRequest>();

  const fetchArticles = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const data = await ProblemApi.listProblemKnowledge(problemId);
      setArticles(data.items ?? []);
    } catch (err) {
      // 反查失败不阻断详情页主体，但必须明示——不能把「读不到」渲染成「没有沉淀」。
      setLoadError(
        err instanceof ApiRequestError && (err.status === 404 || err.code === 4004)
          ? '问题不存在或不属于当前租户'
          : '沉淀知识读取失败'
      );
      setArticles([]);
    } finally {
      setLoading(false);
    }
  }, [problemId]);

  useEffect(() => {
    if (Number.isFinite(problemId) && problemId > 0) {
      void fetchArticles();
    }
  }, [problemId, fetchArticles]);

  const handlePromote = async () => {
    let values: PromoteProblemKnowledgeRequest;
    try {
      values = await form.validateFields();
    } catch {
      return; // 表单校验未过，错误由 Form 自行呈现
    }
    setSubmitting(true);
    try {
      // 全字段可选：空值用问题单内容兜底，空对象即可触发全兜底路径。
      const payload: PromoteProblemKnowledgeRequest = {};
      if (values.title?.trim()) payload.title = values.title.trim();
      if (values.category?.trim()) payload.category = values.category.trim();
      payload.tags = (values.tags ?? [])
        .map((t) => t.trim())
        .filter((t) => t.length > 0);
      const created = await ProblemApi.promoteProblemKnowledge(problemId, payload);
      message.success(
        `已沉淀为知识草稿《${created.title}》，发布须过知识评审流（submit → approve）`
      );
      setModalOpen(false);
      form.resetFields();
      await fetchArticles();
    } catch (err) {
      message.error(describePromoteError(err));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div data-testid="problem-knowledge-tab">
      <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'space-between' }}>
        <Text type="secondary">
          把问题的现象 / 根因 / 临时措施 / 解决方案沉淀为知识草稿（含草稿，创建时间倒序）
        </Text>
        <Button
          type="primary"
          icon={<Plus size={14} />}
          onClick={() => setModalOpen(true)}
          data-testid="promote-knowledge-button"
        >
          沉淀为知识
        </Button>
      </div>

      {loadError && (
        <Alert type="error" showIcon message={loadError} style={{ marginBottom: 12 }} />
      )}

      {/* 反查失败时只呈现 Alert，不渲染空列表——「读取失败」与「尚未沉淀」同屏自相矛盾，
          会把「读不到」误导成「没有沉淀」。 */}
      {!loadError && (
      <List
        loading={loading}
        dataSource={articles}
        locale={{
          emptyText: <Empty description="尚未沉淀知识" image={Empty.PRESENTED_IMAGE_SIMPLE} />,
        }}
        renderItem={(article) => (
          <List.Item
            data-testid="problem-knowledge-item"
            actions={[
              <Button
                key="open"
                type="link"
                onClick={() => router.push(`/knowledge/articles/${article.id}`)}
              >
                查看
              </Button>,
            ]}
          >
            <List.Item.Meta
              title={
                <span>
                  <BookOpen size={14} className="inline mr-1" />
                  {article.title}
                  {article.status === 'published' ? (
                    <Tag color="green" style={{ marginLeft: 8 }}>
                      已发布
                    </Tag>
                  ) : (
                    <Tag color="orange" style={{ marginLeft: 8 }}>
                      草稿
                    </Tag>
                  )}
                </span>
              }
              description={
                <span>
                  {dayjs(article.createdAt).format('YYYY-MM-DD HH:mm')}
                  {article.tags.length > 0 && (
                    <span style={{ marginLeft: 12 }}>
                      {article.tags.map((tag) => (
                        <Tag key={tag}>{tag}</Tag>
                      ))}
                    </span>
                  )}
                </span>
              }
            />
          </List.Item>
        )}
      />
      )}

      <Modal
        title="沉淀为知识"
        open={modalOpen}
        onOk={handlePromote}
        confirmLoading={submitting}
        onCancel={() => {
          setModalOpen(false);
          form.resetFields();
        }}
        okText="沉淀"
        destroyOnClose
      >
        <Alert
          type="info"
          showIcon
          message="内容将按「问题现象 / 根因分析 / 临时措施 / 解决方案」自动组装；产物为草稿，发布须过知识评审流。"
          style={{ marginBottom: 16 }}
        />
        <Form form={form} layout="vertical">
          <Form.Item name="title" label="标题（可选）">
            <Input placeholder="留空则使用问题标题" maxLength={200} />
          </Form.Item>
          <Form.Item name="category" label="分类（可选）">
            <Input placeholder="留空则为「故障排查」" maxLength={64} />
          </Form.Item>
          <Form.Item name="tags" label="标签（可选）">
            <Select
              mode="tags"
              placeholder="留空仅带自动标签「问题沉淀」与问题编号"
              tokenSeparators={[',', '，']}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default ProblemKnowledgeTab;
