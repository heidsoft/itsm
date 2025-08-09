import React from "react";
import AppLayout from "../components/AppLayout";

export default function CMDBLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout title="配置管理" breadcrumb={[{ title: "配置管理" }]}>
      {children}
    </AppLayout>
  );
}
