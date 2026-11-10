'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { App } from 'antd';
import {
  AlertCircle,
  ArrowLeft,
  Calendar,
  CheckCircle2,
  Code,
  Download,
  Globe,
  Shield,
  Star,
  Tag,
  Terminal,
  User,
} from 'lucide-react';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/Card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/Tabs';
import type {
  MarketplaceItem,
  TenantInstallation,
} from '@/lib/services/marketplace-service';
import { serverErrorMessage } from '@/lib/api/http-client';
import marketplaceService from '@/lib/services/marketplace-service';
import { useCapabilities } from '@/lib/hooks/useCapabilities';
import { usePermissions } from '@/lib/hooks/use-permissions';
import {
  MARKETPLACE_TYPE_LABELS,
  marketplaceCategoryLabel,
  marketplaceTagLabel,
} from '@/lib/marketplaceLabels';
import { formatDateTime } from '@/lib/formatters';

const typeNames = MARKETPLACE_TYPE_LABELS;

const formatDate = (value?: string) => formatDateTime(value) || 'None yet';

const normalizeMarkdown = (text?: string) =>
  (text || 'No detailed introduction yet').split('\n').filter(line => line.trim().length > 0);

const MarketplaceDetailPage = () => {
  const params = useParams();
  const router = useRouter();
  const itemId = Number(params.id);
  const { message, modal } = App.useApp();
  const { allows, find: findCapability } = useCapabilities();
  const { hasPermission } = usePermissions();
  // 能力矩阵的 manage 只区分 admin-like 角色（部门经理/审计员也算），
  // 真正的写权限以 RBAC 为准：没有 marketplace:write 时不展示会 403 的安装入口。
  const canManageMarketplace =
    allows('marketplace', 'manage') && hasPermission('marketplace', 'write');

  const [item, setItem] = useState<MarketplaceItem | null>(null);
  const [installation, setInstallation] = useState<TenantInstallation | null>(null);
  const [loading, setLoading] = useState(true);
  const [installing, setInstalling] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const versions = useMemo(() => item?.versions ?? [], [item]);
  const isInstalled = Boolean(installation);
  // 能否安装只由服务端的交付判定决定（连接器/技能为真）。界面不得另写一份 type 白名单：
  // 目录放开新类型时，写死的白名单会把新入口静默拦掉，而 501 只该留给真正不交付的类型。
  const installSupported = Boolean(item?.deliverable);

  // NaN 守卫：验证 itemId 有效性
  useEffect(() => {
    if (!Number.isInteger(itemId) || itemId <= 0) {
      setLoadError('Invalid project ID');
      setLoading(false);
    }
  }, [itemId]);

  useEffect(() => {
    // 如果 ID 无效，不发起请求
    if (!Number.isInteger(itemId) || itemId <= 0) return;

    let cancelled = false;
    const load = async () => {
      setLoading(true);
      setLoadError(null);
      try {
        const [itemRes, installationRes] = await Promise.all([
          marketplaceService.getItem(itemId),
          marketplaceService.getInstallation(itemId),
        ]);
        if (cancelled) return;
        setItem(itemRes);
        setInstallation(installationRes);
      } catch (error: unknown) {
        if (cancelled) return;
        console.error('Failed to fetch marketplace item detail:', error);
        setLoadError(error instanceof Error ? error.message : 'Failed to load application details');
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    if (Number.isFinite(itemId)) {
      load();
    }
    return () => {
      cancelled = true;
    };
  }, [itemId]);

  const handleInstall = async () => {
    setInstalling(true);
    try {
      const installed = await marketplaceService.installItem(itemId);
      setInstallation(installed);
      // 安装接口返回的是「安装记录 + 运行态是否就绪」两件事。连接器没起来时
      // 记录是 failed（不是 4xx），必须如实提示去补配置，不能再报「安装成功」。
      if (installed.status === 'failed') {
        message.warning(
          installed.errorMessage
            ? `Installation not completed:${installed.errorMessage}`
            : 'The installation is not completed. Please complete the configuration in the installation list and try again.',
        );
      } else {
        message.success('Installation successful');
      }
      router.push('/installations');
    } catch (error) {
      console.error('Failed to install item:', error);
      message.error(serverErrorMessage(error, 'Installation failed'));
    } finally {
      setInstalling(false);
    }
  };

  const handleUninstall = async () => {
    if (!item) return;
    const confirmed = await modal.confirm({
      title: 'Confirm uninstallation',
      content: `Confirm to uninstall "${item.title}"?" Relevant functions will not be available after uninstallation.`,
      okText: 'Confirm',
      cancelText: 'Cancel',
    });
    if (!confirmed) return;
    try {
      await marketplaceService.uninstallItem(itemId);
      setInstallation(null);
      message.success('Uninstall successful');
    } catch (error) {
      console.error('Failed to uninstall item:', error);
      message.error(serverErrorMessage(error, 'Uninstall failed'));
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    );
  }

  if (loadError || !item) {
    return (
      <div className="container mx-auto px-4 py-6">
        <div className="bg-white rounded-lg shadow-sm p-10 text-center">
          <AlertCircle className="h-12 w-12 text-red-400 mb-4 mx-auto" />
          <h3 className="text-lg font-medium text-gray-900 mb-2">Application does not exist</h3>
          <p className="text-gray-500 mb-4">{loadError || 'The application you visited may have been removed or deleted'}</p>
          <Link href="/marketplace">
            <Button>Return to App Market</Button>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto px-4 py-6">
      <div className="flex items-center gap-2 mb-6 text-sm">
        <Link href="/marketplace" className="text-gray-500 hover:text-gray-700 flex items-center gap-1">
          <ArrowLeft className="h-4 w-4" />

          Return to market
        </Link>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader>
              <div className="flex flex-col md:flex-row md:items-start gap-4">
                <div className="w-16 h-16 rounded-md overflow-hidden bg-gray-100 flex items-center justify-center">
                  {item.iconUrl ? (
                    <img src={item.iconUrl} alt={item.title} className="w-full h-full object-cover" />
                  ) : (
                    <span className="text-sm">{typeNames[item.type]}</span>
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex flex-wrap items-center gap-3 mb-1">
                    <h1 className="text-2xl font-bold">{item.title}</h1>
                    <Badge variant="secondary">{typeNames[item.type]}</Badge>
                    {item.isOfficial && <Badge className="bg-blue-100 text-blue-800 hover:bg-blue-200">Official</Badge>}
                    {item.isFree ? (
                      <Badge className="bg-green-100 text-green-800 hover:bg-green-200">Free</Badge>
                    ) : (
                      <Badge className="bg-orange-100 text-orange-800 hover:bg-orange-200">¥{item.price || 0}</Badge>
                    )}
                  </div>
                  <p className="text-gray-600">{item.description || 'No description'}</p>
                  <div className="flex flex-wrap items-center gap-4 mt-3 text-sm text-gray-500">
                    {(item.rating || 0) > 0 && (
                      <span className="flex items-center gap-1">
                        <Star className="h-4 w-4 fill-yellow-400 text-yellow-400" />
                        {(item.rating || 0).toFixed(1)}
                      </span>
                    )}
                    {(item.installCount || 0) > 0 && (
                      <span className="flex items-center gap-1">
                        <Download className="h-4 w-4" />
                        {item.installCount}  Installation
                      </span>
                    )}
                    <span className="flex items-center gap-1">
                      <User className="h-4 w-4" />

                      Author:{item.authorName || item.provider}
                    </span>
                    <span className="flex items-center gap-1">
                      <Calendar className="h-4 w-4" />

                      updated on {formatDate(item.updatedAt)}
                    </span>
                  </div>
                </div>
                <div className="md:text-right">
                  {!canManageMarketplace ? (
                    <div className="max-w-52 text-left">
                      <Badge className="bg-yellow-100 text-yellow-800">Preview · Read only</Badge>
                      <p className="text-xs text-gray-500 mt-2">
                        {findCapability('marketplace')?.degradedReason || 'Only tenant administrators can install, uninstall, and configure extensions.'}
                      </p>
                    </div>
                  ) : isInstalled ? (
                    <div className="space-y-2">
                      <div className="flex items-center md:justify-end gap-1 text-green-600">
                        <CheckCircle2 className="h-5 w-5" />
                        <span>Already installed</span>
                      </div>
                      <div className="flex gap-2">
                        <Button onClick={() => router.push('/installations')}>Admin</Button>
                        <Button variant="destructive" onClick={handleUninstall}>Uninstall</Button>
                      </div>
                    </div>
                  ) : !installSupported ? (
                    <div className="max-w-52 text-left">
                      <Button size="lg" disabled>The platform does not deliver this type</Button>
                      <p className="text-xs text-gray-500 mt-2">
                        {typeNames[item.type]}Class extension requires an external host runtime, and the platform does not host third-party code; historical entries can only be viewed and uninstalled.
                      </p>
                    </div>
                  ) : (
                    <Button size="lg" onClick={handleInstall} disabled={installing}>
                      {installing ? 'Installing...' : 'Install now'}
                    </Button>
                  )}
                  <p className="text-xs text-gray-500 mt-2">Version {item.latestVersion || '1.0.0'}</p>
                </div>
              </div>
            </CardHeader>
          </Card>

          <Card>
            <Tabs defaultValue="description">
              <TabsList>
                <TabsTrigger value="description">Detailed introduction</TabsTrigger>
                <TabsTrigger value="capabilities">Features</TabsTrigger>
                <TabsTrigger value="permissions">Permission Description</TabsTrigger>
                <TabsTrigger value="versions">Version History</TabsTrigger>
              </TabsList>
                <TabsContent value="description" className="mt-0 space-y-4">
                  <div className="prose prose-sm max-w-none">
                    {normalizeMarkdown(item.longDescription || item.description).map((line, index) => (
                      <p key={index}>{line}</p>
                    ))}
                  </div>
                  {(item.tags || []).length > 0 && (
                    <div>
                      <h3 className="text-lg font-medium mb-3">Related tags</h3>
                      <div className="flex flex-wrap gap-2">
                        {(item.tags || []).map(tag => (
                          <Badge key={tag} variant="outline">
                            <Tag className="h-3 w-3 mr-1" />
                            {marketplaceTagLabel(tag)}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  )}
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <InfoLink icon={<Globe className="h-5 w-5 text-gray-400 mt-0.5" />} label="Official website" value={item.homepage} />
                    <InfoLink icon={<Code className="h-5 w-5 text-gray-400 mt-0.5" />} label="Code Repository" value={item.repository} />
                    <InfoText icon={<Shield className="h-5 w-5 text-gray-400 mt-0.5" />} label="Open source protocol" value={item.license || 'Not declared'} />
                    <InfoText icon={<Terminal className="h-5 w-5 text-gray-400 mt-0.5" />} label="Minimum system version" value={item.minSystemVersion || 'Not declared'} />
                  </div>
                </TabsContent>

                <TabsContent value="capabilities" className="mt-0">
                  <h3 className="text-lg font-medium mb-3">Supported features</h3>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    {(item.capabilities || []).map(capability => (
                      <div key={capability} className="flex items-center gap-2 p-2 bg-gray-50 rounded-md">
                        <CheckCircle2 className="h-5 w-5 text-green-500 flex-shrink-0" />
                        <span>{capability}</span>
                      </div>
                    ))}
                  </div>
                </TabsContent>

                <TabsContent value="permissions" className="mt-0">
                  <h3 className="text-lg font-medium mb-3">System permissions required</h3>
                  <div className="space-y-3">
                    {(item.requiredPermissions || []).map(permission => (
                      <div key={permission} className="flex items-start gap-2 p-2 bg-gray-50 rounded-md">
                        <Shield className="h-5 w-5 text-blue-500 flex-shrink-0 mt-0.5" />
                        <span>{permission}</span>
                      </div>
                    ))}
                    {(item.requiredPermissions || []).length === 0 && <p className="text-sm text-gray-500">No additional permission description yet</p>}
                  </div>
                </TabsContent>

                <TabsContent value="versions" className="mt-0">
                  <h3 className="text-lg font-medium mb-3">Version History</h3>
                  <div className="space-y-4">
                    {versions.length > 0 ? versions.map(version => (
                      <div key={version.version} className="border-l-2 border-gray-200 pl-4 pb-4">
                        <div className="flex items-center justify-between mb-1">
                          <div className="font-medium">v{version.version}</div>
                          <div className="text-sm text-gray-500">{formatDate(version.releasedAt)}</div>
                        </div>
                        <p className="text-sm text-gray-600 whitespace-pre-line">{version.changelog || 'No update instructions yet'}</p>
                      </div>
                    )) : <p className="text-sm text-gray-500">No version history yet</p>}
                  </div>
                </TabsContent>
            </Tabs>
          </Card>
        </div>

        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Installation information</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <InfoRow label="latest version" value={item.latestVersion || '1.0.0'} />
              <InfoRow label="Release date" value={formatDate(versions[0]?.releasedAt)} />
              <InfoRow label="Minimum system requirements" value={item.minSystemVersion || 'Not declared'} />
              {(item.installCount || 0) > 0 && (
                <InfoRow label="Installation volume" value={String(item.installCount)} />
              )}
              <div>
                <div className="text-sm font-medium mb-1">Category</div>
                <Badge variant="secondary">{marketplaceCategoryLabel(item.category)}</Badge>
              </div>
            </CardContent>
            <CardFooter className="border-t pt-4">
              {!canManageMarketplace ? (
                <Button className="w-full" size="lg" disabled>

                  Only administrators can install
                </Button>
              ) : isInstalled ? (
                <Button className="w-full" onClick={() => router.push('/installations')}>Manage installed applications</Button>
              ) : !installSupported ? (
                <Button className="w-full" size="lg" disabled>

                  The platform does not deliver this type
                </Button>
              ) : (
                <Button className="w-full" size="lg" onClick={handleInstall} disabled={installing}>
                  {installing ? 'Installing...' : 'Install now'}
                </Button>
              )}
            </CardFooter>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Provider information</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 rounded-full bg-blue-100 flex items-center justify-center text-blue-600 font-medium">
                  {(item.provider || item.authorName || item.title).charAt(0)}
                </div>
                <div>
                  <div className="font-medium">{item.provider || item.authorName || 'Unknown provider'}</div>
                  {item.isOfficial && <Badge className="mt-1 bg-blue-100 text-blue-800 hover:bg-blue-200">Official certification</Badge>}
                </div>
              </div>
              <p className="text-sm text-gray-600">
                {item.isOfficial ? 'Developed and maintained by the ITSM official team to ensure compatibility and security.' : 'Contributed by community developers and verified by official review.'}
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
};

const InfoRow = ({ label, value }: { label: string; value: string }) => (
  <div>
    <div className="text-sm font-medium mb-1">{label}</div>
    <div className="text-gray-600">{value}</div>
  </div>
);

const InfoText = ({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) => (
  <div className="flex items-start gap-2">
    {icon}
    <div>
      <div className="text-sm font-medium">{label}</div>
      <div className="text-sm text-gray-600 break-all">{value}</div>
    </div>
  </div>
);

const InfoLink = ({ icon, label, value }: { icon: React.ReactNode; label: string; value?: string }) => (
  <div className="flex items-start gap-2">
    {icon}
    <div>
      <div className="text-sm font-medium">{label}</div>
      {value ? (
        <a href={value} target="_blank" rel="noopener noreferrer" className="text-sm text-blue-600 hover:underline break-all">
          {value}
        </a>
      ) : (
        <div className="text-sm text-gray-600">Not provided</div>
      )}
    </div>
  </div>
);

export default MarketplaceDetailPage;
