import { redirect } from 'next/navigation';

/**
 * 工作流管理页面
 * 重定向到 /admin/workflows
 * 保留 /workflows 路由以兼容旧链接
 */
export default function WorkflowsPage() {
  redirect('/admin/workflows');
}
