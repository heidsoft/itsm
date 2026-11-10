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
      return 'There is no root cause or solution to the problem, and it has been accumulated as knowledge with no reusable value (please add the root cause or solution first)';
    }
    if (err.status === 403 || err.code === 2003) {
      return 'No knowledge base write permission (knowledge:write), unable to precipitate';
    }
    if (err.status === 503 || err.code === 5003) {
      return 'The knowledge accumulation ability is not installed, please contact the administrator';
    }
    if (err.status === 404 || err.code === 4004) {
      return 'The problem does not exist or does not belong to the current tenant';
    }
  }
  return 'Precipitation failed, please try again later.';
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
          ? 'The problem does not exist or does not belong to the current tenant'
          : 'Failed to read precipitated knowledge'
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
        `has been deposited into a knowledge draft "${created.title}》, publication must go through the knowledge review process (submit → approve)`
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

          Precipitate the phenomenon/root cause/temporary measures/solutions of the problem into a knowledge draft (including drafts, in reverse order of creation time)
        </Text>
        <Button
          type="primary"
          icon={<Plus size={14} />}
          onClick={() => setModalOpen(true)}
          data-testid="promote-knowledge-button"
        >

          Precipitation into knowledge
        </Button>
      </div>

      {loadError && (
        <Alert type="error" showIcon title={loadError} style={{ marginBottom: 12 }} />
      )}

      {/* 反查失败时只呈现 Alert，不渲染空列表——「读取失败」与「尚未沉淀」同屏自相矛盾，
          会把「读不到」误导成「没有沉淀」。 */}
      {!loadError && (
      <List
        loading={loading}
        dataSource={articles}
        locale={{
          emptyText: <Empty description="No knowledge has been accumulated yet" image={Empty.PRESENTED_IMAGE_SIMPLE} />,
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

                View
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

                      Published
                    </Tag>
                  ) : (
                    <Tag color="orange" style={{ marginLeft: 8 }}>

                      Draft
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
        title="Precipitation into knowledge"
        open={modalOpen}
        onOk={handlePromote}
        confirmLoading={submitting}
        onCancel={() => {
          setModalOpen(false);
          form.resetFields();
        }}
        okText="Precipitation"
        destroyOnHidden
      >
        <Alert
          type="info"
          showIcon
          title="The content will be automatically assembled according to &quot;Problem phenomenon/Root cause analysis/Temporary measures/Solutions&quot;; the product is a draft, and release must go through the knowledge review process."
          style={{ marginBottom: 16 }}
        />
        <Form form={form} layout="vertical">
          <Form.Item name="title" label="Title (optional)">
            <Input placeholder="Leave blank to use question title" maxLength={200} />
          </Form.Item>
          <Form.Item name="category" label="Classification (optional)">
            <Input placeholder="Leave blank for &quot;Troubleshooting&quot;" maxLength={64} />
          </Form.Item>
          <Form.Item name="tags" label="tag (optional)">
            <Select
              mode="tags"
              placeholder="Leave blank only with automatic label &quot;Problem Precipitation&quot; and problem number"
              tokenSeparators={[',', '，']}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default ProblemKnowledgeTab;
