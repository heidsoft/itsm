'use client';

import React, { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { App, Modal } from 'antd';
import {
  AlertCircle,
  ArrowUpCircle,
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
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card';
import { Input } from '@/components/ui/Input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/Select';
import { httpClient, serverErrorMessage } from '@/lib/api/http-client';
import { listItemVersions } from '@/lib/api/marketplace-catalog-api';
import type { CatalogItemVersion } from '@/lib/api/marketplace-catalog-api';
import type { ConnectorConfig } from '@/lib/services/connector-service';
import connectorService from '@/lib/services/connector-service';
import type { MarketplaceDependencyReport, TenantInstallation } from '@/lib/services/marketplace-service';
import marketplaceService from '@/lib/services/marketplace-service';
import { useCapabilities } from '@/lib/hooks/useCapabilities';
import { usePermissions } from '@/lib/hooks/use-permissions';
import { formatDateTime } from '@/lib/formatters';

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
    label: 'Running',
    color: 'bg-green-100 text-green-800',
    icon: <CheckCircle2 className="h-4 w-4 text-green-500" />,
  },
  disabled: {
    label: 'Deactivated',
    color: 'bg-gray-100 text-gray-800',
    icon: <XCircle className="h-4 w-4 text-gray-500" />,
  },
  failed: {
    label: 'Run failed',
    color: 'bg-red-100 text-red-800',
    icon: <AlertCircle className="h-4 w-4 text-red-500" />,
  },
  installing: {
    label: 'Installing',
    color: 'bg-blue-100 text-blue-800',
    icon: <Clock className="h-4 w-4 text-blue-500" />,
  },
  uninstalled: {
    label: 'Uninstalled',
    color: 'bg-gray-100 text-gray-800',
    icon: <XCircle className="h-4 w-4 text-gray-500" />,
  },
};

const typeNames = {
  connector: 'Connectors',
  skill: 'Skill',
  plugin: 'Plug-in',
};

// 商品展示字段由安装列表接口按商品边扁平带出（itemName/itemTitle/...），
// 界面不再读 installation.edges.item —— DTO 里从来没有 edges，读它只会让每张卡片
// 退化成「应用 #id」并让类型过滤和搜索静默失效。
const connectorRuntimeName = (installation: TenantInstallation) =>
  installation.itemName?.replace(/-connector$/, '') || '';

const dependencyStatusLabel: Record<string, string> = {
  ok: 'Dependencies are aligned',
  unsatisfied: 'Prerequisite not satisfied',
  declaration_missing: 'Directory missing version statement',
};

const maskSensitive = (key: string, value: unknown) => {
  if (value === undefined || value === null || value === '') return 'Not configured';
  if (/secret|token|key|access|refresh/i.test(key)) return '******';
  return String(value);
};

const makeForm = (installation: TenantInstallation): InstallForm => {
  const config = installation.config || {};
  const credentials = (config.credentials || {}) as Record<string, unknown>;
  const settings = (config.settings || {}) as Record<string, unknown>;
  return {
    appId: String(credentials.appId || ''),
    appSecret: String(credentials.appSecret || ''),
    verificationToken: String(credentials.verificationToken || ''),
    encryptKey: String(credentials.encryptKey || ''),
    debugChannel: String(settings.debugChannel || ''),
    region: String(settings.region || 'cn'),
  };
};

