import React from "react";
import AppLayout from "@/components/layout/AppLayout";

export default function CMDBLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout title="配置管理" description="IT资产和配置项管理" showPageHeader={true}>
      {children}
    </AppLayout>
  );
}