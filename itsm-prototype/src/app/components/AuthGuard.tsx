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

  useEffect(() => {
    const checkAuth = () => {
      const authenticated = AuthService.isAuthenticated();
      setIsAuthenticated(authenticated);
      setIsLoading(false);

      // 如果未认证且不在登录页面，重定向到登录页
      if (!authenticated && pathname !== "/login") {
        router.push("/login");
        return;
      }

      // 如果已认证且在登录页面，重定向到dashboard
      if (authenticated && pathname === "/login") {
        router.push("/dashboard");
        return;
      }
    };

    checkAuth();
  }, [router, pathname]);

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  // 如果在登录页面，直接显示内容
  if (pathname === "/login") {
    return <>{children}</>;
  }

  // 如果已认证，显示内容
  if (isAuthenticated) {
    return <>{children}</>;
  }

  // 其他情况显示加载状态
  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-blue-600"></div>
    </div>
  );
}
