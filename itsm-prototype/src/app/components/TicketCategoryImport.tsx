"use client";

import React, { useState } from "react";
import {
  Modal,
  Button,
  Upload,
  message,
  Progress,
  Alert,
  Space,
  Typography,
  Card,
  Table,
  Tag,
  Tooltip,
} from "antd";
import {
  Download,
  FileSpreadsheet,
  FileText,
  CheckCircle,
  XCircle,
  Info,
  AlertTriangle,
  FileSpreadsheet as FileExcel,
} from "lucide-react";
import { UploadOutlined } from "@ant-design/icons";
import {
  ticketCategoryService,
  type CreateCategoryRequest,
} from "../lib/services/ticket-category-service";

const { Text, Title } = Typography;
const { Dragger } = Upload;

interface TicketCategoryImportProps {
  visible: boolean;
  onCancel: () => void;
  onSuccess: () => void;
}

interface ImportResult {
  success: number;
  failed: number;
  total: number;
  errors: string[];
  details: ImportDetail[];
}

interface ImportDetail {
  row: number;
  name: string;
  code: string;
  status: "success" | "failed";
  message: string;
}

const TicketCategoryImport: React.FC<TicketCategoryImportProps> = ({
  visible,
  onCancel,
  onSuccess,
}) => {
  const [fileList, setFileList] = useState<any[]>([]);
  const [uploading, setUploading] = useState(false);
  const [importProgress, setImportProgress] = useState(0);
  const [importResult, setImportResult] = useState<ImportResult | null>(null);
  const [previewData, setPreviewData] = useState<any[]>([]);

  // 处理文件上传
  const handleUpload = async (file: File) => {
    if (!file) return;

    const isExcel =
      file.type ===
        "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" ||
      file.type === "application/vnd.ms-excel";
    const isCsv = file.type === "text/csv" || file.name.endsWith(".csv");

    if (!isExcel && !isCsv) {
      message.error("只支持 Excel (.xlsx, .xls) 或 CSV 文件");
      return false;
    }

    if (file.size > 10 * 1024 * 1024) {
      message.error("文件大小不能超过 10MB");
      return false;
    }

    // 预览文件内容
    await previewFile(file);
    return false; // 阻止自动上传
  };

  // 预览文件内容
  const previewFile = async (file: File) => {
    try {
      const formData = new FormData();
      formData.append("file", file);

      // 这里应该调用后端的预览API
      // const response = await ticketCategoryService.previewImport(formData);
      // setPreviewData(response.data);

      // 模拟预览数据
      const mockData = [
        {
          name: "硬件问题",
          code: "HW001",
          description: "硬件相关的问题",
          parent_code: "",
          sort_order: 1,
          is_active: true,
        },
        {
          name: "软件问题",
          code: "SW001",
          description: "软件相关的问题",
          parent_code: "",
          sort_order: 2,
          is_active: true,
        },
        {
          name: "网络问题",
          code: "NET001",
          description: "网络相关的问题",
          parent_code: "",
          sort_order: 3,
          is_active: true,
        },
        {
          name: "系统问题",
          code: "SYS001",
          description: "系统相关的问题",
          parent_code: "",
          sort_order: 4,
          is_active: true,
        },
      ];
      setPreviewData(mockData);
    } catch (error) {
      message.error("文件预览失败");
    }
  };

  // 开始导入
  const handleImport = async () => {
    if (fileList.length === 0) {
      message.warning("请先选择要导入的文件");
      return;
    }

    try {
      setUploading(true);
      setImportProgress(0);
      setImportResult(null);

      const file = fileList[0].originFileObj;
      const formData = new FormData();
      formData.append("file", file);

      // 模拟导入进度
      const progressInterval = setInterval(() => {
        setImportProgress((prev) => {
          if (prev >= 90) {
            clearInterval(progressInterval);
            return 90;
          }
          return prev + 10;
        });
      }, 200);

      // 调用导入API
      // const response = await ticketCategoryService.importCategories(formData);

      // 模拟导入结果
      await new Promise((resolve) => setTimeout(resolve, 2000));
      clearInterval(progressInterval);
      setImportProgress(100);

      const mockResult: ImportResult = {
        success: 3,
        failed: 1,
        total: 4,
        errors: ["第4行: 分类代码已存在"],
        details: [
          {
            row: 1,
            name: "硬件问题",
            code: "HW001",
            status: "success",
            message: "导入成功",
          },
          {
            row: 2,
            name: "软件问题",
            code: "SW001",
            status: "success",
            message: "导入成功",
          },
          {
            row: 3,
            name: "网络问题",
            code: "NET001",
            status: "success",
            message: "导入成功",
          },
          {
            row: 4,
            name: "系统问题",
            code: "SYS001",
            status: "failed",
            message: "分类代码已存在",
          },
        ],
      };

      setImportResult(mockResult);
      message.success("导入完成");
    } catch (error) {
      message.error(
        "导入失败: " + (error instanceof Error ? error.message : "未知错误")
      );
    } finally {
      setUploading(false);
    }
  };

  // 下载模板
  const downloadTemplate = () => {
    // 创建CSV模板内容
    const template = [
      ["name", "code", "description", "parent_code", "sort_order", "is_active"],
      ["硬件问题", "HW001", "硬件相关的问题", "", "1", "true"],
      ["软件问题", "SW001", "软件相关的问题", "", "2", "true"],
      ["网络问题", "NET001", "网络相关的问题", "", "3", "true"],
    ]
      .map((row) => row.join(","))
      .join("\n");

    const blob = new Blob([template], { type: "text/csv;charset=utf-8;" });
    const link = document.createElement("a");
    const url = URL.createObjectURL(blob);
    link.setAttribute("href", url);
    link.setAttribute("download", "工单分类导入模板.csv");
    link.style.visibility = "hidden";
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  // 下载Excel模板
  const downloadExcelTemplate = () => {
    // 这里应该调用后端API生成Excel模板
    message.info("Excel模板下载功能需要后端支持");
  };

  // 重置状态
  const handleCancel = () => {
    setFileList([]);
    setUploading(false);
    setImportProgress(0);
    setImportResult(null);
    setPreviewData([]);
    onCancel();
  };

  // 完成导入
  const handleFinish = () => {
    if (importResult && importResult.success > 0) {
      onSuccess();
    }
    handleCancel();
  };

  const uploadProps = {
    fileList,
    beforeUpload: handleUpload,
    onChange: ({ fileList }: any) => setFileList(fileList),
    accept: ".xlsx,.xls,.csv",
    multiple: false,
  };

  return (
    <Modal
      title="批量导入工单分类"
      open={visible}
      onCancel={handleCancel}
      footer={null}
      width={800}
      destroyOnClose
    >
      <div className="space-y-6">
        {/* 模板下载 */}
        <Card size="small" title="下载导入模板">
          <Space>
            <Button
              icon={<Download className="w-4 h-4" />}
              onClick={downloadTemplate}
            >
              下载CSV模板
            </Button>
            <Button
              icon={<FileExcel className="w-4 h-4" />}
              onClick={downloadExcelTemplate}
            >
              下载Excel模板
            </Button>
          </Space>
          <div className="mt-2 text-sm text-gray-500">
            请按照模板格式填写数据，支持CSV和Excel文件
          </div>
        </Card>

        {/* 文件上传 */}
        <Card size="small" title="选择文件">
          <Dragger {...uploadProps}>
            <p className="ant-upload-drag-icon">
              <UploadOutlined className="text-4xl text-blue-500" />
            </p>
            <p className="ant-upload-text">点击或拖拽文件到此区域上传</p>
            <p className="ant-upload-hint">
              支持 .xlsx, .xls, .csv 格式，文件大小不超过 10MB
            </p>
          </Dragger>
        </Card>

        {/* 文件预览 */}
        {previewData.length > 0 && (
          <Card size="small" title="数据预览">
            <Table
              dataSource={previewData}
              columns={[
                { title: "分类名称", dataIndex: "name", key: "name" },
                { title: "分类代码", dataIndex: "code", key: "code" },
                { title: "描述", dataIndex: "description", key: "description" },
                {
                  title: "父级代码",
                  dataIndex: "parent_code",
                  key: "parent_code",
                },
                { title: "排序", dataIndex: "sort_order", key: "sort_order" },
                {
                  title: "状态",
                  dataIndex: "is_active",
                  key: "is_active",
                  render: (value: boolean) => (
                    <Tag color={value ? "green" : "red"}>
                      {value ? "启用" : "禁用"}
                    </Tag>
                  ),
                },
              ]}
              size="small"
              pagination={false}
              scroll={{ x: 600 }}
            />
          </Card>
        )}

        {/* 导入进度 */}
        {uploading && (
          <Card size="small" title="导入进度">
            <Progress percent={importProgress} status="active" />
            <div className="text-center text-sm text-gray-500 mt-2">
              正在导入数据，请稍候...
            </div>
          </Card>
        )}

        {/* 导入结果 */}
        {importResult && (
          <Card size="small" title="导入结果">
            <div className="space-y-4">
              {/* 统计信息 */}
              <div className="flex items-center justify-center space-x-8">
                <div className="text-center">
                  <div className="text-2xl font-bold text-green-600">
                    {importResult.success}
                  </div>
                  <div className="text-sm text-gray-500">成功</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-red-600">
                    {importResult.failed}
                  </div>
                  <div className="text-sm text-gray-500">失败</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-blue-600">
                    {importResult.total}
                  </div>
                  <div className="text-sm text-gray-500">总计</div>
                </div>
              </div>

              {/* 错误信息 */}
              {importResult.errors.length > 0 && (
                <Alert
                  message="导入错误"
                  description={
                    <ul className="list-disc list-inside">
                      {importResult.errors.map((error, index) => (
                        <li key={index}>{error}</li>
                      ))}
                    </ul>
                  }
                  type="error"
                  showIcon
                />
              )}

              {/* 详细结果 */}
              <Table
                dataSource={importResult.details}
                columns={[
                  { title: "行号", dataIndex: "row", key: "row", width: 80 },
                  { title: "分类名称", dataIndex: "name", key: "name" },
                  { title: "分类代码", dataIndex: "code", key: "code" },
                  {
                    title: "状态",
                    dataIndex: "status",
                    key: "status",
                    render: (status: string) => (
                      <Tag
                        color={status === "success" ? "green" : "red"}
                        icon={
                          status === "success" ? <CheckCircle /> : <XCircle />
                        }
                      >
                        {status === "success" ? "成功" : "失败"}
                      </Tag>
                    ),
                  },
                  { title: "消息", dataIndex: "message", key: "message" },
                ]}
                size="small"
                pagination={false}
                rowKey="row"
              />
            </div>
          </Card>
        )}

        {/* 操作按钮 */}
        <div className="flex justify-end space-x-2">
          <Button onClick={handleCancel}>取消</Button>
          {fileList.length > 0 && !uploading && !importResult && (
            <Button type="primary" onClick={handleImport}>
              开始导入
            </Button>
          )}
          {importResult && (
            <Button type="primary" onClick={handleFinish}>
              完成
            </Button>
          )}
        </div>
      </div>
    </Modal>
  );
};

export default TicketCategoryImport;
