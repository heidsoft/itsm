import { redirect } from 'next/navigation';

/**
 * 组织架构页面
 * 重定向到 /admin/departments
 * 保留 /system/organization 路由以兼容旧链接
 */
export default function OrganizationPage() {
  redirect('/admin/departments');
}
