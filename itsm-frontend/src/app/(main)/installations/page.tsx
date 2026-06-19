'use client';

import React, { useState, useEffect } from 'react';
import { 
  Card, 
  CardContent, 
  CardDescription, 
  CardFooter, 
  CardHeader, 
  CardTitle 
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { 
  Settings, 
  Trash2, 
  AlertCircle, 
  CheckCircle2, 
  Clock, 
  XCircle,
  PlusCircle,
  Search
} from 'lucide-react';
import Link from 'next/link';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

// 已安装应用类型
interface Installation {
  id: number;
  item_id: number;
  name: string;
  title: string;
  type: 'connector' | 'skill' | 'plugin';
  icon_url: string;
  description: string;
  installed_version: string;
  status: 'active' | 'disabled' | 'failed' | 'installing';
  installed_at: string;
  updated_at: string;
  last_used_at?: string;
  error_message?: string;
  config: any;
}

const InstallationsPage = () => {
  const [installations, setInstallations] = useState<Installation[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [typeFilter, setTypeFilter] = useState('all');

  // 模拟获取已安装应用
  useEffect(() => {
    const fetchInstallations = async () => {
      try {
        // 替换为真实API调用
        // const res = await fetch('/api/v1/marketplace/installations');
        // const data = await res.json();
        // setInstallations(data.data);

        // 模拟数据
        const mockInstallations: Installation[] = [
          {
            id: 1,
            item_id: 1,
            name: 'feishu-connector',
            title: '飞书连接器',
            type: 'connector',
            icon_url: 'https://cdn-icons-png.flaticon.com/512/5968/5968757.png',
            description: '集成飞书消息、审批、组织架构同步',
            installed_version: '1.2.0',
            status: 'active',
            installed_at: '2024-03-15T10:30:00Z',
            updated_at: '2024-05-20T14:20:00Z',
            last_used_at: '2024-06-18T09:15:00Z',
            config: {
              app_id: 'cli_xxxxxx',
              app_secret: '******',
              verification_token: '******'
            }
          },
          {
            id: 2,
            item_id: 4,
            name: 'ticket-classification-skill',
            title: '工单智能分类',
            type: 'skill',
            icon_url: 'https://cdn-icons-png.flaticon.com/512/4712/4712109.png',
            description: '自动识别工单类型、优先级，智能分派到对应团队',
            installed_version: '2.0.0',
            status: 'active',
            installed_at: '2024-02-20T09:15:00Z',
            updated_at: '2024-04-10T16:40:00Z',
            last_used_at: '2024-06-18T10:30:00Z',
            config: {
              confidence_threshold: 0.7,
              auto_assign: true
            }
          },
          {
            id: 3,
            item_id: 7,
            name: 'prometheus-connector',
            title: 'Prometheus连接器',
            type: 'connector',
            icon_url: 'https://cdn-icons-png.flaticon.com/512/9336/9336119.png',
            description: '集成Prometheus告警，自动生成事件工单',
            installed_version: '1.0.0',
            status: 'failed',
            installed_at: '2024-04-05T14:20:00Z',
            updated_at: '2024-04-05T14:20:00Z',
            error_message: '连接Prometheus服务器超时，请检查地址和认证信息',
            config: {
              api_url: 'http://prometheus:9090',
              auth_token: '******'
            }
          },
          {
            id: 4,
            item_id: 5,
            name: 'ticket-summary-skill',
            title: '工单摘要生成',
            type: 'skill',
            icon_url: 'https://cdn-icons-png.flaticon.com/512/1005/1005141.png',
            description: '自动生成工单摘要，提取关键信息，提高处理效率',
            installed_version: '1.1.0',
            status: 'disabled',
            installed_at: '2024-01-10T11:45:00Z',
            updated_at: '2024-03-01T09:30:00Z',
            config: {}
          }
        ];

        setInstallations(mockInstallations);
      } catch (error) {
        console.error('Failed to fetch installations:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchInstallations();
  }, []);

  // 过滤已安装应用
  const filteredInstallations = installations.filter(installation => {
    const matchesSearch = !search || 
      installation.title.toLowerCase().includes(search.toLowerCase()) ||
      installation.description.toLowerCase().includes(search.toLowerCase());

    const matchesStatus = statusFilter === 'all' || installation.status === statusFilter;

    const matchesType = typeFilter === 'all' || installation.type === typeFilter;

    return matchesSearch && matchesStatus && matchesType;
  });

  // 卸载应用
  const handleUninstall = async (id: number, title: string) => {
    if (confirm(`确定要卸载「${title}」吗？卸载后相关功能将无法使用。`)) {
      try {
        // 替换为真实的卸载API调用
        // await fetch(`/api/v1/marketplace/items/${id}/uninstall`, {
        //   method: 'POST'
        // });
        
        setInstallations(installations.filter(item => item.id !== id));
      } catch (error) {
        console.error('Failed to uninstall item:', error);
      }
    }
  };

  // 状态配置
  const statusConfig = {
    active: {
      label: '运行中',
      color: 'bg-green-100 text-green-800',
      icon: <CheckCircle2 className="h-4 w-4 text-green-500" />
    },
    disabled: {
      label: '已禁用',
      color: 'bg-gray-100 text-gray-800',
      icon: <XCircle className="h-4 w-4 text-gray-500" />
    },
    failed: {
      label: '运行失败',
      color: 'bg-red-100 text-red-800',
      icon: <AlertCircle className="h-4 w-4 text-red-500" />
    },
    installing: {
      label: '安装中',
      color: 'bg-blue-100 text-blue-800',
      icon: <Clock className="h-4 w-4 text-blue-500" />
    }
  };

  // 类型名称映射
  const typeNames = {
    connector: '连接器',
    skill: '技能',
    plugin: '插件'
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  return (
    <div className="container mx-auto px-4 py-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">我的应用</h1>
          <p className="text-gray-500 mt-1">管理已安装的连接器、AI技能和扩展插件</p>
        </div>
        <Link href="/marketplace">
          <Button variant="default">
            <PlusCircle className="h-4 w-4 mr-2" />
            安装应用
          </Button>
        </Link>
      </div>

      {/* 搜索和过滤区 */}
      <div className="bg-white rounded-lg shadow-sm p-4 mb-6">
        <div className="flex flex-col md:flex-row gap-4">
          <div className="flex-1">
            <div className="relative">
              <Search className="absolute left-3 top-3 h-4 w-4 text-gray-400" />
              <Input
                placeholder="搜索应用名称或描述..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-10"
              />
            </div>
          </div>
          <div className="w-full md:w-40">
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger>
                <SelectValue placeholder="状态" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部状态</SelectItem>
                <SelectItem value="active">运行中</SelectItem>
                <SelectItem value="disabled">已禁用</SelectItem>
                <SelectItem value="failed">运行失败</SelectItem>
                <SelectItem value="installing">安装中</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="w-full md:w-40">
            <Select value={typeFilter} onValueChange={setTypeFilter}>
              <SelectTrigger>
                <SelectValue placeholder="类型" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部类型</SelectItem>
                <SelectItem value="connector">连接器</SelectItem>
                <SelectItem value="skill">技能</SelectItem>
                <SelectItem value="plugin">插件</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      </div>

      {/* 应用列表 */}
      <div className="grid grid-cols-1 gap-4">
        {filteredInstallations.map(installation => {
          const status = statusConfig[installation.status];
          return (
            <Card key={installation.id} className="hover:shadow-sm transition-shadow">
              <CardHeader className="pb-2">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className="w-12 h-12 rounded-md overflow-hidden bg-gray-100 flex items-center justify-center">
                      {installation.icon_url ? (
                        <img src={installation.icon_url} alt={installation.title} className="w-full h-full object-cover" />
                      ) : (
                        <div className="text-lg">{typeNames[installation.type]}</div>
                      )}
                    </div>
                    <div>
                      <div className="flex items-center gap-3">
                        <CardTitle className="text-xl">{installation.title}</CardTitle>
                        <Badge variant="secondary">{typeNames[installation.type]}</Badge>
                        <Badge className={status.color}>
                          {status.icon}
                          <span className="ml-1">{status.label}</span>
                        </Badge>
                      </div>
                      <CardDescription className="mt-1">
                        {installation.description}
                      </CardDescription>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Button 
                      variant="secondary" 
                      size="sm"
                      // onClick={() => handleConfigure(installation)}
                    >
                      <Settings className="h-4 w-4 mr-1" />
                      配置
                    </Button>
                    <Button 
                      variant="destructive" 
                      size="sm"
                      onClick={() => handleUninstall(installation.id, installation.title)}
                    >
                      <Trash2 className="h-4 w-4 mr-1" />
                      卸载
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="pb-2">
                {installation.status === 'failed' && installation.error_message && (
                  <div className="mb-3 p-2 bg-red-50 border border-red-200 rounded text-sm text-red-700 flex items-start gap-2">
                    <AlertCircle className="h-4 w-4 mt-0.5 flex-shrink-0" />
                    <span>{installation.error_message}</span>
                  </div>
                )}

                <div className="grid grid-cols-1 md:grid-cols-4 gap-4 text-sm">
                  <div>
                    <div className="text-gray-500 mb-1">已安装版本</div>
                    <div>v{installation.installed_version}</div>
                  </div>
                  <div>
                    <div className="text-gray-500 mb-1">安装时间</div>
                    <div>{new Date(installation.installed_at).toLocaleString()}</div>
                  </div>
                  <div>
                    <div className="text-gray-500 mb-1">上次更新</div>
                    <div>{new Date(installation.updated_at).toLocaleString()}</div>
                  </div>
                  <div>
                    <div className="text-gray-500 mb-1">最近使用</div>
                    <div>
                      {installation.last_used_at 
                        ? new Date(installation.last_used_at).toLocaleString() 
                        : '从未使用'}
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>

      {filteredInstallations.length === 0 && (
        <div className="bg-white rounded-lg shadow-sm p-10 text-center">
          <div className="flex flex-col items-center justify-center">
            <Search className="h-12 w-12 text-gray-300 mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">没有找到匹配的应用</h3>
            <p className="text-gray-500 mb-4">尝试调整搜索条件或过滤选项</p>
            <Link href="/marketplace">
              <Button>浏览应用市场</Button>
            </Link>
          </div>
        </div>
      )}
    </div>
  );
};

export default InstallationsPage;
