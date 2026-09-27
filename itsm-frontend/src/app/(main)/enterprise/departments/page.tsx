import { redirect } from 'next/navigation';

/**
 * 部门管理页面
 * 重定向到 /admin/departments
 * 保留 /enterprise/departments 路由以兼容旧链接
 */
export default function EnterpriseDepartmentsPage() {
  redirect('/admin/departments');
}
