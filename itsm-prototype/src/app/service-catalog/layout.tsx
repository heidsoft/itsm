import React from "react";
import AppLayout from "../components/AppLayout";

export default function ServiceCatalogLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout title="服务目录" breadcrumb={[{ title: "服务目录" }]}>
      {children}
    </AppLayout>
  );
}
