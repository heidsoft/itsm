import React from "react";
import AppLayout from "../components/AppLayout";

export default function SLADashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout
      title="SLA监控仪表盘"
      breadcrumb={[
        { title: "仪表盘", href: "/dashboard" },
        { title: "SLA监控", href: "/sla-dashboard" },
      ]}
    >
      {children}
    </AppLayout>
  );
}