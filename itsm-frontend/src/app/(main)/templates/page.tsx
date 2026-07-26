import { redirect } from 'next/navigation';

/**
 * 模板管理唯一入口为 /tickets/templates，本路由仅做兼容跳转。
 */
export default function TemplatesRedirectPage() {
  redirect('/tickets/templates');
}
