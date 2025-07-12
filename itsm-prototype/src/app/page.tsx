"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { AuthService } from "./lib/auth-service";

export default function Home() {
  const router = useRouter();

  useEffect(() => {
    // 检查用户是否已登录
    if (AuthService.isAuthenticated()) {
      router.push("/dashboard");
    } else {
      router.push("/login");
    }
  }, [router]);

  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-blue-600"></div>
    </div>
  );
}
