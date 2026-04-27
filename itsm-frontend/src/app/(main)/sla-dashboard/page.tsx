'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';

/**
 * SLA Dashboard page - redirects to /sla for now
 * The actual SLA dashboard is at /sla
 */
export default function SLADashboardPage() {
  const router = useRouter();

  useEffect(() => {
    // Redirect to /sla which contains the full SLA dashboard
    router.replace('/sla');
  }, [router]);

  return null;
}
