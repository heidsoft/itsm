'use client';

import { redirect } from 'next/navigation';
import { useEffect } from 'react';

/**
 * Admin Overview Page - redirects to /admin
 * This page exists to handle the /admin/overview route defined in sidebar menu
 */
export default function AdminOverviewPage() {
  useEffect(() => {
    redirect('/admin');
  }, []);

  return null;
}
