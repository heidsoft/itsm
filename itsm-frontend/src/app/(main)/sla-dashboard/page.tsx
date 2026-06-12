import { redirect } from 'next/navigation';

/**
 * SLA Dashboard page - redirects to /sla for now
 * The actual SLA dashboard is at /sla
 */
export default function SLADashboardPage() {
  redirect('/sla');
}
