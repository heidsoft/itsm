import { redirect } from 'next/navigation';

// /reports/catalog-usage 是历史/别名入口。
// 实际页面落在 /reports/service-catalog-usage/，与其他报表保持一致
// （/reports/incidents → incident-trends、/reports/changes → change-success 等）。
export default function CatalogUsageRedirectPage() {
  redirect('/reports/service-catalog-usage');
}
