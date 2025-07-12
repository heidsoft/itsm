"use client";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import {
  HardDrive,
  UserCog,
  ShieldCheck,
  Search,
  Clock,
  AlertCircle,
} from "lucide-react";
import { ServiceCatalogApi, ServiceCatalog } from "../lib/service-catalog-api";

// 图标映射
const categoryIcons = {
  云资源服务: HardDrive,
  账号与权限: UserCog,
  安全服务: ShieldCheck,
};

const ServiceItemCard = ({ catalog }: { catalog: ServiceCatalog }) => {
  const IconComponent = categoryIcons[catalog.category] || HardDrive;

  return (
    <div className="bg-white p-6 rounded-lg shadow-md hover:shadow-xl transition-all duration-300 flex flex-col">
      <div className="flex items-center mb-3">
        <IconComponent className="w-5 h-5 mr-2 text-blue-600" />
        <h4 className="text-lg font-semibold text-gray-800">{catalog.name}</h4>
      </div>
      <p className="text-gray-600 mt-2 flex-grow">{catalog.description}</p>
      <div className="flex items-center text-sm text-gray-500 mt-4 pt-4 border-t border-gray-100">
        <Clock className="w-4 h-4 mr-2" />
        <span>预计交付时间: {catalog.delivery_time}</span>
      </div>
      <Link href={`/service-catalog/request/${catalog.id}`} passHref>
        <button className="mt-4 w-full bg-blue-600 text-white font-semibold py-2 rounded-lg hover:bg-blue-700 transition-colors">
          发起请求
        </button>
      </Link>
    </div>
  );
};

const ServiceCatalogPage = () => {
  const [searchTerm, setSearchTerm] = useState("");
  const [catalogs, setCatalogs] = useState<ServiceCatalog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedCategory, setSelectedCategory] = useState<string>("");

  // 获取服务目录数据
  useEffect(() => {
    const fetchCatalogs = async () => {
      try {
        setLoading(true);
        const response = await ServiceCatalogApi.getServiceCatalogs({
          page: 1,
          size: 100,
          category: selectedCategory || undefined,
          status: "enabled",
        });
        setCatalogs(response.data.catalogs);
        setError(null);
      } catch (err) {
        setError("获取服务目录失败，请稍后重试");
        console.error("Failed to fetch service catalogs:", err);
      } finally {
        setLoading(false);
      }
    };

    fetchCatalogs();
  }, [selectedCategory]);

  // 过滤服务目录
  const filteredCatalogs = catalogs.filter(
    (catalog) =>
      catalog.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      catalog.description.toLowerCase().includes(searchTerm.toLowerCase())
  );

  // 按分类分组
  const groupedCatalogs = filteredCatalogs.reduce((acc, catalog) => {
    if (!acc[catalog.category]) {
      acc[catalog.category] = [];
    }
    acc[catalog.category].push(catalog);
    return acc;
  }, {} as Record<string, ServiceCatalog[]>);

  const categories = Object.keys(groupedCatalogs);

  if (loading) {
    return (
      <div className="p-10 bg-gray-50 min-h-full flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-gray-600">加载服务目录中...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-10 bg-gray-50 min-h-full flex items-center justify-center">
        <div className="text-center">
          <AlertCircle className="w-12 h-12 text-red-500 mx-auto mb-4" />
          <p className="text-red-600 mb-4">{error}</p>
          <button
            onClick={() => window.location.reload()}
            className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700"
          >
            重新加载
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="p-10 bg-gray-50 min-h-full">
      <header className="mb-8">
        <h2 className="text-4xl font-bold text-gray-800">服务目录</h2>
        <p className="text-gray-500 mt-1">您的一站式IT服务请求中心</p>
      </header>

      <div className="mb-8 space-y-4">
        <div className="relative">
          <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
          <input
            type="text"
            placeholder="搜索服务，例如：虚拟机、重置密码..."
            className="w-full pl-12 pr-4 py-3 border border-gray-300 rounded-lg shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
          />
        </div>

        <div className="flex flex-wrap gap-2">
          <button
            onClick={() => setSelectedCategory("")}
            className={`px-4 py-2 rounded-lg transition-colors ${
              selectedCategory === ""
                ? "bg-blue-600 text-white"
                : "bg-white text-gray-700 hover:bg-gray-100"
            }`}
          >
            全部分类
          </button>
          {Array.from(new Set(catalogs.map((c) => c.category))).map(
            (category) => (
              <button
                key={category}
                onClick={() => setSelectedCategory(category)}
                className={`px-4 py-2 rounded-lg transition-colors ${
                  selectedCategory === category
                    ? "bg-blue-600 text-white"
                    : "bg-white text-gray-700 hover:bg-gray-100"
                }`}
              >
                {category}
              </button>
            )
          )}
        </div>
      </div>

      <div className="space-y-12">
        {categories.length > 0 ? (
          categories.map((category) => {
            const IconComponent = categoryIcons[category] || HardDrive;
            return (
              <section key={category}>
                <div className="flex items-center mb-4">
                  <IconComponent className="w-6 h-6 mr-3 text-blue-600" />
                  <h3 className="text-2xl font-semibold text-gray-700">
                    {category}
                  </h3>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
                  {groupedCatalogs[category].map((catalog) => (
                    <ServiceItemCard key={catalog.id} catalog={catalog} />
                  ))}
                </div>
              </section>
            );
          })
        ) : (
          <div className="text-center py-16">
            <p className="text-gray-500 text-lg">未找到匹配的服务项。</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default ServiceCatalogPage;
