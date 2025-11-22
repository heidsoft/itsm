import React from "react";
import AppLayout from "@/components/layout/AppLayout";

export default function ServiceCatalogLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <AppLayout title="服务目录" description="IT服务目录和请求" showPageHeader={true}>{children}</AppLayout>;
}