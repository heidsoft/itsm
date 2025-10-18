"use client";

import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import { MockAuthService } from "@/lib/auth/mock-auth-service";
import { useAuthStore } from "@/lib/store/auth-store";

interface AuthGuardProps {
  children: React.ReactNode;
}

export function AuthGuard({ children }: AuthGuardProps) {
  const router = useRouter();
  const pathname = usePathname();
  const [isLoading, setIsLoading] = useState(true);
  const { isAuthenticated } = useAuthStore();

  useEffect(() => {
    const initializeAuth = async () => {
      console.log("AuthGuard: 初始化认证状态", { pathname });

      try {
        // 初始化认证服务
        await MockAuthService.initialize();
        
        const authenticated = MockAuthService.isAuthenticated();
        console.log("AuthGuard: 认证状态", { authenticated, pathname });

        setIsLoading(false);

        // 如果未认证且不在登录页面，重定向到登录页
        if (!authenticated && pathname !== "/login") {
          console.log("AuthGuard: 未认证，重定向到登录页");
          router.push("/login");
          return;
        }

        // 如果已认证且在登录页面，重定向到dashboard
        if (authenticated && pathname === "/login") {
          console.log("AuthGuard: 已认证且在登录页，重定向到dashboard");
          router.push("/dashboard");
          return;
        }

        console.log("AuthGuard: 状态正常，显示内容");
      } catch (error) {
        console.error("AuthGuard: 初始化认证失败", error);
        setIsLoading(false);
        
        // 初始化失败，如果不在登录页则重定向
        if (pathname !== "/login") {
          router.push("/login");
        }
      }
    };

    initializeAuth();
  }, [router, pathname]);

  if (isLoading) {
    console.log("AuthGuard: 显示加载状态");
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-gray-600">正在加载...</p>
        </div>
      </div>
    );
  }

  // 如果在登录页面，直接显示内容
  if (pathname === "/login") {
    console.log("AuthGuard: 在登录页面，直接显示内容");
    return <>{children}</>;
  }

  // 如果已认证，显示内容
  if (isAuthenticated) {
    console.log("AuthGuard: 已认证，显示内容");
    return <>{children}</>;
  }

  // 其他情况显示加载状态
  console.log("AuthGuard: 其他情况，显示加载状态");
  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="text-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
        <p className="text-gray-600">正在验证身份...</p>
      </div>
    </div>
  );
}
