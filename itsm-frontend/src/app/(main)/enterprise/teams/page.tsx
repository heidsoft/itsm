import { redirect } from 'next/navigation';

/**
 * 团队管理页面
 * 重定向到 /admin/teams
 * 保留 /enterprise/teams 路由以兼容旧链接
 */
export default function EnterpriseTeamsPage() {
  redirect('/admin/teams');
}
