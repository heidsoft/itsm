import { redirect } from 'next/navigation';

/**
 * 旧版工作流一体化页面（列表+内嵌 BPMN 设计）。
 * 工作流元数据管理已迁移至 /admin/workflows，流程编排请使用 /workflow/designer。
 * 本页仅作兼容保留，访问时直接重定向到管理页。
 */
export default function WorkflowPage() {
  redirect('/admin/workflows');
}
