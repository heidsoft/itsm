"use client";

import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import { AuthService } from "../lib/auth-service";

interface AuthGuardProps {
  children: React.ReactNode;
}

export function AuthGuard({ children }: AuthGuardProps) {
  const router = useRouter();
  const pathname = usePathname();
  const [isLoading, setIsLoading] = useState(true);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [hasChecked, setHasChecked] = useState(false);

  useEffect(() => {
    const checkAuth = () => {
      console.log("AuthGuard: 检查认证状态", { pathname });

      const authenticated = AuthService.isAuthenticated();
      console.log("AuthGuard: 认证状态", { authenticated, pathname });

      setIsAuthenticated(authenticated);
      setIsLoading(false);
      setHasChecked(true);

      // 只有在已经检查过且状态稳定时才进行重定向
      if (hasChecked) {
        // 如果未认证且不在登录页面，重定向到登录页
        if (
          !authenticated &&
          pathname !== "/login" &&
          pathname !== "/simple-test"
        ) {
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
      }

      console.log("AuthGuard: 状态正常，显示内容");
    };

    // 添加延迟确保状态更新
    const timer = setTimeout(checkAuth, 200);
    return () => clearTimeout(timer);
  }, [router, pathname, hasChecked]);

  if (isLoading) {
    console.log("AuthGuard: 显示加载状态");
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-blue-600"></div>
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
    <div className="min-h-screen flex items-center justify-center">
      <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-blue-600"></div>
    </div>
  );
}
