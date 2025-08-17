import React from "react";
import AppLayout from "../components/AppLayout";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout title="仪表盘" breadcrumb={[{ title: "仪表盘" }]}>
      {children}
    </AppLayout>
  );
}
