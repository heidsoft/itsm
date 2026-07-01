'use client';

import React, { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import {
  AlertCircle,
  CheckCircle2,
  Clock,
  ExternalLink,
  PlusCircle,
  Save,
  Search,
  Settings,
  Trash2,
  XCircle,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { httpClient } from '@/lib/api/http-client';
import connectorService, { ConnectorConfig } from '@/lib/services/connector-service';
import marketplaceService, { TenantInstallation } from '@/lib/services/marketplace-service';

type InstallForm = {
  appId: string;
  appSecret: string;
  verificationToken: string;
  encryptKey: string;
  debugChannel: string;
  region: string;
};

const statusConfig = {
  active: {
    label: '运行中',
    color: 'bg-green-100 text-green-800',
    icon: <CheckCircle2 className="h-4 w-4 text-green-500" />,
  },
  disabled: {
    label: '已禁用',
    color: 'bg-gray-100 text-gray-800',
    icon: <XCircle className="h-4 w-4 text-gray-500" />,
  },
  failed: {
    label: '运行失败',
    color: 'bg-red-100 text-red-800',
    icon: <AlertCircle className="h-4 w-4 text-red-500" />,
  },
  installing: {
    label: '安装中',
    color: 'bg-blue-100 text-blue-800',
    icon: <Clock className="h-4 w-4 text-blue-500" />,
  },
  uninstalled: {
    label: '已卸载',
    color: 'bg-gray-100 text-gray-800',
    icon: <XCircle className="h-4 w-4 text-gray-500" />,
  },
};

const typeNames = {
  connector: '连接器',
  skill: '技能',
  plugin: '插件',
};

const connectorRuntimeName = (installation: TenantInstallation) =>
  installation.edges?.item?.name?.replace(/-connector$/, '') || '';

const maskSensitive = (key: string, value: unknown) => {
  if (value === undefined || value === null || value === '') return '未配置';
  if (/secret|token|key|access|refresh/i.test(key)) return '******';
  return String(value);
};

const makeForm = (installation: TenantInstallation): InstallForm => {
  const config = installation.config || {};
  const credentials = (config.credentials || {}) as Record<string, unknown>;
  const settings = (config.settings || {}) as Record<string, unknown>;
  return {
    appId: String(credentials.app_id || ''),
    appSecret: String(credentials.app_secret || ''),
    verificationToken: String(credentials.verification_token || ''),
    encryptKey: String(credentials.encrypt_key || ''),
    debugChannel: String(settings.debug_channel || ''),
    region: String(settings.region || 'cn'),
  };
};

const InstallationsPage = () => {
  const [installations, setInstallations] = useState<TenantInstallation[]>([]);
  const [connectorConfigs, setConnectorConfigs] = useState<ConnectorConfig[]>([]);
  const [loading, setLoading] = useState(true);
  const [savingId, setSavingId] = useState<number | null>(null);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [forms, setForms] = useState<Record<number, InstallForm>>({});
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [typeFilter, setTypeFilter] = useState('all');
  const [loadError, setLoadError] = useState<string | null>(null);

  const loadInstallations = async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const [installationRes, configRes] = await Promise.all([
        marketplaceService.listInstallations(),
        connectorService.configs().catch(() => []),
      ]);
      setInstallations(installationRes);
      setConnectorConfigs(configRes);
      setForms(Object.fromEntries(installationRes.map(item => [item.id, makeForm(item)])));
    } catch (error: any) {
      console.error('Failed to fetch installations:', error);
      setLoadError(error?.message || '加载已安装应用失败');
      setInstallations([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadInstallations();
  }, []);

  const connectorConfigByName = useMemo(() => {
    const map = new Map<string, ConnectorConfig>();
    connectorConfigs.forEach(cfg => map.set(cfg.name, cfg));
    return map;
  }, [connectorConfigs]);

  const filteredInstallations = installations.filter(installation => {
    const item = installation.edges?.item;
    const title = item?.title || item?.name || '';
    const description = item?.description || '';
    const matchesSearch =
      !search ||
      title.toLowerCase().includes(search.toLowerCase()) ||
      description.toLowerCase().includes(search.toLowerCase());
    const matchesStatus = statusFilter === 'all' || installation.status === statusFilter;
    const matchesType = typeFilter === 'all' || item?.type === typeFilter;
    return matchesSearch && matchesStatus && matchesType;
  });

  const handleUninstall = async (installation: TenantInstallation) => {
    const title = installation.edges?.item?.title || installation.edges?.item?.name || '应用';
    if (!confirm(`确定要卸载「${title}」吗？卸载后相关功能将无法使用。`)) return;
    try {
      await marketplaceService.uninstallItem(installation.itemId);
      setInstallations(items => items.filter(item => item.id !== installation.id));
    } catch (error) {
      console.error('Failed to uninstall item:', error);
      alert(error instanceof Error ? error.message : '卸载失败');
    }
  };

  const updateForm = (id: number, patch: Partial<InstallForm>) => {
    setForms(prev => ({
      ...prev,
      [id]: {
        ...prev[id],
        ...patch,
      },
    }));
  };

  const handleSaveConfig = async (installation: TenantInstallation) => {
    const form = forms[installation.id] || makeForm(installation);
    const currentConfig = installation.config || {};
    const nextConfig = {
      ...currentConfig,
      credentials: {
        app_id: form.appId.trim(),
        app_secret: form.appSecret.trim(),
        verification_token: form.verificationToken.trim(),
        encrypt_key: form.encryptKey.trim(),
      },
      settings: {
        ...((currentConfig.settings || {}) as Record<string, unknown>),
        debug_channel: form.debugChannel.trim(),
        region: form.region,
      },
    };
    setSavingId(installation.id);
    try {
      const updated = await marketplaceService.updateInstallationConfig(installation.itemId, nextConfig);
      setInstallations(items => items.map(item => (item.id === installation.id ? updated : item)));
      setForms(prev => ({ ...prev, [updated.id]: makeForm(updated) }));
      setEditingId(null);
      const configRes = await connectorService.configs().catch(() => []);
      setConnectorConfigs(configRes);
    } catch (error) {
      console.error('Failed to save installation config:', error);
      alert(error instanceof Error ? error.message : '保存配置失败');
    } finally {
      setSavingId(null);
    }
  };

  const handleFeishuOAuth = async () => {
    try {
      const response = await httpClient.get<{ authUrl: string }>('/api/v1/feishu/oauth/auth-url');
      if (response.authUrl) {
        window.location.href = response.authUrl;
      }
    } catch (error) {
      console.error('Failed to start Feishu OAuth:', error);
      alert(error instanceof Error ? error.message : '无法获取飞书授权链接，请先保存 App ID 和 App Secret');
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
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
          <Button>
            <PlusCircle className="h-4 w-4 mr-2" />
            安装应用
          </Button>
        </Link>
      </div>

      <div className="bg-white rounded-lg shadow-sm p-4 mb-6">
        <div className="flex flex-col md:flex-row gap-4">
          <div className="flex-1">
            <div className="relative">
              <Search className="absolute left-3 top-3 h-4 w-4 text-gray-400" />
              <Input
                placeholder="搜索应用名称或描述..."
                value={search}
                onChange={event => setSearch(event.target.value)}
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

      {loadError && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-sm text-red-700 flex items-start gap-2">
          <AlertCircle className="h-4 w-4 mt-0.5 flex-shrink-0" />
          <span>{loadError}</span>
        </div>
      )}

      <div className="grid grid-cols-1 gap-4">
        {filteredInstallations.map(installation => {
          const item = installation.edges?.item;
          const status = statusConfig[installation.status] || statusConfig.installing;
          const runtimeName = connectorRuntimeName(installation);
          const runtimeConfig = connectorConfigByName.get(runtimeName);
          const isFeishu = runtimeName === 'feishu';
          const isEditing = editingId === installation.id;
          const form = forms[installation.id] || makeForm(installation);
          return (
            <Card key={installation.id} className="hover:shadow-sm transition-shadow">
              <CardHeader className="pb-2">
                <div className="flex flex-col lg:flex-row lg:items-start justify-between gap-4">
                  <div className="flex items-center gap-3">
                    <div className="w-12 h-12 rounded-md overflow-hidden bg-gray-100 flex items-center justify-center">
                      {item?.iconUrl ? (
                        <img src={item.iconUrl} alt={item.title} className="w-full h-full object-cover" />
                      ) : (
                        <span className="text-sm">{item?.type ? typeNames[item.type] : '应用'}</span>
                      )}
                    </div>
                    <div>
                      <div className="flex flex-wrap items-center gap-3">
                        <CardTitle className="text-xl">{item?.title || item?.name || `应用 #${installation.itemId}`}</CardTitle>
                        {item?.type && <Badge variant="secondary">{typeNames[item.type]}</Badge>}
                        <Badge className={status.color}>
                          {status.icon}
                          <span className="ml-1">{status.label}</span>
                        </Badge>
                        {runtimeConfig && (
                          <Badge className={runtimeConfig.healthy ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'}>
                            {runtimeConfig.lifecycle || (runtimeConfig.healthy ? 'healthy' : 'configured')}
                          </Badge>
                        )}
                      </div>
                      <CardDescription className="mt-1">{item?.description || '暂无描述'}</CardDescription>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    {item?.type === 'connector' && (
                      <Button variant="secondary" size="sm" onClick={() => setEditingId(isEditing ? null : installation.id)}>
                        <Settings className="h-4 w-4 mr-1" />
                        配置
                      </Button>
                    )}
                    <Button variant="destructive" size="sm" onClick={() => handleUninstall(installation)}>
                      <Trash2 className="h-4 w-4 mr-1" />
                      卸载
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                {(installation.status === 'failed' || runtimeConfig?.lastError) && (
                  <div className="p-2 bg-red-50 border border-red-200 rounded text-sm text-red-700 flex items-start gap-2">
                    <AlertCircle className="h-4 w-4 mt-0.5 flex-shrink-0" />
                    <span>{installation.errorMessage || runtimeConfig?.lastError}</span>
                  </div>
                )}

                <div className="grid grid-cols-1 md:grid-cols-4 gap-4 text-sm">
                  <Info label="已安装版本" value={`v${installation.installedVersion}`} />
                  <Info label="安装时间" value={new Date(installation.installedAt).toLocaleString()} />
                  <Info label="上次更新" value={new Date(installation.updatedAt || installation.lastUpdatedAt || installation.installedAt).toLocaleString()} />
                  <Info label="最近使用" value={installation.lastUsedAt ? new Date(installation.lastUsedAt).toLocaleString() : '从未使用'} />
                </div>

                {item?.type === 'connector' && (
                  <div className="rounded-md border bg-gray-50 p-3">
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-3 text-sm">
                      <Info label="运行时名称" value={runtimeName || '未识别'} />
                      <Info label="启用状态" value={runtimeConfig?.enabled ? '已启用' : '未启用'} />
                      <Info label="健康状态" value={runtimeConfig ? (runtimeConfig.healthy ? '健康' : '待检查') : '未初始化'} />
                    </div>
                    <div className="mt-3 grid grid-cols-1 md:grid-cols-3 gap-3 text-sm">
                      {Object.entries((installation.config?.credentials || {}) as Record<string, unknown>).map(([key, value]) => (
                        <Info key={key} label={`凭据 ${key}`} value={maskSensitive(key, value)} />
                      ))}
                      {Object.entries((installation.config?.oauth || {}) as Record<string, unknown>).map(([key, value]) => (
                        <Info key={key} label={`OAuth ${key}`} value={maskSensitive(key, value)} />
                      ))}
                    </div>
                  </div>
                )}

                {isEditing && (
                  <div className="rounded-md border p-4 space-y-4">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <Field label="App ID" value={form.appId} onChange={value => updateForm(installation.id, { appId: value })} />
                      <Field label="App Secret" type="password" value={form.appSecret} onChange={value => updateForm(installation.id, { appSecret: value })} />
                      <Field label="Verification Token" type="password" value={form.verificationToken} onChange={value => updateForm(installation.id, { verificationToken: value })} />
                      <Field label="Encrypt Key" type="password" value={form.encryptKey} onChange={value => updateForm(installation.id, { encryptKey: value })} />
                      <Field label="Debug Channel" value={form.debugChannel} onChange={value => updateForm(installation.id, { debugChannel: value })} />
                      <div>
                        <div className="text-sm font-medium mb-1">区域</div>
                        <Select value={form.region} onValueChange={value => updateForm(installation.id, { region: value })}>
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="cn">中国大陆</SelectItem>
                            <SelectItem value="intl">海外 Lark</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <Button onClick={() => handleSaveConfig(installation)} disabled={savingId === installation.id}>
                        <Save className="h-4 w-4 mr-2" />
                        {savingId === installation.id ? '保存中...' : '保存配置'}
                      </Button>
                      {isFeishu && (
                        <Button variant="secondary" onClick={handleFeishuOAuth}>
                          <ExternalLink className="h-4 w-4 mr-2" />
                          飞书 OAuth 授权
                        </Button>
                      )}
                    </div>
                  </div>
                )}
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

const Info = ({ label, value }: { label: string; value: string }) => (
  <div>
    <div className="text-gray-500 mb-1">{label}</div>
    <div className="break-all">{value}</div>
  </div>
);

const Field = ({
  label,
  value,
  onChange,
  type = 'text',
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
}) => (
  <div>
    <div className="text-sm font-medium mb-1">{label}</div>
    <Input type={type} value={value} onChange={event => onChange(event.target.value)} />
  </div>
);

export default InstallationsPage;
