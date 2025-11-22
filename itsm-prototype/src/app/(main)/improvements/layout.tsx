import React from "react";
import AppLayout from "@/components/layout/AppLayout";

export default function ImprovementsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout title="持续改进" breadcrumb={[{ title: "持续改进" }]}>
      {children}
    </AppLayout>
  );
}
