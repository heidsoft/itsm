"use client";

import React, { useState, useEffect } from "react";
import { ChevronDown, Building2, Check } from "lucide-react";
import { useAuthStore } from "../lib/store";
import { TenantAPI } from "../lib/tenant-api";
import { Tenant } from "../lib/api-config";

interface TenantSelectorProps {
  className?: string;
}

const TenantSelector: React.FC<TenantSelectorProps> = ({ className = "" }) => {
  const { currentTenant, setCurrentTenant, user } = useAuthStore();
  const [isOpen, setIsOpen] = useState(false);
  const [availableTenants, setAvailableTenants] = useState<Tenant[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    loadAvailableTenants();
  }, []);

  const loadAvailableTenants = async () => {
    if (!user) return;

    try {
      setLoading(true);
      // 如果用户是超级管理员，获取所有租户；否则只获取用户所属的租户
      const response = await TenantAPI.getTenants();
      setAvailableTenants(response.tenants);
    } catch (error) {
      console.error("Failed to load tenants:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleTenantSwitch = async (tenant: Tenant) => {
    try {
      await TenantAPI.switchTenant(tenant.id);
      setCurrentTenant(tenant);
      setIsOpen(false);
      // 刷新页面以重新加载租户相关数据
      window.location.reload();
    } catch (error) {
      console.error("Failed to switch tenant:", error);
    }
  };

  if (!currentTenant || availableTenants.length <= 1) {
    return null;
  }

  return (
    <div className={`relative ${className}`}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center space-x-2 px-3 py-2 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
      >
        <Building2 className="w-4 h-4 text-gray-500" />
        <span className="text-sm font-medium text-gray-700">
          {currentTenant.name}
        </span>
        <ChevronDown className="w-4 h-4 text-gray-500" />
      </button>

      {isOpen && (
        <div className="absolute top-full left-0 mt-1 w-64 bg-white border border-gray-200 rounded-lg shadow-lg z-50">
          <div className="p-2">
            <div className="text-xs font-medium text-gray-500 px-2 py-1 mb-1">
              切换租户
            </div>
            {loading ? (
              <div className="px-2 py-2 text-sm text-gray-500">加载中...</div>
            ) : (
              availableTenants.map((tenant) => (
                <button
                  key={tenant.id}
                  onClick={() => handleTenantSwitch(tenant)}
                  className="w-full flex items-center justify-between px-2 py-2 text-sm text-gray-700 hover:bg-gray-100 rounded"
                >
                  <div className="flex items-center space-x-2">
                    <Building2 className="w-4 h-4 text-gray-400" />
                    <div className="text-left">
                      <div className="font-medium">{tenant.name}</div>
                      <div className="text-xs text-gray-500">{tenant.code}</div>
                    </div>
                  </div>
                  {currentTenant.id === tenant.id && (
                    <Check className="w-4 h-4 text-blue-600" />
                  )}
                </button>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default TenantSelector;
