"use client";

import React, { useState } from "react";
import {
  Modal,
  Button,
  Form,
  Select,
  Switch,
  Checkbox,
  Space,
  Typography,
  Card,
  Divider,
  message,
  Spin,
} from "antd";
import {
  Download,
  FileSpreadsheet,
  FileText,
  FileSpreadsheet as FileCsv,
  Database,
  Settings,
  CheckCircle,
  FileSpreadsheet as FileExcel,
} from "lucide-react";
import {
  ticketCategoryService,
  type CategoryTreeItem,
} from "../lib/services/ticket-category-service";

const { Option } = Select;
const { Text, Title } = Typography;
const { CheckboxGroup } = Checkbox;

interface TicketCategoryExportProps {
  visible: boolean;
  onCancel: () => void;
  onSuccess?: () => void;
}

interface ExportOptions {
  format: "csv" | "excel" | "json";
  includeInactive: boolean;
  includeSystem: boolean;
  includeMetadata: boolean;
  flattenStructure: boolean;
  selectedFields: string[];
  encoding: "utf8" | "gbk";
}

const TicketCategoryExport: React.FC<TicketCategoryExportProps> = ({
  visible,
  onCancel,
  onSuccess,
}) => {
  const [form] = Form.useForm();
  const [exporting, setExporting] = useState(false);
  const [exportProgress, setExportProgress] = useState(0);

  // 默认导出选项
  const defaultOptions: ExportOptions = {
    format: "excel",
    includeInactive: true,
    includeSystem: true,
    includeMetadata: true,
    flattenStructure: false,
    selectedFields: [
      "name",
      "code",
      "description",
      "parent_id",
      "level",
      "sort_order",
      "is_active",
    ],
    encoding: "utf8",
  };

  // 可选择的字段
  const availableFields = [
    { key: "name", label: "分类名称", required: true },
    { key: "code", label: "分类代码", required: true },
    { key: "description", label: "分类描述" },
    { key: "parent_id", label: "父分类ID" },
    { key: "parent_name", label: "父分类名称" },
    { key: "level", label: "层级" },
    { key: "sort_order", label: "排序顺序" },
    { key: "is_active", label: "是否启用" },
    { key: "tenant_id", label: "租户ID" },
    { key: "created_at", label: "创建时间" },
    { key: "updated_at", label: "更新时间" },
    { key: "created_by", label: "创建人" },
    { key: "updated_by", label: "更新人" },
  ];

  // 处理导出
  const handleExport = async (values: ExportOptions) => {
    try {
      setExporting(true);
      setExportProgress(0);

      // 模拟导出进度
      const progressInterval = setInterval(() => {
        setExportProgress((prev) => {
          if (prev >= 90) {
            clearInterval(progressInterval);
            return 90;
          }
          return prev + 10;
        });
      }, 200);

      // 获取分类数据
      const categories = await ticketCategoryService.getCategoryTree();

      // 处理数据格式
      const processedData = processExportData(categories, values);

      // 执行导出
      await performExport(processedData, values);

      clearInterval(progressInterval);
      setExportProgress(100);

      message.success("导出完成");

      if (onSuccess) {
        onSuccess();
      }

      // 延迟关闭模态框
      setTimeout(() => {
        onCancel();
      }, 1000);
    } catch (error) {
      message.error(
        "导出失败: " + (error instanceof Error ? error.message : "未知错误")
      );
    } finally {
      setExporting(false);
      setExportProgress(0);
    }
  };

  // 处理导出数据
  const processExportData = (
    categories: CategoryTreeItem[],
    options: ExportOptions
  ): any[] => {
    const result: any[] = [];

    const processCategory = (
      category: CategoryTreeItem,
      parentName: string = ""
    ) => {
      const item: any = {};

      // 根据选择的字段构建数据
      options.selectedFields.forEach((field) => {
        switch (field) {
          case "name":
            item.name = category.name;
            break;
          case "code":
            item.code = category.code;
            break;
          case "description":
            item.description = category.description || "";
            break;
          case "parent_id":
            item.parent_id = category.parent_id;
            break;
          case "parent_name":
            item.parent_name = parentName;
            break;
          case "level":
            item.level = category.level;
            break;
          case "sort_order":
            item.sort_order = category.sort_order;
            break;
          case "is_active":
            item.is_active = category.is_active ? "是" : "否";
            break;
          case "tenant_id":
            item.tenant_id = category.tenant_id;
            break;
          case "created_at":
            item.created_at = category.created_at;
            break;
          case "updated_at":
            item.updated_at = category.updated_at;
            break;
          case "created_by":
            item.created_by = category.created_by || "";
            break;
          case "updated_by":
            item.updated_by = category.updated_by || "";
            break;
        }
      });

      result.push(item);

      // 处理子分类
      if (category.children && category.children.length > 0) {
        category.children.forEach((child) => {
          processCategory(child, category.name);
        });
      }
    };

    // 处理所有顶级分类
    categories.forEach((category) => {
      if (options.includeInactive || category.is_active) {
        processCategory(category);
      }
    });

    return result;
  };

  // 执行导出
  const performExport = async (data: any[], options: ExportOptions) => {
    switch (options.format) {
      case "csv":
        exportToCSV(data, options);
        break;
      case "excel":
        exportToExcel(data, options);
        break;
      case "json":
        exportToJSON(data, options);
        break;
    }
  };

  // 导出为CSV
  const exportToCSV = (data: any[], options: ExportOptions) => {
    if (data.length === 0) return;

    const headers = options.selectedFields.map((field) => {
      const fieldInfo = availableFields.find((f) => f.key === field);
      return fieldInfo ? fieldInfo.label : field;
    });

    const csvContent = [
      headers.join(","),
      ...data.map((row) =>
        options.selectedFields
          .map((field) => {
            const value = row[field];
            // 处理包含逗号或引号的值
            if (
              typeof value === "string" &&
              (value.includes(",") || value.includes('"'))
            ) {
              return `"${value.replace(/"/g, '""')}"`;
            }
            return value;
          })
          .join(",")
      ),
    ].join("\n");

    const blob = new Blob([csvContent], {
      type: `text/csv;charset=${options.encoding === "gbk" ? "gbk" : "utf-8"}`,
    });
    downloadFile(
      blob,
      `工单分类_${new Date().toISOString().split("T")[0]}.csv`
    );
  };

  // 导出为Excel
  const exportToExcel = (data: any[], options: ExportOptions) => {
    // 这里应该使用库如 xlsx 来生成Excel文件
    // 暂时使用CSV格式，但文件扩展名为.xlsx
    exportToCSV(data, options);

    // 提示用户
    message.info("Excel导出功能需要安装xlsx库，当前使用CSV格式");
  };

  // 导出为JSON
  const exportToJSON = (data: any[], options: ExportOptions) => {
    const jsonContent = JSON.stringify(data, null, 2);
    const blob = new Blob([jsonContent], { type: "application/json" });
    downloadFile(
      blob,
      `工单分类_${new Date().toISOString().split("T")[0]}.json`
    );
  };

  // 下载文件
  const downloadFile = (blob: Blob, filename: string) => {
    const link = document.createElement("a");
    const url = URL.createObjectURL(blob);
    link.setAttribute("href", url);
    link.setAttribute("download", filename);
    link.style.visibility = "hidden";
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  // 重置表单
  const handleReset = () => {
    form.setFieldsValue(defaultOptions);
  };

  // 关闭模态框
  const handleCancel = () => {
    if (!exporting) {
      form.resetFields();
      onCancel();
    }
  };

  return (
    <Modal
      title="导出工单分类"
      open={visible}
      onCancel={handleCancel}
      footer={null}
      width={700}
      destroyOnHidden
    >
      <div className="space-y-6">
        {/* 导出选项表单 */}
        <Form
          form={form}
          layout="vertical"
          initialValues={defaultOptions}
          onFinish={handleExport}
        >
          {/* 基本选项 */}
          <Card size="small" title="基本选项">
            <div className="grid grid-cols-2 gap-4">
              <Form.Item
                name="format"
                label="导出格式"
                rules={[{ required: true, message: "请选择导出格式" }]}
              >
                <Select>
                  <Option value="excel">
                    <Space>
                      <FileExcel className="w-4 h-4 text-green-600" />
                      Excel (.xlsx)
                    </Space>
                  </Option>
                  <Option value="csv">
                    <Space>
                      <FileCsv className="w-4 h-4 text-blue-600" />
                      CSV (.csv)
                    </Space>
                  </Option>
                  <Option value="json">
                    <Space>
                      <FileText className="w-4 h-4 text-orange-600" />
                      JSON (.json)
                    </Space>
                  </Option>
                </Select>
              </Form.Item>

              <Form.Item name="encoding" label="文件编码">
                <Select>
                  <Option value="utf8">UTF-8</Option>
                  <Option value="gbk">GBK (中文)</Option>
                </Select>
              </Form.Item>
            </div>
          </Card>

          {/* 数据选项 */}
          <Card size="small" title="数据选项">
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <Form.Item
                  name="includeInactive"
                  label="包含禁用分类"
                  valuePropName="checked"
                >
                  <Switch />
                </Form.Item>

                <Form.Item
                  name="includeSystem"
                  label="包含系统分类"
                  valuePropName="checked"
                >
                  <Switch />
                </Form.Item>
              </div>

              <Form.Item
                name="includeMetadata"
                label="包含元数据"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>

              <Form.Item
                name="flattenStructure"
                label="扁平化结构"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
            </div>
          </Card>

          {/* 字段选择 */}
          <Card size="small" title="导出字段">
            <Form.Item
              name="selectedFields"
              rules={[{ required: true, message: "请选择至少一个字段" }]}
            >
              <CheckboxGroup className="grid grid-cols-2 gap-2">
                {availableFields.map((field) => (
                  <Checkbox
                    key={field.key}
                    value={field.key}
                    disabled={field.required}
                  >
                    <Space>
                      {field.label}
                      {field.required && <Text type="danger">*</Text>}
                    </Space>
                  </Checkbox>
                ))}
              </CheckboxGroup>
            </Form.Item>
          </Card>

          {/* 操作按钮 */}
          <div className="flex justify-between items-center">
            <Button onClick={handleReset} disabled={exporting}>
              重置选项
            </Button>

            <Space>
              <Button onClick={handleCancel} disabled={exporting}>
                取消
              </Button>
              <Button
                type="primary"
                htmlType="submit"
                loading={exporting}
                icon={<Download className="w-4 h-4" />}
              >
                {exporting ? "导出中..." : "开始导出"}
              </Button>
            </Space>
          </div>
        </Form>

        {/* 导出进度 */}
        {exporting && (
          <Card size="small" title="导出进度">
            <div className="space-y-4">
              <div className="flex items-center justify-center">
                <Spin size="large" />
              </div>
              <div className="text-center text-sm text-gray-500">
                正在导出数据，请稍候...
              </div>
            </div>
          </Card>
        )}
      </div>
    </Modal>
  );
};

export default TicketCategoryExport;
