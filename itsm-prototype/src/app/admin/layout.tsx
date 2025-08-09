"use client";

import React from "react";
import AppLayout from "../components/AppLayout";

interface AdminLayoutProps {
  children: React.ReactNode;
}

const AdminLayout = ({ children }: AdminLayoutProps) => {
  return (
    <AppLayout title="系统管理" breadcrumb={[{ title: "系统管理" }]}>
      {children}
    </AppLayout>
  );
};

export default AdminLayout;
