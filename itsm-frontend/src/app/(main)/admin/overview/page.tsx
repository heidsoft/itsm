import { redirect } from 'next/navigation';

/**
 * Admin Overview Page - redirects to /admin
 * This page exists to handle the /admin/overview route defined in sidebar menu
 */
export default function AdminOverviewPage() {
  redirect('/admin');
}
