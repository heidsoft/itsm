"use client";

import { Plus, Search, Trash2, Edit } from 'lucide-react';

import React, { useState, useEffect } from "react";
import  from 'lucide-react';
import {
  ServiceCatalogApi,
  ServiceCatalog,
  CreateServiceCatalogRequest,
  UpdateServiceCatalogRequest,
} from "../../lib/service-catalog-api";

const ServiceCatalogManagement = () => {
  const [catalogs, setCatalogs] = useState<ServiceCatalog[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editingCatalog, setEditingCatalog] = useState<ServiceCatalog | null>(
    null
  );
  const [searchTerm, setSearchTerm] = useState("");

  const [formData, setFormData] = useState<CreateServiceCatalogRequest>({
    name: "",
    category: "",
    description: "",
    delivery_time: "",
    status: "enabled",
  });

  // 获取服务目录列表
  const fetchCatalogs = async () => {
    try {
      setLoading(true);
      const response = await ServiceCatalogApi.getServiceCatalogs({
        page: 1,
        size: 100,
      });
      setCatalogs(response.catalogs);
    } catch (error) {
      console.error("Failed to fetch catalogs:", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchCatalogs();
  }, []);

  // 处理表单提交
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      if (editingCatalog) {
        await ServiceCatalogApi.updateServiceCatalog(
          editingCatalog.id,
          formData
        );
      } else {
        await ServiceCatalogApi.createServiceCatalog(formData);
      }
      setShowModal(false);
      setEditingCatalog(null);
      resetForm();
      fetchCatalogs();
    } catch (error) {
      console.error("Failed to save catalog:", error);
    }
  };

  // 重置表单
  const resetForm = () => {
    setFormData({
      name: "",
      category: "",
      description: "",
      delivery_time: "",
      status: "enabled",
    });
  };

  // 编辑服务目录
  const handleEdit = (catalog: ServiceCatalog) => {
    setEditingCatalog(catalog);
    setFormData({
      name: catalog.name,
      category: catalog.category,
      description: catalog.description,
      delivery_time: catalog.delivery_time,
      status: catalog.status as "enabled" | "disabled",
    });
    setShowModal(true);
  };

  // 删除服务目录
  const handleDelete = async (id: number) => {
    if (confirm("确定要删除这个服务目录吗？")) {
      try {
        await ServiceCatalogApi.deleteServiceCatalog(id);
        fetchCatalogs();
      } catch (error) {
        console.error("Failed to delete catalog:", error);
      }
    }
  };

  // 过滤服务目录
  const filteredCatalogs = catalogs.filter(
    (catalog) =>
      catalog.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      catalog.category.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">服务目录管理</h1>
        <button
          onClick={() => {
            setEditingCatalog(null);
            resetForm();
            setShowModal(true);
          }}
          className="bg-blue-600 text-white px-4 py-2 rounded-lg flex items-center gap-2 hover:bg-blue-700"
        >
          <Plus className="w-4 h-4" />
          新建服务目录
        </button>
      </div>

      {/* 搜索框 */}
      <div className="mb-6">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-4 h-4" />
          <input
            type="text"
            placeholder="搜索服务目录..."
            className="pl-10 pr-4 py-2 border border-gray-300 rounded-lg w-full max-w-md"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
          />
        </div>
      </div>

      {/* 服务目录列表 */}
      <div className="bg-white rounded-lg shadow overflow-hidden">
        {loading ? (
          <div className="p-8 text-center">
            <div className="text-gray-500">加载中...</div>
          </div>
        ) : filteredCatalogs.length === 0 ? (
          <div className="p-8 text-center">
            <div className="text-gray-500">暂无服务目录数据</div>
          </div>
        ) : (
          <table className="min-w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  名称
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  分类
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  交付时间
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  状态
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  操作
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {filteredCatalogs.map((catalog) => (
                <tr key={catalog.id}>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div>
                      <div className="text-sm font-medium text-gray-900">
                        {catalog.name}
                      </div>
                      <div className="text-sm text-gray-500">
                        {catalog.description}
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {catalog.category}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {catalog.delivery_time}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                        catalog.status === "enabled"
                          ? "bg-green-100 text-green-800"
                          : "bg-red-100 text-red-800"
                      }`}
                    >
                      {catalog.status === "enabled" ? "启用" : "禁用"}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                    <button
                      onClick={() => handleEdit(catalog)}
                      className="text-blue-600 hover:text-blue-900 mr-4"
                    >
                      <Edit className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => handleDelete(catalog.id)}
                      className="text-red-600 hover:text-red-900"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* 模态框 */}
      {showModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h2 className="text-lg font-semibold mb-4">
              {editingCatalog ? "编辑服务目录" : "新建服务目录"}
            </h2>
            <form onSubmit={handleSubmit}>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    名称
                  </label>
                  <input
                    type="text"
                    required
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    value={formData.name}
                    onChange={(e) =>
                      setFormData({ ...formData, name: e.target.value })
                    }
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    分类
                  </label>
                  <select
                    required
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    value={formData.category}
                    onChange={(e) =>
                      setFormData({ ...formData, category: e.target.value })
                    }
                  >
                    <option value="">请选择分类</option>
                    <option value="云服务">云服务</option>
                    <option value="基础设施">基础设施</option>
                    <option value="应用服务">应用服务</option>
                    <option value="数据服务">数据服务</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    描述
                  </label>
                  <textarea
                    required
                    rows={3}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    value={formData.description}
                    onChange={(e) =>
                      setFormData({ ...formData, description: e.target.value })
                    }
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    交付时间
                  </label>
                  <input
                    type="text"
                    required
                    placeholder="例如：1-3个工作日"
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    value={formData.delivery_time}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        delivery_time: e.target.value,
                      })
                    }
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    状态
                  </label>
                  <select
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    value={formData.status}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        status: e.target.value as "enabled" | "disabled",
                      })
                    }
                  >
                    <option value="enabled">启用</option>
                    <option value="disabled">禁用</option>
                  </select>
                </div>
              </div>
              <div className="flex justify-end space-x-3 mt-6">
                <button
                  type="button"
                  onClick={() => {
                    setShowModal(false);
                    setEditingCatalog(null);
                    resetForm();
                  }}
                  className="px-4 py-2 text-gray-600 border border-gray-300 rounded-md hover:bg-gray-50"
                >
                  取消
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
                >
                  {editingCatalog ? "更新" : "创建"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default ServiceCatalogManagement;
