import React from "react";
import AppLayout from "../components/AppLayout";

export default function ServiceCatalogLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <AppLayout>{children}</AppLayout>;
}