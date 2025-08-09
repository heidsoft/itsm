import React from "react";
import AppLayout from "../components/AppLayout";

export default function MyRequestsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout title="我的请求" breadcrumb={[{ title: "我的请求" }]}>
      {children}
    </AppLayout>
  );
}
