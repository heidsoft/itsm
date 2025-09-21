import React from "react";
import AppLayout from "../components/AppLayout";

export default function CMDBLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout>
      {children}
    </AppLayout>
  );
}