const InstallationsPage = () => {
  const [installations, setInstallations] = useState<TenantInstallation[]>([]);
  const [connectorConfigs, setConnectorConfigs] = useState<ConnectorConfig[]>([]);
  const [loading, setLoading] = useState(true);
  const [savingId, setSavingId] = useState<number | null>(null);
  const [upgradingId, setUpgradingId] = useState<number | null>(null);
  const [autoUpgradeId, setAutoUpgradeId] = useState<number | null>(null);
  const [dependencyReport, setDependencyReport] = useState<MarketplaceDependencyReport | null>(null);
  const [dependencyError, setDependencyError] = useState<string | null>(null);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [forms, setForms] = useState<Record<number, InstallForm>>({});
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [typeFilter, setTypeFilter] = useState('all');
  const [loadError, setLoadError] = useState<string | null>(null);

  // 升级弹窗状态。
  //
  // 2026-11-04 第一轮修复：给出可选已发布版本的弹窗 + 二次确认，让「指名旧版本回滚」
  // 有界面入口（后端 upgrade 端点本就支持 targetVersion），换版本会重新装载运行态。
  // 同日第二轮实测：按钮门控曾写成 `upgradeAvailable || upgradeVersions.length > 0`，
  // 而版本清单只在这个按钮的 openUpgrade() 里才拉取 —— 已安装=最新版时按钮永不出现，
  // 回滚入口恰好在最需要它的时候消失；且 upgradeVersions 是全局态，打开过 A 的弹窗
  // 会让 B 也冒出按钮。现改为入口恒常渲染、版本清单按商品独立加载，
  // 由弹窗自己判断「有没有可切换的目标」，没有就明说并禁用确认。
  const [upgradeTarget, setUpgradeTarget] = useState<TenantInstallation | null>(null);
  const [upgradeVersions, setUpgradeVersions] = useState<CatalogItemVersion[]>([]);
  const [upgradeVersion, setUpgradeVersion] = useState<string>('');
  const [upgradeVersionsLoading, setUpgradeVersionsLoading] = useState(false);
  const [upgradeVersionsError, setUpgradeVersionsError] = useState<string | null>(null);
  const { message, modal } = App.useApp();
  const { allows } = useCapabilities();
  const { hasPermission } = usePermissions();
  const canManageMarketplace = allows('marketplace', 'manage') && hasPermission('marketplace', 'write');

  const loadInstallations = async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const [installationRes, configRes, dependencyRes] = await Promise.all([
        marketplaceService.listInstallations(),
        connectorService.configs().catch(() => []),
        // 依赖对齐是独立读面：它失败时不能说「依赖健康」，要说查不到。
        marketplaceService.dependencyStatus().catch((error: unknown) => {
          setDependencyError(error instanceof Error ? error.message : 'Dependency alignment status query failed');
          return null;
        }),
      ]);
      setInstallations(installationRes);
      setConnectorConfigs(configRes);
      setDependencyReport(dependencyRes);
      if (dependencyRes) setDependencyError(null);
      setForms(Object.fromEntries(installationRes.map(item => [item.id, makeForm(item)])));
    } catch (error: unknown) {
      console.error('Failed to fetch installations:', error);
      setLoadError(error instanceof Error ? error.message : 'Failed to load installed application');
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
    const title = installation.itemTitle || installation.itemName || '';
    const description = installation.itemDescription || '';
    const matchesSearch =
      !search ||
      title.toLowerCase().includes(search.toLowerCase()) ||
      description.toLowerCase().includes(search.toLowerCase());
    const matchesStatus = statusFilter === 'all' || installation.status === statusFilter;
    const matchesType = typeFilter === 'all' || installation.itemType === typeFilter;
    return matchesSearch && matchesStatus && matchesType;
  });

  // 可切换目标 = 已发布且不是当前已安装版本。弹窗据此判断「有没有得换」，
  // 而不是拿「版本清单是否非空」当入口门控（那会构成循环依赖）。
  const switchableVersions = upgradeVersions.filter(
    v => v.version !== upgradeTarget?.installedVersion,
  );

  const handleUninstall = async (installation: TenantInstallation) => {
    const title = installation.itemTitle || installation.itemName || 'Application';
    if (!confirm(`Confirm to uninstall "${title}"?" Relevant functions will not be available after uninstallation.`)) return;
    try {
      await marketplaceService.uninstallItem(installation.itemId);
      setInstallations(items => items.filter(item => item.id !== installation.id));
      await refreshDependencyStatus();
    } catch (error) {
      console.error('Failed to uninstall item:', error);
      // 卸载被拒时后端会说明原因（例如「连接器运行实例未能注销，卸载已中止」），
      // 折叠成「卸载失败」会让人以为网络问题而反复重试。
      message.error(serverErrorMessage(error, 'Uninstall failed'));
    }
  };

  const updateInstallationRow = (updated: TenantInstallation) => {
    setInstallations(items => items.map(item => (item.id === updated.id ? updated : item)));
  };

  // 换版本会重新执行依赖闸门，因此升级/卸载之后要重新取一次对齐报告；
  // 失败时保留旧报告并把原因显示出来，不能显示成「依赖已对齐」。
  const refreshDependencyStatus = async () => {
    try {
      setDependencyReport(await marketplaceService.dependencyStatus());
      setDependencyError(null);
    } catch (error: unknown) {
      setDependencyError(error instanceof Error ? error.message : 'Dependency alignment status query failed');
    }
  };

  // 打开升级弹窗：按当前商品拉取已发布版本（可指名旧版本回滚）。
  // 每次打开都先清空上一件商品的清单与报错，避免跨卡片串扰。
  const openUpgrade = async (installation: TenantInstallation) => {
    const initialTarget = installation.latestVersion ?? '';
    setUpgradeTarget(installation);
    setUpgradeVersions([]);
    setUpgradeVersionsError(null);
    setUpgradeVersion(initialTarget);
    setUpgradeVersionsLoading(true);
    try {
      const versions = (await listItemVersions(installation.itemId)).filter(v => v.status === 'published');
      setUpgradeVersions(versions);
      // 已安装=最新版时 initialTarget 就是当前版本，照它执行等于原地重装一遍。
      // 自动改选一个确实不同的已发布版本，让「刚升完想退一版」一步可达。
      const candidates = versions.filter(v => v.version !== installation.installedVersion);
      if (initialTarget === installation.installedVersion || initialTarget === '') {
        setUpgradeVersion(
          candidates.find(v => v.version === installation.latestVersion)?.version ?? candidates[0]?.version ?? '',
        );
      }
    } catch {
      // 清单拉不到但目录确实有更新时，仍允许升到 latestVersion；
      // 无更新（已最新）则必须禁用确认，不能静默按当前版本重跑一次升级。
      setUpgradeVersionsError('Failed to load the version list. Currently, you can only upgrade to the latest version of the directory.');
      if (initialTarget === installation.installedVersion || initialTarget === '') {
        setUpgradeVersion('');
      }
    } finally {
      setUpgradeVersionsLoading(false);
    }
  };

  const handleUpgrade = async (installation: TenantInstallation, targetVersion?: string) => {
    const target = targetVersion ?? installation.latestVersion ?? '';
    setUpgradingId(installation.itemId);
    try {
      updateInstallationRow(await marketplaceService.upgradeItem(installation.itemId, targetVersion));
      // 目标版本可能比当前更旧（回滚），统一说「切换到」，不要报成「已升级到」。
      message.success(`Version switched to v ${target}`);
      setUpgradeTarget(null);
      await refreshDependencyStatus();
    } catch (error) {
      console.error('Failed to upgrade item:', error);
      // 409 会说「升级未完成，仍在使用当前版本」：这条必须原样展示，
      // 否则管理员以为已经换了版本，而运行态仍停在旧发布物上。
      message.error(serverErrorMessage(error, 'Escalation failed'));
    } finally {
      setUpgradingId(null);
    }
  };

  const handleToggleAutoUpgrade = async (installation: TenantInstallation) => {
    const next = !installation.autoUpgrade;
    setAutoUpgradeId(installation.itemId);
    try {
      updateInstallationRow(await marketplaceService.setAutoUpgrade(installation.itemId, next));
      message.success(next ? 'Automatic upgrade is enabled' : 'Automatic upgrade is turned off');
    } catch (error) {
      console.error('Failed to update auto upgrade preference:', error);
      message.error(serverErrorMessage(error, next ? 'Failed to enable automatic upgrade' : 'Failed to turn off automatic upgrade'));
    } finally {
      setAutoUpgradeId(null);
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
        appId: form.appId.trim(),
        appSecret: form.appSecret.trim(),
        verificationToken: form.verificationToken.trim(),
        encryptKey: form.encryptKey.trim(),
      },
      settings: {
        ...((currentConfig.settings || {}) as Record<string, unknown>),
        debugChannel: form.debugChannel.trim(),
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
      // 后端在「新配置没能让连接器起来」时返回 409 并说明原因，且明确告知
      // 已保留原配置——这条信息必须原样展示，否则用户会以为保存成功了。
      message.error(serverErrorMessage(error, 'Failed to save configuration'));
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
      message.error(error instanceof Error ? error.message : 'Unable to obtain Feishu authorization link, please save App ID and App Secret first');
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
          <h1 className="text-3xl font-bold tracking-tight">My application</h1>
          <p className="text-gray-500 mt-1">Manage installed connectors, AI skills and extensions</p>
        </div>
        {canManageMarketplace && <Link href="/marketplace">
          <Button>
            <PlusCircle className="h-4 w-4 mr-2" />

            Install application
          </Button>
        </Link>}
      </div>

      <div className="bg-white rounded-lg shadow-sm p-4 mb-6">
        <div className="flex flex-col md:flex-row gap-4">
          <div className="flex-1">
            <div className="relative">
              <Search className="absolute left-3 top-3 h-4 w-4 text-gray-400" />
              <Input
                placeholder="Search app name or description..."
                value={search}
                onChange={event => setSearch(event.target.value)}
                className="pl-10"
              />
            </div>
          </div>
          <div className="w-full md:w-40">
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger>
                <SelectValue placeholder="Status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Status</SelectItem>
                <SelectItem value="active">Running</SelectItem>
                <SelectItem value="disabled">Deactivated</SelectItem>
                <SelectItem value="failed">Run failed</SelectItem>
                <SelectItem value="installing">Installing</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="w-full md:w-40">
            <Select value={typeFilter} onValueChange={setTypeFilter}>
              <SelectTrigger>
                <SelectValue placeholder="Type" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Types</SelectItem>
                <SelectItem value="connector">Connectors</SelectItem>
                <SelectItem value="skill">Skill</SelectItem>
                <SelectItem value="plugin">Plug-in</SelectItem>
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

      {dependencyError && (
        <div className="mb-4 p-3 bg-yellow-50 border border-yellow-200 rounded text-sm text-yellow-800 flex items-start gap-2">
          <AlertCircle className="h-4 w-4 mt-0.5 flex-shrink-0" />
          <span>The dependency alignment status is temporarily unavailable:{dependencyError}. The list below is not intended to be used as a guide to health.</span>
        </div>
      )}

      {dependencyReport && dependencyReport.unsatisfied + dependencyReport.declarationMissing > 0 && (
        <div className="mb-4 p-3 bg-yellow-50 border border-yellow-200 rounded text-sm">
          <div className="flex items-start gap-2">
            <AlertCircle className="h-4 w-4 mt-0.5 flex-shrink-0 text-yellow-700" />
            <div>
              <div className="font-medium text-yellow-800">
                Installed components: {dependencyReport.total}; unmet prerequisites: {dependencyReport.unsatisfied};
                missing version declarations: {dependencyReport.declarationMissing}
              </div>
              <ul className="mt-2 space-y-1 text-yellow-900">
                {dependencyReport.findings
                  .filter(finding => finding.status !== 'ok')
                  .map(finding => (
                    <li key={finding.itemId}>
                      <span className="font-medium">{finding.itemTitle || finding.itemName}</span>
                      {`（${dependencyStatusLabel[finding.status] ?? finding.status}${finding.installedVersion ? ` · v${finding.installedVersion}` : ''}）`}
                      {(finding.problems || []).map(problem => (
                        <div key={problem} className="ml-2 text-xs text-yellow-800">
                          {problem}
                        </div>
                      ))}
                    </li>
                  ))}
              </ul>
              <p className="mt-2 text-xs text-yellow-800">

                The upgrade will not be released until the pre-requisite components are completed; the directory lacks a version statement and a version line needs to be published in the application management first.
              </p>
            </div>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 gap-4">
        {filteredInstallations.map(installation => {
          const status = statusConfig[installation.status] || statusConfig.installing;
          const runtimeName = connectorRuntimeName(installation);
          const runtimeConfig = connectorConfigByName.get(runtimeName);
          const isFeishu = runtimeName === 'feishu';
          const isEditing = editingId === installation.id;
          const form = forms[installation.id] || makeForm(installation);
          const displayName = installation.itemTitle || installation.itemName || `Application #${installation.itemId}`;
          // 配置面只有连接器有：技能没有租户级配置项，后端会明确拒绝，
          // 摆一个点了必然失败的按钮比不摆更糟。
          const isConnector = installation.itemType === 'connector';
          return (
            <Card key={installation.id} className="hover:shadow-sm transition-shadow">
              <CardHeader className="pb-2">
                <div className="flex flex-col lg:flex-row lg:items-start justify-between gap-4">
                  <div className="flex items-center gap-3">
                    <div className="w-12 h-12 rounded-md overflow-hidden bg-gray-100 flex items-center justify-center">
                      {installation.itemIconUrl ? (
                        <img src={installation.itemIconUrl} alt={displayName} className="w-full h-full object-cover" />
                      ) : (
                        <span className="text-sm">{installation.itemType ? typeNames[installation.itemType] : 'Application'}</span>
                      )}
                    </div>
                    <div>
                      <div className="flex flex-wrap items-center gap-3">
                        <CardTitle className="text-xl">{displayName}</CardTitle>
                        {installation.itemType && <Badge variant="secondary">{typeNames[installation.itemType]}</Badge>}
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
                      <CardDescription className="mt-1">{installation.itemDescription || 'No description'}</CardDescription>
                    </div>
                  </div>
                  {canManageMarketplace && <div className="flex items-center gap-2">
                    {/* 入口恒常渲染：能否切换版本由弹窗里的已发布版本清单决定，
                        不能用「只有点开弹窗才会填充的 upgradeVersions」当门控。 */}
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => void openUpgrade(installation)}
                      disabled={upgradingId === installation.itemId}
                    >
                      <ArrowUpCircle className="h-4 w-4 mr-1" />
                      {upgradingId === installation.itemId
                        ? 'Upgrading...'
                        : installation.upgradeAvailable
                          ? `Upgrade to v ${installation.latestVersion}`
                          : 'Upgrade/Rollback'}
                    </Button>
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => handleToggleAutoUpgrade(installation)}
                      disabled={autoUpgradeId === installation.itemId}
                    >

                      Automatic upgrade:{installation.autoUpgrade ? 'On' : 'Off'}
                    </Button>
                    {isConnector && (
                      <Button variant="secondary" size="sm" onClick={() => setEditingId(isEditing ? null : installation.id)}>
                        <Settings className="h-4 w-4 mr-1" />

                        Configure
                      </Button>
                    )}
                    <Button variant="destructive" size="sm" onClick={() => handleUninstall(installation)}>
                      <Trash2 className="h-4 w-4 mr-1" />

                      Uninstall
                    </Button>
                  </div>}
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
                  <Info label="Installed version" value={`v${installation.installedVersion}`} />
                  <Info label="latest version of directory" value={installation.latestVersion ? `v${installation.latestVersion}` : 'Unknown'} />
                  <Info label="Installation time" value={formatDateTime(installation.installedAt)} />
                  {/* 上次更新：从未更新过的记录用安装时间兜底，lastUpdatedAt 是契约里的可选字段 */}
                  <Info label="Last updated" value={formatDateTime(installation.lastUpdatedAt || installation.installedAt)} />
                  <Info label="Recently used" value={formatDateTime(installation.lastUsedAt) || 'Never Used'} />
                </div>

                {isConnector && (
                  <div className="rounded-md border bg-gray-50 p-3">
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-3 text-sm">
                      <Info label="runtime name" value={runtimeName || 'Not recognized'} />
                      <Info label="Enabled" value={runtimeConfig?.enabled ? 'Enabled' : 'Not Enabled'} />
                      <Info label="Health status" value={runtimeConfig ? (runtimeConfig.healthy ? 'Healthy' : 'To be checked') : 'not initialized'} />
                    </div>
                    <div className="mt-3 grid grid-cols-1 md:grid-cols-3 gap-3 text-sm">
                      {Object.entries((installation.config?.credentials || {}) as Record<string, unknown>).map(([key, value]) => (
                        <Info key={key} label={`Credentials ${key}`} value={maskSensitive(key, value)} />
                      ))}
                      {Object.entries((installation.config?.oauth || {}) as Record<string, unknown>).map(([key, value]) => (
                        <Info key={key} label={`OAuth ${key}`} value={maskSensitive(key, value)} />
                      ))}
                    </div>
                  </div>
                )}

                {canManageMarketplace && isEditing && (
                  <div className="rounded-md border p-4 space-y-4">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <Field label="App ID" value={form.appId} onChange={value => updateForm(installation.id, { appId: value })} />
                      <Field label="App Secret" type="password" value={form.appSecret} onChange={value => updateForm(installation.id, { appSecret: value })} />
                      <Field label="Verification Token" type="password" value={form.verificationToken} onChange={value => updateForm(installation.id, { verificationToken: value })} />
                      <Field label="Encrypt Key" type="password" value={form.encryptKey} onChange={value => updateForm(installation.id, { encryptKey: value })} />
                      <Field label="Debug Channel" value={form.debugChannel} onChange={value => updateForm(installation.id, { debugChannel: value })} />
                      <div>
                        <div className="text-sm font-medium mb-1">Area</div>
                        <Select value={form.region} onValueChange={value => updateForm(installation.id, { region: value })}>
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="cn">Mainland China</SelectItem>
                            <SelectItem value="intl">Overseas Lark</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <Button onClick={() => handleSaveConfig(installation)} disabled={savingId === installation.id}>
                        <Save className="h-4 w-4 mr-2" />
                        {savingId === installation.id ? 'Saving...' : 'Save Configuration'}
                      </Button>
                      {isFeishu && (
                        <Button variant="secondary" onClick={handleFeishuOAuth}>
                          <ExternalLink className="h-4 w-4 mr-2" />

                          Feishu OAuth authorization
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
            <h3 className="text-lg font-medium text-gray-900 mb-2">No matching application found</h3>
            <p className="text-gray-500 mb-4">Try adjusting search criteria or filtering options</p>
            <Link href="/marketplace">
              <Button>Browse the App Market</Button>
            </Link>
          </div>
        </div>
      )}

      {/* 升级/回滚弹窗：可选目标版本 + 二次确认写明版本迁移。
          换版本会重新装载运行态（可能重启连接器/技能），属于不可忽略的写操作，
          必须先让操作人看清「从哪个版本到哪个版本」再执行。 */}
      <Modal
        open={upgradeTarget !== null}
        title={`Upgrade/Rollback:${upgradeTarget?.itemTitle || upgradeTarget?.itemName || ''}`}
        onCancel={() => setUpgradeTarget(null)}
        footer={null}
        destroyOnHidden
      >
        {upgradeTarget && (
          <div className="space-y-4">
            <div className="text-sm text-gray-600">

              Currently installed version:
              <span className="font-medium text-gray-900">v{upgradeTarget.installedVersion}</span>
            </div>

            <div>
              <div className="text-sm font-medium mb-1">target version</div>
              <Select
                value={upgradeVersion || undefined}
                onValueChange={setUpgradeVersion}
                disabled={upgradeVersionsLoading || switchableVersions.length === 0}
                placeholder="Select target version"
              >
                {upgradeVersions.length === 0 ? (
                  <SelectItem value="__no_published_version__" disabled>

                    No other released version yet
                  </SelectItem>
                ) : (
                  upgradeVersions.map(v => (
                    <SelectItem
                      key={v.version}
                      value={v.version}
                      disabled={v.version === upgradeTarget.installedVersion}
                    >
                      v{v.version}
                      {v.version === upgradeTarget.installedVersion ? '(current version)' : ''}
                    </SelectItem>
                  ))
                )}
              </Select>
              {upgradeVersionsLoading && (
                <div className="text-xs text-gray-500 mt-1">Loading available versions…</div>
              )}
              {upgradeVersionsError && (
                <div className="text-xs text-red-600 mt-1">{upgradeVersionsError}</div>
              )}
              {!upgradeVersionsLoading && !upgradeVersionsError && switchableVersions.length === 0 && (
                <div className="text-xs text-gray-500 mt-1">

                  This product currently only has v{upgradeTarget.installedVersion}  A released version, there is currently no target version for upgrade or rollback.
                </div>
              )}
            </div>

            {upgradeVersion ? (
              <div className="text-sm">

                Will be from
                <span className="font-medium"> v{upgradeTarget.installedVersion} </span>

                changed to
                <span className="font-medium"> v{upgradeVersion} </span>

                . An upgrade will reload the running state, and a rollback will return the old publication.
              </div>
            ) : (
              <div className="text-sm text-gray-500">

                There is currently no target version to switch to. Please publish other versions in the application management first and then try again.
              </div>
            )}

            <div className="flex justify-end gap-2">
              <Button variant="secondary" onClick={() => setUpgradeTarget(null)}>

                Cancel
              </Button>
              <Button
                onClick={() => void handleUpgrade(upgradeTarget, upgradeVersion)}
                disabled={
                  upgradingId === upgradeTarget.itemId ||
                  !upgradeVersion ||
                  upgradeVersion === upgradeTarget.installedVersion
                }
              >
                {upgradingId === upgradeTarget.itemId ? 'Executing...' : 'Confirm execution'}
              </Button>
            </div>
          </div>
        )}
      </Modal>
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
